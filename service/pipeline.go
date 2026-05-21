package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"lumor_puls/tools"
	"lumor_puls/types"
)

// Pipeline runs capture → diff → persist for one task.
type Pipeline struct {
	deps      Deps
	store     *Store
	browser   *Browser
	extractor *Extractor
}

func NewPipeline(deps Deps) *Pipeline {
	return &Pipeline{
		deps:      deps,
		store:     NewStore(deps.DB),
		browser:   NewBrowser(deps.Config.Browser),
		extractor: NewExtractor(deps.Config),
	}
}

// RunTask executes the full monitor loop for one task id.
func (p *Pipeline) RunTask(ctx context.Context, taskID string) error {
	task, err := p.store.GetTask(taskID)
	if err != nil {
		return err
	}
	if !task.Enabled {
		return fmt.Errorf("task %s disabled", taskID)
	}

	runErr := p.run(ctx, task)
	if markErr := p.store.MarkTaskRun(taskID, runErr); markErr != nil {
		log.Printf("mark task run %s: %v", taskID, markErr)
	}
	return runErr
}

func (p *Pipeline) run(ctx context.Context, task *types.MonitorTask) error {
	log.Printf("pipeline: task=%s step=capture url=%s", task.ID, task.URL)

	cap, err := p.browser.Capture(ctx, task.ID, task.URL)
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}
	log.Printf("pipeline: task=%s step=capture_done title=%q text_len=%d", task.ID, cap.Title, len(cap.Text))

	hash := tools.ContentHash(cap.Text)
	now := time.Now()
	cur := &types.Snapshot{
		TaskID:      task.ID,
		URL:         cap.URL,
		Title:       cap.Title,
		Text:        cap.Text,
		ContentHash: hash,
		CapturedAt:  now,
	}

	log.Printf("pipeline: task=%s step=load_previous_snapshot", task.ID)
	prev, err := p.store.LastSnapshot(task.ID)
	if err != nil {
		return err
	}
	if prev == nil {
		if err := p.store.SaveSnapshot(cur); err != nil {
			return err
		}
		log.Printf("pipeline: task=%s step=baseline_saved hash=%s", task.ID, hash[:12])
		return nil
	}

	if prev.ContentHash == hash {
		log.Printf("pipeline: task=%s step=unchanged hash=%s", task.ID, hash[:12])
		return nil
	}

	log.Printf("pipeline: task=%s step=content_changed prev_hash=%s new_hash=%s", task.ID, prev.ContentHash[:12], hash[:12])
	if err := p.store.SaveSnapshot(cur); err != nil {
		return err
	}

	log.Printf("pipeline: task=%s step=llm_diff", task.ID)
	result, err := p.extractor.Diff(ctx, *task, prev, cur)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	rows := signalsFromExtract(task, cap.URL, result)
	if len(rows) == 0 {
		log.Printf("pipeline: task=%s step=no_signals summary=%q", task.ID, resultSummary(result))
		return nil
	}
	if err := p.store.InsertSignals(rows); err != nil {
		return err
	}
	log.Printf("pipeline: task=%s step=signals_saved count=%d", task.ID, len(rows))
	return nil
}

func resultSummary(r *types.ExtractResult) string {
	if r == nil {
		return ""
	}
	return r.Summary
}

func signalsFromExtract(task *types.MonitorTask, url string, r *types.ExtractResult) []types.Signal {
	if r == nil || len(r.Changes) == 0 {
		if r != nil && strings.Contains(strings.ToLower(r.Summary), "no meaningful change") {
			return nil
		}
		if r != nil && r.Summary != "" {
			return []types.Signal{{
				TaskID:     task.ID,
				URL:        url,
				SignalType: types.SignalTypeOther,
				Summary:    r.Summary,
				Severity:   types.SeverityLow,
			}}
		}
		return nil
	}
	rows := make([]types.Signal, 0, len(r.Changes))
	for _, c := range r.Changes {
		st := c.Type
		if st == "" {
			st = types.SignalTypeOther
		}
		sev := c.Severity
		if sev == "" {
			sev = types.SeverityMedium
		}
		sum := r.Summary
		if c.New != "" {
			sum = strings.TrimSpace(c.Type + ": " + c.New)
		}
		rows = append(rows, types.Signal{
			TaskID:     task.ID,
			URL:        url,
			SignalType: st,
			Summary:    sum,
			Severity:   sev,
			OldExcerpt: c.Old,
			NewExcerpt: c.New,
		})
	}
	return rows
}
