package extractor_test

import (
	"context"
	"os"
	"testing"

	"sidelight/internal/extractor"
)

// Common setup for raw path
const rawPath = "../../images/raw/raw-1.ARW"

func TestExifToolExtractor_ExtractPreview(t *testing.T) {
	// Skip if exiftool is not in PATH (simple check)
	// In a real env, we assume it's there or skip.
	
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Skipf("RAW file not found at %s, skipping integration test", rawPath)
	}

	ext := extractor.NewExifToolExtractor()
	data, err := ext.ExtractPreview(context.Background(), rawPath)
	if err != nil {
		t.Fatalf("Failed to extract preview: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Extracted data is empty")
	}

	// Basic check for JPEG header
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Errorf("Extracted data does not look like a JPEG (header: %X %X)", data[0], data[1])
	}

	t.Logf("Successfully extracted %d bytes from %s", len(data), rawPath)
}

func TestExifToolExtractor_ExtractMetadata(t *testing.T) {
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		t.Skipf("RAW file not found at %s, skipping integration test", rawPath)
	}

	ext := extractor.NewExifToolExtractor()
	meta, err := ext.ExtractMetadata(context.Background(), rawPath)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	if meta == nil {
		t.Fatal("Returned metadata is nil")
	}

	t.Logf("Extracted Metadata: %+v", meta)

	// Check for some expected fields (assuming raw-1.ARW is a valid raw file)
	// We might not know exact values, but Make/Model/ISO shouldn't be empty/zero usually.
	if meta.Make == "" && meta.Model == "" {
		t.Error("Make and Model are both empty")
	}
	if meta.ISO == 0 {
		t.Log("Warning: ISO is 0, this might be valid but unusual")
	}
}
