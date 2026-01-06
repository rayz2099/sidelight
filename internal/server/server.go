package server

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
}

func NewServer(processor *app.Processor, port int) *Server {
	return &Server{
		processor: processor,
		port:      port,
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

	// 2. Save uploaded file to temp
	tempDir, err := os.MkdirTemp("", "sidelight-web-*")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // Cleanup

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

	// 3. Process image using existing app logic
	// We want to generate a PREVIEW, so we'll use a special flow
	// Currently Processor.ProcessFile writes sidecars.
	// For web preview, we ideally want to apply the look to a small JPG.
	// Since we don't have a Go-based render engine (we rely on Lightroom/RT),
	// we will use the RAW preview extraction + AI grading, 
	// BUT since we can't easily "render" the XMP/PP3 in-memory without external tools like RT CLI,
	// we will try to use the RawTherapee CLI if available to render a JPEG.

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Update processor formats to generate PP3 for rendering
	s.processor.Formats = []string{"pp3"}
	
	_, err = s.processor.ProcessFile(ctx, tempPath, ai.AnalysisOptions{
		Style:      style,
		UserPrompt: prompt,
	})
	if err != nil {
		log.Printf("Processing error: %v", err)
		http.Error(w, "Processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

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
		}
	}

	// Generate output path
	outputPath := filepath.Join(tempDir, "output_preview.jpg")
	
	// Execute RT CLI
	// rawtherapee-cli -o <output> -s -Y -c <input>
	// -s uses the sidecar file (which we just generated alongside the temp file)
	cmd := exec.CommandContext(ctx, rtPath, "-o", outputPath, "-s", "-Y", "-c", tempPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("RT CLI failed: %s\nOutput: %s", err, string(out))
		http.Error(w, "Rendering failed", http.StatusInternalServerError)
		return
	}

	// 5. Serve the rendered image
	http.ServeFile(w, r, outputPath)
}
