package service

import (
	"context"
	"log"
	"sync"
)

// Runner runs pipeline tasks one at a time (single browser).
type Runner struct {
	pipeline *Pipeline
	mu       sync.Mutex
}

// NewRunner creates a runner backed by the shared pipeline.
func NewRunner(deps Deps) *Runner {
	return &Runner{pipeline: NewPipeline(deps)}
}

// RunTask executes a task while holding the global browser lock.
func (r *Runner) RunTask(ctx context.Context, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	log.Printf("runner: start task=%s (browser lock held)", taskID)
	err := r.pipeline.RunTask(ctx, taskID)
	if err != nil {
		log.Printf("runner: task=%s failed: %v", taskID, err)
		return err
	}
	log.Printf("runner: task=%s done", taskID)
	return nil
}
