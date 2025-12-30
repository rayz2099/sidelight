# AGENTS.md - Context & Instructions for AI Coding Assistants

## 1. Project Overview
**Name:** SideLight
**Description:** A CLI tool for photographers written in Go. It automates RAW photo color grading using AI (LLM with Vision capabilities).
**Core Logic:**
1.  **Extract:** Reads a RAW file (ARW, CR3, etc.), extracts the embedded JPEG preview using `exiftool`.
2.  **Analyze:** Sends the compressed preview to an AI provider (OpenAI/Gemini/Claude).
3.  **Generate:** Receives color grading parameters (JSON) and generates an XMP sidecar file.
4.  **Result:** The original RAW file is untouched; the XMP file applies the look in Lightroom/Camera Raw.

## 2. Technical Stack
* **Language:** Go (Golang) 1.22+
* **CLI Framework:** `github.com/spf13/cobra`
* **Image Extraction:** Wrapper around `exiftool` (via `os/exec`).
* **Concurrency:** Native Goroutines & Channels (Worker Pool pattern).
* **Configuration:** `github.com/spf13/viper` (for API keys and defaults).
* **UI:** `github.com/schollz/progressbar/v3` (for CLI progress).

## 3. Coding Standards & Rules
* **Project Layout:** Follow standard Go project layout (`cmd/`, `internal/`, `pkg/`).
* **Error Handling:** precise error wrapping (`fmt.Errorf("...: %w", err)`). No silent failures.
* **Type Safety:** strictly define JSON and XML structs. Do not use `map[string]interface{}` unless absolutely necessary.
* **Dependency Injection:** Interfaces should be defined where they are used.
* **Comments:** All exported functions must have GoDoc style comments.

## 4. Constraint Checklist
* [ ] **No OpenCV**: Do not use `gocv` or CGO bindings for image processing to ensure easy distribution. Use standard library `image` or `exiftool` only.
* [ ] **Sidecar Only**: Never attempt to write/modify the source RAW file.
* [ ] **Mapping**: XMP parameter mapping must match Adobe Camera Raw standards (e.g., `crs:Exposure2012`).
