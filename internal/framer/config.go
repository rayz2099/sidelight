package framer

// FrameConfig defines the layout and style of the output image.
type FrameConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Background Configuration
	BackgroundType  string  `json:"background_type"`  // "color" or "image_blur"
	BackgroundColor string  `json:"background_color"` // Hex color, e.g., "#FFFFFF" (used if type is color, or as overlay)
	BlurSigma       float64 `json:"blur_sigma"`       // Sigma for Gaussian Blur (e.g., 20.0)

	PaddingRatio   float64        `json:"padding_ratio"`    // Border width relative to image short side (0.0 - 1.0)
	BottomBarRatio float64        `json:"bottom_bar_ratio"` // Extra space at bottom relative to image height (0.0 - 1.0)
	Elements       []FrameElement `json:"elements"`
}

// FrameElement represents a single item drawn on the frame.
type FrameElement struct {
	Type    string `json:"type"`    // "text", "image" (original photo), "static_image" (logo)
	Content string `json:"content"` // Text template (e.g. "{{Make}}") or file path for static_image

	// Font Styling (for text)
	FontFile string  `json:"font_file"` // Path relative to assets/fonts/
	FontSize float64 `json:"font_size"` // Size relative to image width (0.0 - 1.0)
	Color    string  `json:"color"`     // Hex color

	// Positioning
	// "top-left", "top-center", "top-right", "middle-...", "bottom-..."
	Anchor string `json:"anchor"`

	// Margins relative to the canvas size
	MarginX float64 `json:"margin_x"`
	MarginY float64 `json:"margin_y"`

	// For image/logo resizing
	Scale float64 `json:"scale"` // Scale factor relative to available space
}
