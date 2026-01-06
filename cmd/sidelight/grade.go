package main

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	formats     []string
)

var gradeCmd = &cobra.Command{
	Use:   "grade [files...]",
	Short: "AI-powered color grading for photos (RAW & Standard)",
	Long:  `Analyze photos (RAW, JPG, PNG) using Gemini AI to generate XMP/PP3 sidecar files with professional color grading parameters.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runGrade,
}

func init() {
	// Flags specific to the 'grade' command
	gradeCmd.Flags().StringVar(&apiKey, "api-key", "", "Gemini API Key (or set GEMINI_API_KEY env var)")
	gradeCmd.Flags().String("endpoint", "", "Gemini Endpoint URL")
	gradeCmd.Flags().String("model", "", "Gemini Model Name")
	gradeCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 4, "Number of concurrent files to process")
	gradeCmd.Flags().StringVarP(&style, "style", "s", "natural", "Grading style (natural, cinematic, film, bw, portrait)")
	gradeCmd.Flags().StringVarP(&userPrompt, "prompt", "p", "", "Custom instructions (e.g., 'warmer', 'high contrast')")
	gradeCmd.Flags().StringSliceVarP(&formats, "format", "f", []string{"xmp"}, "Output formats (xmp, pp3, rt, all)")

	// Env vars - 设置环境变量作为最低优先级的默认值
	viper.SetEnvPrefix("GEMINI")
	viper.BindEnv("gemini_api_key", "GEMINI_API_KEY")
	viper.BindEnv("gemini_endpoint_url", "GEMINI_ENDPOINT_URL")
	viper.BindEnv("gemini_model_name", "GEMINI_MODEL_NAME")
}

// GradeParams 包含执行 grade 操作所需的所有参数
type GradeParams struct {
	Files        []string
	AIClient     ai.Client
	Extractor    extractor.Extractor
	Concurrency  int
	Style        string
	UserPrompt   string
	Formats      []string
	ShowProgress bool
}

// processGrading 执行实际的图片处理逻辑
func processGrading(ctx context.Context, params GradeParams) []error {
	processor := app.NewProcessor(params.Extractor, params.AIClient)
	processor.Formats = params.Formats

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
	var errorsList []error
	for i := 0; i < len(files); i++ {
		err := <-results
		if err != nil {
			errorsList = append(errorsList, err)
		}
		if bar != nil {
			bar.Add(1)
		}
	}

	return errorsList
}

func runGrade(cmd *cobra.Command, args []string) {
	key := viper.GetString("gemini_api_key")
	if cmd.Flags().Changed("api-key") {
		flagValue, _ := cmd.Flags().GetString("api-key")
		if flagValue != "" {
			key = flagValue
		}
	}

	endpoint := viper.GetString("gemini_endpoint_url")
	if cmd.Flags().Changed("endpoint") {
		flagValue, _ := cmd.Flags().GetString("endpoint")
		if flagValue != "" {
			endpoint = flagValue
		}
	}

	modelName := viper.GetString("gemini_model_name")
	if cmd.Flags().Changed("model") {
		flagValue, _ := cmd.Flags().GetString("model")
		if flagValue != "" {
			modelName = flagValue
		}
	}

	if key == "" {
		log.Fatal("API Key is required.")
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

	// Handle "all" format
	finalFormats := formats
	for _, f := range formats {
		if strings.ToLower(f) == "all" {
			finalFormats = []string{"xmp", "pp3"}
			break
		}
	}

	params := GradeParams{
		Files:        files,
		AIClient:     aiClient,
		Extractor:    ext,
		Concurrency:  concurrency,
		Style:        style,
		UserPrompt:   userPrompt,
		Formats:      finalFormats,
		ShowProgress: true,
	}

	allErrs := processGrading(ctx, params)

	fmt.Printf("\nFinished grading %d files.\n", len(files))
	if len(allErrs) > 0 {
		fmt.Printf("Encountered %d errors:\n", len(allErrs))
		for _, e := range allErrs {
			fmt.Printf("- %v\n", e)
		}
	}
}
