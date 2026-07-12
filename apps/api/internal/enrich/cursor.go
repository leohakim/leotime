package enrich

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultCursorModelID = "composer-2.5"

type CursorClient struct {
	BaseURL    string
	HTTPClient *http.Client
	ModelID    string
	PollEvery  time.Duration
	Timeout    time.Duration
}

func DefaultCursorClient() *CursorClient {
	return &CursorClient{
		BaseURL:    "https://api.cursor.com",
		HTTPClient: &http.Client{Timeout: 45 * time.Second},
		ModelID:    defaultCursorModelID,
		PollEvery:  2 * time.Second,
		Timeout:    90 * time.Second,
	}
}

type cursorCreateRequest struct {
	Prompt cursorPromptBody `json:"prompt"`
	Model  *cursorModelBody `json:"model,omitempty"`
}

type cursorPromptBody struct {
	Text string `json:"text"`
}

type cursorModelBody struct {
	ID string `json:"id"`
}

type cursorCreateResponse struct {
	Agent struct {
		ID string `json:"id"`
	} `json:"agent"`
	Run struct {
		ID string `json:"id"`
	} `json:"run"`
}

type cursorRunResponse struct {
	Status string `json:"status"`
	Result string `json:"result"`
}

func (c *CursorClient) PromptOnce(ctx context.Context, apiKey, prompt string) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	prompt = strings.TrimSpace(prompt)
	if apiKey == "" {
		return "", errors.New("cursor api key is required")
	}
	if prompt == "" {
		return "", errors.New("cursor prompt is required")
	}

	client := c.httpClient()
	deadlineCtx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	agentID, runID, err := c.createAgent(deadlineCtx, client, apiKey, prompt)
	if err != nil {
		return "", err
	}
	return c.waitForRun(deadlineCtx, client, apiKey, agentID, runID)
}

func (c *CursorClient) createAgent(ctx context.Context, client *http.Client, apiKey, prompt string) (string, string, error) {
	payload, err := json.Marshal(cursorCreateRequest{
		Prompt: cursorPromptBody{Text: prompt},
		Model:  &cursorModelBody{ID: c.modelID()},
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal cursor create request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/v1/agents", bytes.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("build cursor create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, "")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("cursor create request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("read cursor create response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("cursor create failed: status %d", resp.StatusCode)
	}

	var created cursorCreateResponse
	if err := json.Unmarshal(body, &created); err != nil {
		return "", "", fmt.Errorf("decode cursor create response: %w", err)
	}
	if created.Agent.ID == "" || created.Run.ID == "" {
		return "", "", errors.New("cursor create response missing agent or run id")
	}
	return created.Agent.ID, created.Run.ID, nil
}

func (c *CursorClient) waitForRun(ctx context.Context, client *http.Client, apiKey, agentID, runID string) (string, error) {
	ticker := time.NewTicker(c.pollEvery())
	defer ticker.Stop()

	for {
		text, done, err := c.fetchRun(ctx, client, apiKey, agentID, runID)
		if err != nil {
			return "", err
		}
		if done {
			if strings.TrimSpace(text) == "" {
				return "", errors.New("cursor run finished without result text")
			}
			return strings.TrimSpace(text), nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("cursor run timed out: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *CursorClient) fetchRun(ctx context.Context, client *http.Client, apiKey, agentID, runID string) (string, bool, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/runs/%s", c.baseURL(), agentID, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, fmt.Errorf("build cursor run request: %w", err)
	}
	req.SetBasicAuth(apiKey, "")

	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("cursor run request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false, fmt.Errorf("read cursor run response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("cursor run failed: status %d", resp.StatusCode)
	}

	var run cursorRunResponse
	if err := json.Unmarshal(body, &run); err != nil {
		return "", false, fmt.Errorf("decode cursor run response: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(run.Status)) {
	case "FINISHED":
		return run.Result, true, nil
	case "ERROR", "CANCELLED", "EXPIRED":
		return "", false, fmt.Errorf("cursor run ended with status %s", run.Status)
	default:
		return "", false, nil
	}
}

func (c *CursorClient) baseURL() string {
	if strings.TrimSpace(c.BaseURL) == "" {
		return "https://api.cursor.com"
	}
	return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
}

func (c *CursorClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *CursorClient) modelID() string {
	if strings.TrimSpace(c.ModelID) == "" {
		return defaultCursorModelID
	}
	return strings.TrimSpace(c.ModelID)
}

func (c *CursorClient) pollEvery() time.Duration {
	if c.PollEvery > 0 {
		return c.PollEvery
	}
	return 2 * time.Second
}

func (c *CursorClient) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 90 * time.Second
}
