package models

// GradingParams defines the color grading parameters returned by the AI.
type GradingParams struct {
	Exposure2012   float64 `json:"exposure"`
	Contrast2012   int     `json:"contrast"`
	Highlights2012 int     `json:"highlights"`
	Shadows2012    int     `json:"shadows"`
	Whites2012     int     `json:"whites"`
	Blacks2012     int     `json:"blacks"`
	Texture        int     `json:"texture"`
	Clarity2012    int     `json:"clarity"`
	Dehaze         int     `json:"dehaze"`
	Vibrance       int     `json:"vibrance"`
	Saturation     int     `json:"saturation"`
	Temperature    int     `json:"temperature"`
	Tint           int     `json:"tint"`
	Sharpness      int     `json:"sharpness"`
}

// ProcessingResult holds the outcome of processing a single file.
type ProcessingResult struct {
	SourcePath string
	XmpPath    string
	Params     GradingParams
	Error      error
}
