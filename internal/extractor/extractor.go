package extractor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Extractor defines the interface for extracting previews from RAW files.
type Extractor interface {
	ExtractPreview(ctx context.Context, rawPath string) ([]byte, error)
}

// ExifToolExtractor implements Extractor using the external exiftool command.
type ExifToolExtractor struct {
	// Path to exiftool binary, defaults to "exiftool"
	BinPath string
}

// NewExifToolExtractor creates a new ExifToolExtractor.
func NewExifToolExtractor() *ExifToolExtractor {
	return &ExifToolExtractor{
		BinPath: "exiftool",
	}
}

// ExtractPreview extracts the embedded preview image from a RAW file.
// It uses exiftool to extract the binary data of the PreviewImage or JpgFromRaw.
func (e *ExifToolExtractor) ExtractPreview(ctx context.Context, rawPath string) ([]byte, error) {
	// Attempt to get PreviewImage first, then fallback to JpgFromRaw if needed.
	// -b: output binary data
	// -PreviewImage: specifically target the preview image
	// -JpgFromRaw: target high-res preview if PreviewImage is not what we want
	// We'll try PreviewImage first as it's common.
	
	// Command: exiftool -b -PreviewImage <path>
	cmd := exec.CommandContext(ctx, e.BinPath, "-b", "-PreviewImage", rawPath)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exiftool failed: %w (stderr: %s)", err, stderr.String())
	}

	data := out.Bytes()
	if len(data) == 0 {
		// Fallback to JpgFromRaw
		cmd = exec.CommandContext(ctx, e.BinPath, "-b", "-JpgFromRaw", rawPath)
		out.Reset()
		stderr.Reset()
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("exiftool fallback failed: %w (stderr: %s)", err, stderr.String())
		}
		data = out.Bytes()
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no preview image found in %s", rawPath)
	}

	return data, nil
}
