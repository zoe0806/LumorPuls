package types

// CaptureResult is in-memory output from the browser worker.
type CaptureResult struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

// ExtractChange is one structured change from the LLM extractor.
type ExtractChange struct {
	Type     string `json:"type"`
	Old      string `json:"old"`
	New      string `json:"new"`
	Severity string `json:"severity"`
}

// ExtractResult is LLM output for diff between snapshots.
type ExtractResult struct {
	Changes []ExtractChange `json:"changes"`
	Summary string          `json:"summary"`
}
