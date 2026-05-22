package types

import "encoding/json"

// CaptureResult is in-memory output from the browser worker.
type CaptureResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

// ExtractChange is one change from the ecosystem/generic extractor.
type ExtractChange struct {
	Type     string `json:"type"`
	Old      string `json:"old"`
	New      string `json:"new"`
	Severity string `json:"severity"`
}

// ExtractResult is LLM output for ecosystem-style diff.
type ExtractResult struct {
	Changes []ExtractChange `json:"changes"`
	Summary string          `json:"summary"`
}

// PricingChange is one pricing delta from the pricing extractor.
type PricingChange struct {
	Model     string `json:"model"`
	OldPrice  string `json:"old_price"`
	NewPrice  string `json:"new_price"`
	Currency  string `json:"currency,omitempty"`
	Severity  string `json:"severity"`
}

// PricingExtractResult is LLM output for pricing category.
type PricingExtractResult struct {
	Changes []PricingChange `json:"changes"`
	Summary string          `json:"summary"`
}

// ReleaseChange is one release delta from the release extractor.
type ReleaseChange struct {
	Product    string `json:"product"`
	Version    string `json:"version"`
	OldVersion string `json:"old_version,omitempty"`
	Breaking   bool   `json:"breaking"`
	Notes      string `json:"notes,omitempty"`
	Severity   string `json:"severity"`
}

// ReleaseExtractResult is LLM output for release category.
type ReleaseExtractResult struct {
	Changes []ReleaseChange `json:"changes"`
	Summary string          `json:"summary"`
}

// CreateTaskRequest is the body for POST /tasks.
type CreateTaskRequest struct {
	ID             string `json:"id"`
	URL            string `json:"url"`
	Interval       string `json:"interval"`
	Type           string `json:"type"`
	SignalCategory string `json:"signalCategory"`
	Enabled        *bool  `json:"enabled"`
}

// UpdateTaskRequest is the body for PUT /tasks/:id (all fields optional).
type UpdateTaskRequest struct {
	URL            *string `json:"url"`
	Interval       *string `json:"interval"`
	Type           *string `json:"type"`
	SignalCategory *string `json:"signalCategory"`
	Enabled        *bool   `json:"enabled"`
}

// MarshalPayload JSON-encodes a struct for signals.payload_json.
func MarshalPayload(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
