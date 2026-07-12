package enrich

import (
	"context"
	"log"
	"os"
	"strings"
)

var cursorPromptClient = DefaultCursorClient()

func tryCursorAI(bundle ContextBundle, apiKey string, enabled bool) (string, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("CURSOR_API_KEY"))
	}
	if apiKey == "" {
		return "", false
	}
	if !enabled && strings.TrimSpace(os.Getenv("CURSOR_API_KEY")) == "" {
		return "", false
	}

	prompt := BuildCursorPrompt(bundle)
	ctx, cancel := context.WithTimeout(context.Background(), cursorPromptClient.timeout())
	defer cancel()

	text, err := cursorPromptClient.PromptOnce(ctx, apiKey, prompt)
	if err != nil {
		log.Printf("cursor enrich failed: %v", err)
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	return text, true
}
