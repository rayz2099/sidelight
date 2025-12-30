package extractor_test

import (
	"context"
	"os"
	"testing"

	"sidelight/internal/extractor"
)

func TestExifToolExtractor_ExtractPreview(t *testing.T) {
	// Skip if exiftool is not in PATH
	if _, err := os.Stat("/usr/local/bin/exiftool"); os.IsNotExist(err) {
		// Just a heuristic check, better to check exec.LookPath
	}

	rawPath := "../../images/raw/KAI01144.ARW"
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
