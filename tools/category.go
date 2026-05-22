package tools

import (
	"fmt"
	"strings"

	"lumor_puls/types"
)

// NormalizeSignalCategory validates and normalizes a category string.
func NormalizeSignalCategory(raw string) (string, error) {
	c := strings.TrimSpace(strings.ToLower(raw))
	if c == "" {
		return types.SignalCategoryEcosystem, nil
	}
	for _, v := range types.ValidSignalCategories {
		if c == v {
			return c, nil
		}
	}
	return "", fmt.Errorf("invalid signalCategory %q, allowed: %v", raw, types.ValidSignalCategories)
}
