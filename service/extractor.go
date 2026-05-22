package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lumor_puls/config"
	"lumor_puls/tools"
	"lumor_puls/types"
)

// Extractor runs category-specific semantic diff via LLM.
type Extractor struct {
	cfg    config.LLMConfig
	dir    string
	client *http.Client
}

func NewExtractor(cfg config.Config) *Extractor {
	timeout := time.Duration(cfg.LLM.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	dir := "prompts"
	if cfg.Prompts.Dir != "" {
		dir = cfg.Prompts.Dir
	}
	return &Extractor{
		cfg:    cfg.LLM,
		dir:    dir,
		client: &http.Client{Timeout: timeout},
	}
}

// Extract compares snapshots and returns signals for the task's signal_category.
func (e *Extractor) Extract(ctx context.Context, task types.MonitorTask, prev, cur *types.Snapshot) ([]types.Signal, error) {
	if e.cfg.APIKey == "" {
		apiKey := os.Getenv("LUMOR_LLM_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("llm apiKey empty (set config or LUMOR_LLM_API_KEY)")
		}
		e.cfg.APIKey = apiKey
	}

	category, err := tools.NormalizeSignalCategory(task.SignalCategory)
	if err != nil {
		return nil, err
	}

	prompt, err := e.loadPrompt(category)
	if err != nil {
		return nil, err
	}

	user := buildDiffUserMessage(task, prev, cur)
	raw, err := e.chat(ctx, prompt, user)
	if err != nil {
		return nil, err
	}

	log.Printf("extractor: task=%s category=%s", task.ID, category)
	switch category {
	case types.SignalCategoryPricing:
		return e.parsePricingSignals(task, cur.URL, category, raw)
	case types.SignalCategoryRelease:
		return e.parseReleaseSignals(task, cur.URL, category, raw)
	default:
		return e.parseEcosystemSignals(task, cur.URL, category, raw)
	}
}

func buildDiffUserMessage(task types.MonitorTask, prev, cur *types.Snapshot) string {
	return fmt.Sprintf("Task: %s\nCategory: %s\nURL: %s\n\n=== PREVIOUS (%s) ===\n%s\n\n=== CURRENT (%s) ===\n%s",
		task.ID, task.SignalCategory, task.URL,
		prev.CapturedAt.Format(time.RFC3339), clipForLLM(prev.Text, 24000),
		cur.CapturedAt.Format(time.RFC3339), clipForLLM(cur.Text, 24000),
	)
}

func (e *Extractor) loadPrompt(category string) (string, error) {
	name := "diff_" + category + ".txt"
	if category == types.SignalCategoryProtocol || category == types.SignalCategoryCapability {
		name = "diff_ecosystem.txt"
	}
	path := filepath.Join(e.dir, name)
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(wd, path)
	}
	return tools.LoadPrompt(path)
}

func (e *Extractor) parsePricingSignals(task types.MonitorTask, pageURL, category, raw string) ([]types.Signal, error) {
	var result types.PricingExtractResult
	if err := decodeLLMJSON(raw, &result); err != nil {
		return nil, err
	}
	if isNoChangeSummary(result.Summary) && len(result.Changes) == 0 {
		return nil, nil
	}
	rows := make([]types.Signal, 0, len(result.Changes))
	for _, c := range result.Changes {
		sev := c.Severity
		if sev == "" {
			sev = types.SeverityMedium
		}
		payload := map[string]string{
			"model":     c.Model,
			"old_price": c.OldPrice,
			"new_price": c.NewPrice,
			"currency":  c.Currency,
		}
		sum := result.Summary
		if c.Model != "" && c.NewPrice != "" {
			sum = fmt.Sprintf("%s: %s → %s", c.Model, c.OldPrice, c.NewPrice)
		}
		rows = append(rows, types.Signal{
			TaskID:         task.ID,
			URL:            pageURL,
			SignalCategory: category,
			SignalType:     types.SignalTypePricing,
			Summary:        sum,
			Severity:       sev,
			PayloadJSON:    types.MarshalPayload(payload),
			OldExcerpt:     c.OldPrice,
			NewExcerpt:     c.NewPrice,
		})
	}
	if len(rows) == 0 && result.Summary != "" {
		rows = append(rows, types.Signal{
			TaskID:         task.ID,
			URL:            pageURL,
			SignalCategory: category,
			SignalType:     types.SignalTypePricing,
			Summary:        result.Summary,
			Severity:       types.SeverityLow,
			PayloadJSON:    types.MarshalPayload(map[string]string{"summary": result.Summary}),
		})
	}
	return rows, nil
}

func (e *Extractor) parseReleaseSignals(task types.MonitorTask, pageURL, category, raw string) ([]types.Signal, error) {
	var result types.ReleaseExtractResult
	if err := decodeLLMJSON(raw, &result); err != nil {
		return nil, err
	}
	if isNoChangeSummary(result.Summary) && len(result.Changes) == 0 {
		return nil, nil
	}
	rows := make([]types.Signal, 0, len(result.Changes))
	for _, c := range result.Changes {
		sev := c.Severity
		if sev == "" {
			sev = types.SeverityMedium
		}
		if c.Breaking {
			sev = types.SeverityHigh
		}
		payload := map[string]interface{}{
			"product":     c.Product,
			"version":     c.Version,
			"old_version": c.OldVersion,
			"breaking":    c.Breaking,
			"notes":       c.Notes,
		}
		sum := result.Summary
		if c.Version != "" {
			sum = fmt.Sprintf("%s %s", c.Product, c.Version)
			if c.Notes != "" {
				sum += ": " + c.Notes
			}
		}
		rows = append(rows, types.Signal{
			TaskID:         task.ID,
			URL:            pageURL,
			SignalCategory: category,
			SignalType:     types.SignalTypeRelease,
			Summary:        sum,
			Severity:       sev,
			PayloadJSON:    types.MarshalPayload(payload),
			OldExcerpt:     c.OldVersion,
			NewExcerpt:     c.Version,
		})
	}
	return rows, nil
}

func (e *Extractor) parseEcosystemSignals(task types.MonitorTask, pageURL, category, raw string) ([]types.Signal, error) {
	var result types.ExtractResult
	if err := decodeLLMJSON(raw, &result); err != nil {
		return nil, err
	}
	if isNoChangeSummary(result.Summary) && len(result.Changes) == 0 {
		return nil, nil
	}
	rows := make([]types.Signal, 0, len(result.Changes))
	for _, c := range result.Changes {
		st := c.Type
		if st == "" {
			st = types.SignalTypeOther
		}
		sev := c.Severity
		if sev == "" {
			sev = types.SeverityMedium
		}
		payload := map[string]string{
			"type": c.Type,
			"old":  c.Old,
			"new":  c.New,
		}
		sum := result.Summary
		if c.New != "" {
			sum = strings.TrimSpace(c.Type + ": " + c.New)
		}
		rows = append(rows, types.Signal{
			TaskID:         task.ID,
			URL:            pageURL,
			SignalCategory: category,
			SignalType:     st,
			Summary:        sum,
			Severity:       sev,
			PayloadJSON:    types.MarshalPayload(payload),
			OldExcerpt:     c.Old,
			NewExcerpt:     c.New,
		})
	}
	if len(rows) == 0 && result.Summary != "" {
		rows = append(rows, types.Signal{
			TaskID:         task.ID,
			URL:            pageURL,
			SignalCategory: category,
			SignalType:     types.SignalTypeOther,
			Summary:        result.Summary,
			Severity:       types.SeverityLow,
			PayloadJSON:    types.MarshalPayload(map[string]string{"summary": result.Summary}),
		})
	}
	return rows, nil
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

func decodeLLMJSON(raw string, dest interface{}) error {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("parse llm json: %w (raw=%s)", err, clipForLLM(raw, 200))
	}
	return nil
}

func isNoChangeSummary(s string) bool {
	return strings.Contains(strings.ToLower(s), "no meaningful change")
}

func clipForLLM(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}
