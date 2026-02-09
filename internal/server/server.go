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
	"sync/atomic"
	"time"

	"sidelight/internal/ai"
	"sidelight/internal/app"
	"sidelight/internal/extractor"
)

//go:embed static/*
var staticFiles embed.FS

// requestCounter generates unique request IDs for log tracing.
var requestCounter uint64

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

// clientIP extracts the real client IP, respecting Cloudflare/proxy headers.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return r.RemoteAddr
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	fsys, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to load static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(fsys)))
	mux.HandleFunc("/api/grade", s.handleGrade)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       10 * time.Minute, // large RAW uploads
		WriteTimeout:      10 * time.Minute, // AI processing + RT rendering
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB headers
	}

	log.Printf("[INFO] server starting on :%d (temp=%s, keep=%v)", s.port, s.tempDir, s.keepTemp)
	return srv.ListenAndServe()
}

// sendRTFallback extracts the original preview and returns it with PP3 content.
// Returns true if the fallback succeeded and the response was written.
func (s *Server) sendRTFallback(ctx context.Context, w http.ResponseWriter, reqID uint64, tempPath, pp3Path string) bool {
	previewData, err := extractor.NewExifToolExtractor().ExtractPreview(ctx, tempPath)
	if err != nil {
		return false
	}
	pp3Content, _ := os.ReadFile(pp3Path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"format":      "rt",
		"image_data":  base64.StdEncoding.EncodeToString(previewData),
		"pp3_content": string(pp3Content),
	})
	log.Printf("[INFO] req#%d fallback preview sent", reqID)
	return true
}

func (s *Server) handleGrade(w http.ResponseWriter, r *http.Request) {
	reqID := atomic.AddUint64(&requestCounter, 1)
	start := time.Now()
	ip := clientIP(r)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse upload (200MB limit for large RAW files)
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		log.Printf("[INFO] req#%d parse form failed from %s: %v", reqID, ip, err)
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("[INFO] req#%d no image in form from %s: %v", reqID, ip, err)
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
		format = "rt"
	}

	log.Printf("[INFO] req#%d from %s file=%s size=%d style=%s format=%s",
		reqID, ip, header.Filename, header.Size, style, format)

	// 2. Save uploaded file to temp dir
	var tempDir string
	if s.tempDir != "" {
		tempDir = filepath.Join(s.tempDir, fmt.Sprintf("sidelight-web-%d", time.Now().UnixNano()))
		err = os.MkdirAll(tempDir, 0755)
	} else {
		tempDir, err = os.MkdirTemp("", "sidelight-web-*")
	}
	if err != nil {
		log.Printf("[INFO] req#%d mkdir failed: %v", reqID, err)
		http.Error(w, "Server error: Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	if !s.keepTemp {
		defer os.RemoveAll(tempDir)
	}

	tempPath := filepath.Join(tempDir, header.Filename)
	out, err := os.Create(tempPath)
	if err != nil {
		log.Printf("[INFO] req#%d create temp file failed: %v", reqID, err)
		http.Error(w, "Server error: Failed to create temp file", http.StatusInternalServerError)
		return
	}

	written, err := io.Copy(out, file)
	out.Close()
	if err != nil {
		log.Printf("[INFO] req#%d save file failed: %v", reqID, err)
		http.Error(w, "Server error: Failed to save file", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] req#%d saved %d bytes to %s", reqID, written, tempDir)

	// 3. AI processing (5 min timeout for large files)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	opts := app.ProcessOptions{
		AnalysisOptions: ai.AnalysisOptions{
			Style:      style,
			UserPrompt: prompt,
		},
	}

	baseName := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))

	if format == "xmp" {
		s.processor.Formats = []string{"xmp"}
		if _, err = s.processor.ProcessFile(ctx, tempPath, opts); err != nil {
			log.Printf("[INFO] req#%d xmp processing failed: %v", reqID, err)
			http.Error(w, "XMP Processing failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		xmpPath := filepath.Join(tempDir, baseName+".xmp")
		xmpContent, err := os.ReadFile(xmpPath)
		if err != nil {
			log.Printf("[INFO] req#%d read xmp failed: %v", reqID, err)
			http.Error(w, "Failed to read XMP configuration", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"format":  "xmp",
			"content": string(xmpContent),
		})
		log.Printf("[INFO] req#%d xmp done in %s", reqID, time.Since(start))
		return
	}

	// RT/PP3 mode
	s.processor.Formats = []string{"pp3"}
	if _, err = s.processor.ProcessFile(ctx, tempPath, opts); err != nil {
		log.Printf("[INFO] req#%d pp3 processing failed: %v", reqID, err)
		http.Error(w, "PP3 Processing failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pp3Path := filepath.Join(tempDir, baseName+".pp3")
	if _, err := os.Stat(pp3Path); os.IsNotExist(err) {
		log.Printf("[INFO] req#%d pp3 file not created at %s", reqID, pp3Path)
		http.Error(w, "PP3 file was not generated", http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] req#%d pp3 generated", reqID)

	// 4. Render preview using RT CLI
	rtPath := os.Getenv("RT_CLI_PATH")
	if rtPath == "" {
		log.Printf("[INFO] req#%d RT_CLI_PATH not set, using fallback preview", reqID)
		if s.sendRTFallback(ctx, w, reqID, tempPath, pp3Path) {
			return
		}
		http.Error(w, "RT_CLI_PATH not set and preview extraction failed", http.StatusServiceUnavailable)
		return
	}

	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		log.Printf("[INFO] req#%d RT CLI not found at %s, using fallback", reqID, rtPath)
		if s.sendRTFallback(ctx, w, reqID, tempPath, pp3Path) {
			return
		}
		http.Error(w, "RT CLI not found and preview extraction failed", http.StatusServiceUnavailable)
		return
	}

	outputPath := filepath.Join(tempDir, "output_preview.jpg")
	cmd := exec.CommandContext(ctx, rtPath, "-o", outputPath, "-p", pp3Path, "-Y", "-c", tempPath)
	cmd.Env = append(os.Environ(), "TMPDIR="+tempDir)

	if cmdOut, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[INFO] req#%d RT CLI failed: %v output=%s", reqID, err, string(cmdOut))
		if s.sendRTFallback(ctx, w, reqID, tempPath, pp3Path) {
			return
		}
		http.Error(w, fmt.Sprintf("Rendering failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// 5. Return rendered image + PP3 config
	imageData, err := os.ReadFile(outputPath)
	if err != nil {
		log.Printf("[INFO] req#%d read rendered image failed: %v", reqID, err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	pp3Content, _ := os.ReadFile(pp3Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"format":      "rt",
		"image_data":  base64.StdEncoding.EncodeToString(imageData),
		"pp3_content": string(pp3Content),
	})
	log.Printf("[INFO] req#%d done in %s (rt rendered)", reqID, time.Since(start))
}
