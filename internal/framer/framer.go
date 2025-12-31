package framer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // Ensure JPEG decoding is available
	_ "image/png"  // Ensure PNG decoding is available
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"sidelight/pkg/models"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

// Framer handles the image generation.
type Framer struct {
	BaseDir string // Root directory for resolving assets
}

// NewFramer creates a new Framer.
func NewFramer(baseDir string) *Framer {
	return &Framer{BaseDir: baseDir}
}

// LoadStyle loads a FrameConfig from a JSON file in assets/styles.
func (f *Framer) LoadStyle(name string) (FrameConfig, error) {
	// If name doesn't have extension, add .json
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	path := filepath.Join(f.BaseDir, "assets/styles", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return FrameConfig{}, fmt.Errorf("failed to read style file %s: %w", path, err)
	}

	var config FrameConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return FrameConfig{}, fmt.Errorf("failed to parse style json: %w", err)
	}

	return config, nil
}

// Render processes the input image and applies the frame configuration.
func (f *Framer) Render(srcImage image.Image, metadata models.Metadata, config FrameConfig) (image.Image, error) {
	srcWidth := srcImage.Bounds().Dx()
	srcHeight := srcImage.Bounds().Dy()

	// Determine "short side" for padding calculations
	shortSide := float64(srcWidth)
	if srcHeight < srcWidth {
		shortSide = float64(srcHeight)
	}

	padding := shortSide * config.PaddingRatio
	bottomBar := float64(srcHeight) * config.BottomBarRatio

	// Calculate Canvas Size
	// The canvas wraps the image + padding on all sides + extra bottom bar
	// Note: Padding is usually applied to all 4 sides.
	// Total Width = Image + 2 * Padding
	// Total Height = Image + 2 * Padding + BottomBar
	canvasWidth := int(float64(srcWidth) + (2 * padding))
	canvasHeight := int(float64(srcHeight) + (2 * padding) + bottomBar)

	dc := gg.NewContext(canvasWidth, canvasHeight)

	// Draw Background
	if config.BackgroundType == "image_blur" {
		// 1. Resize source to fill the canvas (Aspect Fill)
		// We use imaging.Fill to crop/resize to exact canvas dimensions
		bgImage := imaging.Fill(srcImage, canvasWidth, canvasHeight, imaging.Center, imaging.Lanczos)

		// 2. Blur
		sigma := config.BlurSigma
		if sigma <= 0 {
			sigma = 20.0 // Default robust blur
		}
		bgImage = imaging.Blur(bgImage, sigma)

		// 3. Draw
		dc.DrawImage(bgImage, 0, 0)

		// Optional: Add a dimming overlay if background color is specified with alpha?
		// For now, if BackgroundColor is set, maybe overlay it with transparency?
		// Let's keep it simple: just blur.
	} else {
		// Standard Color Background
		bgColor, err := parseHexColor(config.BackgroundColor)
		if err != nil {
			return nil, fmt.Errorf("invalid background color: %w", err)
		}
		dc.SetColor(bgColor)
		dc.Clear()
	}

	// Process Elements
	for _, el := range config.Elements {
		switch el.Type {
		case "image":
			// The main photo. Typically centered horizontally and top-padded.
			// Position: (Padding, Padding)
			dc.DrawImage(srcImage, int(padding), int(padding))

		case "text":
			if err := f.drawText(dc, el, metadata, canvasWidth, canvasHeight, srcWidth); err != nil {
				return nil, fmt.Errorf("failed to draw text: %w", err)
			}

		case "static_image":
			// Logos etc.
			// Not fully implemented for this MVP, but placeholder logic:
			// f.drawStaticImage(dc, el, ...)
		}
	}

	return dc.Image(), nil
}

func (f *Framer) drawText(dc *gg.Context, el FrameElement, meta models.Metadata, canvasW, canvasH, srcW int) error {
	// 1. Resolve Content (Template)
	tmpl, err := template.New("text").Parse(el.Content)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, meta); err != nil {
		return err
	}
	text := buf.String()
	if text == "" {
		return nil
	}

	// 2. Load Font
	fontPath := filepath.Join(f.BaseDir, "assets/fonts", el.FontFile)
	// Font size is relative to the source image width for consistency
	points := float64(srcW) * el.FontSize
	if err := dc.LoadFontFace(fontPath, points); err != nil {
		return fmt.Errorf("failed to load font %s: %w", fontPath, err)
	}

	// 3. Set Color
	c, err := parseHexColor(el.Color)
	if err != nil {
		return err
	}
	dc.SetColor(c)

	// 4. Calculate Position
	var x, y float64
	marginX := float64(canvasW) * el.MarginX
	marginY := float64(canvasH) * el.MarginY

	parts := strings.Split(el.Anchor, "-")
	if len(parts) != 2 {
		parts = []string{"top", "left"} // default
	}
	vert, horz := parts[0], parts[1]

	// Determine reference Point (x,y) on the canvas
	switch horz {
	case "left":
		x = marginX
	case "center":
		x = float64(canvasW) / 2
	case "right":
		x = float64(canvasW) - marginX
	}

	switch vert {
	case "top":
		y = marginY
	case "middle":
		y = float64(canvasH) / 2
	case "bottom":
		y = float64(canvasH) - marginY
	}

	// Determine Anchor (ax, ay) for the text box
	// 0.0 = Top/Left of text matches point
	// 1.0 = Bottom/Right of text matches point
	var ax, ay float64

	switch horz {
	case "left":
		ax = 0.0
	case "center":
		ax = 0.5
	case "right":
		ax = 1.0
	}

	switch vert {
	case "top":
		ay = 0.0 // Top of text at y
	case "middle":
		ay = 0.5
	case "bottom":
		ay = 1.0 // Bottom of text at y
	}

	dc.DrawStringAnchored(text, x, y, ax, ay)
	return nil
}

func parseHexColor(s string) (color.Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		// RRGGBB
		v, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return nil, err
		}
		return color.RGBA{
			R: uint8(v >> 16),
			G: uint8(v >> 8),
			B: uint8(v),
			A: 255,
		}, nil
	}
	return nil, fmt.Errorf("invalid color format: %s", s)
}
