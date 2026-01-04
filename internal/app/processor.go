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

// ProcessFile handles a single photo file (RAW or standard).
func (p *Processor) ProcessFile(ctx context.Context, rawPath string, opts ai.AnalysisOptions) (*models.ProcessingResult, error) {
	result := &models.ProcessingResult{
		SourcePath: rawPath,
	}

	// 1. Extract Preview
	previewData, err := p.extractor.ExtractPreview(ctx, rawPath)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// 1.5 Extract Metadata
	metadata, err := p.extractor.ExtractMetadata(ctx, rawPath)
	if err != nil {
		// Log warning but continue? Or fail?
		// For now, let's just log it and proceed with empty metadata if possible,
		// but since it returns error, we should probably fail or handle it.
		// However, ExtractMetadata is robust. If it fails, maybe exiftool is missing or file is bad.
		return nil, fmt.Errorf("metadata extraction failed: %w", err)
	}
	result.Metadata = *metadata

	// 2. Analyze with AI
	params, err := p.aiClient.AnalyzeImage(ctx, previewData, *metadata, opts)
	if err != nil {
		return nil, fmt.Errorf("ai analysis failed: %w", err)
	}
	result.Params = *params

	// 3. Map params to XMP settings
	settings := xmp.NewCameraRawSettings()

	// Special handling for non-RAW files (JPG/PNG)
	// AI generates absolute Kelvin values (e.g., 5000) for Temperature.
	// For non-RAW files, XMP expects relative values (-100 to +100).
	// Applying 5000 to a JPG causes Lightroom to crash or the image to disappear.
	// Since we can't easily convert absolute to relative without original metadata,
	// we disable WB adjustments for non-RAW files to ensure safety.
	checkExt := strings.ToLower(strings.TrimSpace(filepath.Ext(rawPath)))
	if checkExt == ".jpg" || checkExt == ".jpeg" || checkExt == ".png" {
		params.Temperature = 0
		params.Tint = 0
		settings.CameraProfile = "Embedded"
	}

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
	settings.LuminanceSmoothing = params.LuminanceSmoothing
	settings.ColorNoiseReduction = params.ColorNoiseReduction

	settings.PostCropVignetteAmount = params.PostCropVignetteAmount

	// HSL - Hue
	settings.HueAdjustmentRed = params.HueAdjustmentRed
	settings.HueAdjustmentOrange = params.HueAdjustmentOrange
	settings.HueAdjustmentYellow = params.HueAdjustmentYellow
	settings.HueAdjustmentGreen = params.HueAdjustmentGreen
	settings.HueAdjustmentAqua = params.HueAdjustmentAqua
	settings.HueAdjustmentBlue = params.HueAdjustmentBlue
	settings.HueAdjustmentPurple = params.HueAdjustmentPurple
	settings.HueAdjustmentMagenta = params.HueAdjustmentMagenta

	// HSL - Saturation
	settings.SaturationAdjustmentRed = params.SaturationAdjustmentRed
	settings.SaturationAdjustmentOrange = params.SaturationAdjustmentOrange
	settings.SaturationAdjustmentYellow = params.SaturationAdjustmentYellow
	settings.SaturationAdjustmentGreen = params.SaturationAdjustmentGreen
	settings.SaturationAdjustmentAqua = params.SaturationAdjustmentAqua
	settings.SaturationAdjustmentBlue = params.SaturationAdjustmentBlue
	settings.SaturationAdjustmentPurple = params.SaturationAdjustmentPurple
	settings.SaturationAdjustmentMagenta = params.SaturationAdjustmentMagenta

	// HSL - Luminance
	settings.LuminanceAdjustmentRed = params.LuminanceAdjustmentRed
	settings.LuminanceAdjustmentOrange = params.LuminanceAdjustmentOrange
	settings.LuminanceAdjustmentYellow = params.LuminanceAdjustmentYellow
	settings.LuminanceAdjustmentGreen = params.LuminanceAdjustmentGreen
	settings.LuminanceAdjustmentAqua = params.LuminanceAdjustmentAqua
	settings.LuminanceAdjustmentBlue = params.LuminanceAdjustmentBlue
	settings.LuminanceAdjustmentPurple = params.LuminanceAdjustmentPurple
	settings.LuminanceAdjustmentMagenta = params.LuminanceAdjustmentMagenta

	// Split Toning
	settings.SplitToningShadowHue = params.SplitToningShadowHue
	settings.SplitToningShadowSaturation = params.SplitToningShadowSaturation
	settings.SplitToningHighlightHue = params.SplitToningHighlightHue
	settings.SplitToningHighlightSaturation = params.SplitToningHighlightSaturation
	settings.SplitToningBalance = params.SplitToningBalance

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

	// 6. For JPG/PNG, we MUST embed the XMP into the file for Lightroom to see it.
	// LR generally ignores sidecars for non-RAW files.
	if checkExt == ".jpg" || checkExt == ".jpeg" || checkExt == ".png" {
		if err := p.extractor.EmbedXMP(ctx, rawPath, xmpPath); err != nil {
			return nil, fmt.Errorf("failed to embed xmp metadata: %w", err)
		}
	}

	return result, nil
}
