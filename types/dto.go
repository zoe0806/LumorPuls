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

// CreateTaskRequest is the body for POST /tasks.
type CreateTaskRequest struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Interval string `json:"interval"`
	Type     string `json:"type"`
	Enabled  *bool  `json:"enabled"`
}

// UpdateTaskRequest is the body for PUT /tasks/:id (all fields optional).
type UpdateTaskRequest struct {
	URL      *string `json:"url"`
	Interval *string `json:"interval"`
	Type     *string `json:"type"`
	Enabled  *bool   `json:"enabled"`
}
