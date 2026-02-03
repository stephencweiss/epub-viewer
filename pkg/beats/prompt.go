package beats

import "fmt"

const beatAnalysisPromptTemplate = `You are a literary analyst extracting narrative beats from fiction.

A "beat" is a semantic unit of story progression. For the following scene, identify:

1. SUMMARY: What happens in this scene? (1-3 sentences, focusing on key events)
2. CONFLICT: What is the main tension or conflict? (internal, external, or both)
3. CHOICE: Who makes a significant choice or decision? What is it?
4. CONSEQUENCE: What is the result or consequence of that choice?
5. PERSPECTIVE_SHIFT: Does the reader's understanding of any character change? (optional - only if applicable)

IMPORTANT GUIDELINES:
- Be specific and concrete, not vague
- Focus on character actions and motivations
- If there's no clear choice/consequence (e.g., purely descriptive scene), note "No significant choice in this scene"
- Keep responses concise but substantive

SCENE TEXT:
---
%s
---

Respond with ONLY a JSON object in this exact format:
{
    "summary": "...",
    "conflict": "...",
    "choice": "...",
    "consequence": "...",
    "perspective_shift": "..."
}

Note: perspective_shift can be empty string if not applicable.

JSON response:`

// DefaultMaxSceneChars is the default maximum characters to send to the LLM per scene.
const DefaultMaxSceneChars = 8000

// BuildBeatPrompt constructs the prompt for beat analysis.
// If the scene text exceeds maxChars, it will be truncated.
func BuildBeatPrompt(sceneText string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = DefaultMaxSceneChars
	}

	// Truncate scene text if too long
	if len(sceneText) > maxChars {
		sceneText = sceneText[:maxChars] + "\n\n[Scene truncated for analysis...]"
	}

	return fmt.Sprintf(beatAnalysisPromptTemplate, sceneText)
}
