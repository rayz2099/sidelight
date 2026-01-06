package main

import (
	"context"
	"fmt"
	"log"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sidelight/internal/ai"
	"sidelight/internal/app"
	"sidelight/internal/extractor"
)

var (
	apiKey      string
	concurrency int
	style       string
	userPrompt  string
)

var gradeCmd = &cobra.Command{
	Use:   "grade [files...]",
	Short: "AI-powered color grading for photos (RAW & Standard)",
	Long:  `Analyze photos (RAW, JPG, PNG) using Gemini AI to generate XMP sidecar files with professional color grading parameters.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runGrade,
}

func init() {
	// Flags specific to the 'grade' command
	gradeCmd.Flags().StringVar(&apiKey, "api-key", "", "Gemini API Key (or set SL_GEMINI_API_KEY env var)")
	gradeCmd.Flags().String("endpoint", "", "Gemini Endpoint URL")
	gradeCmd.Flags().String("model", "", "Gemini Model Name")
	gradeCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 4, "Number of concurrent files to process")
	gradeCmd.Flags().StringVarP(&style, "style", "s", "natural", "Grading style (natural, cinematic, film, bw, portrait)")
	gradeCmd.Flags().StringVarP(&userPrompt, "prompt", "p", "", "Custom instructions (e.g., 'warmer', 'high contrast')")

	// 不使用 BindPFlag，避免命令行参数自动覆盖配置文件
	// 优先级将在 runGrade 中手动控制：配置文件 > 命令行 > 环境变量

	// Env vars - 设置环境变量作为最低优先级的默认值
	viper.SetEnvPrefix("GEMINI")
	viper.BindEnv("gemini_api_key", "SL_GEMINI_API_KEY")
	viper.BindEnv("gemini_endpoint_url", "SL_GEMINI_ENDPOINT_URL")
	viper.BindEnv("gemini_model_name", "SL_GEMINI_MODEL_NAME")
}

// GradeParams 包含执行 grade 操作所需的所有参数
type GradeParams struct {
	Files        []string
	AIClient     ai.Client
	Extractor    extractor.Extractor
	Concurrency  int
	Style        string
	UserPrompt   string
	ShowProgress bool
}

// processGrading 执行实际的图片处理逻辑，可以被测试直接调用
func processGrading(ctx context.Context, params GradeParams) []error {
	processor := app.NewProcessor(params.Extractor, params.AIClient)

	opts := ai.AnalysisOptions{
		Style:      params.Style,
		UserPrompt: params.UserPrompt,
	}

	files := params.Files
	if len(files) == 0 {
		return []error{fmt.Errorf("no files to process")}
	}

	var bar *progressbar.ProgressBar
	if params.ShowProgress {
		bar = progressbar.Default(int64(len(files)))
	}

	jobs := make(chan string, len(files))
	results := make(chan error, len(files))

	// Start workers
	for w := 1; w <= params.Concurrency; w++ {
		go func() {
			for file := range jobs {
				_, err := processor.ProcessFile(ctx, file, opts)
				results <- err
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Collect results
	var errs []error
	for i := 0; i < len(files); i++ {
		err := <-results
		if err != nil {
			errs = append(errs, err)
		}
		if bar != nil {
			bar.Add(1)
		}
	}

	return errs
}

func runGrade(cmd *cobra.Command, args []string) {
	// 配置优先级：配置文件 > 命令行参数 > 环境变量
	// 策略：先从配置文件读取（viper 自动处理），如果命令行有显式设置值则覆盖

	// 读取 API Key
	key := viper.GetString("gemini_api_key")
	if cmd.Flags().Changed("api-key") {
		// 命令行显式设置了值，使用命令行的值
		flagValue, _ := cmd.Flags().GetString("api-key")
		if flagValue != "" {
			key = flagValue
		}
	}

	// 读取 Endpoint
	endpoint := viper.GetString("gemini_endpoint_url")
	if cmd.Flags().Changed("endpoint") {
		// 命令行显式设置了值，使用命令行的值
		flagValue, _ := cmd.Flags().GetString("endpoint")
		if flagValue != "" {
			endpoint = flagValue
		}
	}

	// 读取 Model Name
	modelName := viper.GetString("gemini_model_name")
	if cmd.Flags().Changed("model") {
		// 命令行显式设置了值，使用命令行的值
		flagValue, _ := cmd.Flags().GetString("model")
		if flagValue != "" {
			modelName = flagValue
		}
	}

	if key == "" {
		log.Fatal("API Key is required. Provide it via config file (highest priority), --api-key flag, or SL_GEMINI_API_KEY environment variable.")
	}

	ctx := context.Background()
	ext := extractor.NewExifToolExtractor()
	aiClient, err := ai.NewGeminiClient(ctx, key, endpoint, modelName)
	if err != nil {
		log.Fatalf("Failed to initialize AI client: %v", err)
	}
	defer aiClient.Close()

	files := collectFiles(args)
	if len(files) == 0 {
		log.Fatal("No supported files found to process.")
	}

	params := GradeParams{
		Files:        files,
		AIClient:     aiClient,
		Extractor:    ext,
		Concurrency:  concurrency,
		Style:        style,
		UserPrompt:   userPrompt,
		ShowProgress: true,
	}

	errs := processGrading(ctx, params)

	fmt.Printf("\nFinished grading %d files.\n", len(files))
	if len(errs) > 0 {
		fmt.Printf("Encountered %d errors:\n", len(errs))
		for _, err := range errs {
			fmt.Printf("- %v\n", err)
		}
	}
}
