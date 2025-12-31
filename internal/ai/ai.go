package ai

import (
	"context"
	"sidelight/pkg/models"
)

// Client defines the interface for AI color grading services.
type Client interface {
	AnalyzeImage(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.GradingParams, error)
}

// AnalysisOptions contains parameters to control the AI analysis.
type AnalysisOptions struct {
	Style      string
	UserPrompt string
}
