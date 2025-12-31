package framer

// DefaultConfig returns a classic white border configuration.
func DefaultConfig() FrameConfig {
	return FrameConfig{
		ID:              "classic_white",
		Name:            "Classic White with EXIF",
		BackgroundColor: "#FFFFFF",
		PaddingRatio:    0.05, // 5% border
		BottomBarRatio:  0.10, // Extra 10% at bottom for text
		Elements: []FrameElement{
			{
				Type:   "image",
				Anchor: "center-middle", // Not used for main image logic currently, but good for future
			},
			{
				Type:     "text",
				Content:  "{{.Make}} {{.Model}}",
				FontFile: "MapleMono-NF-CN-Regular.ttf",
				FontSize: 0.025, // 2.5% of image width
				Color:    "#000000",
				Anchor:   "bottom-left",
				MarginX:  0.05,
				MarginY:  0.065, // Positioned in the bottom bar
			},
			{
				Type:     "text",
				Content:  "{{.FocalLength}}  {{.Aperture}}  {{.ShutterSpeed}}  ISO{{.ISO}}",
				FontFile: "MapleMono-NF-CN-Regular.ttf",
				FontSize: 0.018,
				Color:    "#666666",
				Anchor:   "bottom-left",
				MarginX:  0.05,
				MarginY:  0.035,
			},
		},
	}
}
