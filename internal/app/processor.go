package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sidelight/internal/ai"
	"sidelight/internal/extractor"
	"sidelight/internal/xmp"
	"sidelight/pkg/models"
)

// Processor coordinates the extraction, analysis, and sidecar generation.
type Processor struct {
	extractor extractor.Extractor
	aiClient  ai.Client
}

// NewProcessor creates a new Processor.
func NewProcessor(ext extractor.Extractor, ai ai.Client) *Processor {
	return &Processor{
		extractor: ext,
		aiClient:  ai,
	}
}

// ProcessFile handles a single RAW file.
func (p *Processor) ProcessFile(ctx context.Context, rawPath string, opts ai.AnalysisOptions) (*models.ProcessingResult, error) {
	result := &models.ProcessingResult{
		SourcePath: rawPath,
	}

	// 1. Extract Preview
	previewData, err := p.extractor.ExtractPreview(ctx, rawPath)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// 2. Analyze with AI
	params, err := p.aiClient.AnalyzeImage(ctx, previewData, opts)
	if err != nil {
		return nil, fmt.Errorf("ai analysis failed: %w", err)
	}
	result.Params = *params

	// 3. Map params to XMP settings
	settings := xmp.NewCameraRawSettings()
	settings.Exposure2012 = params.Exposure2012
	settings.Contrast2012 = params.Contrast2012
	settings.Highlights2012 = params.Highlights2012
	settings.Shadows2012 = params.Shadows2012
	settings.Whites2012 = params.Whites2012
	settings.Blacks2012 = params.Blacks2012
	settings.Texture = params.Texture
	settings.Clarity2012 = params.Clarity2012
	settings.Dehaze = params.Dehaze
	settings.Vibrance = params.Vibrance
	settings.Saturation = params.Saturation
	settings.Temperature = params.Temperature
	settings.Tint = params.Tint
	settings.Sharpness = params.Sharpness

	// 4. Marshal XMP
	xmpData, err := xmp.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("xmp marshaling failed: %w", err)
	}

	// 5. Write XMP sidecar
	// Sidecar path: replace extension with .xmp
	ext := filepath.Ext(rawPath)
	xmpPath := strings.TrimSuffix(rawPath, ext) + ".xmp"
	result.XmpPath = xmpPath

	if err := os.WriteFile(xmpPath, xmpData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write xmp file: %w", err)
	}

	return result, nil
}
