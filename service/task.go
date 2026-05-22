package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"lumor_puls/tools"
	"lumor_puls/types"

	"gorm.io/gorm"
)

var taskIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// CreateTask validates and inserts a monitor task.
func CreateTask(store *Store, req types.CreateTaskRequest) (*types.MonitorTask, error) {
	id := strings.TrimSpace(req.ID)
	if err := validateTaskID(id); err != nil {
		return nil, err
	}
	pageURL, err := validateTaskURL(req.URL)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(req.Interval)
	if err := validateTaskInterval(interval); err != nil {
		return nil, err
	}
	taskType := strings.TrimSpace(req.Type)
	if taskType == "" {
		taskType = types.TaskTypeBrowserSnapshot
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	category, err := tools.NormalizeSignalCategory(req.SignalCategory)
	if err != nil {
		return nil, err
	}

	t := &types.MonitorTask{
		ID:             id,
		URL:            pageURL,
		Interval:       interval,
		Type:           taskType,
		SignalCategory: category,
		Enabled:        enabled,
	}
	if err := store.CreateTask(t); err != nil {
		return nil, err
	}
	return t, nil
}

// UpdateTask applies partial updates to a task.
func UpdateTask(store *Store, id string, req types.UpdateTaskRequest) (*types.MonitorTask, error) {
	id = strings.TrimSpace(id)
	if err := validateTaskID(id); err != nil {
		return nil, err
	}
	t, err := store.GetTask(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, err
	}
	if req.URL != nil {
		u, err := validateTaskURL(*req.URL)
		if err != nil {
			return nil, err
		}
		t.URL = u
	}
	if req.Interval != nil {
		if err := validateTaskInterval(*req.Interval); err != nil {
			return nil, err
		}
		t.Interval = strings.TrimSpace(*req.Interval)
	}
	if req.Type != nil {
		tt := strings.TrimSpace(*req.Type)
		if tt == "" {
			return nil, fmt.Errorf("type cannot be empty")
		}
		t.Type = tt
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	if req.SignalCategory != nil {
		category, err := tools.NormalizeSignalCategory(*req.SignalCategory)
		if err != nil {
			return nil, err
		}
		t.SignalCategory = category
	}
	if err := store.UpdateTask(t); err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask removes a task by id.
func DeleteTask(store *Store, id string) error {
	id = strings.TrimSpace(id)
	if err := validateTaskID(id); err != nil {
		return err
	}
	return store.DeleteTask(id)
}

func validateTaskID(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	if !taskIDPattern.MatchString(id) {
		return fmt.Errorf("id must match [a-z][a-z0-9_], max 64 chars")
	}
	return nil
}

func validateTaskURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("url must use http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("url must have a host")
	}
	return raw, nil
}

func validateTaskInterval(interval string) error {
	interval = strings.TrimSpace(interval)
	if interval == "" {
		return fmt.Errorf("interval is required")
	}
	if _, err := tools.ParseInterval(interval); err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}
	return nil
}
