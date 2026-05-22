package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"lumor_puls/tools"
	"lumor_puls/types"
)

// Pipeline runs capture → category extract → persist for one task.
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
	category := task.SignalCategory
	if category == "" {
		category = types.SignalCategoryEcosystem
	}
	log.Printf("pipeline: task=%s category=%s step=capture url=%s", task.ID, category, task.URL)

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
		log.Printf("pipeline: task=%s step=baseline_saved hash=%s", task.ID, hashPrefix(hash))
		return nil
	}

	if prev.ContentHash == hash {
		log.Printf("pipeline: task=%s step=unchanged hash=%s", task.ID, hashPrefix(hash))
		return nil
	}

	log.Printf("pipeline: task=%s step=content_changed prev=%s new=%s", task.ID, hashPrefix(prev.ContentHash), hashPrefix(hash))
	if err := p.store.SaveSnapshot(cur); err != nil {
		return err
	}

	log.Printf("pipeline: task=%s step=extract category=%s", task.ID, category)
	rows, err := p.extractor.Extract(ctx, *task, prev, cur)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	if len(rows) == 0 {
		log.Printf("pipeline: task=%s step=no_signals", task.ID)
		return nil
	}
	if err := p.store.InsertSignals(rows); err != nil {
		return err
	}
	log.Printf("pipeline: task=%s step=signals_saved count=%d category=%s", task.ID, len(rows), category)
	return nil
}

func hashPrefix(h string) string {
	if len(h) >= 12 {
		return h[:12]
	}
	return h
}
