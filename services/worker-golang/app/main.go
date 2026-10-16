package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	if err := runWorkersWithHealth(ctx, cfg, workers); err != nil {
		log.Error(ctx, "Worker-golang service exited with error", zap.Error(err))
		os.Exit(1)
	}

	log.Info(ctx, "Worker-golang service completed")
}

func runWorkersWithHealth(ctx context.Context, cfg *config.Config, workers []*worker.Worker) error {
	address := net.JoinHostPort(cfg.ServerIP, strconv.Itoa(cfg.ServerPort))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on worker health address %q: %w", address, err)
	}
	return runWorkersWithHealthOnListener(ctx, listener, cfg.LivenessWindow, cfg.ParallelN, workers)
}

func runWorkersWithHealthOnListener(ctx context.Context, listener net.Listener, window time.Duration, parallelN int, workers []*worker.Worker) error {
	server := &http.Server{
		Handler:           worker.NewHealthHandler(window, workers...),
		ReadHeaderTimeout: 5 * time.Second,
	}
	runCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	serveErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			cancelWorkers()
		}
	}()

	workersErr := worker.RunWorkers(runCtx, workers, parallelN)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		_ = server.Close()
	}

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve worker health endpoint: %w", err)
	default:
	}
	if workersErr != nil {
		return workersErr
	}
	if shutdownErr != nil {
		return fmt.Errorf("shut down worker health endpoint: %w", shutdownErr)
	}
	return nil
}
