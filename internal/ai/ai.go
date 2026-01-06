package ai

import (
	"context"
	"sidelight/pkg/models"
)

// Client defines the interface for AI color grading services.
type Client interface {
	AnalyzeImageLR(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.GradingParams, error)
	// AnalyzeImageForPP3 generates RawTherapee PP3 native parameters directly
	AnalyzeImageForPP3(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.PP3Params, error)
}

// AnalysisOptions contains parameters to control the AI analysis.
type AnalysisOptions struct {
	Style      string
	UserPrompt string
}
