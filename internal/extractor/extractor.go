package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sidelight/pkg/models"
)

// Extractor defines the interface for extracting previews from RAW files or reading standard images.
type Extractor interface {
	ExtractPreview(ctx context.Context, rawPath string) ([]byte, error)
	ExtractMetadata(ctx context.Context, rawPath string) (*models.Metadata, error)
	EmbedXMP(ctx context.Context, imagePath, xmpPath string) error
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

// ExtractPreview returns the image data for analysis.
// For standard images (JPG, PNG), it reads the file directly.
// For RAW files, it uses exiftool to extract the embedded preview image.
func (e *ExifToolExtractor) ExtractPreview(ctx context.Context, rawPath string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(rawPath))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		return os.ReadFile(rawPath)
	}

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

type exiftoolOutput struct {
	Make             string      `json:"Make"`
	Model            string      `json:"Model"`
	Lens             string      `json:"Lens"`
	LensID           string      `json:"LensID"`
	LensModel        string      `json:"LensModel"`
	ISO              interface{} `json:"ISO"` // Can be int or string depending on -n
	Aperture         interface{} `json:"Aperture"`
	ShutterSpeed     interface{} `json:"ShutterSpeed"`
	FocalLength      interface{} `json:"FocalLength"`
	DateTimeOriginal string      `json:"DateTimeOriginal"`
}

// ExtractMetadata extracts technical details from the image file.
func (e *ExifToolExtractor) ExtractMetadata(ctx context.Context, rawPath string) (*models.Metadata, error) {
	// -j: JSON output
	// -n: Numerical output for values where appropriate (prevents "1/100" string for shutter speed if it can be decimal, but we might want the string for readable context.
	// Actually, for AI context, readable strings like "1/100" are better than 0.01.
	// So we will NOT use -n generally, but maybe for specific calculations.
	// Let's stick to standard formatting (human readable) as it's for LLM context.

	args := []string{
		"-j",
		"-Make",
		"-Model",
		"-Lens",
		"-LensID",
		"-LensModel",
		"-ISO",
		"-Aperture",
		"-ShutterSpeed",
		"-FocalLength",
		"-DateTimeOriginal",
		rawPath,
	}

	cmd := exec.CommandContext(ctx, e.BinPath, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exiftool metadata extraction failed: %w (stderr: %s)", err, stderr.String())
	}

	var outputs []exiftoolOutput
	if err := json.Unmarshal(out.Bytes(), &outputs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exiftool output: %w", err)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no metadata found for %s", rawPath)
	}

	o := outputs[0]

	// Determine best lens string
	lens := o.Lens
	if lens == "" {
		lens = o.LensModel
	}
	if lens == "" {
		lens = o.LensID
	}

	// Helper to stringify interface{} safely
	toString := func(v interface{}) string {
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}

	// Helper to int safely
	toInt := func(v interface{}) int {
		if v == nil {
			return 0
		}
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case string:
			// try parsing if strictly needed, but usually ISO is just a number in JSON if simple
			// but exiftool without -n might give "100" or "Hi 100".
			// For now, let's just fmt.Sscanf if it's a string, or 0.
			var i int
			fmt.Sscanf(val, "%d", &i)
			return i
		default:
			return 0
		}
	}

	return &models.Metadata{
		Make:         o.Make,
		Model:        o.Model,
		Lens:         lens,
		ISO:          toInt(o.ISO),
		Aperture:     toString(o.Aperture),
		ShutterSpeed: toString(o.ShutterSpeed),
		FocalLength:  toString(o.FocalLength),
		DateTime:     o.DateTimeOriginal,
	}, nil
}

// EmbedXMP embeds the XMP metadata from xmpPath into the image at imagePath.
func (e *ExifToolExtractor) EmbedXMP(ctx context.Context, imagePath, xmpPath string) error {
	// Command: exiftool -overwrite_original -tagsfromfile <xmpPath> -all:all <imagePath>
	// This copies all tags from the XMP sidecar into the image file.
	// We use -overwrite_original to avoid creating _original backup files, assuming the user knows this modifies the file.
	args := []string{
		"-overwrite_original",
		"-tagsfromfile", xmpPath,
		"-all:all",
		imagePath,
	}

	cmd := exec.CommandContext(ctx, e.BinPath, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exiftool embed failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
