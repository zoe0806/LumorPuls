package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

const defaultBrowserSession = "lumor_worker"

// Capture opens url, waits for load, extracts title/url/text, then closes session.
// All tasks share one browser session (Runner already serializes) to avoid many Chrome daemons on Windows.
func (b *Browser) Capture(ctx context.Context, taskID, url string) (*types.CaptureResult, error) {
	session := b.sessionName()
	runCtx, cancel, budget := b.captureContext(ctx)
	defer cancel()

	// Reset stale daemon/socket from a previous crash or per-task sessions.
	_ = b.run(context.Background(), session, "close")

	log.Printf("browser: open session=%s task=%s url=%s (budget=%s)", session, taskID, url, budget)
	if err := b.run(runCtx, session, "open", url); err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = b.run(context.Background(), session, "close") }()

	// networkidle on SPAs (OpenAI etc.) often never fires — use fixed wait by default.
	if b.cfg.WaitNetworkIdle {
		log.Printf("browser: wait networkidle (may hang on heavy sites; prefer waitNetworkIdle=false)")
		if err := b.run(runCtx, session, "wait", "--load", "networkidle"); err != nil {
			log.Printf("browser: networkidle failed, fallback wait 8s: %v", err)
			_ = b.run(runCtx, session, "wait", "8000")
		}
	} else {
		log.Printf("browser: wait 8s for render")
		_ = b.run(runCtx, session, "wait", "8000")
	}

	title, err := b.getString(runCtx, session, "get", "title")
	if err != nil {
		title = ""
	}
	pageURL, err := b.getString(runCtx, session, "get", "url")
	if err != nil {
		pageURL = url
	}
	log.Printf("browser: extract text")
	text, err := b.evalText(runCtx, session)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}
	log.Printf("browser: done text_len=%d", len(text))
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
	cmd.Env = browserEnv(b.cfg)
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
	// Single-line only: multiline scripts break on Windows argv quoting.
	script := `document.body?document.body.innerText:''`
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
	cmd.Env = browserEnv(b.cfg)
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

// captureContext builds a timeout that is not shorter than the parent when parent allows more time.
// Fixes: open alone can exceed 120s on Windows; a fixed 120s cap caused eval to hit deadline exceeded.
func (b *Browser) captureContext(ctx context.Context) (context.Context, context.CancelFunc, time.Duration) {
	cfgSec := b.cfg.TimeoutSec
	if cfgSec <= 0 {
		cfgSec = 300
	}
	budget := time.Duration(cfgSec) * time.Second
	const floor = 5 * time.Minute
	if budget < floor {
		budget = floor
	}
	if dl, ok := ctx.Deadline(); ok {
		remaining := time.Until(dl)
		if remaining > 0 && remaining > budget {
			budget = remaining
		}
	}
	c, cancel := context.WithTimeout(ctx, budget)
	return c, cancel, budget
}

func (b *Browser) sessionName() string {
	if s := strings.TrimSpace(b.cfg.SessionName); s != "" {
		return s
	}
	return defaultBrowserSession
}

// browserEnv passes browser-related env vars to agent-browser CLI.
func browserEnv(cfg config.BrowserConfig) []string {
	env := os.Environ()
	path := cfg.ExecutablePath
	if path == "" {
		path = os.Getenv("AGENT_BROWSER_EXECUTABLE_PATH")
	}
	if path != "" {
		env = append(env, "AGENT_BROWSER_EXECUTABLE_PATH="+path)
	}
	// Shut down idle daemon so sessions do not accumulate after each task.
	env = append(env, "AGENT_BROWSER_IDLE_TIMEOUT_MS=120000")
	return env
}

// CloseAllSessions runs agent-browser close --all (use when repairing stuck daemons).
func CloseAllSessions(cfg config.BrowserConfig) error {
	bin := cfg.Bin
	if bin == "" {
		bin = "agent-browser"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "close", "--all")
	cmd.Env = browserEnv(cfg)
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
