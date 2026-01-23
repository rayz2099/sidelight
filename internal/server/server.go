package server

import (
	"context"
	"embed"
<<<<<<< Updated upstream
	"encoding/json"
||||||| Stash base
=======
	"encoding/base64"
	"encoding/json"
>>>>>>> Stashed changes
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sidelight/internal/ai"
	"sidelight/internal/app"
	"sidelight/internal/extractor"
	"sidelight/pkg/models"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	processor *app.Processor
	port      int
<<<<<<< Updated upstream
	outputDir string
}

type GradeResponse struct {
	ImageURL string            `json:"image_url"`
	Params   *models.PP3Params `json:"params"`
||||||| Stash base
=======
	tempDir   string
	keepTemp  bool
>>>>>>> Stashed changes
}

<<<<<<< Updated upstream
func NewServer(processor *app.Processor, port int) *Server {
	// Create persistent output directory for serving images
	outDir, err := os.MkdirTemp("", "sidelight-outputs-*")
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Start cleanup routine (delete files older than 30 mins)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			entries, err := os.ReadDir(outDir)
			if err != nil {
				continue
			}
			threshold := time.Now().Add(-30 * time.Minute)
			for _, e := range entries {
				info, err := e.Info()
				if err == nil && info.ModTime().Before(threshold) {
					os.Remove(filepath.Join(outDir, e.Name()))
				}
			}
		}
	}()

||||||| Stash base
func NewServer(processor *app.Processor, port int) *Server {
=======
func NewServer(processor *app.Processor, port int, tempDir string, keepTemp bool) *Server {
>>>>>>> Stashed changes
	return &Server{
		processor: processor,
		port:      port,
<<<<<<< Updated upstream
		outputDir: outDir,
||||||| Stash base
=======
		tempDir:   tempDir,
		keepTemp:  keepTemp,
>>>>>>> Stashed changes
	}
}

func (s *Server) Start() error {
	// Serve static files
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	http.Handle("/", http.FileServer(http.FS(fsys)))
	
	// Serve output images
	http.Handle("/outputs/", http.StripPrefix("/outputs/", http.FileServer(http.Dir(s.outputDir))))

	// API endpoints
	http.HandleFunc("/api/grade", s.handleGrade)

	log.Printf("Starting server on http://localhost:%d", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) handleGrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "No image file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	style := r.FormValue("style")
	if style == "" {
		style = "natural"
	}
	prompt := r.FormValue("prompt")
	format := r.FormValue("format")
	if format == "" {
		format = "rt" // Default to RT mode for backward compatibility
	}

	// 2. Save uploaded file to temp
	var tempDir string
	if s.tempDir != "" {
		tempDir = filepath.Join(s.tempDir, fmt.Sprintf("sidelight-web-%d", time.Now().UnixNano()))
		err = os.MkdirAll(tempDir, 0755)
	} else {
		tempDir, err = os.MkdirTemp("", "sidelight-web-*")
	}
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	// Cleanup unless keepTemp is set
	if !s.keepTemp {
		defer os.RemoveAll(tempDir)
	}

	tempPath := filepath.Join(tempDir, header.Filename)
	out, err := os.Create(tempPath)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	
	if n, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	} else {
		log.Printf("Uploaded file saved to %s (%d bytes)", tempPath, n)
	}
	out.Close()

<<<<<<< Updated upstream
	// 3. Process image using existing app logic
||||||| Stash base
	// 3. Process image using existing app logic
	// We want to generate a PREVIEW, so we'll use a special flow
	// Currently Processor.ProcessFile writes sidecars.
	// For web preview, we ideally want to apply the look to a small JPG.
	// Since we don't have a Go-based render engine (we rely on Lightroom/RT),
	// we will use the RAW preview extraction + AI grading, 
	// BUT since we can't easily "render" the XMP/PP3 in-memory without external tools like RT CLI,
	// we will try to use the RawTherapee CLI if available to render a JPEG.

=======
	// 3. Process image based on format selection
>>>>>>> Stashed changes
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	if format == "xmp" {
		// XMP mode: Generate XMP configuration and return as JSON
		s.processor.Formats = []string{"xmp"}

		_, err = s.processor.ProcessFile(ctx, tempPath, app.ProcessOptions{
			AnalysisOptions: ai.AnalysisOptions{
				Style:      style,
				UserPrompt: prompt,
			},
		})
		if err != nil {
			log.Printf("Processing error: %v", err)
			http.Error(w, "Processing failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Read generated XMP file
		// Sidecar files are created with base filename (without extension)
		baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
		xmpPath := filepath.Join(tempDir, baseName+".xmp")
		xmpContent, err := os.ReadFile(xmpPath)
		if err != nil {
			log.Printf("Failed to read XMP file: %v", err)
			http.Error(w, "Failed to read XMP configuration", http.StatusInternalServerError)
			return
		}

		// Return XMP content as JSON
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"format": "xmp",
			"content": string(xmpContent),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
		}
		return
	}

	// RT mode: Generate PP3 and render preview
	s.processor.Formats = []string{"pp3"}
<<<<<<< Updated upstream
	
	log.Printf("Processing file: %s (Style: %s)", tempPath, style)

	// DEBUG: Verify preview extraction manually first
	debugPreview, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
	if err != nil {
		log.Printf("DEBUG: Manual preview extraction failed: %v", err)
	} else {
		log.Printf("DEBUG: Manual preview extraction success: %d bytes", len(debugPreview))
		if len(debugPreview) > 4 {
			log.Printf("DEBUG: Preview Magic Bytes: %X %X %X %X", debugPreview[0], debugPreview[1], debugPreview[2], debugPreview[3])
		}
	}
	
	result, err := s.processor.ProcessFile(ctx, tempPath, ai.AnalysisOptions{
		Style:      style,
		UserPrompt: prompt,
||||||| Stash base
	
	_, err = s.processor.ProcessFile(ctx, tempPath, ai.AnalysisOptions{
		Style:      style,
		UserPrompt: prompt,
=======

	_, err = s.processor.ProcessFile(ctx, tempPath, app.ProcessOptions{
		AnalysisOptions: ai.AnalysisOptions{
			Style:      style,
			UserPrompt: prompt,
		},
>>>>>>> Stashed changes
	})
	if err != nil {
		log.Printf("Processing error: %v", err)
		http.Error(w, "Processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

<<<<<<< Updated upstream
	// 4. Render preview using RT CLI
	// Priority: PATH > RT_CLI_PATH > Default macOS location
	rtPath, err := exec.LookPath("rawtherapee-cli")
	if err != nil {
		// Not in PATH, try env var
		if envPath := os.Getenv("RT_CLI_PATH"); envPath != "" {
			rtPath = envPath
		} else {
			// Fallback to default macOS location
			rtPath = "/Applications/RawTherapee.app/Contents/MacOS/rawtherapee-cli"
||||||| Stash base
	// 4. Render preview using RT CLI
	// Check for RT CLI
	rtPath := "/Applications/RawTherapee.app/Contents/MacOS/rawtherapee-cli" // Default macOS
	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		// Try generic command or env var
		if envPath := os.Getenv("RT_CLI_PATH"); envPath != "" {
			rtPath = envPath
		} else {
			// Fallback: just return the extracted preview (original) if we can't render
			// This is not ideal but better than error.
			// Actually, let's try to extract the embedded preview from the RAW first.
			previewData, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
			if err == nil {
				w.Header().Set("Content-Type", "image/jpeg")
				w.Write(previewData)
				return
			}
			http.Error(w, "Rendering engine not found (RawTherapee CLI)", http.StatusServiceUnavailable)
			return
=======
	// Verify PP3 file was created
	// Sidecar files are created with base filename (without extension)
	baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
	pp3Path := filepath.Join(tempDir, baseName+".pp3")
	if _, err := os.Stat(pp3Path); os.IsNotExist(err) {
		log.Printf("Error: PP3 file was not created at %s after processing", pp3Path)
		// List files in temp directory for debugging
		if files, err := os.ReadDir(tempDir); err == nil {
			log.Printf("Files in temp directory:")
			for _, file := range files {
				log.Printf("  - %s", file.Name())
			}
		}
		http.Error(w, "PP3 file was not generated", http.StatusInternalServerError)
		return
	}
	log.Printf("PP3 file successfully created: %s", pp3Path)

	// 4. Render preview using RT CLI
	// Only use RT_CLI_PATH environment variable
	rtPath := os.Getenv("RT_CLI_PATH")
	if rtPath == "" {
		log.Printf("RT_CLI_PATH not set, falling back to original preview")
		previewData, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
		if err == nil {
			// Read PP3 configuration if available
			baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
			pp3Path := filepath.Join(tempDir, baseName+".pp3")
			pp3Content, _ := os.ReadFile(pp3Path)

			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"format":       "rt",
				"image_data":   base64.StdEncoding.EncodeToString(previewData),
				"pp3_content":  string(pp3Content),
			}
			json.NewEncoder(w).Encode(response)
			return
>>>>>>> Stashed changes
		}
		http.Error(w, "RT_CLI_PATH not set and preview extraction failed", http.StatusServiceUnavailable)
		return
	}

	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		log.Printf("RT CLI not found at %s, falling back to original preview", rtPath)
		previewData, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
		if err == nil {
			// Read PP3 configuration if available
			baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
			pp3Path := filepath.Join(tempDir, baseName+".pp3")
			pp3Content, _ := os.ReadFile(pp3Path)

			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"format":       "rt",
				"image_data":   base64.StdEncoding.EncodeToString(previewData),
				"pp3_content":  string(pp3Content),
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "RT CLI not found and preview extraction failed", http.StatusServiceUnavailable)
		return
	}

	var imageURL string
	
<<<<<<< Updated upstream
	// Generate unique output filename
	outFilename := fmt.Sprintf("preview_%d.jpg", time.Now().UnixNano())
	outputPath := filepath.Join(s.outputDir, outFilename)

	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		// Fallback: extract preview and save it to outputDir
		previewData, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
		if err == nil {
			if err := os.WriteFile(outputPath, previewData, 0644); err != nil {
				http.Error(w, "Failed to save preview", http.StatusInternalServerError)
				return
			}
			imageURL = "/outputs/" + outFilename
		} else {
			http.Error(w, "Rendering engine not found (RawTherapee CLI)", http.StatusServiceUnavailable)
			return
		}
	} else {
		// Construct expected sidecar path (must match what Processor generated)
		ext := filepath.Ext(tempPath)
		pp3Path := strings.TrimSuffix(tempPath, ext) + ".pp3"

		// Execute RT CLI
		// -o <file> writes strictly to that file (if single input)
		// -j100: JPEG quality 100
		cmd := exec.CommandContext(ctx, rtPath, "-o", outputPath, "-p", pp3Path, "-j100", "-Y", "-c", tempPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("RT CLI failed: %s\nOutput: %s", err, string(out))
			http.Error(w, "Rendering failed", http.StatusInternalServerError)
			return
		}
		
		imageURL = "/outputs/" + outFilename
||||||| Stash base
	// Execute RT CLI
	// rawtherapee-cli -o <output> -s -Y -c <input>
	// -s uses the sidecar file (which we just generated alongside the temp file)
	cmd := exec.CommandContext(ctx, rtPath, "-o", outputPath, "-s", "-Y", "-c", tempPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("RT CLI failed: %s\nOutput: %s", err, string(out))
		http.Error(w, "Rendering failed", http.StatusInternalServerError)
		return
=======
	// Execute RT CLI
	// rawtherapee-cli -o <output> -s -Y -c <input>
	// -s uses the sidecar file (which we just generated alongside the temp file)
	log.Printf("Using RT CLI: %s", rtPath)
	log.Printf("Processing file: %s", tempPath)
	log.Printf("Output will be: %s", outputPath)

	// Check if PP3 file exists (pp3Path already declared above)
	if _, err := os.Stat(pp3Path); os.IsNotExist(err) {
		log.Printf("Warning: PP3 file not found at %s", pp3Path)
	} else {
		log.Printf("PP3 file found: %s", pp3Path)
	}

	cmd := exec.CommandContext(ctx, rtPath, "-o", outputPath, "-p", pp3Path, "-Y", "-c", tempPath)
	cmd.Env = append(os.Environ(), "TMPDIR="+tempDir) // Ensure RT uses our temp dir

	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("RT CLI failed: %s\nCommand: %s\nOutput: %s", err, cmd.String(), string(out))

		// Try fallback to original preview
		log.Printf("Falling back to original preview...")
		previewData, extractErr := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
		if extractErr == nil {
			// Read PP3 configuration if available
			baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
			pp3Path := filepath.Join(tempDir, baseName+".pp3")
			pp3Content, _ := os.ReadFile(pp3Path)

			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"format":       "rt",
				"image_data":   base64.StdEncoding.EncodeToString(previewData),
				"pp3_content":  string(pp3Content),
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		http.Error(w, fmt.Sprintf("Rendering failed: %s", err.Error()), http.StatusInternalServerError)
		return
>>>>>>> Stashed changes
	}

<<<<<<< Updated upstream
	// 5. Construct JSON response
	resp := GradeResponse{
		ImageURL: imageURL,
		Params:   result.PP3Params,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
||||||| Stash base
	// 5. Serve the rendered image
	http.ServeFile(w, r, outputPath)
=======
	log.Printf("RT CLI succeeded, output file: %s", outputPath)

	// 5. Read the rendered image and PP3 configuration
	imageData, err := os.ReadFile(outputPath)
	if err != nil {
		log.Printf("Failed to read rendered image: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Read PP3 configuration (pp3Path already declared above)
	var pp3Content []byte
	pp3Content, err = os.ReadFile(pp3Path)
	if err != nil {
		log.Printf("Failed to read PP3 file: %v", err)
		// Still continue with just the image
		pp3Content = []byte("")
	}

	// Return both image and configuration as JSON response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"format":       "rt",
		"image_data":   base64.StdEncoding.EncodeToString(imageData),
		"pp3_content":  string(pp3Content),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
	}
>>>>>>> Stashed changes
}
