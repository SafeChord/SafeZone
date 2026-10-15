package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"go.uber.org/zap"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/logger"
	"safezone.service.worker-golang/app/worker"
)

// setupSignalContext returns a context that is canceled when SIGINT or SIGTERM is received.
func setupSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

var newWorker = worker.NewWorker

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := setupSignalContext()
	defer stop()

	log := logger.NewContextLogger(cfg.ServiceName, cfg.ServiceVersion, cfg.Environment)
	log.Info(ctx, "Worker-golang service started")

	workers := make([]*worker.Worker, 0, cfg.WorkerCount)
	for i := 0; i < cfg.WorkerCount; i++ {
		w, err := newWorker(i, cfg, log)
		if err != nil {
			workerCtx := context.WithValue(ctx, log.WorkerIDKey, strconv.Itoa(i))
			log.Error(workerCtx, "Failed to initialize worker", zap.Error(err))
			os.Exit(1)
		}
		workers = append(workers, w)
	}

	if err := worker.RunWorkers(ctx, workers, cfg.ParallelN); err != nil {
		log.Error(ctx, "Worker-golang service exited with error", zap.Error(err))
		os.Exit(1)
	}

	log.Info(ctx, "Worker-golang service completed")
}
