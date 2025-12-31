package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"sidelight/internal/extractor"
	"sidelight/internal/framer"
)

var (
	frameQuality int
	frameFormat  string
	frameStyle   string
)

var frameCmd = &cobra.Command{
	Use:   "frame [files...]",
	Short: "Add a stylish frame with EXIF data to your photos",
	Args:  cobra.MinimumNArgs(1),
	Run:   runFrame,
}

func init() {
	frameCmd.Flags().IntVarP(&frameQuality, "quality", "q", 100, "Output image quality (0-100) for JPG")
	frameCmd.Flags().StringVarP(&frameFormat, "format", "f", "jpg", "Output format (jpg, png)")
	frameCmd.Flags().StringVarP(&frameStyle, "style", "s", "", "Frame style name (e.g., Modern-Glass, Gallery-Minimal)")
}

func runFrame(cmd *cobra.Command, args []string) {
	files := collectFiles(args)
	if len(files) == 0 {
		log.Fatal("No valid files found to process.")
	}

	ctx := context.Background()
	ext := extractor.NewExifToolExtractor()
	
	// Assume we are running from project root for asset loading, 
	// or try to find where the binary is.
	cwd, _ := os.Getwd()
	fr := framer.NewFramer(cwd)
	
	var config framer.FrameConfig
	var err error

	if frameStyle != "" {
		config, err = fr.LoadStyle(frameStyle)
		if err != nil {
			log.Fatalf("Failed to load style '%s': %v", frameStyle, err)
		}
	} else {
		config = framer.DefaultConfig()
	}

	bar := progressbar.Default(int64(len(files)))
	
	// Simple sequential processing for framing (drawing can be CPU bound, but disk I/O too)
	concurrency := 4 
	jobs := make(chan string, len(files))
	results := make(chan error, len(files))

	for w := 1; w <= concurrency; w++ {
		go func() {
			for path := range jobs {
				results <- processSingleFrame(ctx, path, ext, fr, config)
			}
		}()
	}

	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	var errs []error
	for i := 0; i < len(files); i++ {
		err := <-results
		if err != nil {
			errs = append(errs, err)
		}
		bar.Add(1)
	}

	fmt.Printf("\nFinished framing %d files.\n", len(files))
	if len(errs) > 0 {
		fmt.Printf("Encountered %d errors:\n", len(errs))
		for _, err := range errs {
			fmt.Printf("- %v\n", err)
		}
	}
}

func processSingleFrame(ctx context.Context, path string, ext extractor.Extractor, fr *framer.Framer, config framer.FrameConfig) error {
	// 1. Extract/Load Image Data
	var imgData []byte
	isRaw := isRawExtension(path)

	if isRaw {
		var err error
		imgData, err = ext.ExtractPreview(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to extract preview from %s: %w", filepath.Base(path), err)
		}
	} else {
		// For JPG/PNG, read directly
		var err error
		imgData, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filepath.Base(path), err)
		}
	}

	// 2. Extract Metadata
	// ExifTool works on both RAW and JPG
	meta, err := ext.ExtractMetadata(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to extract metadata from %s: %w", filepath.Base(path), err)
	}

	// 3. Decode Image
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("failed to decode image %s: %w", filepath.Base(path), err)
	}

	// 4. Render Frame
	framedImg, err := fr.Render(img, *meta, config)
	if err != nil {
		return fmt.Errorf("failed to render frame for %s: %w", filepath.Base(path), err)
	}

	// 5. Save
	extStr := filepath.Ext(path)
	// raw-1.ARW -> raw-1_framed.jpg (or .png)
	baseName := strings.TrimSuffix(path, extStr)
	outPath := fmt.Sprintf("%s_framed.%s", baseName, frameFormat)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outPath, err)
	}
	defer f.Close()

	switch strings.ToLower(frameFormat) {
	case "png":
		if err := png.Encode(f, framedImg); err != nil {
			return fmt.Errorf("failed to save png %s: %w", outPath, err)
		}
	case "jpg", "jpeg":
		if err := jpeg.Encode(f, framedImg, &jpeg.Options{Quality: frameQuality}); err != nil {
			return fmt.Errorf("failed to save jpg %s: %w", outPath, err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", frameFormat)
	}

	return nil
}

func isRawExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	rawExts := map[string]bool{
		".arw": true, ".cr2": true, ".cr3": true, 
		".nef": true, ".orf": true, ".dng": true, ".raf": true,
	}
	return rawExts[ext]
}
