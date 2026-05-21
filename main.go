package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lumor_puls/api"
	"lumor_puls/config"
	"lumor_puls/service"
	"lumor_puls/tools"
)

func main() {
	runOnce := flag.String("run", "", "run a single task id once and exit")
	flag.Parse()

	cfg := config.GetConfig()
	if err := tools.InitDB(cfg); err != nil {
		panic(fmt.Errorf("init db: %w", err))
	}
	defer func() { _ = tools.CloseDB() }()

	deps := service.Deps{DB: tools.DB(), Config: cfg}
	runner := service.NewRunner(deps)

	if *runOnce != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if err := runner.RunTask(ctx, *runOnce); err != nil {
			log.Fatalf("run task %s: %v", *runOnce, err)
		}
		log.Printf("task %s finished", *runOnce)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if service.SchedulerEnabled(cfg) {
		sched := service.NewScheduler(deps, runner)
		tools.SafeGo(func() { sched.Start(ctx) })
	}

	router := api.SetupRouter(deps, runner)
	addr := api.ListenAddr(cfg)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 20 * time.Minute,
	}

	go func() {
		log.Printf("lumor_puls listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Errorf("listen: %w", err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
