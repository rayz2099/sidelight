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

// PP3Params defines RawTherapee native parameters for direct PP3 generation.
// These are designed to work with RT's processing pipeline without conversion loss.
type PP3Params struct {
	// Exposure
	Compensation float64 `json:"compensation"` // -5.0 to +12.0
	Contrast     int     `json:"contrast"`     // -100 to 100
	Saturation   int     `json:"saturation"`   // -100 to 100
	Black        int     `json:"black"`        // 0 to 32768

	// Highlight & Shadow Recovery
	HighlightCompr    int `json:"highlight_compr"`    // 0 to 500
	ShadowRecovery    int `json:"shadow_recovery"`    // 0 to 100
	HighlightRecovery int `json:"highlight_recovery"` // 0 to 100

	// White Balance
	Temperature int     `json:"temperature"` // 1500 to 60000
	Tint        float64 `json:"tint"`        // 0.02 to 10.0 (1.0 = neutral)

	// Lab Adjustments (key for color pop)
	LabBrightness   int `json:"lab_brightness"`   // -100 to 100
	LabContrast     int `json:"lab_contrast"`     // -100 to 100
	LabChromaticity int `json:"lab_chromaticity"` // -100 to 100

	// Local Contrast / Clarity (SharpenMicro)
	SharpenMicroStrength   int `json:"sharpenmicro_strength"`   // 0 to 100
	SharpenMicroContrast   int `json:"sharpenmicro_contrast"`   // 0 to 100
	SharpenMicroUniformity int `json:"sharpenmicro_uniformity"` // 0 to 100

	// Dehaze
	DehazeStrength int `json:"dehaze_strength"` // -100 to 100

	// Vibrance
	VibPastels   int `json:"vib_pastels"`   // -100 to 100
	VibSaturated int `json:"vib_saturated"` // -100 to 100

	// Sharpening (output sharpening)
	SharpenEnabled  bool    `json:"sharpen_enabled"`  // enable sharpening
	SharpenAmount   int     `json:"sharpen_amount"`   // 0 to 1000
	SharpenRadius   float64 `json:"sharpen_radius"`   // 0.3 to 3.0
	SharpenContrast int     `json:"sharpen_contrast"` // 0 to 100

	// Edge Sharpening
	EdgeSharpenEnabled bool `json:"edge_sharpen_enabled"` // enable edge sharpening
	EdgeSharpenAmount  int  `json:"edge_sharpen_amount"`  // 0 to 100
	EdgeSharpenPasses  int  `json:"edge_sharpen_passes"`  // 1 to 4

	// Capture Sharpening (demosaic level - critical for perceived sharpness)
	CaptureSharpEnabled bool    `json:"capture_sharp_enabled"` // enable capture sharpening
	CaptureSharpAmount  int     `json:"capture_sharp_amount"`  // 0 to 200
	CaptureSharpRadius  float64 `json:"capture_sharp_radius"`  // 0.5 to 2.0

	// Noise Reduction
	NRLuminance   int `json:"nr_luminance"`   // 0 to 100
	NRChrominance int `json:"nr_chrominance"` // 0 to 100

	// Tone Curve (S-curve control points)
	// Format: array of [x, y] pairs, x and y are 0.0 to 1.0
	ToneCurve [][]float64 `json:"tone_curve"`

	// L Curve (Luminance curve in Lab space)
	LCurve [][]float64 `json:"l_curve"`

	// RGB Curves for color grading
	RCurve [][]float64 `json:"r_curve"`
	GCurve [][]float64 `json:"g_curve"`
	BCurve [][]float64 `json:"b_curve"`

	// Color Toning (Split Toning)
	ColorToningShadowR    int `json:"ct_shadow_r"`    // -100 to 100
	ColorToningShadowG    int `json:"ct_shadow_g"`    // -100 to 100
	ColorToningShadowB    int `json:"ct_shadow_b"`    // -100 to 100
	ColorToningHighlightR int `json:"ct_highlight_r"` // -100 to 100
	ColorToningHighlightG int `json:"ct_highlight_g"` // -100 to 100
	ColorToningHighlightB int `json:"ct_highlight_b"` // -100 to 100
	ColorToningBalance    int `json:"ct_balance"`     // 0 to 100

	// Vignette
	VignetteAmount int `json:"vignette_amount"` // -100 to 100
}

// ProcessingResult holds the outcome of processing a single file.
type ProcessingResult struct {
	SourcePath string
	XmpPath    string
	Params     GradingParams
	PP3Params  *PP3Params
	Metadata   Metadata
	Error      error
}
