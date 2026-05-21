package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadPrompt reads a prompt file; env LUMOR_PROMPT_<NAME> overrides (NAME = upper basename).
func LoadPrompt(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", path, err)
	}
	text := string(raw)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	envKey := "LUMOR_PROMPT_" + strings.ToUpper(strings.ReplaceAll(base, "-", "_"))
	if v := os.Getenv(envKey); v != "" {
		return v, nil
	}
	return text, nil
}
