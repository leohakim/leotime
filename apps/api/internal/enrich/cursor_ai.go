package enrich

import (
	"context"
	"log"
	"os"
	"strings"
)

var cursorPromptClient = DefaultCursorClient()

type CursorAIResult struct {
	Text    string
	ModelID string
	Usage   TokenUsage
}

func tryCursorAI(bundle ContextBundle, apiKey string, enabled bool) (CursorAIResult, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("CURSOR_API_KEY"))
	}
	if apiKey == "" {
		return CursorAIResult{}, false
	}
	if !enabled && strings.TrimSpace(os.Getenv("CURSOR_API_KEY")) == "" {
		return CursorAIResult{}, false
	}

	prompt := BuildCursorPrompt(bundle)
	ctx, cancel := context.WithTimeout(context.Background(), cursorPromptClient.timeout())
	defer cancel()

	result, err := cursorPromptClient.PromptOnce(ctx, apiKey, prompt)
	if err != nil {
		log.Printf("cursor enrich failed: %v", err)
		return CursorAIResult{}, false
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		return CursorAIResult{}, false
	}
	return CursorAIResult{
		Text:    text,
		ModelID: result.ModelID,
		Usage:   result.Usage,
	}, true
}
