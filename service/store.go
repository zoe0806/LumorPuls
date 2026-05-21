package service

import (
	"errors"
	"time"

	"lumor_puls/types"

	"gorm.io/gorm"
)

// Store wraps persistence for tasks, snapshots, and signals.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// SeedTasks inserts config seed tasks when missing.
func (s *Store) SeedTasks(seeds []types.MonitorTask) error {
	for _, t := range seeds {
		var n int64
		if err := s.db.Model(&types.MonitorTask{}).Where("id = ?", t.ID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if err := s.db.Create(&t).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListTasks returns all monitor tasks.
func (s *Store) ListTasks() ([]types.MonitorTask, error) {
	var list []types.MonitorTask
	err := s.db.Order("id ASC").Find(&list).Error
	return list, err
}

// ListEnabledTasks returns tasks that are enabled.
func (s *Store) ListEnabledTasks() ([]types.MonitorTask, error) {
	var list []types.MonitorTask
	err := s.db.Where("enabled = ?", true).Find(&list).Error
	return list, err
}

// GetTask loads a task by id.
func (s *Store) GetTask(id string) (*types.MonitorTask, error) {
	var t types.MonitorTask
	err := s.db.Where("id = ?", id).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkTaskRun updates last run time and optional error message.
func (s *Store) MarkTaskRun(id string, runErr error) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_run_at": now,
		"last_error":  "",
	}
	if runErr != nil {
		updates["last_error"] = truncateErr(runErr.Error(), 1000)
	}
	return s.db.Model(&types.MonitorTask{}).Where("id = ?", id).Updates(updates).Error
}

// LastSnapshot returns the newest snapshot for a task.
func (s *Store) LastSnapshot(taskID string) (*types.Snapshot, error) {
	var snap types.Snapshot
	err := s.db.Where("task_id = ?", taskID).Order("captured_at DESC").First(&snap).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// SaveSnapshot persists a new snapshot row.
func (s *Store) SaveSnapshot(snap *types.Snapshot) error {
	return s.db.Create(snap).Error
}

// InsertSignals batch-inserts signal rows.
func (s *Store) InsertSignals(rows []types.Signal) error {
	if len(rows) == 0 {
		return nil
	}
	return s.db.Create(&rows).Error
}

// ListSignals queries signals with optional filters.
func (s *Store) ListSignals(signalType, taskID string, limit int) ([]types.Signal, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.Order("created_at DESC").Limit(limit)
	if signalType != "" {
		q = q.Where("signal_type = ?", signalType)
	}
	if taskID != "" {
		q = q.Where("task_id = ?", taskID)
	}
	var list []types.Signal
	return list, q.Find(&list).Error
}

func truncateErr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
