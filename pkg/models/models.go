package models

// Metadata holds technical details extracted from the image.
type Metadata struct {
	Make         string `json:"make"`
	Model        string `json:"model"`
	Lens         string `json:"lens"`
	ISO          int    `json:"iso"`
	Aperture     string `json:"aperture"` // Stored as string to handle "f/1.8" etc.
	ShutterSpeed string `json:"shutter_speed"`
	FocalLength  string `json:"focal_length"`
	DateTime     string `json:"date_time"`
}

// GradingParams defines the color grading parameters returned by the AI.
type GradingParams struct {
	// Basic Tone
	Exposure2012   float64 `json:"exposure"`
	Contrast2012   int     `json:"contrast"`
	Highlights2012 int     `json:"highlights"`
	Shadows2012    int     `json:"shadows"`
	Whites2012     int     `json:"whites"`
	Blacks2012     int     `json:"blacks"`

	// Presence
	Texture     int `json:"texture"`
	Clarity2012 int `json:"clarity"`
	Dehaze      int `json:"dehaze"`
	Vibrance    int `json:"vibrance"`
	Saturation  int `json:"saturation"`

	// White Balance
	Temperature int `json:"temperature"`
	Tint        int `json:"tint"`

	// Detail & Noise
	Sharpness           int `json:"sharpness"`
	LuminanceSmoothing  int `json:"luminance_noise_reduction"`
	ColorNoiseReduction int `json:"color_noise_reduction"`

	// Vignette
	PostCropVignetteAmount int `json:"vignette_amount"`

	// HSL - Hue
	HueAdjustmentRed     int `json:"hue_red"`
	HueAdjustmentOrange  int `json:"hue_orange"`
	HueAdjustmentYellow  int `json:"hue_yellow"`
	HueAdjustmentGreen   int `json:"hue_green"`
	HueAdjustmentAqua    int `json:"hue_aqua"`
	HueAdjustmentBlue    int `json:"hue_blue"`
	HueAdjustmentPurple  int `json:"hue_purple"`
	HueAdjustmentMagenta int `json:"hue_magenta"`

	// HSL - Saturation
	SaturationAdjustmentRed     int `json:"saturation_red"`
	SaturationAdjustmentOrange  int `json:"saturation_orange"`
	SaturationAdjustmentYellow  int `json:"saturation_yellow"`
	SaturationAdjustmentGreen   int `json:"saturation_green"`
	SaturationAdjustmentAqua    int `json:"saturation_aqua"`
	SaturationAdjustmentBlue    int `json:"saturation_blue"`
	SaturationAdjustmentPurple  int `json:"saturation_purple"`
	SaturationAdjustmentMagenta int `json:"saturation_magenta"`

	// HSL - Luminance
	LuminanceAdjustmentRed     int `json:"luminance_red"`
	LuminanceAdjustmentOrange  int `json:"luminance_orange"`
	LuminanceAdjustmentYellow  int `json:"luminance_yellow"`
	LuminanceAdjustmentGreen   int `json:"luminance_green"`
	LuminanceAdjustmentAqua    int `json:"luminance_aqua"`
	LuminanceAdjustmentBlue    int `json:"luminance_blue"`
	LuminanceAdjustmentPurple  int `json:"luminance_purple"`
	LuminanceAdjustmentMagenta int `json:"luminance_magenta"`

	// Split Toning (Simple)
	SplitToningShadowHue           int `json:"split_shadow_hue"`
	SplitToningShadowSaturation    int `json:"split_shadow_saturation"`
	SplitToningHighlightHue        int `json:"split_highlight_hue"`
	SplitToningHighlightSaturation int `json:"split_highlight_saturation"`
	SplitToningBalance             int `json:"split_balance"`
}

// ProcessingResult holds the outcome of processing a single file.
type ProcessingResult struct {
	SourcePath string
	XmpPath    string
	Params     GradingParams
	Metadata   Metadata
	Error      error
}
