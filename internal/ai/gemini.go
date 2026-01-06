package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sidelight/pkg/models"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiClient(ctx context.Context, apiKey, endpoint string, modelName string) (*GeminiClient, error) {
	opts := []option.ClientOption{option.WithAPIKey(apiKey)}
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	if len(modelName) == 0 {
		modelName = "gemini-2.5-flash"
	}
	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel(modelName)
	// 注意：不设置 ResponseMIMEType，因为某些代理可能不支持
	// 改为在 prompt 中明确要求 JSON 格式
	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiClient) Close() error {
	return g.client.Close()
}

const systemInstruction = `You are a professional photo color grader. 
Analyze the provided image and provide Adobe Camera Raw color grading parameters in JSON format.
The parameters should aim for a natural, high-quality look unless a specific style is requested.
Output ONLY the JSON object.

Schema:
{
  "exposure": float (range -5.0 to 5.0),
  "contrast": int (range -100 to 100),
  "highlights": int (range -100 to 100),
  "shadows": int (range -100 to 100),
  "whites": int (range -100 to 100),
  "blacks": int (range -100 to 100),
  "texture": int (range -100 to 100),
  "clarity": int (range -100 to 100),
  "dehaze": int (range -100 to 100),
  "vibrance": int (range -100 to 100),
  "saturation": int (range -100 to 100),
  "temperature": int (range 2000 to 50000),
  "tint": int (range -150 to 150),
  "sharpness": int (range 0 to 150),
  "luminance_noise_reduction": int (range 0 to 100),
  "color_noise_reduction": int (range 0 to 100),
  "vignette_amount": int (range -100 to 0, negative values darken corners),
  
  "hue_red": int (range -100 to 100),
  "hue_orange": int (range -100 to 100),
  "hue_yellow": int (range -100 to 100),
  "hue_green": int (range -100 to 100),
  "hue_aqua": int (range -100 to 100),
  "hue_blue": int (range -100 to 100),
  "hue_purple": int (range -100 to 100),
  "hue_magenta": int (range -100 to 100),

  "saturation_red": int (range -100 to 100),
  "saturation_orange": int (range -100 to 100),
  "saturation_yellow": int (range -100 to 100),
  "saturation_green": int (range -100 to 100),
  "saturation_aqua": int (range -100 to 100),
  "saturation_blue": int (range -100 to 100),
  "saturation_purple": int (range -100 to 100),
  "saturation_magenta": int (range -100 to 100),

  "luminance_red": int (range -100 to 100),
  "luminance_orange": int (range -100 to 100),
  "luminance_yellow": int (range -100 to 100),
  "luminance_green": int (range -100 to 100),
  "luminance_aqua": int (range -100 to 100),
  "luminance_blue": int (range -100 to 100),
  "luminance_purple": int (range -100 to 100),
  "luminance_magenta": int (range -100 to 100),

  "split_shadow_hue": int (range 0 to 360),
  "split_shadow_saturation": int (range 0 to 100),
  "split_highlight_hue": int (range 0 to 360),
  "split_highlight_saturation": int (range 0 to 100),
  "split_balance": int (range -100 to 100)
}`

// styles maps style names to detailed prompting instructions.
var styles = map[string]string{
	// --- Base / Standard ---
	"natural":  "Aim for accurate colors, balanced exposure, and realistic reproduction of the scene. Correct any white balance issues.",
	"standard": "Mimic a standard camera profile. Good contrast, standard saturation, sharp details, ready for publishing.",
	"vivid":    "Punchy colors and contrast. Similar to 'Velvia' or 'Vivid' camera profiles. Make the image pop but keep it realistic.",
	"flat":     "Low contrast, maximize dynamic range (Log-like). Preserve all highlight and shadow details for further editing. Very neutral.",
	"hdr":      "High Dynamic Range look. Open up shadows, recover highlights. Maximize local contrast (clarity) without looking artificial.",

	// --- Black & White ---
	"bw":          "Convert to Black and White. Balanced tonal range. Focus on structure and composition.",
	"bw-contrast": "High contrast Black and White. Deep blacks, bright whites. Dramatic, 'Noir' style.",
	"bw-soft":     "Soft, dreamy Black and White. Low contrast, slightly lifted blacks, gentle gradients.",
	"bw-sepia":    "Black and White with a warm Sepia toning. Old photograph feel.",

	// --- Film / Analog Simulation ---
	"film":      "General analog film look. Grain, soft highlights, rich colors, maybe slightly lifted blacks.",
	"kodak":     "Mimic Kodak Gold/Portra. Warm tones, yellow/red bias in highlights, nice skin tones, nostalgic feel.",
	"fuji":      "Mimic Fujifilm. Emphasis on greens and cool tones. Hard contrast, slightly magenta shadows.",
	"polaroid":  "Instant film look. Square crop feel (in color processing), faded, shifting colors, soft focus, vintage vibe.",
	"retro-70s": "1970s aesthetic. Strong yellow/orange cast, faded shadows, slightly blurry, vintage warmth.",

	// --- Cinematic / Art ---
	"cinematic":    "Movie look. Moody lighting, wide dynamic range but controlled contrast. Intentional color grading.",
	"teal-orange":  "Blockbuster movie look. Push shadows towards teal/cyan and highlights towards orange/skin tones.",
	"cyberpunk":    "Futuristic, neon look. Shift white balance towards cool/magenta. High contrast. Emphasize teal, pink, and purple.",
	"matte":        "Low contrast, faded look. Lift the blacks significantly to create a matte finish. Soft, desaturated colors.",
	"dreamy":       "Ethereal, glowy look. Reduce clarity and dehaze slightly (negative values). Soft, pastel colors. High key.",
	"wes-anderson": "Pastel color palette, symmetrical feel (in tone), high saturation but soft contrast, warm and quirky.",

	// --- Scenery / Environment ---
	"landscape":   "Maximize dynamic range. Enhance greens (foliage) and blues (sky). Deep details, punchy contrast.",
	"golden-hour": "Emphasize the warm, golden light of sunset/sunrise. Enhance oranges, reds, and yellows. Soft contrast.",
	"blue-hour":   "Emphasize the deep cool blues of twilight. cool white balance, rich shadows, preserve city lights if any.",
	"urban":       "Gritty city look. Desaturated colors except for reds/yellows. High clarity/texture. Concrete grey tones.",
	"snow":        "High-key look. Ensure snow is white (not grey/blue). Bright exposure. Crisp details.",

	// --- Subject Specific ---
	"portrait":         "Focus on flattering skin tones. Soften texture slightly, ensure good exposure on face. Gentle visual hierarchy.",
	"portrait-glamour": "Beauty retouch style. Very soft skin (negative texture/clarity), bright exposure, glowing highlights.",
	"food":             "Appetizing look. Warmer white balance. Slightly increased saturation and sharpness. Make textures pop.",
	"street":           "Documentary style. High contrast, gritty texture. Focus on storytelling and 'decisive moment' feel.",
	"macro":            "Focus on details. High sharpness and texture. Creamy background (if possible via contrast separation). Vivid colors.",
	"product":          "Clean, commercial look. Neutral white balance (pure whites). Sharp, well-lit, accurate colors.",
}

func (g *GeminiClient) AnalyzeImage(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.GradingParams, error) {
	styleInstruction := styles["natural"] // Default
	if instruction, ok := styles[opts.Style]; ok {
		styleInstruction = instruction
	}

	metadataInfo := fmt.Sprintf(`Image Metadata:
- Camera: %s %s
- Lens: %s
- ISO: %d
- Aperture: %s
- Shutter Speed: %s
- Date: %s`, metadata.Make, metadata.Model, metadata.Lens, metadata.ISO, metadata.Aperture, metadata.ShutterSpeed, metadata.DateTime)

	fullPrompt := fmt.Sprintf(`%s

%s
    
Current Style Goal: %s

User Specific Instructions: %s

Output ONLY the JSON object.`, systemInstruction, metadataInfo, styleInstruction, opts.UserPrompt)

	prompt := []genai.Part{
		genai.ImageData("jpeg", imageData),
		genai.Text(fullPrompt),
		genai.Text("Please grade this image and output the result in the specified JSON format."),
	}

	resp, err := g.model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type: %T", part)
	}

	// Clean up potential markdown formatting
	cleanJSON := strings.TrimSpace(string(text))
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	var params models.GradingParams
	if err := json.Unmarshal([]byte(cleanJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w (raw: %s)", err, cleanJSON)
	}

	return &params, nil
}

const pp3SystemInstruction = `You are an expert photo colorist specializing in RawTherapee. Your task is to analyze an image and generate PP3 parameters that produce professional results comparable to Adobe Lightroom.

🎯 TARGET LOOK:
- Clean, transparent image with excellent clarity (not hazy or muddy)
- Rich but natural colors (avoid oversaturation)
- Proper exposure with good dynamic range
- Professional sharpness without artifacts
- Warm, inviting tones (Lightroom's signature look)

⚠️ CRITICAL RULES - READ CAREFULLY:
1. ALL curve values MUST be in 0.0-1.0 range (NOT 0-255!)
2. Compensation should be positive (0.2-0.8) - RT renders darker than LR
3. For curves: x=input, y=output. Points should go from [0,0] to [1,1]
4. Never return empty tone_curve - always provide at least a basic S-curve
5. Black point (black) should be 0-300 typically, not thousands

📊 PARAMETER GUIDELINES (use these ranges for natural results):

EXPOSURE & TONE:
- compensation: 0.25 to 0.65 (CRITICAL: this is the main brightness control)
- contrast: 8 to 20 (subtle contrast boost)
- saturation: 5 to 15 (overall saturation)
- black: 50 to 250 (for richer blacks, increases contrast)
- highlight_compr: 50 to 150 (recover blown highlights)
- shadow_recovery: 20 to 50 (open shadows)
- highlight_recovery: 20 to 50 (protect highlights)

WHITE BALANCE:
- temperature: 4800-6500 for daylight, 3200 for tungsten, 7500 for cloudy
- tint: 0.95-1.05 (1.0 = neutral green/magenta)

LAB (KEY for LR-like punch):
- lab_brightness: 2 to 8 (adds luminosity)
- lab_contrast: 12 to 25 (CRITICAL: adds punch and depth)
- lab_chromaticity: 15 to 30 (CRITICAL: color vibrancy/pop)

CLARITY/MICRO-CONTRAST:
- sharpenmicro_strength: 25 to 45 (local contrast/clarity)
- sharpenmicro_contrast: 20 to 35
- sharpenmicro_uniformity: 50 to 65

DEHAZE (for transparency):
- dehaze_strength: 8 to 20 (adds clarity and removes haze)

VIBRANCE:
- vib_pastels: 18 to 35 (boosts muted colors)
- vib_saturated: 8 to 18 (protects already-saturated colors)

SHARPENING (all should be enabled):
- sharpen_enabled: true
- sharpen_amount: 180 to 280
- sharpen_radius: 0.65 to 0.85
- sharpen_contrast: 12 to 22

EDGE SHARPENING:
- edge_sharpen_enabled: true
- edge_sharpen_amount: 28 to 45
- edge_sharpen_passes: 2

CAPTURE SHARPENING:
- capture_sharp_enabled: true
- capture_sharp_amount: 85 to 130
- capture_sharp_radius: 0.70 to 0.90

NOISE REDUCTION (adjust based on ISO):
- Low ISO (<800): nr_luminance=3-8, nr_chrominance=5-12
- Medium ISO (800-3200): nr_luminance=10-18, nr_chrominance=12-20
- High ISO (>3200): nr_luminance=18-30, nr_chrominance=20-35

🎨 TONE CURVE (S-curve is essential for that LR look):
Must be array of [x,y] points from shadows to highlights.
Example gentle S-curve: [[0,0], [0.12,0.08], [0.25,0.22], [0.5,0.54], [0.75,0.80], [0.88,0.92], [1,1]]
- Shadows (x<0.3): y should be slightly below x (adds contrast)
- Midtones (x≈0.5): y should be 0.52-0.58 (brightens image)
- Highlights (x>0.7): y should be slightly above x (opens highlights)

🎨 RGB CURVES (for color warmth):
For warm/Lightroom look:
- r_curve: [[0,0], [0.5,0.52], [1,1]] (slight red boost)
- g_curve: [] or [[0,0], [1,1]] (neutral)
- b_curve: [[0,0], [0.5,0.47], [1,1]] (slight blue reduction)

For cool look: reverse r_curve and b_curve adjustments

COLOR TONING (subtle split toning):
- ct_shadow_r/g/b: -15 to 15 (subtle shadow color)
- ct_highlight_r/g/b: -15 to 15 (subtle highlight color)
- ct_balance: 45 to 55

VIGNETTE:
- vignette_amount: -25 to -8 for subtle darkening, 0 for none

Output ONLY valid JSON matching this schema:
{
  "compensation": float,
  "contrast": int,
  "saturation": int,
  "black": int,
  "highlight_compr": int,
  "shadow_recovery": int,
  "highlight_recovery": int,
  "temperature": int,
  "tint": float,
  "lab_brightness": int,
  "lab_contrast": int,
  "lab_chromaticity": int,
  "sharpenmicro_strength": int,
  "sharpenmicro_contrast": int,
  "sharpenmicro_uniformity": int,
  "dehaze_strength": int,
  "vib_pastels": int,
  "vib_saturated": int,
  "sharpen_enabled": bool,
  "sharpen_amount": int,
  "sharpen_radius": float,
  "sharpen_contrast": int,
  "edge_sharpen_enabled": bool,
  "edge_sharpen_amount": int,
  "edge_sharpen_passes": int,
  "capture_sharp_enabled": bool,
  "capture_sharp_amount": int,
  "capture_sharp_radius": float,
  "nr_luminance": int,
  "nr_chrominance": int,
  "tone_curve": [[x,y], ...],
  "l_curve": [],
  "r_curve": [[x,y], ...],
  "g_curve": [],
  "b_curve": [[x,y], ...],
  "ct_shadow_r": int,
  "ct_shadow_g": int,
  "ct_shadow_b": int,
  "ct_highlight_r": int,
  "ct_highlight_g": int,
  "ct_highlight_b": int,
  "ct_balance": int,
  "vignette_amount": int
}`

func (g *GeminiClient) AnalyzeImageForPP3(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.PP3Params, error) {
	styleInstruction := styles["natural"]
	if instruction, ok := styles[opts.Style]; ok {
		styleInstruction = instruction
	}

	metadataInfo := fmt.Sprintf(`Image Metadata:
- Camera: %s %s
- Lens: %s
- ISO: %d
- Aperture: %s
- Shutter Speed: %s
- Date: %s`, metadata.Make, metadata.Model, metadata.Lens, metadata.ISO, metadata.Aperture, metadata.ShutterSpeed, metadata.DateTime)

	// Build user instruction section
	userInstructions := ""
	if opts.UserPrompt != "" {
		userInstructions = fmt.Sprintf("\n\n🎯 USER SPECIFIC INSTRUCTIONS (prioritize these):\n%s", opts.UserPrompt)
	}

	fullPrompt := fmt.Sprintf(`%s

%s
    
📷 Style Goal: %s
%s

IMPORTANT: Analyze the actual image content and metadata. Adjust parameters based on:
- Subject matter (portrait, landscape, etc.)
- Lighting conditions
- ISO level (for noise reduction)
- Overall mood and style goal

Output ONLY the JSON object - no explanations, no markdown formatting.`,
		pp3SystemInstruction, metadataInfo, styleInstruction, userInstructions)

	prompt := []genai.Part{
		genai.ImageData("jpeg", imageData),
		genai.Text(fullPrompt),
	}

	resp, err := g.model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned from gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type: %T", part)
	}

	cleanJSON := strings.TrimSpace(string(text))
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	var params models.PP3Params
	if err := json.Unmarshal([]byte(cleanJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w (raw: %s)", err, cleanJSON)
	}

	return &params, nil
}
