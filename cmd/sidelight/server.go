package main

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sidelight/internal/ai"
	"sidelight/internal/app"
	"sidelight/internal/extractor"
	"sidelight/internal/server"
)

var (
	serverPort int
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web interface",
	Long:  `Start a local web server to use SideLight via a graphical interface in your browser.`,
	Run:   runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 8080, "Port to listen on")
}

func runServer(cmd *cobra.Command, args []string) {
	key := viper.GetString("gemini_api_key")
	if key == "" {
		log.Fatal("Error: GEMINI_API_KEY is not set. Please set it via environment variable or config file.")
	}

	endpoint := viper.GetString("gemini_endpoint_url")
	modelName := viper.GetString("gemini_model_name")

	ctx := context.Background()

	// Initialize dependencies
	aiClient, err := ai.NewGeminiClient(ctx, key, endpoint, modelName)
	if err != nil {
		log.Fatalf("Failed to initialize AI client: %v", err)
	}
	defer aiClient.Close()

	ext := extractor.NewExifToolExtractor()
	processor := app.NewProcessor(ext, aiClient)

	// Start server
	srv := server.NewServer(processor, serverPort)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
