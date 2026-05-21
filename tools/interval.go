package tools

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseInterval parses durations like 30m, 6h, 1d.
func ParseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty interval")
	}
	var unit time.Duration
	switch s[len(s)-1] {
	case 'm':
		unit = time.Minute
	case 'h':
		unit = time.Hour
	case 'd':
		unit = 24 * time.Hour
	default:
		return time.ParseDuration(s)
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, fmt.Errorf("parse interval %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("interval must be positive: %q", s)
	}
	return time.Duration(n) * unit, nil
}
