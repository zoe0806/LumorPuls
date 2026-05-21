package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumor_puls/config"
	"lumor_puls/tools"
	"lumor_puls/types"
)

// Extractor runs semantic diff via LLM (OpenAI-compatible API).
type Extractor struct {
	cfg        config.LLMConfig
	promptPath string
	client     *http.Client
}

func NewExtractor(cfg config.Config) *Extractor {
	timeout := time.Duration(cfg.LLM.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &Extractor{
		cfg:        cfg.LLM,
		promptPath: cfg.Prompts.DiffPath,
		client:     &http.Client{Timeout: timeout},
	}
}

// Diff compares previous and current snapshot text and returns structured changes.
func (e *Extractor) Diff(ctx context.Context, task types.MonitorTask, prev, cur *types.Snapshot) (*types.ExtractResult, error) {
	if e.cfg.APIKey == "" {
		apiKey := os.Getenv("LUMOR_LLM_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("llm apiKey empty (set config or LUMOR_LLM_API_KEY)")
		}
		e.cfg.APIKey = apiKey
	}

	prompt, err := e.loadPrompt()
	if err != nil {
		return nil, err
	}

	user := fmt.Sprintf("Task: %s\nURL: %s\n\n=== PREVIOUS (%s) ===\n%s\n\n=== CURRENT (%s) ===\n%s",
		task.ID, task.URL,
		prev.CapturedAt.Format(time.RFC3339), clipForLLM(prev.Text, 24000),
		cur.CapturedAt.Format(time.RFC3339), clipForLLM(cur.Text, 24000),
	)

	raw, err := e.chat(ctx, prompt, user)
	if err != nil {
		return nil, err
	}
	return parseExtractJSON(raw)
}

func (e *Extractor) loadPrompt() (string, error) {
	path := e.promptPath
	if path == "" {
		path = "prompts/diff.txt"
	}
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(wd, path)
	}
	return tools.LoadPrompt(path)
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e *Extractor) chat(ctx context.Context, system, user string) (string, error) {
	base := strings.TrimSuffix(e.cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := e.cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	body, _ := json.Marshal(chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, string(raw))
	}

	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("llm error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm empty choices")
	}
	return out.Choices[0].Message.Content, nil
}

func parseExtractJSON(raw string) (*types.ExtractResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result types.ExtractResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("parse llm json: %w (raw=%s)", err, clipForLLM(raw, 200))
	}
	return &result, nil
}

func clipForLLM(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}
