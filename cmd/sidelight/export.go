package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg" // Support decoding
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sidelight/internal/extractor"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	exportQuality int
	exportFormat  string
)

var exportCmd = &cobra.Command{
	Use:   "export [files...]",
	Short: "Export embedded preview images from RAW files",
	Args:  cobra.MinimumNArgs(1),
	Run:   runExport,
}

func init() {
	exportCmd.Flags().IntVarP(&exportQuality, "quality", "q", 100, "Output image quality (0-100) for JPG")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "jpg", "Output format (jpg, png)")
}

func runExport(cmd *cobra.Command, args []string) {
	files := collectFiles(args)
	if len(files) == 0 {
		log.Fatal("No files found to export.")
	}

	ctx := context.Background()
	ext := extractor.NewExifToolExtractor()

	bar := progressbar.Default(int64(len(files)))

	var rawFiles []string
	for _, f := range files {
		if isRawExtension(f) {
			rawFiles = append(rawFiles, f)
		}
	}

	if len(rawFiles) == 0 {
		log.Println("No RAW files found in the selection. Export command is for extracting previews from RAWs.")
		return
	}

	jobs := make(chan string, len(rawFiles))
	results := make(chan error, len(rawFiles))

	concurrency := 4
	for w := 1; w <= concurrency; w++ {
		go func() {
			for path := range jobs {
				results <- processExport(ctx, path, ext)
			}
		}()
	}

	for _, file := range rawFiles {
		jobs <- file
	}
	close(jobs)

	var errs []error
	for i := 0; i < len(rawFiles); i++ {
		err := <-results
		if err != nil {
			errs = append(errs, err)
		}
		bar.Add(1)
	}

	fmt.Printf("\nFinished exporting %d files.\n", len(rawFiles))
	if len(errs) > 0 {
		fmt.Printf("Encountered %d errors:\n", len(errs))
		for _, err := range errs {
			fmt.Printf("- %v\n", err)
		}
	}
}

func processExport(ctx context.Context, path string, ext extractor.Extractor) error {
	data, err := ext.ExtractPreview(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to extract preview from %s: %w", filepath.Base(path), err)
	}

	// Decode to image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to decode extracted preview: %w", err)
	}

	// Save with specified format and quality
	extStr := filepath.Ext(path)
	baseName := strings.TrimSuffix(path, extStr)
	outPath := fmt.Sprintf("%s.%s", baseName, exportFormat)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	switch strings.ToLower(exportFormat) {
	case "png":
		if err := png.Encode(f, img); err != nil {
			return fmt.Errorf("failed to save png %s: %w", outPath, err)
		}
	case "jpg", "jpeg":
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: exportQuality}); err != nil {
			return fmt.Errorf("failed to save jpg %s: %w", outPath, err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", exportFormat)
	}

	return nil
}
