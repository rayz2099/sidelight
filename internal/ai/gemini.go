package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"sidelight/pkg/models"
)

type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-flash-latest")
	// Set the response MIME type to JSON
	model.ResponseMIMEType = "application/json"
	
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
The parameters should aim for a natural, high-quality look.
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
  "sharpness": int (range 0 to 150)
}`

// styles maps style names to detailed prompting instructions.
var styles = map[string]string{
	"natural":   "Aim for accurate colors, balanced exposure, and realistic reproduction of the scene. Correct any white balance issues.",
	"cinematic": "Create a cinematic look. Use complementary colors (like teal/orange where appropriate), moody lighting, and slightly crushed blacks for depth.",
	"film":      "Emulate analog film stock. Soft highlights, rich colors, maybe slightly lifted blacks, and a bit of texture.",
	"bw":        "Convert to Black and White. Focus on contrast, structure, and tonal separation. Ensure true blacks and bright whites.",
	"portrait":  "Focus on flattering skin tones. Soften texture slightly, ensure good exposure on the subject, and create a gentle visual hierarchy.",
}

func (g *GeminiClient) AnalyzeImage(ctx context.Context, imageData []byte, opts AnalysisOptions) (*models.GradingParams, error) {
	styleInstruction := styles["natural"] // Default
	if instruction, ok := styles[opts.Style]; ok {
		styleInstruction = instruction
	}

	fullInstruction := fmt.Sprintf(`%s
    
Current Style Goal: %s

User Specific Instructions: %s

Output ONLY the JSON object.`, systemInstruction, styleInstruction, opts.UserPrompt)

	g.model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(fullInstruction)},
	}

	prompt := []genai.Part{
		genai.ImageData("jpeg", imageData),
		genai.Text("Please grade this image."),
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
