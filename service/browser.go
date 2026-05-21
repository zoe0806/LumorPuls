package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"lumor_puls/config"
	"lumor_puls/types"
)

// Browser captures pages via agent-browser CLI.
type Browser struct {
	cfg config.BrowserConfig
}

func NewBrowser(cfg config.BrowserConfig) *Browser {
	return &Browser{cfg: cfg}
}

// Capture opens url, waits for load, extracts title/url/text, then closes session.
func (b *Browser) Capture(ctx context.Context, taskID, url string) (*types.CaptureResult, error) {
	session := "lumor_" + taskID
	timeout := time.Duration(b.cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := b.run(runCtx, session, "open", url); err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = b.run(context.Background(), session, "close") }()

	if b.cfg.WaitNetworkIdle {
		_ = b.run(runCtx, session, "wait", "--load", "networkidle")
	} else {
		_ = b.run(runCtx, session, "wait", "3000")
	}

	title, err := b.getString(runCtx, session, "get", "title")
	if err != nil {
		title = ""
	}
	pageURL, err := b.getString(runCtx, session, "get", "url")
	if err != nil {
		pageURL = url
	}
	text, err := b.evalText(runCtx, session)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}
	text = normalizeText(text)
	if len(text) > 120000 {
		text = text[:120000]
	}

	return &types.CaptureResult{
		URL:       pageURL,
		Title:     title,
		Text:      text,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (b *Browser) run(ctx context.Context, session string, args ...string) error {
	bin := b.cfg.Bin
	if bin == "" {
		bin = "agent-browser"
	}
	full := append([]string{"--session", session}, args...)
	cmd := exec.CommandContext(ctx, bin, full...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func (b *Browser) getString(ctx context.Context, session string, args ...string) (string, error) {
	out, err := b.runOutput(ctx, session, append(args, "--json")...)
	if err != nil {
		return "", err
	}
	return parseCLIJSONValue(out), nil
}

func (b *Browser) evalText(ctx context.Context, session string) (string, error) {
	script := `(function(){
		var b=document.body;
		if(!b)return '';
		return (b.innerText||b.textContent||'').replace(/\s+/g,' ').trim();
	})()`
	out, err := b.runOutput(ctx, session, "eval", script, "--json")
	if err != nil {
		return "", err
	}
	return parseCLIJSONValue(out), nil
}

func (b *Browser) runOutput(ctx context.Context, session string, args ...string) ([]byte, error) {
	bin := b.cfg.Bin
	if bin == "" {
		bin = "agent-browser"
	}
	full := append([]string{"--session", session}, args...)
	cmd := exec.CommandContext(ctx, bin, full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

// parseCLIJSONValue handles {"success":true,"data":"..."} or plain string JSON.
func parseCLIJSONValue(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return ""
	}
	var wrap struct {
		Data    json.RawMessage `json:"data"`
		Success bool            `json:"success"`
	}
	if err := json.Unmarshal([]byte(s), &wrap); err == nil && len(wrap.Data) > 0 {
		var str string
		if err := json.Unmarshal(wrap.Data, &str); err == nil {
			return str
		}
		return strings.Trim(string(wrap.Data), `"`)
	}
	var str string
	if err := json.Unmarshal([]byte(s), &str); err == nil {
		return str
	}
	return s
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
