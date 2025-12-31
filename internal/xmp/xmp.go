package xmp

import (
	"encoding/xml"
	"fmt"
)

const (
	// Namespaces
	NsX   = "adobe:ns:meta/"
	NsRdf = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	NsCrs = "http://ns.adobe.com/camera-raw-settings/1.0/"

	// Standard Header
	XmpHeader = `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
)

// CameraRawSettings defines the subset of CRS parameters we want to control via AI.
// These map directly to attributes in the rdf:Description element.
type CameraRawSettings struct {
	ProcessVersion string `xml:"crs:ProcessVersion,attr,omitempty"`

	// Basic Tone
	Exposure2012   float64 `xml:"crs:Exposure2012,attr,omitempty"`
	Contrast2012   int     `xml:"crs:Contrast2012,attr,omitempty"`
	Highlights2012 int     `xml:"crs:Highlights2012,attr,omitempty"`
	Shadows2012    int     `xml:"crs:Shadows2012,attr,omitempty"`
	Whites2012     int     `xml:"crs:Whites2012,attr,omitempty"`
	Blacks2012     int     `xml:"crs:Blacks2012,attr,omitempty"`

	// Presence
	Texture     int `xml:"crs:Texture,attr,omitempty"`
	Clarity2012 int `xml:"crs:Clarity2012,attr,omitempty"`
	Dehaze      int `xml:"crs:Dehaze,attr,omitempty"`
	Vibrance    int `xml:"crs:Vibrance,attr,omitempty"`
	Saturation  int `xml:"crs:Saturation,attr,omitempty"`

	// White Balance
	// Note: If WhiteBalance is "Custom", Temperature and Tint are used.
	// If "As Shot", they might be ignored or offsets.
	// We assume we are providing specific values (Custom).
	Temperature int `xml:"crs:Temperature,attr,omitempty"`
	Tint        int `xml:"crs:Tint,attr,omitempty"`

	// Detail
	Sharpness           int `xml:"crs:Sharpness,attr,omitempty"`
	LuminanceSmoothing  int `xml:"crs:LuminanceSmoothing,attr,omitempty"`
	ColorNoiseReduction int `xml:"crs:ColorNoiseReduction,attr,omitempty"`

	// Vignette
	PostCropVignetteAmount int `xml:"crs:PostCropVignetteAmount,attr,omitempty"`

	// HSL - Hue
	HueAdjustmentRed     int `xml:"crs:HueAdjustmentRed,attr,omitempty"`
	HueAdjustmentOrange  int `xml:"crs:HueAdjustmentOrange,attr,omitempty"`
	HueAdjustmentYellow  int `xml:"crs:HueAdjustmentYellow,attr,omitempty"`
	HueAdjustmentGreen   int `xml:"crs:HueAdjustmentGreen,attr,omitempty"`
	HueAdjustmentAqua    int `xml:"crs:HueAdjustmentAqua,attr,omitempty"`
	HueAdjustmentBlue    int `xml:"crs:HueAdjustmentBlue,attr,omitempty"`
	HueAdjustmentPurple  int `xml:"crs:HueAdjustmentPurple,attr,omitempty"`
	HueAdjustmentMagenta int `xml:"crs:HueAdjustmentMagenta,attr,omitempty"`

	// HSL - Saturation
	SaturationAdjustmentRed     int `xml:"crs:SaturationAdjustmentRed,attr,omitempty"`
	SaturationAdjustmentOrange  int `xml:"crs:SaturationAdjustmentOrange,attr,omitempty"`
	SaturationAdjustmentYellow  int `xml:"crs:SaturationAdjustmentYellow,attr,omitempty"`
	SaturationAdjustmentGreen   int `xml:"crs:SaturationAdjustmentGreen,attr,omitempty"`
	SaturationAdjustmentAqua    int `xml:"crs:SaturationAdjustmentAqua,attr,omitempty"`
	SaturationAdjustmentBlue    int `xml:"crs:SaturationAdjustmentBlue,attr,omitempty"`
	SaturationAdjustmentPurple  int `xml:"crs:SaturationAdjustmentPurple,attr,omitempty"`
	SaturationAdjustmentMagenta int `xml:"crs:SaturationAdjustmentMagenta,attr,omitempty"`

	// HSL - Luminance
	LuminanceAdjustmentRed     int `xml:"crs:LuminanceAdjustmentRed,attr,omitempty"`
	LuminanceAdjustmentOrange  int `xml:"crs:LuminanceAdjustmentOrange,attr,omitempty"`
	LuminanceAdjustmentYellow  int `xml:"crs:LuminanceAdjustmentYellow,attr,omitempty"`
	LuminanceAdjustmentGreen   int `xml:"crs:LuminanceAdjustmentGreen,attr,omitempty"`
	LuminanceAdjustmentAqua    int `xml:"crs:LuminanceAdjustmentAqua,attr,omitempty"`
	LuminanceAdjustmentBlue    int `xml:"crs:LuminanceAdjustmentBlue,attr,omitempty"`
	LuminanceAdjustmentPurple  int `xml:"crs:LuminanceAdjustmentPurple,attr,omitempty"`
	LuminanceAdjustmentMagenta int `xml:"crs:LuminanceAdjustmentMagenta,attr,omitempty"`

	// Split Toning
	SplitToningShadowHue           int `xml:"crs:SplitToningShadowHue,attr,omitempty"`
	SplitToningShadowSaturation    int `xml:"crs:SplitToningShadowSaturation,attr,omitempty"`
	SplitToningHighlightHue        int `xml:"crs:SplitToningHighlightHue,attr,omitempty"`
	SplitToningHighlightSaturation int `xml:"crs:SplitToningHighlightSaturation,attr,omitempty"`
	SplitToningBalance             int `xml:"crs:SplitToningBalance,attr,omitempty"`
}

// rdfDescription represents the inner content of the RDF.
type rdfDescription struct {
	XMLName  xml.Name `xml:"rdf:Description"`
	About    string   `xml:"rdf:about,attr"`
	XmlnsCrs string   `xml:"xmlns:crs,attr"`
	CameraRawSettings
}

// rdfRDF represents the <rdf:RDF> container.
type rdfRDF struct {
	XMLName     xml.Name        `xml:"rdf:RDF"`
	XmlnsRdf    string          `xml:"xmlns:rdf,attr"`
	Description *rdfDescription 
}

// xmpMeta represents the root <x:xmpmeta> element.
type xmpMeta struct {
	XMLName xml.Name `xml:"x:xmpmeta"`
	XmlnsX  string   `xml:"xmlns:x,attr"`
	XmpTk   string   `xml:"x:xmptk,attr"`
	RDF     *rdfRDF
}

// NewCameraRawSettings returns a struct with default ProcessVersion.
func NewCameraRawSettings() CameraRawSettings {
	return CameraRawSettings{
		ProcessVersion: "11.0",
	}
}

// Marshal generates the full XMP byte slice for the given settings.
func Marshal(settings CameraRawSettings) ([]byte, error) {
	// Wrap the settings in the XMP envelope
	xmp := &xmpMeta{
		XmlnsX: NsX,
		XmpTk:  "SideLight", // Tool name
		RDF: &rdfRDF{
			XmlnsRdf: NsRdf,
			Description: &rdfDescription{
				About:             "",
				XmlnsCrs:          NsCrs,
				CameraRawSettings: settings,
			},
		},
	}

	output, err := xml.MarshalIndent(xmp, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XMP: %w", err)
	}

	// Combine header and XML body
	return append([]byte(XmpHeader), output...), nil
}
