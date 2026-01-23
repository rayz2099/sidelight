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
	// --- Base / Standard ---
	"natural": `Natural look: accurate colors, balanced exposure.
<<<<<<< Updated upstream
<<<<<<< Updated upstream
compensation=0.45, contrast=12, lab_contrast=20, lab_chromaticity=20, nr_luminance=10, nr_chrominance=15`,
||||||| Stash base
compensation=0.45, contrast=10, lab_contrast=15, lab_chromaticity=15, nr_luminance=10, nr_chrominance=15`,
=======
compensation=0.45, contrast=12, lab_contrast=15, lab_chromaticity=20, nr_luminance=10, nr_chrominance=15`,
>>>>>>> Stashed changes
||||||| Stash base
compensation=0.45, contrast=10, lab_contrast=15, lab_chromaticity=15, nr_luminance=10, nr_chrominance=15`,
=======
compensation=0.45, contrast=12, lab_contrast=15, lab_chromaticity=20, nr_luminance=10, nr_chrominance=15`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"vivid": `Vibrant colors, punchy contrast.
compensation=0.48, contrast=18, lab_contrast=25, lab_chromaticity=40, vib_pastels=30, nr_luminance=10`,
||||||| Stash base
	"vivid": `Vibrant colors, punchy contrast.
compensation=0.48, contrast=15, lab_contrast=20, lab_chromaticity=25, vib_pastels=20, nr_luminance=10`,
=======
	"standard": `Standard camera profile. Good contrast, standard saturation, sharp details.
compensation=0.48, contrast=15, lab_contrast=18, lab_chromaticity=18, vib_pastels=12, nr_luminance=12`,
>>>>>>> Stashed changes
||||||| Stash base
	"vivid": `Vibrant colors, punchy contrast.
compensation=0.48, contrast=15, lab_contrast=20, lab_chromaticity=25, vib_pastels=20, nr_luminance=10`,
=======
	"standard": `Standard camera profile. Good contrast, standard saturation, sharp details.
compensation=0.48, contrast=15, lab_contrast=18, lab_chromaticity=18, vib_pastels=12, nr_luminance=12`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"film": `Film look: warm tones, lifted blacks, soft roll-off.
compensation=0.50, contrast=12, lab_chromaticity=25, temperature=5800, tint=1.02, nr_luminance=5`,
||||||| Stash base
	"film": `Film look: warm tones, lifted blacks, soft roll-off.
compensation=0.50, contrast=12, lab_chromaticity=20, temperature=5800, tint=1.02, nr_luminance=5`,
=======
	"vivid": `Vibrant Velvia-like colors, punchy contrast.
compensation=0.50, contrast=18, lab_contrast=22, lab_chromaticity=30, vib_pastels=25, nr_luminance=8`,
>>>>>>> Stashed changes
||||||| Stash base
	"film": `Film look: warm tones, lifted blacks, soft roll-off.
compensation=0.50, contrast=12, lab_chromaticity=20, temperature=5800, tint=1.02, nr_luminance=5`,
=======
	"vivid": `Vibrant Velvia-like colors, punchy contrast.
compensation=0.50, contrast=18, lab_contrast=22, lab_chromaticity=30, vib_pastels=25, nr_luminance=8`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"kodak": `Kodak Portra style: warm, creamy skin tones, slight overexposure look.
compensation=0.52, contrast=12, lab_chromaticity=22, temperature=5600, tint=0.98, vib_pastels=20`,
||||||| Stash base
	"kodak": `Kodak Portra style: warm, creamy skin tones, slight overexposure look.
compensation=0.52, contrast=10, lab_chromaticity=18, temperature=5600, tint=0.98, vib_pastels=15`,
=======
	"flat": `Low contrast Log-like for maximum dynamic range.
compensation=0.38, contrast=3, lab_contrast=8, lab_chromaticity=12, black=0, nr_luminance=8`,
>>>>>>> Stashed changes
||||||| Stash base
	"kodak": `Kodak Portra style: warm, creamy skin tones, slight overexposure look.
compensation=0.52, contrast=10, lab_chromaticity=18, temperature=5600, tint=0.98, vib_pastels=15`,
=======
	"flat": `Low contrast Log-like for maximum dynamic range.
compensation=0.38, contrast=3, lab_contrast=8, lab_chromaticity=12, black=0, nr_luminance=8`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"fuji": `Fujifilm style: high transparency, punchy greens, rich details.
compensation=0.48, contrast=15, lab_contrast=25, lab_chromaticity=35, temperature=5400, tint=1.02, dehaze_strength=15, sharpenmicro_strength=20`,
||||||| Stash base
	"fuji": `Fujifilm style: cool shadows, high contrast, punchy greens.
compensation=0.45, contrast=18, lab_contrast=22, lab_chromaticity=20, temperature=5200, tint=0.96`,
=======
	"hdr": `High Dynamic Range look with local contrast.
compensation=0.35, contrast=8, lab_contrast=25, shadow_recovery=40, highlight_compr=60, vib_pastels=15`,
>>>>>>> Stashed changes
||||||| Stash base
	"fuji": `Fujifilm style: cool shadows, high contrast, punchy greens.
compensation=0.45, contrast=18, lab_contrast=22, lab_chromaticity=20, temperature=5200, tint=0.96`,
=======
	"hdr": `High Dynamic Range look with local contrast.
compensation=0.35, contrast=8, lab_contrast=25, shadow_recovery=40, highlight_compr=60, vib_pastels=15`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"cinematic": `Movie look: teal/orange vibe, controlled contrast, moody.
compensation=0.42, contrast=18, lab_contrast=22, lab_chromaticity=20, vib_pastels=15`,
||||||| Stash base
	"cinematic": `Movie look: teal/orange vibe, controlled contrast, moody.
compensation=0.42, contrast=15, lab_contrast=18, lab_chromaticity=15, vib_pastels=10`,
=======
	// --- Black & White ---
	"bw": `High contrast Black and White with rich tonal range.
compensation=0.45, contrast=22, saturation=-100, lab_contrast=28, nr_luminance=15`,
>>>>>>> Stashed changes
||||||| Stash base
	"cinematic": `Movie look: teal/orange vibe, controlled contrast, moody.
compensation=0.42, contrast=15, lab_contrast=18, lab_chromaticity=15, vib_pastels=10`,
=======
	// --- Black & White ---
	"bw": `High contrast Black and White with rich tonal range.
compensation=0.45, contrast=22, saturation=-100, lab_contrast=28, nr_luminance=15`,
>>>>>>> Stashed changes

	"bw-contrast": `Dramatic high contrast Black and White, deep blacks and bright whites.
compensation=0.42, contrast=28, saturation=-100, lab_contrast=35, black=80, nr_luminance=18`,

	"bw-soft": `Soft dreamy Black and White with lifted blacks.
compensation=0.52, contrast=8, saturation=-100, lab_contrast=15, black=0, nr_luminance=20`,

	"bw-sepia": `Black and White with warm sepia toning.
compensation=0.48, contrast=15, saturation=-100, lab_contrast=20, temperature=6200, tint=0.94, nr_luminance=15`,

	// --- Film / Analog Simulation ---
	"film": `General analog film look with soft highlights and rich colors.
compensation=0.52, contrast=12, lab_chromaticity=22, temperature=5800, tint=1.02, black=0, nr_luminance=5`,

	"kodak": `Kodak Gold/Portra style: warm, creamy skin tones, nostalgic feel.
compensation=0.55, contrast=10, lab_chromaticity=20, temperature=5600, tint=0.98, vib_pastels=18, nr_luminance=8`,

	"fuji": `Fujifilm style: PUNCHY cool shadows, HIGH contrast, VIVID greens and blues. Strong film look.
compensation=0.35, contrast=25, lab_contrast=30, lab_chromaticity=35, temperature=5600, tint=1.05, vib_pastels=25, shadow_recovery=25, highlight_compr=60, saturation_green=40, saturation_blue=30, hue_green=-15, luminance_yellow=-15, nr_luminance=8`,

	"polaroid": `Instant film look: faded, shifting colors, vintage vibe.
compensation=0.58, contrast=6, lab_chromaticity=18, temperature=5900, tint=1.05, vib_pastels=8, nr_luminance=12`,

	"retro-70s": `EXTREME 70s golden hour: MAXIMUM warmth 9000K+, INTENSE yellow/orange saturation, thick faded shadows!
compensation=0.50, contrast=-35, lab_chromaticity=30, temperature=9200, tint=0.92, vib_pastels=25, black=150, saturation_orange=40, saturation_yellow=50, saturation_red=20, luminance_yellow=15, luminance_orange=10, nr_luminance=5`,

	// --- Cinematic / Art ---
	"cinematic": `Movie look: moody lighting, wide dynamic range, controlled contrast.
compensation=0.42, contrast=15, lab_contrast=20, lab_chromaticity=18, vib_pastels=12, nr_luminance=10`,

	"teal-orange": `Blockbuster EXTREME teal/orange split! MAXIMUM color separation - teal shadows, burning orange highlights!
compensation=0.30, contrast=25, lab_contrast=25, lab_chromaticity=40, temperature=6500, tint=1.10, vib_pastels=20, highlight_compr=100, shadow_recovery=50, ct_shadow_b=60, ct_shadow_g=-40, ct_shadow_r=-20, ct_highlight_r=50, ct_highlight_g=20, ct_highlight_b=-30, ct_balance=20`,

	"cyberpunk": `EXTREME cyberpunk neon: ICE COLD 3500K, MAXIMUM magenta tint, INTENSE purple/cyan saturation!
compensation=0.35, contrast=30, lab_contrast=35, lab_chromaticity=45, temperature=3500, tint=1.15, vib_pastels=30, saturation_aqua=50, saturation_blue=60, saturation_purple=70, saturation_magenta=60, saturation_yellow=-60, saturation_green=-40, ct_shadow_b=40, ct_shadow_r=-30, ct_highlight_r=30, ct_highlight_b=20`,

	"matte": `EXTREME faded matte: MASSIVE black lift for thick hazy film fog! Washed-out, desaturated vintage film.
compensation=0.55, contrast=-30, lab_contrast=0, lab_chromaticity=5, black=200, shadow_recovery=60, vib_pastels=-30, saturation=-25, temperature=5800, nr_luminance=15`,

	"dreamy": `Ethereal glowy look with soft contrast and pastel colors.
compensation=0.58, contrast=6, lab_contrast=8, lab_chromaticity=15, vib_pastels=12, nr_luminance=8`,

	"wes-anderson": `Pastel color palette with soft contrast, warm and quirky.
compensation=0.50, contrast=10, lab_contrast=12, lab_chromaticity=25, temperature=5700, vib_pastels=18`,

	// --- Scenery / Environment ---
	"landscape": `Landscape: clear sky, enhanced foliage, detailed.
<<<<<<< Updated upstream
<<<<<<< Updated upstream
compensation=0.40, contrast=18, lab_contrast=25, lab_chromaticity=35, vib_pastels=25, nr_luminance=10`,
||||||| Stash base
compensation=0.40, contrast=15, lab_contrast=20, lab_chromaticity=25, vib_pastels=20, nr_luminance=10`,
=======
compensation=0.40, contrast=16, lab_contrast=22, lab_chromaticity=28, vib_pastels=22, nr_luminance=8`,
>>>>>>> Stashed changes
||||||| Stash base
compensation=0.40, contrast=15, lab_contrast=20, lab_chromaticity=25, vib_pastels=20, nr_luminance=10`,
=======
compensation=0.40, contrast=16, lab_contrast=22, lab_chromaticity=28, vib_pastels=22, nr_luminance=8`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"portrait": `Portrait: flattering skin tones, soft contrast, reduced texture.
compensation=0.48, contrast=10, lab_contrast=15, lab_chromaticity=18, vib_pastels=10, nr_luminance=20, nr_chrominance=20`,
||||||| Stash base
	"portrait": `Portrait: flattering skin tones, soft contrast, reduced texture.
compensation=0.48, contrast=8, lab_contrast=10, lab_chromaticity=15, vib_pastels=10, nr_luminance=20, nr_chrominance=20`,
=======
	"golden-hour": `Golden sunset/sunrise light with warm oranges and reds.
compensation=0.48, contrast=12, lab_chromaticity=25, temperature=6200, tint=0.95, vib_pastels=20`,
>>>>>>> Stashed changes
||||||| Stash base
	"portrait": `Portrait: flattering skin tones, soft contrast, reduced texture.
compensation=0.48, contrast=8, lab_contrast=10, lab_chromaticity=15, vib_pastels=10, nr_luminance=20, nr_chrominance=20`,
=======
	"golden-hour": `Golden sunset/sunrise light with warm oranges and reds.
compensation=0.48, contrast=12, lab_chromaticity=25, temperature=6200, tint=0.95, vib_pastels=20`,
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"bw": `Black and white: strong contrast, rich tonal range.
compensation=0.45, contrast=25, saturation=-100, lab_contrast=35, nr_luminance=15`,
||||||| Stash base
	"bw": `Black and white: strong contrast, rich tonal range.
compensation=0.45, contrast=20, saturation=-100, lab_contrast=25, nr_luminance=15`,
=======
	"blue-hour": `Deep cool twilight blues with preserved city lights.
compensation=0.38, contrast=14, lab_contrast=18, lab_chromaticity=22, temperature=4600, tint=1.02, vib_pastels=15`,
>>>>>>> Stashed changes
||||||| Stash base
	"bw": `Black and white: strong contrast, rich tonal range.
compensation=0.45, contrast=20, saturation=-100, lab_contrast=25, nr_luminance=15`,
=======
	"blue-hour": `Deep cool twilight blues with preserved city lights.
compensation=0.38, contrast=14, lab_contrast=18, lab_chromaticity=22, temperature=4600, tint=1.02, vib_pastels=15`,
>>>>>>> Stashed changes

	"urban": `Gritty city look with desaturated colors and high texture.
compensation=0.42, contrast=18, lab_contrast=25, lab_chromaticity=15, vib_pastels=5, nr_luminance=15`,

	"snow": `High-key snow scenes with crisp details and pure whites.
compensation=0.35, contrast=12, lab_contrast=16, lab_chromaticity=15, vib_pastels=10, nr_luminance=12`,

	// --- Subject Specific ---
	"portrait": `Portrait: flattering skin tones, soft contrast.
compensation=0.50, contrast=8, lab_contrast=12, lab_chromaticity=18, vib_pastels=12, nr_luminance=22, nr_chrominance=25`,

	"portrait-glamour": `Beauty/glamour style with very soft skin and glowing highlights.
compensation=0.55, contrast=5, lab_contrast=8, lab_chromaticity=15, vib_pastels=8, nr_luminance=30, nr_chrominance=35`,

	"food": `Appetizing food photography with warmer white balance.
compensation=0.48, contrast=14, lab_contrast=18, lab_chromaticity=22, temperature=5800, vib_pastels=18, nr_luminance=10`,

	"street": `Documentary street style with high contrast and gritty texture.
compensation=0.42, contrast=20, lab_contrast=25, lab_chromaticity=18, vib_pastels=8, nr_luminance=12`,

	"macro": `Macro photography: high sharpness, vivid colors, creamy backgrounds.
compensation=0.45, contrast=16, lab_contrast=20, lab_chromaticity=25, vib_pastels=20, nr_luminance=8`,

	"product": `Clean commercial product photography with neutral colors.
compensation=0.48, contrast=12, lab_contrast=15, lab_chromaticity=16, temperature=5500, tint=1.00, vib_pastels=10, nr_luminance=10`,
}

<<<<<<< Updated upstream
<<<<<<< Updated upstream
const pp3SystemInstruction = `You are an expert photo color grader for RawTherapee. 
Analyze the image and output professional color grading parameters in JSON format.
||||||| Stash base
const pp3SystemInstruction = `You are a RawTherapee color grading expert. Generate high-quality PP3 parameters.
=======
const pp3SystemInstruction = `You are a professional RawTherapee color grading artist. Analyze the image and create stunning color grading that brings out the best in every photo.
>>>>>>> Stashed changes
||||||| Stash base
const pp3SystemInstruction = `You are a RawTherapee color grading expert. Generate high-quality PP3 parameters.
=======
const pp3SystemInstruction = `You are a professional RawTherapee color grading artist. Analyze the image and create stunning color grading that brings out the best in every photo.
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
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
||||||| Stash base
⚠️ CRITICAL QUALITY RULES:
- **Noise Reduction**: ALWAYS apply 'nr_luminance' (10-25) and 'nr_chrominance' (15-30) unless ISO is very low. Grainy images look bad.
- **Exposure**: 'compensation' MUST be 0.35-0.55. RT renders dark by default.
- **Saturation**: Be conservative. Use 'vib_pastels' for natural color boosts instead of 'saturation'.
- **Contrast**: Avoid high 'contrast' (>20) or 'lab_contrast' (>30) to prevent harsh artifacts.
=======
🎯 YOUR MISSION:
Create DRAMATIC, VISUALLY STRIKING color grading with BOLD artistic choices. Make each style INSTANTLY recognizable and SIGNIFICANTLY different from others. Don't be subtle - create strong visual impact!
>>>>>>> Stashed changes
||||||| Stash base
⚠️ CRITICAL QUALITY RULES:
- **Noise Reduction**: ALWAYS apply 'nr_luminance' (10-25) and 'nr_chrominance' (15-30) unless ISO is very low. Grainy images look bad.
- **Exposure**: 'compensation' MUST be 0.35-0.55. RT renders dark by default.
- **Saturation**: Be conservative. Use 'vib_pastels' for natural color boosts instead of 'saturation'.
- **Contrast**: Avoid high 'contrast' (>20) or 'lab_contrast' (>30) to prevent harsh artifacts.
=======
🎯 YOUR MISSION:
Create DRAMATIC, VISUALLY STRIKING color grading with BOLD artistic choices. Make each style INSTANTLY recognizable and SIGNIFICANTLY different from others. Don't be subtle - create strong visual impact!
>>>>>>> Stashed changes

<<<<<<< Updated upstream
<<<<<<< Updated upstream
Output ONLY the JSON object.`
||||||| Stash base
📊 ALLOWED PARAMETERS:
- compensation: 0.35-0.55 (brightness, REQUIRED)
- contrast: 5-25 (global contrast)
- saturation: -100 to 20 (color saturation, -100 for B&W)
- black: 0-150 (black point, higher = darker blacks)
- highlight_compr: 0-100 (recover highlights)
- temperature: 4000-7500 (white balance)
- tint: 0.90-1.10 (green-magenta balance)
- lab_brightness: -10 to 10 (luminance adjust)
- lab_contrast: 0-30 (local contrast/clarity)
- lab_chromaticity: 0-30 (color vibrancy)
- vib_pastels: 0-30 (boost muted colors)
- vib_saturated: 0-15 (protect saturated colors)
- nr_luminance: 5-40 (reduce grain/noise)
- nr_chrominance: 10-40 (remove color noise)
||||||| Stash base
📊 ALLOWED PARAMETERS:
- compensation: 0.35-0.55 (brightness, REQUIRED)
- contrast: 5-25 (global contrast)
- saturation: -100 to 20 (color saturation, -100 for B&W)
- black: 0-150 (black point, higher = darker blacks)
- highlight_compr: 0-100 (recover highlights)
- temperature: 4000-7500 (white balance)
- tint: 0.90-1.10 (green-magenta balance)
- lab_brightness: -10 to 10 (luminance adjust)
- lab_contrast: 0-30 (local contrast/clarity)
- lab_chromaticity: 0-30 (color vibrancy)
- vib_pastels: 0-30 (boost muted colors)
- vib_saturated: 0-15 (protect saturated colors)
- nr_luminance: 5-40 (reduce grain/noise)
- nr_chrominance: 10-40 (remove color noise)
=======
⚡ DRAMATIC GUIDELINES:
- **Be EXTREME**: Use the full parameter ranges! Subtle adjustments won't show.
- **Color Toning CRITICAL**: For film/cinematic styles, use STRONG ct_shadow/ct_highlight values (±30 to ±60 range)
- **Temperature EXTREMES**: 3000K for cool cyberpunk, 9000K+ for warm vintage
- **Black Point DRAMATIC**: 150+ for matte/faded looks, 0 for punchy contrast
- **Saturation BOLD**: -50 for desaturated, +30 for vivid
- **Lab Chromaticity**: Push to 40-50 for color pop
>>>>>>> Stashed changes

🎨 AVAILABLE TOOLS:
- compensation: 0.25-0.70 (overall brightness)
- contrast: -40 to 40 (global contrast, negative for low-contrast vintage)
- saturation: -100 to 40 (-100 for B&W, up to 40 for vivid)
- black: 0-250 (black point depth, high values for matte/faded looks)
- highlight_compr: 0-150 (highlight recovery)
- shadow_recovery: 0-100 (shadow lift)
- highlight_recovery: 0-100 (additional highlight control)
- temperature: 3000-9500 (white balance warmth, allow extreme values for vintage)
- tint: 0.80-1.20 (green-magenta balance)
- lab_brightness: -15 to 20 (luminance adjustment)
- lab_contrast: 0-40 (local contrast/clarity)
- lab_chromaticity: 0-40 (color saturation/vibrancy)
- vib_pastels: 0-40 (enhance muted colors)
- vib_saturated: 0-25 (control already saturated colors)
- nr_luminance: 5-35 (grain/noise reduction)
- nr_chrominance: 10-35 (color noise reduction)
- dehaze_strength: 0-40 (atmospheric haze removal)
- HSL color adjustments: hue_[color], saturation_[color], luminance_[color] for red, orange, yellow, green, aqua, blue, purple, magenta (range -100 to 100 each)
- Color Toning (Split Toning): ct_shadow_r/g/b, ct_highlight_r/g/b, ct_balance (range -100 to 100 for RGB, 0-100 for balance)

Focus on creating a compelling final image. Use HSL adjustments and Color Toning for cinematic effects. Be BOLD with parameters - don't be conservative! Output ONLY JSON:
{
  "compensation": 0.45,
  "contrast": 15,
  "saturation": 8,
  "black": 60,
  "highlight_compr": 40,
  "shadow_recovery": 20,
  "highlight_recovery": 15,
  "temperature": 5500,
  "tint": 1.0,
  "lab_brightness": 5,
  "lab_contrast": 18,
  "lab_chromaticity": 22,
  "vib_pastels": 18,
  "vib_saturated": 8,
  "nr_luminance": 15,
  "nr_chrominance": 20,
  "dehaze_strength": 10,
  "hue_red": 0,
  "hue_orange": 0,
  "hue_yellow": 0,
  "hue_green": 0,
  "hue_aqua": 0,
  "hue_blue": 0,
  "hue_purple": 0,
  "hue_magenta": 0,
  "saturation_red": 0,
  "saturation_orange": 0,
  "saturation_yellow": 0,
  "saturation_green": 0,
  "saturation_aqua": 0,
  "saturation_blue": 0,
  "saturation_purple": 0,
  "saturation_magenta": 0,
  "luminance_red": 0,
  "luminance_orange": 0,
  "luminance_yellow": 0,
  "luminance_green": 0,
  "luminance_aqua": 0,
  "luminance_blue": 0,
  "luminance_purple": 0,
  "luminance_magenta": 0,
  "ct_shadow_r": 0,
  "ct_shadow_g": 0,
  "ct_shadow_b": 0,
  "ct_highlight_r": 0,
  "ct_highlight_g": 0,
  "ct_highlight_b": 0,
  "ct_balance": 50
}`
=======
⚡ DRAMATIC GUIDELINES:
- **Be EXTREME**: Use the full parameter ranges! Subtle adjustments won't show.
- **Color Toning CRITICAL**: For film/cinematic styles, use STRONG ct_shadow/ct_highlight values (±30 to ±60 range)
- **Temperature EXTREMES**: 3000K for cool cyberpunk, 9000K+ for warm vintage
- **Black Point DRAMATIC**: 150+ for matte/faded looks, 0 for punchy contrast
- **Saturation BOLD**: -50 for desaturated, +30 for vivid
- **Lab Chromaticity**: Push to 40-50 for color pop

🎨 AVAILABLE TOOLS:
- compensation: 0.25-0.70 (overall brightness)
- contrast: -40 to 40 (global contrast, negative for low-contrast vintage)
- saturation: -100 to 40 (-100 for B&W, up to 40 for vivid)
- black: 0-250 (black point depth, high values for matte/faded looks)
- highlight_compr: 0-150 (highlight recovery)
- shadow_recovery: 0-100 (shadow lift)
- highlight_recovery: 0-100 (additional highlight control)
- temperature: 3000-9500 (white balance warmth, allow extreme values for vintage)
- tint: 0.80-1.20 (green-magenta balance)
- lab_brightness: -15 to 20 (luminance adjustment)
- lab_contrast: 0-40 (local contrast/clarity)
- lab_chromaticity: 0-40 (color saturation/vibrancy)
- vib_pastels: 0-40 (enhance muted colors)
- vib_saturated: 0-25 (control already saturated colors)
- nr_luminance: 5-35 (grain/noise reduction)
- nr_chrominance: 10-35 (color noise reduction)
- dehaze_strength: 0-40 (atmospheric haze removal)
- HSL color adjustments: hue_[color], saturation_[color], luminance_[color] for red, orange, yellow, green, aqua, blue, purple, magenta (range -100 to 100 each)
- Color Toning (Split Toning): ct_shadow_r/g/b, ct_highlight_r/g/b, ct_balance (range -100 to 100 for RGB, 0-100 for balance)

Focus on creating a compelling final image. Use HSL adjustments and Color Toning for cinematic effects. Be BOLD with parameters - don't be conservative! Output ONLY JSON:
{
  "compensation": 0.45,
  "contrast": 15,
  "saturation": 8,
  "black": 60,
  "highlight_compr": 40,
  "shadow_recovery": 20,
  "highlight_recovery": 15,
  "temperature": 5500,
  "tint": 1.0,
  "lab_brightness": 5,
  "lab_contrast": 18,
  "lab_chromaticity": 22,
  "vib_pastels": 18,
  "vib_saturated": 8,
  "nr_luminance": 15,
  "nr_chrominance": 20,
  "dehaze_strength": 10,
  "hue_red": 0,
  "hue_orange": 0,
  "hue_yellow": 0,
  "hue_green": 0,
  "hue_aqua": 0,
  "hue_blue": 0,
  "hue_purple": 0,
  "hue_magenta": 0,
  "saturation_red": 0,
  "saturation_orange": 0,
  "saturation_yellow": 0,
  "saturation_green": 0,
  "saturation_aqua": 0,
  "saturation_blue": 0,
  "saturation_purple": 0,
  "saturation_magenta": 0,
  "luminance_red": 0,
  "luminance_orange": 0,
  "luminance_yellow": 0,
  "luminance_green": 0,
  "luminance_aqua": 0,
  "luminance_blue": 0,
  "luminance_purple": 0,
  "luminance_magenta": 0,
  "ct_shadow_r": 0,
  "ct_shadow_g": 0,
  "ct_shadow_b": 0,
  "ct_highlight_r": 0,
  "ct_highlight_g": 0,
  "ct_highlight_b": 0,
  "ct_balance": 50
}`
>>>>>>> Stashed changes

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
