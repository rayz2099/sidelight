package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"sidelight/pkg/models"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client    *genai.Client
	model     *genai.GenerativeModel
	modelName string
}

// bearerTokenTransport adds Bearer token authentication for proxy endpoints
type bearerTokenTransport struct {
	apiKey    string
	transport http.RoundTripper
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.transport.RoundTrip(newReq)
}

func NewGeminiClient(ctx context.Context, apiKey, endpoint string, modelName string) (*GeminiClient, error) {
	opts := []option.ClientOption{option.WithAPIKey(apiKey)}
	// If using a custom endpoint (proxy), add Bearer token auth
	//fmt.Println("apikey:", apiKey)
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
		opts = append(opts, option.WithHTTPClient(&http.Client{
			Transport: &bearerTokenTransport{
				apiKey:    apiKey,
				transport: http.DefaultTransport,
			},
		}))
	}

	if len(modelName) == 0 {
		modelName = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel(modelName)
	return &GeminiClient{
		client:    client,
		model:     model,
		modelName: modelName,
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
	"fuji":      "Mimic Fujifilm. High transparency, emphasis on greens and natural skin tones. Punchy contrast and rich details.",
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

func (g *GeminiClient) AnalyzeImageLR(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.GradingParams, error) {
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

// pp3Styles contains RawTherapee-specific style instructions with RT parameter guidance
var pp3Styles = map[string]string{
	"natural": `Natural look: accurate colors, balanced exposure.
compensation=0.45, contrast=12, lab_contrast=20, lab_chromaticity=20, nr_luminance=10, nr_chrominance=15`,

	"vivid": `Vibrant colors, punchy contrast.
compensation=0.48, contrast=18, lab_contrast=25, lab_chromaticity=40, vib_pastels=30, nr_luminance=10`,

	"film": `Film look: warm tones, lifted blacks, soft roll-off.
compensation=0.50, contrast=12, lab_chromaticity=25, temperature=5800, tint=1.02, nr_luminance=5`,

	"kodak": `Kodak Portra style: warm, creamy skin tones, slight overexposure look.
compensation=0.52, contrast=12, lab_chromaticity=22, temperature=5600, tint=0.98, vib_pastels=20`,

	"fuji": `Fujifilm style: high transparency, punchy greens, rich details.
compensation=0.48, contrast=15, lab_contrast=25, lab_chromaticity=35, temperature=5400, tint=1.02, dehaze_strength=15, sharpenmicro_strength=20`,

	"cinematic": `Movie look: teal/orange vibe, controlled contrast, moody.
compensation=0.42, contrast=18, lab_contrast=22, lab_chromaticity=20, vib_pastels=15`,

	"landscape": `Landscape: clear sky, enhanced foliage, detailed.
compensation=0.40, contrast=18, lab_contrast=25, lab_chromaticity=35, vib_pastels=25, nr_luminance=10`,

	"portrait": `Portrait: flattering skin tones, soft contrast, reduced texture.
compensation=0.48, contrast=10, lab_contrast=15, lab_chromaticity=18, vib_pastels=10, nr_luminance=20, nr_chrominance=20`,

	"bw": `Black and white: strong contrast, rich tonal range.
compensation=0.45, contrast=25, saturation=-100, lab_contrast=35, nr_luminance=15`,

	"matte": `Matte/faded look: lifted blacks, low contrast, desaturated.
compensation=0.52, contrast=5, lab_contrast=10, lab_chromaticity=10, black=0`,
}

const pp3SystemInstruction = `You are an expert photo color grader for RawTherapee. 
Analyze the image and output professional color grading parameters in JSON format.

Key Parameters to include:
- compensation: (0.35 to 0.60) controls brightness.
- contrast: (0 to 30)
- saturation: (-100 to 20)
- black: (0 to 100)
- highlight_compr: (0 to 100)
- temperature: (2000 to 10000)
- tint: (0.8 to 1.2)
- lab_brightness, lab_contrast, lab_chromaticity: (-20 to 20)
- dehaze_strength: (0 to 30) for transparency and clarity.
- sharpenmicro_strength: (0 to 40) for local contrast/clarity.
- nr_luminance, nr_chrominance: (0 to 40) for noise reduction.

Output ONLY the JSON object.`

func (g *GeminiClient) AnalyzeImageForPP3(ctx context.Context, imageData []byte, metadata models.Metadata, opts AnalysisOptions) (*models.PP3Params, error) {
	// Use RT-specific styles instead of generic Adobe styles
	styleInstruction := pp3Styles["natural"]
	if instruction, ok := pp3Styles[opts.Style]; ok {
		styleInstruction = instruction
	}

	metadataInfo := fmt.Sprintf(`Image Metadata:
- Camera: %s %s
- ISO: %d
- Aperture: %s
- Shutter Speed: %s`, metadata.Make, metadata.Model, metadata.ISO, metadata.Aperture, metadata.ShutterSpeed)

	// Build user instruction section
	userInstructions := ""
	if opts.UserPrompt != "" {
		userInstructions = fmt.Sprintf("\n\nUser Goal: %s", opts.UserPrompt)
	}

	fullPrompt := fmt.Sprintf(`%s

%s
    
Desired Style: %s
%s

Analyze the image and generate the JSON for RawTherapee parameters.`,
		pp3SystemInstruction, metadataInfo, styleInstruction, userInstructions)

	prompt := []genai.Part{
		genai.ImageData("image/jpeg", imageData),
		genai.Text(fullPrompt),
		genai.Text("Output the JSON object now."),
	}

	var resp *genai.GenerateContentResponse
	var err error

	// Retry logic: try up to 3 times
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = g.model.GenerateContent(ctx, prompt...)
		if err != nil {
			if attempt == 3 {
				return nil, fmt.Errorf("gemini generation failed after 3 attempts: %w", err)
			}
			continue
		}

		if resp == nil || len(resp.Candidates) == 0 {
			if attempt == 3 {
				feedback := "<nil>"
				if resp != nil {
					feedback = fmt.Sprintf("%+v", resp.PromptFeedback)
				}
				return nil, fmt.Errorf("no candidates returned after 3 attempts (imageSize=%d, feedback=%s)", len(imageData), feedback)
			}
			continue
		}
		
		// Success
		break
	}

	// Check if candidate was blocked

	// Check if candidate was blocked
	if resp.Candidates[0].FinishReason != 0 && resp.Candidates[0].FinishReason != 1 {
		return nil, fmt.Errorf("candidate finished with reason: %v (feedback=%+v)", resp.Candidates[0].FinishReason, resp.PromptFeedback)
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("candidate has no content parts")
	}

	part := resp.Candidates[0].Content.Parts[0]
	text, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected response part type: %T", part)
	}

	cleanJSON := strings.TrimSpace(string(text))
	if idx := strings.Index(cleanJSON, "{"); idx != -1 {
		cleanJSON = cleanJSON[idx:]
	}
	if idx := strings.LastIndex(cleanJSON, "}"); idx != -1 {
		cleanJSON = cleanJSON[:idx+1]
	}

	var params models.PP3Params
	if err := json.Unmarshal([]byte(cleanJSON), &params); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w (raw: %s)", err, cleanJSON)
	}

	return &params, nil
}
