package service

import (
	"context"
	"log"
	"time"

	"lumor_puls/config"
	"lumor_puls/tools"
	"lumor_puls/types"
)

// Scheduler ticks and runs due monitor tasks (serial via Runner).
type Scheduler struct {
	deps    Deps
	store   *Store
	runner  *Runner
	tick    time.Duration
}

func NewScheduler(deps Deps, runner *Runner) *Scheduler {
	sec := deps.Config.Scheduler.TickSec
	if sec <= 0 {
		sec = 60
	}
	return &Scheduler{
		deps:   deps,
		store:  NewStore(deps.DB),
		runner: runner,
		tick:   time.Duration(sec) * time.Second,
	}
}

// Start runs the scheduler loop until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	log.Printf("scheduler: started tick=%s", s.tick)

	s.tickOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler: stopped")
			return
		case <-ticker.C:
			s.tickOnce(ctx)
		}
	}
}

func (s *Scheduler) tickOnce(ctx context.Context) {
	tasks, err := s.store.ListEnabledTasks()
	if err != nil {
		log.Printf("scheduler: list tasks: %v", err)
		return
	}
	var due []types.MonitorTask
	for i := range tasks {
		if s.isDue(&tasks[i]) {
			due = append(due, tasks[i])
		}
	}
	if len(due) == 0 {
		return
	}
	log.Printf("scheduler: %d due task(s), running serially", len(due))
	for i := range due {
		if ctx.Err() != nil {
			return
		}
		t := due[i]
		runCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		err := s.runner.RunTask(runCtx, t.ID)
		cancel()
		if err != nil {
			log.Printf("scheduler: task %s failed: %v", t.ID, err)
		}
	}
}

func (s *Scheduler) isDue(t *types.MonitorTask) bool {
	d, err := tools.ParseInterval(t.Interval)
	if err != nil {
		log.Printf("scheduler: bad interval task=%s: %v", t.ID, err)
		return false
	}
	if t.LastRunAt == nil {
		return true
	}
	return time.Since(*t.LastRunAt) >= d
}

// SchedulerEnabled reports whether scheduler should run.
func SchedulerEnabled(cfg config.Config) bool {
	return cfg.Scheduler.Enabled
}
