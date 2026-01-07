package rt

import (
	"fmt"
	"math"
	"strings"

	"sidelight/pkg/models"
)

// clamp limits a value to a given range
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// abs returns the absolute value of an integer
func abs(val int) int {
	if val < 0 {
		return -val
	}
	return val
}

// normalizeCurve detects if values are 0-255 and converts to 0.0-1.0
// Returns the normalized points
func normalizeCurve(points [][]float64) [][]float64 {
	if len(points) == 0 {
		return points
	}

	// Detect if values are in 0-255 range (any value > 1.5)
	needsNormalize := false
	for _, p := range points {
		for _, v := range p {
			if v > 1.5 {
				needsNormalize = true
				break
			}
		}
		if needsNormalize {
			break
		}
	}

	if !needsNormalize {
		return points
	}

	normalized := make([][]float64, len(points))
	for i, p := range points {
		if len(p) >= 2 {
			normalized[i] = []float64{p[0] / 255.0, p[1] / 255.0}
		}
	}
	return normalized
}

// validateCurve checks if a curve would produce very dark/bright output
// Returns validated curve, fixing dangerous values
func validateCurve(points [][]float64, curveType string) [][]float64 {
	if len(points) < 2 {
		// Return neutral curve (no effect)
		return [][]float64{{0, 0}, {1, 1}}
	}

	// Normalize first
	points = normalizeCurve(points)

	// Clamp all values to valid range
	for i := range points {
		if len(points[i]) >= 2 {
			// Clamp x
			if points[i][0] < 0 {
				points[i][0] = 0
			}
			if points[i][0] > 1 {
				points[i][0] = 1
			}
			// Clamp y
			if points[i][1] < 0 {
				points[i][1] = 0
			}
			if points[i][1] > 1 {
				points[i][1] = 1
			}
		}
	}

	// Check for problematic curves that would cause black/white images
	// Find the approximate midpoint output
	var midY float64 = 0.5 // default
	var blackY float64 = 0 // output when input is 0
	var whiteY float64 = 1 // output when input is 1

	for _, p := range points {
		if len(p) >= 2 {
			x, y := p[0], p[1]
			// Check black point
			if x <= 0.05 {
				blackY = y
			}
			// Check white point
			if x >= 0.95 {
				whiteY = y
			}
			// Check midpoint (x around 0.45-0.55)
			if x >= 0.45 && x <= 0.55 {
				midY = y
			}
		}
	}

	// Detect problematic curves:
	// 1. Midpoint output too dark (< 0.3) or too bright (> 0.8)
	// 2. Black point raised too high (> 0.4) - would create washed out look
	// 3. White point too low (< 0.6) - would create dark image
	// 4. Inverted curve (black > white)
	isProblem := false
	reason := ""

	if midY < 0.25 {
		isProblem = true
		reason = "midpoint too dark"
	} else if midY > 0.85 {
		isProblem = true
		reason = "midpoint too bright"
	} else if blackY > 0.5 {
		isProblem = true
		reason = "black point too high"
	} else if whiteY < 0.5 {
		isProblem = true
		reason = "white point too low"
	} else if blackY > whiteY {
		isProblem = true
		reason = "inverted curve"
	}

	if isProblem {
		// Log the issue (could be made more visible)
		_ = reason // suppress unused warning, but reason is useful for debugging

		// Return a safe default based on curve type
		switch curveType {
		case "tone":
			// Gentle S-curve for transparency and punch
			return [][]float64{{0, 0}, {0.15, 0.12}, {0.5, 0.52}, {0.85, 0.88}, {1, 1}}
		case "rgb_r":
			// Slight warmth
			return [][]float64{{0, 0}, {0.5, 0.52}, {1, 1}}
		case "rgb_b":
			// Slight warmth (reduce blue)
			return [][]float64{{0, 0}, {0.5, 0.48}, {1, 1}}
		default:
			// Neutral
			return [][]float64{{0, 0}, {1, 1}}
		}
	}

	return points
}

// formatCurve converts [[x,y], ...] to RT format "1;x1;y1;x2;y2;..."
// RT curves expect values in 0.0-1.0 range
func formatCurve(points [][]float64) string {
	if len(points) == 0 {
		return "0;"
	}

	var parts []string
	parts = append(parts, "1") // Curve type: Standard

	for _, p := range points {
		if len(p) >= 2 {
			x, y := p[0], p[1]
			parts = append(parts, fmt.Sprintf("%.4f", x))
			parts = append(parts, fmt.Sprintf("%.4f", y))
		}
	}
	return strings.Join(parts, ";") + ";"
}

// formatCurveValidated validates and formats a curve for PP3 output
func formatCurveValidated(points [][]float64, curveType string) string {
	validated := validateCurve(points, curveType)
	return formatCurve(validated)
}

// clampFloat limits a float value to a given range
func clampFloat(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// sanitizeParams ensures all PP3 params are within safe ranges
// This prevents dark/purple/oversaturated images from extreme AI values
func sanitizeParams(params *models.PP3Params) {
	// === EXPOSURE - CRITICAL FOR BRIGHTNESS ===
	// RT renders darker than LR, compensation MUST be positive for normal exposure
	if params.Compensation < 0.25 {
		params.Compensation = 0.4 // Safe default
	}
	if params.Compensation > 1.5 {
		params.Compensation = 1.5
	}

	// === CONTRAST - keep moderate ===
	if params.Contrast < -20 {
		params.Contrast = -20
	}
	if params.Contrast > 30 {
		params.Contrast = 30
	}

	// === BLACK POINT - too high crushes shadows ===
	if params.Black > 200 {
		params.Black = 200
	}

	// === HIGHLIGHT COMPRESSION - too high makes flat image ===
	if params.HighlightCompr > 180 {
		params.HighlightCompr = 180
	}

	// === HIGHLIGHT/SHADOW RECOVERY ===
	if params.HighlightRecovery > 70 {
		params.HighlightRecovery = 70
	}
	if params.ShadowRecovery > 60 {
		params.ShadowRecovery = 60
	}

	// === LAB ADJUSTMENTS ===
	// Brightness should not be negative (makes image dark)
	if params.LabBrightness < 0 {
		params.LabBrightness = 0
	}
	if params.LabBrightness > 20 {
		params.LabBrightness = 20
	}

	// Lab contrast
	if params.LabContrast < 0 {
		params.LabContrast = 0
	}
	if params.LabContrast > 40 {
		params.LabContrast = 40
	}

	// Lab chromaticity - too high causes oversaturation
	if params.LabChromaticity < 0 {
		params.LabChromaticity = 0
	}
	if params.LabChromaticity > 45 {
		params.LabChromaticity = 45
	}

	// === DEHAZE ===
	if params.DehazeStrength < 0 {
		params.DehazeStrength = 0
	}
	if params.DehazeStrength > 30 {
		params.DehazeStrength = 30
	}

	// === WHITE BALANCE ===
	// Temperature: allow cooler temps for Fuji-style looks
	if params.Temperature < 4200 {
		params.Temperature = 4200
	}
	if params.Temperature > 7500 {
		params.Temperature = 7500
	}

	// Tint: allow slightly more range for stylistic effects
	if params.Tint < 0.90 {
		params.Tint = 0.90
	}
	if params.Tint > 1.10 {
		params.Tint = 1.10
	}

	// === COLOR TONING - CRITICAL FOR SKIN TONES ===
	// Limit color toning to avoid purple/green color casts
	maxCT := 12
	params.ColorToningShadowR = clamp(params.ColorToningShadowR, -maxCT, maxCT)
	params.ColorToningShadowG = clamp(params.ColorToningShadowG, -maxCT, maxCT)
	params.ColorToningShadowB = clamp(params.ColorToningShadowB, -maxCT, maxCT)
	params.ColorToningHighlightR = clamp(params.ColorToningHighlightR, -maxCT, maxCT)
	params.ColorToningHighlightG = clamp(params.ColorToningHighlightG, -maxCT, maxCT)
	params.ColorToningHighlightB = clamp(params.ColorToningHighlightB, -maxCT, maxCT)

	// === SATURATION ===
	if params.Saturation < -100 {
		params.Saturation = -100
	}
	if params.Saturation > 25 {
		params.Saturation = 25
	}

	// === VIBRANCE ===
	if params.VibPastels > 45 {
		params.VibPastels = 45
	}
	if params.VibSaturated > 25 {
		params.VibSaturated = 25
	}

	// === SHARPENING - keep moderate ===
	if params.SharpenMicroStrength > 50 {
		params.SharpenMicroStrength = 50
	}
	if params.SharpenMicroContrast > 40 {
		params.SharpenMicroContrast = 40
	}

	// === TONE CURVE VALIDATION ===
	// Ensure tone curve doesn't make image too dark
	if len(params.ToneCurve) > 0 {
		for i := range params.ToneCurve {
			if len(params.ToneCurve[i]) >= 2 {
				x := params.ToneCurve[i][0]
				y := params.ToneCurve[i][1]
				// Midtones (around 0.5) should not be darker than input
				if x >= 0.4 && x <= 0.6 && y < x-0.1 {
					params.ToneCurve[i][1] = x // Reset to linear
				}
				// Clamp all values to valid range
				params.ToneCurve[i][0] = clampFloat(x, 0, 1)
				params.ToneCurve[i][1] = clampFloat(y, 0, 1)
			}
		}
	}
}

// GeneratePP3FromNative creates a RawTherapee PP3 file from native PP3 parameters
// SIMPLIFIED version - only essential parameters to avoid artifacts
func GeneratePP3FromNative(params *models.PP3Params, isRaw bool) []byte {
	// Sanitize parameters first
	sanitizeParams(params)

	var sb strings.Builder

	// === HEADER ===
	sb.WriteString("[Version]\n")
	sb.WriteString("Version=346\n")
	sb.WriteString("Build=SideLight-AI\n\n")

	sb.WriteString("[General]\n")
	sb.WriteString("Rank=0\n")
	sb.WriteString("ColorLabel=0\n")
	sb.WriteString("InTrash=false\n\n")

	// === EXPOSURE (core) ===
	sb.WriteString("[Exposure]\n")
	sb.WriteString("Enabled=true\n")
	sb.WriteString(fmt.Sprintf("Compensation=%.2f\n", params.Compensation))
	sb.WriteString(fmt.Sprintf("Contrast=%d\n", clamp(params.Contrast, -50, 50)))
	sb.WriteString(fmt.Sprintf("Saturation=%d\n", clamp(params.Saturation, -50, 50)))
	sb.WriteString(fmt.Sprintf("Black=%d\n", clamp(params.Black, 0, 300)))
	sb.WriteString(fmt.Sprintf("HighlightCompr=%d\n", clamp(params.HighlightCompr, 0, 200)))
	sb.WriteString("HighlightComprThreshold=0\n")
	// Use Blend method for natural highlight rolloff (similar to LR)
	sb.WriteString("HighlightReconstruction=true\n")
	sb.WriteString("HighlightReconstructionMethod=blend\n\n")

	// === TONE CURVE (simple) ===
	sb.WriteString("[ToneCurve]\n")
	sb.WriteString("Enabled=true\n")
	// Use Film-Like / Perceptual curve mode for smoother transitions
	sb.WriteString("CurveMode=FilmLike\n") 
	sb.WriteString("Curve=0;\n")
	sb.WriteString("Curve2=0;\n\n")

	// === WHITE BALANCE ===
	sb.WriteString("[White Balance]\n")
	if isRaw {
		sb.WriteString("Enabled=true\n")
		sb.WriteString("Setting=Custom\n")
		temp := params.Temperature
		if temp < 3500 || temp > 9000 {
			temp = 5500
		}
		sb.WriteString(fmt.Sprintf("Temperature=%d\n", temp))
		tint := params.Tint
		if tint < 0.7 || tint > 1.5 {
			tint = 1.0
		}
		sb.WriteString(fmt.Sprintf("Green=%.3f\n", tint))
		sb.WriteString("Equal=1\n\n")
	} else {
		// For JPEG, applying RAW WB values causes severe color casts (especially blue).
		// Disable WB module to preserve original colors, or use "Camera" if enabled.
		sb.WriteString("Enabled=false\n\n")
	}

	// === LAB ADJUSTMENTS (for color/contrast) ===
	sb.WriteString("[Luminance Curve]\n")
	sb.WriteString("Enabled=true\n")
	sb.WriteString(fmt.Sprintf("Brightness=%d\n", clamp(params.LabBrightness, -20, 20)))
	sb.WriteString(fmt.Sprintf("Contrast=%d\n", clamp(params.LabContrast, 0, 40)))
	sb.WriteString(fmt.Sprintf("Chromaticity=%d\n", clamp(params.LabChromaticity, 0, 40)))
	sb.WriteString("LCurve=0;\n\n")

	// === VIBRANCE ===
	sb.WriteString("[Vibrance]\n")
	sb.WriteString("Enabled=true\n")
	sb.WriteString(fmt.Sprintf("Pastels=%d\n", clamp(params.VibPastels, -30, 50)))
	sb.WriteString(fmt.Sprintf("Saturated=%d\n", clamp(params.VibSaturated, -30, 30)))
	sb.WriteString("PSThreshold=0;75;\n")
	sb.WriteString("ProtectSkins=true\n")
	sb.WriteString("AvoidColorShift=true\n")
	sb.WriteString("PastSatTog=true\n\n")

	// === NOISE REDUCTION ===
	sb.WriteString("[Denoise]\n")
	sb.WriteString("Enabled=true\n")
	
	// Chrominance noise reduction is safe for all files (removes color blotches)
	chroma := 10 // Default safe value
	if params.NRChrominance > 0 {
		chroma = params.NRChrominance
	}
	sb.WriteString(fmt.Sprintf("Chrominance=%d\n", clamp(chroma, 0, 50)))
	sb.WriteString("ChrominanceMethod=Automatic\n")

	// Luminance noise reduction logic
	luma := 0
	if isRaw {
		// For RAW, use AI param
		luma = params.NRLuminance
	} else {
		// For JPEG, strictly avoid Luma denoise unless Dehaze is strong
		if params.DehazeStrength > 10 {
			// Dehaze introduces noise, counteract slightly
			luma = 5 
		}
	}
	sb.WriteString(fmt.Sprintf("Luminance=%d\n", clamp(luma, 0, 50)))
	sb.WriteString("\n")
	
	// === IMPULSE NOISE REDUCTION (Hot pixels / Salt & Pepper) ===
	// Always good to have on
	sb.WriteString("[Impulse Denoise]\n")
	sb.WriteString("Enabled=true\n")
	sb.WriteString("Threshold=50\n\n")

	// === RAW PROCESSING (use defaults) ===
	if isRaw {
		sb.WriteString("[RAW]\n")
		sb.WriteString("CA=true\n")
		sb.WriteString("CAAutoIterations=2\n")
		sb.WriteString("HotPixelFilter=true\n")
		sb.WriteString("DeadPixelFilter=true\n\n")

		sb.WriteString("[RAW Bayer]\n")
		sb.WriteString("Method=rcd\n")
		sb.WriteString("Border=4\n")
		sb.WriteString("ImageNum=1\n")
		sb.WriteString("CcSteps=0\n\n")
	}

	// === DISABLE ALL SHARPENING (causes artifacts) ===
	sb.WriteString("[Sharpening]\n")
	sb.WriteString("Enabled=false\n\n")

	sb.WriteString("[SharpenEdge]\n")
	sb.WriteString("Enabled=false\n\n")

	sb.WriteString("[SharpenMicro]\n")
	// For JPEG, Micro Contrast creates harsh edges/halos.
	// Limit strictly or disable if not RAW.
	if params.SharpenMicroStrength > 0 {
		strength := params.SharpenMicroStrength
		if !isRaw {
			// Cap strength for JPEGs to avoid artifacts
			if strength > 20 {
				strength = 20
			}
		}
		sb.WriteString("Enabled=true\n")
		sb.WriteString(fmt.Sprintf("Strength=%d\n", clamp(strength, 0, 100)))
		sb.WriteString(fmt.Sprintf("Contrast=%d\n", clamp(params.SharpenMicroContrast, 0, 100)))
		sb.WriteString(fmt.Sprintf("Uniformity=%d\n", clamp(params.SharpenMicroUniformity, 0, 100)))
	} else {
		sb.WriteString("Enabled=false\n")
	}
	sb.WriteString("\n")

	sb.WriteString("[PostDemosaicSharpening]\n")
	sb.WriteString("Enabled=false\n\n")

	sb.WriteString("[Dehaze]\n")
	if params.DehazeStrength != 0 {
		sb.WriteString("Enabled=true\n")
		sb.WriteString(fmt.Sprintf("Strength=%d\n", clamp(params.DehazeStrength, -100, 100)))
	} else {
		sb.WriteString("Enabled=false\n")
	}
	sb.WriteString("\n")

	// === COLOR MANAGEMENT ===
	sb.WriteString("[Color Management]\n")
	sb.WriteString("InputProfile=(cameraICC)\n")
	sb.WriteString("ToneCurve=false\n")
	sb.WriteString("ApplyLookTable=true\n")
	sb.WriteString("ApplyBaselineExposureOffset=true\n")
	sb.WriteString("ApplyHueSatMap=true\n")
	sb.WriteString("WorkingProfile=ProPhoto\n")
	sb.WriteString("OutputProfile=RTv4_sRGB\n")
	sb.WriteString("OutputProfileIntent=Relative\n")
	sb.WriteString("OutputBPC=true\n\n")

	// === RESIZE ===
	sb.WriteString("[Resize]\n")
	sb.WriteString("Enabled=false\n")

	return []byte(sb.String())
}

// GeneratePP3 creates a RawTherapee sidecar from Adobe-style GradingParams
// This is the fallback method with parameter conversion
func GeneratePP3(params models.GradingParams) []byte {
	// === EXPOSURE CONVERSION ===
	// Adobe Exposure range: -5.0 to +5.0 (stops)
	// RT Compensation is similar but RT renders slightly darker by default
	// Also need to factor in Whites/Blacks for overall brightness
	compensation := params.Exposure2012
	// RT needs a slight boost to match Adobe's default rendering
	compensation += 0.25
	// Factor in Whites - positive whites in Adobe add brightness
	if params.Whites2012 > 0 {
		compensation += float64(params.Whites2012) * 0.005 // Subtle addition
	}

	// === CONTRAST CONVERSION ===
	// Adobe Contrast: -100 to +100
	// RT Contrast: -100 to +100 (similar but applies differently)
	// RT's contrast is more aggressive, so we scale it down slightly
	contrast := int(float64(params.Contrast2012) * 0.8)

	// === WHITE BALANCE CONVERSION ===
	// Adobe Temperature: 2000-50000K
	// RT Temperature: same range, direct mapping
	temperature := params.Temperature
	if temperature == 0 {
		temperature = 5500 // Default daylight
	}

	// Adobe Tint: -150 to +150 (positive = magenta, negative = green)
	// RT Green: 0.02 to 10.0 (>1.0 = green, <1.0 = magenta)
	// Conversion: Adobe +150 -> RT ~0.5 (strong magenta)
	//             Adobe -150 -> RT ~2.0 (strong green)
	//             Adobe 0 -> RT 1.0 (neutral)
	tint := 1.0
	if params.Tint != 0 {
		// Map -150..+150 to approximately 2.0..0.5
		// Using exponential for more natural feel
		tint = 1.0 * math.Pow(0.5, float64(params.Tint)/150.0)
		tint = clampFloat(tint, 0.5, 2.0)
	}

	// === SATURATION CONVERSION ===
	// Adobe Saturation: -100 to +100
	// RT Saturation in [Exposure]: -100 to +100
	// Direct mapping but RT is slightly more aggressive
	saturation := int(float64(params.Saturation) * 0.85)

	// === BLACK POINT CONVERSION ===
	// Adobe Blacks: -100 to +100 (negative = darker blacks)
	// RT Black: 0 to 32768 (higher = more clipping)
	black := 0
	if params.Blacks2012 < 0 {
		// Adobe negative blacks = crush blacks = higher RT Black value
		black = -params.Blacks2012 * 50 // More conservative scaling
	}

	// === HIGHLIGHTS/SHADOWS RECOVERY ===
	// Adobe Highlights: -100 to +100 (negative = recover)
	// Adobe Shadows: -100 to +100 (positive = recover)
	// RT HighlightCompr: 0-500 (higher = more compression)
	// RT ShadowRecovery: 0-100 (higher = more recovery)
	// RT HighlightRecovery: 0-100 (different from HighlightCompr)

	highlightCompr := 0
	highlightRecovery := 0
	shadowRecovery := 0

	if params.Highlights2012 < 0 {
		// Negative highlights = recover highlights
		highlightCompr = -params.Highlights2012 * 3 // Scale to 0-300 range
		highlightRecovery = clamp(-params.Highlights2012, 0, 100)
	} else if params.Highlights2012 > 0 {
		// Positive highlights = boost brightness in highlights
		// This is handled through tone curve
	}

	if params.Shadows2012 > 0 {
		// Positive shadows = lift shadows
		shadowRecovery = params.Shadows2012
	} else if params.Shadows2012 < 0 {
		// Negative shadows = deepen shadows
		// Factor into black point
		black += -params.Shadows2012 * 20
	}

	// === CLARITY/TEXTURE CONVERSION ===
	// Adobe Clarity: -100 to +100 (local contrast)
	// Adobe Texture: -100 to +100 (fine detail)
	// RT SharpenMicro is closest to Clarity
	// RT uses different algorithm, so scale accordingly

	// Clarity -> SharpenMicro (local contrast)
	clarityStrength := 0
	clarityContrast := 0
	if params.Clarity2012 > 0 {
		clarityStrength = params.Clarity2012 / 2 // 0-50 range
		clarityContrast = params.Clarity2012 / 4 // 0-25 range
	} else if params.Clarity2012 < 0 {
		// Negative clarity = soften, we can't really do this with SharpenMicro
		// Just leave it at 0
	}

	// Texture adds to clarity effect
	if params.Texture > 0 {
		clarityStrength += params.Texture / 4
	}

	// === VIBRANCE/SATURATION FINE TUNING ===
	// Adobe Vibrance: -100 to +100 (protects skin tones)
	// RT Vibrance Pastels/Saturated: -100 to +100
	vibPastels := params.Vibrance       // Direct for less saturated colors
	vibSaturated := params.Vibrance / 2 // Half effect on already saturated

	// === DEHAZE CONVERSION ===
	// Adobe Dehaze: -100 to +100
	// RT Dehaze Strength: -100 to +100
	// Direct mapping, but RT's dehaze is more aggressive
	dehazeStrength := int(float64(params.Dehaze) * 0.7)

	// === SHARPENING CONVERSION ===
	// Adobe Sharpness: 0 to 150
	// RT Sharpen Amount: 0 to 1000
	// RT uses deconvolution which is different from Adobe's USM
	sharpenAmount := params.Sharpness * 3 // Scale to ~0-450 range
	if sharpenAmount == 0 && params.Sharpness == 0 {
		sharpenAmount = 100 // Sensible default
	}

	// === LAB ADJUSTMENTS ===
	// These don't have direct Adobe equivalents but help match the look
	labBrightness := 0
	labContrast := params.Clarity2012 / 10 // Subtle contrast boost from clarity
	labChromaticity := params.Vibrance / 5 // Subtle color boost from vibrance

	// Factor in Whites/Blacks for Lab brightness
	if params.Whites2012 != 0 || params.Blacks2012 != 0 {
		// Whites brighten, Blacks darken
		labBrightness = (params.Whites2012 - params.Blacks2012) / 10
		labBrightness = clamp(labBrightness, -20, 20)
	}

	// === BUILD TONE CURVE FROM ADOBE PARAMETERS ===
	// Adobe's tone curve is implicitly defined by Exposure, Contrast, Highlights, Shadows, Whites, Blacks
	// We need to approximate this with RT's tone curve
	toneCurve := buildToneCurveFromAdobe(params)

	pp3 := &models.PP3Params{
		Compensation:           compensation,
		Contrast:               contrast,
		Saturation:             saturation,
		Black:                  clamp(black, 0, 3000),
		HighlightCompr:         clamp(highlightCompr, 0, 300),
		ShadowRecovery:         clamp(shadowRecovery, 0, 100),
		HighlightRecovery:      clamp(highlightRecovery, 0, 100),
		Temperature:            temperature,
		Tint:                   tint,
		LabBrightness:          clamp(labBrightness, -50, 50),
		LabContrast:            clamp(labContrast, -30, 30),
		LabChromaticity:        clamp(labChromaticity, -30, 30),
		SharpenMicroStrength:   clamp(clarityStrength, 0, 60),
		SharpenMicroContrast:   clamp(clarityContrast, 0, 40),
		SharpenMicroUniformity: 50,
		DehazeStrength:         clamp(dehazeStrength, -70, 70),
		VibPastels:             clamp(vibPastels, -100, 100),
		VibSaturated:           clamp(vibSaturated, -100, 100),
		SharpenAmount:          clamp(sharpenAmount, 0, 500),
		SharpenRadius:          0.75,
		NRLuminance:            params.LuminanceSmoothing,
		NRChrominance:          params.ColorNoiseReduction,
		VignetteAmount:         params.PostCropVignetteAmount,
		ToneCurve:              toneCurve,
	}

	// Color toning from split toning
	if params.SplitToningShadowSaturation > 0 || params.SplitToningHighlightSaturation > 0 {
		shR, shG, shB := hueToRGB(params.SplitToningShadowHue, params.SplitToningShadowSaturation)
		hlR, hlG, hlB := hueToRGB(params.SplitToningHighlightHue, params.SplitToningHighlightSaturation)
		pp3.ColorToningShadowR = shR
		pp3.ColorToningShadowG = shG
		pp3.ColorToningShadowB = shB
		pp3.ColorToningHighlightR = hlR
		pp3.ColorToningHighlightG = hlG
		pp3.ColorToningHighlightB = hlB
		pp3.ColorToningBalance = 50 + params.SplitToningBalance/2
	}

	// Assume RAW for Adobe conversion fallback, as Adobe params (like Temp K) are RAW-centric
	return GeneratePP3FromNative(pp3, true)
}

// buildToneCurveFromAdobe creates a tone curve that approximates Adobe's parametric adjustments
func buildToneCurveFromAdobe(params models.GradingParams) [][]float64 {
	// Adobe's tone controls affect different parts of the curve:
	// - Blacks: affects the darkest tones (0-25%)
	// - Shadows: affects dark tones (10-40%)
	// - Highlights: affects bright tones (60-90%)
	// - Whites: affects the brightest tones (75-100%)
	// - Contrast: affects the S-curve steepness

	// Start with 5-point linear curve
	curve := [][]float64{
		{0.0, 0.0},   // Black point
		{0.25, 0.25}, // Shadows region
		{0.5, 0.5},   // Midpoint
		{0.75, 0.75}, // Highlights region
		{1.0, 1.0},   // White point
	}

	// Apply Blacks adjustment (affects point 0 and 1)
	if params.Blacks2012 != 0 {
		// Positive blacks = lift blacks, negative = crush
		blacksOffset := float64(params.Blacks2012) / 200.0 // ±0.5 max
		curve[0][1] = clampFloat(curve[0][1]+blacksOffset*0.5, 0.0, 0.15)
		curve[1][1] = clampFloat(curve[1][1]+blacksOffset*0.3, 0.05, 0.4)
	}

	// Apply Shadows adjustment (affects points 1 and 2)
	if params.Shadows2012 != 0 {
		// Positive shadows = lift shadows
		shadowsOffset := float64(params.Shadows2012) / 150.0
		curve[1][1] = clampFloat(curve[1][1]+shadowsOffset*0.15, 0.1, 0.45)
		curve[2][1] = clampFloat(curve[2][1]+shadowsOffset*0.05, 0.35, 0.65)
	}

	// Apply Highlights adjustment (affects points 3 and 4)
	if params.Highlights2012 != 0 {
		// Negative highlights = recover/compress highlights
		highlightsOffset := float64(params.Highlights2012) / 150.0
		curve[3][1] = clampFloat(curve[3][1]+highlightsOffset*0.1, 0.6, 0.9)
		curve[4][1] = clampFloat(curve[4][1]+highlightsOffset*0.05, 0.85, 1.0)
	}

	// Apply Whites adjustment (affects points 3 and 4)
	if params.Whites2012 != 0 {
		// Positive whites = push whites brighter
		whitesOffset := float64(params.Whites2012) / 200.0
		curve[3][1] = clampFloat(curve[3][1]+whitesOffset*0.05, 0.6, 0.95)
		curve[4][1] = clampFloat(curve[4][1]+whitesOffset*0.1, 0.9, 1.0)
	}

	// Apply Contrast (S-curve adjustment)
	if params.Contrast2012 != 0 {
		// Positive contrast = steeper S-curve
		contrastFactor := float64(params.Contrast2012) / 200.0
		// Pull shadows down and highlights up
		curve[1][1] = clampFloat(curve[1][1]-contrastFactor*0.08, 0.05, 0.4)
		curve[3][1] = clampFloat(curve[3][1]+contrastFactor*0.08, 0.6, 0.95)
	}

	return curve
}

// hueToRGB converts Adobe's Hue (0-360) and Saturation (0-100) to RT's RGB values
func hueToRGB(hue, saturation int) (r, g, b int) {
	if saturation == 0 {
		return 0, 0, 0
	}

	h := float64(hue) / 360.0
	s := float64(saturation) / 100.0

	var rf, gf, bf float64
	h6 := h * 6.0
	i := int(h6) % 6
	f := h6 - float64(int(h6))

	switch i {
	case 0:
		rf, gf, bf = 1, f, 0
	case 1:
		rf, gf, bf = 1-f, 1, 0
	case 2:
		rf, gf, bf = 0, 1, f
	case 3:
		rf, gf, bf = 0, 1-f, 1
	case 4:
		rf, gf, bf = f, 0, 1
	case 5:
		rf, gf, bf = 1, 0, 1-f
	}

	scale := s * 100.0
	r = int((rf - 0.5) * 2 * scale)
	g = int((gf - 0.5) * 2 * scale)
	b = int((bf - 0.5) * 2 * scale)

	return clamp(r, -100, 100), clamp(g, -100, 100), clamp(b, -100, 100)
}
