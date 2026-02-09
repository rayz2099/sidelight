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
	serverPort    int
	serverTempDir string
	keepTemp      bool
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
	serverCmd.Flags().StringVarP(&serverTempDir, "temp-dir", "t", "", "Temporary directory for file processing (default: system temp)")
	serverCmd.Flags().BoolVarP(&keepTemp, "keep", "k", false, "Keep temporary files after processing")
}

func runServer(cmd *cobra.Command, args []string) {
	key := viper.GetString("gemini_api_key")
	if key == "" {
		log.Fatal("[INFO] gemini_api_key is not set, please set via env or config file")
	}

	endpoint := viper.GetString("gemini_endpoint_url")
	modelName := viper.GetString("gemini_model_name")

	log.Printf("[INFO] initializing ai client (model=%s)", modelName)

	ctx := context.Background()
	aiClient, err := ai.NewGeminiClient(ctx, key, endpoint, modelName)
	if err != nil {
		log.Fatalf("[INFO] ai client init failed: %v", err)
	}
	defer aiClient.Close()

	ext := extractor.NewExifToolExtractor()
	processor := app.NewProcessor(ext, aiClient)

	log.Printf("[INFO] starting server on port %d", serverPort)
	srv := server.NewServer(processor, serverPort, serverTempDir, keepTemp)
	if err := srv.Start(); err != nil {
		log.Fatalf("[INFO] server failed: %v", err)
	}
}
