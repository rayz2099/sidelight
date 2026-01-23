package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sidelight/internal/ai"
	"sidelight/internal/extractor"
	"sidelight/internal/rt"
	"sidelight/internal/xmp"
	"sidelight/pkg/models"
)

// Processor coordinates the extraction, analysis, and sidecar generation.
type Processor struct {
	extractor extractor.Extractor
	aiClient  ai.Client
	Formats   []string // e.g., ["xmp", "pp3"]
}

// NewProcessor creates a new Processor.
func NewProcessor(ext extractor.Extractor, ai ai.Client) *Processor {
	return &Processor{
		extractor: ext,
		aiClient:  ai,
		Formats:   []string{"xmp"},
	}
}

// ProcessOptions controls the processing behavior including AI analysis and output paths.
type ProcessOptions struct {
	ai.AnalysisOptions
	OutputDir  string
	OutputFile string
}

// ProcessFile handles a single photo file (RAW or standard).
func (p *Processor) ProcessFile(ctx context.Context, rawPath string, opts ProcessOptions) (*models.ProcessingResult, error) {
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
		return nil, fmt.Errorf("metadata extraction failed: %w", err)
	}
	result.Metadata = *metadata

	// 2. Generate sidecars based on requested formats independently
	// Deduplicate formats to avoid redundant processing
	uniqueFormats := make(map[string]bool)
	for _, f := range p.Formats {
		uniqueFormats[strings.ToLower(f)] = true
	}

	// Handle XMP (Adobe)
	if uniqueFormats["xmp"] {
		params, err := p.aiClient.AnalyzeImageLR(ctx, previewData, *metadata, opts.AnalysisOptions)
		if err != nil {
			return nil, fmt.Errorf("ai analysis (LR) failed: %w", err)
		}
		result.Params = *params

		if err := p.generateXMP(ctx, rawPath, params, result, opts); err != nil {
			return nil, err
		}
	}

	// Handle PP3 (RawTherapee)
	if uniqueFormats["pp3"] || uniqueFormats["rt"] {
		if err := p.generatePP3Native(ctx, rawPath, previewData, metadata, opts, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (p *Processor) resolveOutputPath(rawPath, ext string, opts ProcessOptions) string {
	dir := filepath.Dir(rawPath)
	if opts.OutputDir != "" {
		dir = opts.OutputDir
	}

	// Create directory if it doesn't exist
	if opts.OutputDir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	baseName := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))
	if opts.OutputFile != "" {
		// If OutputFile is specified, use it as the base name
		// If it has an extension, we might want to respect it or strip it depending on usage.
		// For simplicity, if user provides "output.pp3", we use "output" as base.
		customBase := filepath.Base(opts.OutputFile)
		customExt := filepath.Ext(customBase)
		if customExt != "" {
			baseName = strings.TrimSuffix(customBase, customExt)
		} else {
			baseName = customBase
		}
	}

	return filepath.Join(dir, baseName+ext)
}

func (p *Processor) generateXMP(ctx context.Context, rawPath string, params *models.GradingParams, result *models.ProcessingResult, opts ProcessOptions) error {
	settings := xmp.NewCameraRawSettings()

	checkExt := strings.ToLower(strings.TrimSpace(filepath.Ext(rawPath)))
	if checkExt == ".jpg" || checkExt == ".jpeg" || checkExt == ".png" {
		// Non-RAW logic remains the same
		params.Temperature = 0
		params.Tint = 0
		settings.CameraProfile = "Embedded"
	}

	// Mapping logic... (omitted for brevity, will include full content in actual tool call)
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

	// HSL & Split Toning mapping
	settings.HueAdjustmentRed = params.HueAdjustmentRed
	settings.HueAdjustmentOrange = params.HueAdjustmentOrange
	settings.HueAdjustmentYellow = params.HueAdjustmentYellow
	settings.HueAdjustmentGreen = params.HueAdjustmentGreen
	settings.HueAdjustmentAqua = params.HueAdjustmentAqua
	settings.HueAdjustmentBlue = params.HueAdjustmentBlue
	settings.HueAdjustmentPurple = params.HueAdjustmentPurple
	settings.HueAdjustmentMagenta = params.HueAdjustmentMagenta

	settings.SaturationAdjustmentRed = params.SaturationAdjustmentRed
	settings.SaturationAdjustmentOrange = params.SaturationAdjustmentOrange
	settings.SaturationAdjustmentYellow = params.SaturationAdjustmentYellow
	settings.SaturationAdjustmentGreen = params.SaturationAdjustmentGreen
	settings.SaturationAdjustmentAqua = params.SaturationAdjustmentAqua
	settings.SaturationAdjustmentBlue = params.SaturationAdjustmentBlue
	settings.SaturationAdjustmentPurple = params.SaturationAdjustmentPurple
	settings.SaturationAdjustmentMagenta = params.SaturationAdjustmentMagenta

	settings.LuminanceAdjustmentRed = params.LuminanceAdjustmentRed
	settings.LuminanceAdjustmentOrange = params.LuminanceAdjustmentOrange
	settings.LuminanceAdjustmentYellow = params.LuminanceAdjustmentYellow
	settings.LuminanceAdjustmentGreen = params.LuminanceAdjustmentGreen
	settings.LuminanceAdjustmentAqua = params.LuminanceAdjustmentAqua
	settings.LuminanceAdjustmentBlue = params.LuminanceAdjustmentBlue
	settings.LuminanceAdjustmentPurple = params.LuminanceAdjustmentPurple
	settings.LuminanceAdjustmentMagenta = params.LuminanceAdjustmentMagenta

	settings.SplitToningShadowHue = params.SplitToningShadowHue
	settings.SplitToningShadowSaturation = params.SplitToningShadowSaturation
	settings.SplitToningHighlightHue = params.SplitToningHighlightHue
	settings.SplitToningHighlightSaturation = params.SplitToningHighlightSaturation
	settings.SplitToningBalance = params.SplitToningBalance

	xmpData, err := xmp.Marshal(settings)
	if err != nil {
		return fmt.Errorf("xmp marshaling failed: %w", err)
	}

	xmpPath := p.resolveOutputPath(rawPath, ".xmp", opts)
	result.XmpPath = xmpPath

	if err := os.WriteFile(xmpPath, xmpData, 0644); err != nil {
		return fmt.Errorf("failed to write xmp file: %w", err)
	}

	if checkExt == ".jpg" || checkExt == ".jpeg" || checkExt == ".png" {
		if err := p.extractor.EmbedXMP(ctx, rawPath, xmpPath); err != nil {
			return fmt.Errorf("failed to embed xmp metadata: %w", err)
		}
	}
	return nil
}

func (p *Processor) generatePP3(rawPath string, params *models.GradingParams, result *models.ProcessingResult) error {
	pp3Data := rt.GeneratePP3(*params)

	// RawTherapee sidecar path: [filename].pp3
	ext := filepath.Ext(rawPath)
	pp3Path := strings.TrimSuffix(rawPath, ext) + ".pp3"

	if err := os.WriteFile(pp3Path, pp3Data, 0644); err != nil {
		return fmt.Errorf("failed to write pp3 file: %w", err)
	}
	return nil
}

// PP3NativeAnalyzer is an interface for AI clients that support native PP3 generation
type PP3NativeAnalyzer interface {
	AnalyzeImageForPP3(ctx context.Context, imageData []byte, metadata models.Metadata, opts ai.AnalysisOptions) (*models.PP3Params, error)
}

func (p *Processor) generatePP3Native(ctx context.Context, rawPath string, previewData []byte, metadata *models.Metadata, opts ProcessOptions, result *models.ProcessingResult) error {
	// Check if AI client supports native PP3 generation
	nativeAnalyzer, ok := p.aiClient.(PP3NativeAnalyzer)
	if !ok {
		return fmt.Errorf("AI client does not support native PP3 generation")
	}

	// Get PP3 native params from AI
	pp3Params, err := nativeAnalyzer.AnalyzeImageForPP3(ctx, previewData, *metadata, opts.AnalysisOptions)
	if err != nil {
		return fmt.Errorf("PP3 native analysis failed: %w", err)
	}
	result.PP3Params = pp3Params

	// Detect if file is RAW or standard image (JPG/PNG)
	ext := strings.ToLower(filepath.Ext(rawPath))
	isRaw := true
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		isRaw = false
	}

<<<<<<< Updated upstream
	// Generate PP3 file using native params
	pp3Data := rt.GeneratePP3FromNative(pp3Params, isRaw)

	pp3Path := strings.TrimSuffix(rawPath, filepath.Ext(rawPath)) + ".pp3"
||||||| Stash base
	ext := filepath.Ext(rawPath)
	pp3Path := strings.TrimSuffix(rawPath, ext) + ".pp3"
=======
	pp3Path := p.resolveOutputPath(rawPath, ".pp3", opts)
>>>>>>> Stashed changes
	if err := os.WriteFile(pp3Path, pp3Data, 0644); err != nil {
		return fmt.Errorf("failed to write pp3 file: %w", err)
	}
	return nil
}
