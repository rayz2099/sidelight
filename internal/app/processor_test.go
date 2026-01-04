package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sidelight/internal/ai"
	"sidelight/pkg/models"
)

// MockExtractor
type MockExtractor struct{}

func (m *MockExtractor) ExtractPreview(ctx context.Context, rawPath string) ([]byte, error) {
	return []byte("fake-image-data"), nil
}
func (m *MockExtractor) ExtractMetadata(ctx context.Context, rawPath string) (*models.Metadata, error) {
	return &models.Metadata{}, nil
}
func (m *MockExtractor) EmbedXMP(ctx context.Context, imagePath, xmpPath string) error {
	return nil
}

// MockAIClient
type MockAIClient struct{}

func (m *MockAIClient) AnalyzeImage(ctx context.Context, imageData []byte, metadata models.Metadata, opts ai.AnalysisOptions) (*models.GradingParams, error) {
	return &models.GradingParams{
		Exposure2012: 1.0,
		Temperature:  5000, // Should be zeroed out for JPG
		Tint:         50,   // Should be zeroed out for JPG
	}, nil
}

func TestProcessFile_JPG_Logic(t *testing.T) {
	// Setup
	proc := NewProcessor(&MockExtractor{}, &MockAIClient{})
	ctx := context.Background()
	
	// Create a dummy JPG file
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(jpgPath, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run ProcessFile
	res, err := proc.ProcessFile(ctx, jpgPath, ai.AnalysisOptions{})
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Verify XMP content
	xmpContent, err := os.ReadFile(res.XmpPath)
	if err != nil {
		t.Fatal(err)
	}
	
sContent := string(xmpContent)
	
	// Check Temperature is 0 (indicated by attribute absence or explicit 0, but our mock returns 5000)
	// If it was 5000, it would be crs:Temperature="5000"
	// If it is 0, it depends on omitempty? 
	// In CameraRawSettings, Temperature is int `xml:"crs:Temperature,attr,omitempty"`
	// 0 is "empty" for int. So it should be OMITTED.
	// Wait, if it's omitted, that's fine. 
	// But if it was 5000, it would be present.
	
	if strings.Contains(sContent, `crs:Temperature="5000"`) {
		t.Error("XMP should not contain Temperature=5000 for JPG")
	}
	
	// Check CameraProfile="Embedded"
	if !strings.Contains(sContent, `crs:CameraProfile="Embedded"`) {
		t.Error("XMP should contain crs:CameraProfile=\"Embedded\" for JPG")
	}
	
	// Check HasSettings="True"
	if !strings.Contains(sContent, `crs:HasSettings="True"`) {
		t.Error("XMP should contain crs:HasSettings=\"True\"")
	}
}
