package enrich

type TokenUsage struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens"`
	CacheWriteTokens int `json:"cacheWriteTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type CursorPromptResult struct {
	Text    string
	ModelID string
	Usage   TokenUsage
}

func (u TokenUsage) IsZero() bool {
	return u.TotalTokens == 0 && u.InputTokens == 0 && u.OutputTokens == 0 &&
		u.CacheReadTokens == 0 && u.CacheWriteTokens == 0
}
