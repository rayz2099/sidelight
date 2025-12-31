package main

import (
	"context"
	"testing"

	"sidelight/internal/ai"
	"sidelight/internal/extractor"
)

// TestProcessGrading 演示如何使用 processGrading 函数进行测试
func TestProcessGrading(t *testing.T) {
	ctx := context.Background()

	// 手动构造 Gemini Client
	apiKey := "sk-46ea1b91490847db83461a03139db840"
	modelName := "gemini-2.5-flash"

	aiClient, err := ai.NewGeminiClient(ctx, apiKey, endpoint, modelName)
	if err != nil {
		t.Fatalf("Failed to create AI client: %v", err)
	}
	defer aiClient.Close()

	// 手动构造 Extractor
	ext := extractor.NewExifToolExtractor()

	// 准备测试参数
	params := GradeParams{
		Files:        []string{"../../images/raw/raw-1.ARW"}, // 替换为实际的测试文件路径
		AIClient:     aiClient,
		Extractor:    ext,
		Concurrency:  2,
		Style:        "natural",
		UserPrompt:   "warm and bright",
		ShowProgress: false, // 测试时关闭进度条
	}

	// 执行处理
	errs := processGrading(ctx, params)

	// 验证结果
	if len(errs) > 0 {
		t.Errorf("Expected no errors, but got %d errors:", len(errs))
		for _, err := range errs {
			t.Logf("  - %v", err)
		}
	}
}

// TestProcessGradingWithMultipleFiles 测试多文件处理
func TestProcessGradingWithMultipleFiles(t *testing.T) {
	t.Skip("Skipping integration test - uncomment to run with real files")

	ctx := context.Background()

	// 手动构造参数
	apiKey := "your-api-key"
	endpoint := ""
	modelName := "gemini-1.5-flash"

	aiClient, err := ai.NewGeminiClient(ctx, apiKey, endpoint, modelName)
	if err != nil {
		t.Fatalf("Failed to create AI client: %v", err)
	}
	defer aiClient.Close()

	ext := extractor.NewExifToolExtractor()

	params := GradeParams{
		Files: []string{
			"../../images/raw/raw-1.ARW",
			"../../images/raw/raw-2.ARW",
		},
		AIClient:     aiClient,
		Extractor:    ext,
		Concurrency:  4,
		Style:        "cinematic",
		UserPrompt:   "",
		ShowProgress: false,
	}

	errs := processGrading(ctx, params)

	// 允许部分失败，只要不是全部失败
	if len(errs) == len(params.Files) {
		t.Errorf("All files failed to process")
	}
}

// TestProcessGradingEmptyFiles 测试空文件列表
func TestProcessGradingEmptyFiles(t *testing.T) {
	ctx := context.Background()

	// 即使没有 AI Client 也可以测试空文件的情况
	params := GradeParams{
		Files:        []string{},
		AIClient:     nil,
		Extractor:    nil,
		Concurrency:  1,
		Style:        "natural",
		UserPrompt:   "",
		ShowProgress: false,
	}

	errs := processGrading(ctx, params)

	if len(errs) != 1 {
		t.Errorf("Expected 1 error for empty files, got %d", len(errs))
	}

	if errs[0].Error() != "no files to process" {
		t.Errorf("Expected 'no files to process' error, got: %v", errs[0])
	}
}
