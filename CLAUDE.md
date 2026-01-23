# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SideLight is an AI-powered professional color grading and framing tool for photographers, written in Go. It analyzes photos using Google Gemini's multimodal vision capabilities to generate cinema-quality color grading parameters (XMP sidecar files for Adobe Lightroom or PP3 files for RawTherapee) and creates professional art frames with EXIF metadata.

**Key Principles:**
- Non-destructive workflow: Never modifies source RAW files
- Supports multiple RAW formats (ARW, CR3, NEF) and standard formats (JPG, PNG)
- Dual-format output: Adobe Lightroom (XMP) and RawTherapee native PP3 parameters
- Interactive web UI via embedded server

## Common Development Commands

### Build and Installation
```bash
# Compile binary (outputs to ./bin/sidelight)
just build
# or: go build -o bin/sidelight ./cmd/sidelight

# Install to system (copies to $GOPATH/bin)
just install
# or: go install ./cmd/sidelight

# Preview all frame styles on a test image
just preview-styles [file]

# Preview RawTherapee PP3 output
just preview-rt [file] [style] [prompt]
```

### Testing
```bash
# Run tests for all internal packages
just test
# or: go test -v ./internal/...

# Run all tests including integration tests (requires API key)
go test -v ./...

# Run tests for specific package
go test -v ./internal/app
```

### Configuration
```bash
# Set up API key (required for AI grading features)
export SL_GEMINI_API_KEY="your_api_key_here"

# Or create config file (recommended)
mkdir -p ~/.config/sidelight
echo '{"gemini_api_key": "your_api_key_here"}' > ~/.config/sidelight/config.json
```

## Architecture Overview

### Core Data Flow
```
Input File (RAW/JPG/PNG)
    ↓
[Extractor] - Extract embedded preview + metadata via exiftool
    ↓
[AI Client] - Send preview to Gemini API for analysis
    ↓
[Processor] - Convert AI response to color grading parameters
    ↓
[XMP/PP3 Generator] - Write sidecar files (XMP or PP3)
    ↓
Output: Original file untouched + sidecar with grading data
```

### Key Components

**Processor (`internal/app/processor.go`)**
- Main orchestrator for extraction, analysis, and generation
- Entry point: `ProcessFile(ctx, rawPath, opts)`
- Handles both XMP and PP3 format generation

**Extractor Interface (`internal/extractor/extractor.go`)**
- Implementation: `ExifToolExtractor` (wraps external exiftool binary)
- `ExtractPreview()` - Get image bytes for AI analysis
- `ExtractMetadata()` - Get EXIF data for frame rendering
- `EmbedXMP()` - Write XMP into JPG/PNG files

**AI Client Interface (`internal/ai/ai.go`)**
- Implementation: `GeminiClient` for Google Gemini API
- `AnalyzeImageLR()` - Generate Adobe Lightroom parameters
- `AnalyzeImageForPP3()` - Generate RawTherapee native parameters

**Format Generators**
- XMP: `internal/xmp/xmp.go` - Adobe Camera Raw Settings XML
- PP3: `internal/rt/pp3.go` - RawTherapee configuration format

**Framer (`internal/framer/framer.go`)**
- Renders artistic frames with EXIF overlays
- Loads JSON configs from `assets/styles/`
- 25+ built-in frame styles

**Server (`internal/server/server.go`)**
- HTTP server with embedded web UI
- POST `/api/grade` endpoint for interactive grading

### Directory Structure
```
cmd/sidelight/          # CLI entry points (main.go, grade.go, frame.go, server.go)
internal/
  ├── app/              # Core processor logic
  ├── ai/               # AI client implementations (Gemini)
  ├── xmp/              # Adobe XMP generation
  ├── rt/               # RawTherapee PP3 conversion
  ├── extractor/        # Image/metadata extraction (exiftool wrapper)
  ├── framer/           # Frame rendering engine
  └── server/           # Web UI server
pkg/models/             # Shared data structures
assets/styles/          # 25+ JSON frame style definitions
```

## Key Technologies

- **Go 1.25.4** - Core language
- **Cobra** - CLI framework with subcommands
- **Viper** - Configuration management
- **Google Generative AI** - Gemini multimodal vision API
- **exiftool** (external) - RAW preview extraction, EXIF read/write
- **github.com/disintegration/imaging** - Image manipulation
- **github.com/fogleman/gg** - 2D graphics for frame rendering

## Important Design Constraints

### From AGENTS.md
- **No OpenCV**: Do not use `gocv` or CGO bindings. Use standard library `image` or `exiftool` only
- **Sidecar Only**: Never modify source RAW files
- **XMP Mapping**: Must match Adobe Camera Raw standards (e.g., `crs:Exposure2012`)

### Additional Constraints
- **Non-destructive workflow**: All operations preserve original files
- **Interface-first design**: Use dependency injection for testability
- **Context propagation**: All operations accept `context.Context` for cancellation
- **Concurrent processing**: Worker pool pattern with configurable concurrency

## Configuration Sources (Priority Order)
1. Command-line flags (highest priority)
2. Environment variables (`SL_GEMINI_API_KEY`, `SL_GEMINI_ENDPOINT_URL`, `SL_GEMINI_MODEL_NAME`)
3. Config file `config.json` (searched in: current dir → executable dir → `~/.config/sidelight/`)
4. Defaults (lowest priority)

## External Dependencies
- **exiftool** - Must be installed on system PATH
  - macOS: `brew install exiftool`
  - Linux: `sudo apt-get install libimage-exiftool-perl`
  - Windows: Download `exiftool.exe` and add to PATH
- **Google Gemini API key** - Required for AI grading features

## Main CLI Commands

1. **`sidelight grade [files...] [flags]`**
   - AI color grading with XMP/PP3 output
   - Key flags: `-s/--style`, `-f/--format`, `-p/--prompt`, `-j/--concurrency`

2. **`sidelight frame [files...] [flags]`**
   - Artistic frame generation with EXIF metadata
   - Key flags: `-s/--style`, `-f/--format`, `-q/--quality`, `-o/--output`

3. **`sidelight server [flags]`**
   - Web UI server with interactive grading
   - Key flags: `-p/--port` (default 8080)

## Development Workflow

1. **For feature implementation**: Start with interfaces in the relevant `internal/` package
2. **For testing**: Use `internal/` package tests; integration tests require API key
3. **For new AI features**: Extend `internal/ai/ai.go` interface and implement in `gemini.go`
4. **For new output formats**: Add generator in appropriate `internal/` subdirectory
5. **For new frame styles**: Add JSON config to `assets/styles/`

## Important Files for Context

- `/cmd/sidelight/main.go` - Entry point and config loading
- `/internal/app/processor.go` - Main workflow orchestration
- `/internal/ai/gemini.go` - AI prompts and analysis logic
- `/pkg/models/models.go` - Shared data structures
- `AGENTS.md` - Technical constraints and coding standards
- `Justfile` - Build and development commands