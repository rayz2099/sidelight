package server

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
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
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	processor *app.Processor
	port      int
	tempDir   string
	keepTemp  bool
}

func NewServer(processor *app.Processor, port int, tempDir string, keepTemp bool) *Server {
	return &Server{
		processor: processor,
		port:      port,
		tempDir:   tempDir,
		keepTemp:  keepTemp,
	}
}

func (s *Server) Start() error {
	// Serve static files
	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	http.Handle("/", http.FileServer(http.FS(fsys)))

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
	
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	out.Close()

	// 3. Process image based on format selection
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

	// Generate output path
	outputPath := filepath.Join(tempDir, "output_preview.jpg")
	
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
	}

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
}
