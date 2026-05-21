package tools

import (
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns SHA-256 hex of normalized page text.
func ContentHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
