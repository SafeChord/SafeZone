package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := logger.NewContextLogger(cfg.ServiceName, cfg.ServiceVersion, cfg.Environment)
	log.Info(ctx, "Worker-golang service started")

	workers := make([]*service.Worker, 0, cfg.WorkerCount)
	for i := 0; i < cfg.WorkerCount; i++ {
		workers = append(workers, service.NewWorker(i, cfg, log))
	}

	service.RunWorkers(ctx, workers, cfg.ParallelN)

	log.Info(ctx, "Worker-golang service completed")
}
