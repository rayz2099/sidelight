package server

import (
	"context"
	"embed"
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
	"sidelight/pkg/models"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	processor *app.Processor
	port      int
	outputDir string
}

type GradeResponse struct {
	ImageURL string            `json:"image_url"`
	Params   *models.PP3Params `json:"params"`
}

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

	return &Server{
		processor: processor,
		port:      port,
		outputDir: outDir,
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
	
	if n, err := io.Copy(out, file); err != nil {
		out.Close()
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	} else {
		log.Printf("Uploaded file saved to %s (%d bytes)", tempPath, n)
	}
	out.Close()

	// 3. Process image using existing app logic
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Update processor formats to generate PP3 for rendering
	s.processor.Formats = []string{"pp3"}
	
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
	})
	if err != nil {
		log.Printf("Processing error: %v", err)
		http.Error(w, "Processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

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
		}
	}

	var imageURL string
	
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
	}

	// 5. Construct JSON response
	resp := GradeResponse{
		ImageURL: imageURL,
		Params:   result.PP3Params,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}
