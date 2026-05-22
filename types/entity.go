package types

import "time"

// MonitorTask is a scheduled browser snapshot target.
type MonitorTask struct {
	ID              string     `gorm:"column:id;primaryKey;size:64" json:"id"`
	URL             string     `gorm:"column:url;size:512;not null" json:"url"`
	Interval        string     `gorm:"column:interval;size:16;not null" json:"interval"`
	Type            string     `gorm:"column:type;size:32;not null" json:"type"`
	SignalCategory  string     `gorm:"column:signal_category;size:32;index;not null;default:ecosystem" json:"signalCategory"`
	Enabled         bool       `gorm:"column:enabled;not null;default:true" json:"enabled"`
	LastRunAt       *time.Time `gorm:"column:last_run_at" json:"lastRunAt,omitempty"`
	LastError       string     `gorm:"column:last_error;size:1024" json:"lastError,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (MonitorTask) TableName() string { return "tasks" }

// Snapshot is page state at a point in time.
type Snapshot struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID      string    `gorm:"column:task_id;size:64;index;not null" json:"taskId"`
	URL         string    `gorm:"column:url;size:512" json:"url"`
	Title       string    `gorm:"column:title;size:512" json:"title"`
	Text        string    `gorm:"column:text;type:longtext" json:"-"`
	ContentHash string    `gorm:"column:content_hash;size:64;index" json:"contentHash"`
	CapturedAt  time.Time `gorm:"column:captured_at;index;not null" json:"capturedAt"`
}

func (Snapshot) TableName() string { return "snapshots" }

// Signal is a structured change detected between two snapshots.
type Signal struct {
	ID              uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID          string    `gorm:"column:task_id;size:64;index;not null" json:"taskId"`
	URL             string    `gorm:"column:url;size:512" json:"url"`
	SignalCategory  string    `gorm:"column:signal_category;size:32;index;not null" json:"category"`
	SignalType      string    `gorm:"column:signal_type;size:64;index;not null" json:"type"`
	Summary         string    `gorm:"column:summary;type:text" json:"summary"`
	Severity        string    `gorm:"column:severity;size:16;index" json:"severity"`
	PayloadJSON     string    `gorm:"column:payload_json;type:json" json:"payload,omitempty"`
	OldExcerpt      string    `gorm:"column:old_excerpt;type:text" json:"oldExcerpt,omitempty"`
	NewExcerpt      string    `gorm:"column:new_excerpt;type:text" json:"newExcerpt,omitempty"`
	CreatedAt       time.Time `gorm:"column:created_at;index;autoCreateTime" json:"createdAt"`
}

func (Signal) TableName() string { return "signals" }
