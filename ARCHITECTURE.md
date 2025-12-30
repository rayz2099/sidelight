# ARCHITECTURE.md - System Design & Data Flow

## 1. Directory Structure
```text
.
├── cmd/
│   └── sidelight/
│       └── main.go        # Entry point, initializes Cobra
├── internal/
│   ├── app/               # Application wiring
│   ├── extractor/         # Logic to extract JPG from RAW (exiftool wrapper)
│   ├── ai/                # AI Client (OpenAI/Gemini adapter)
│   ├── xmp/               # XMP generation and struct definitions
│   └── config/            # Viper configuration loading
├── pkg/
│   └── models/            # Shared structs (GradingParams, ProcessingResult)
├── AGENTS.md
└── go.mod
