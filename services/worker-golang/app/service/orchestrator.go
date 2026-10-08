package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RunWorkers runs workers with at most parallelN goroutines concurrently.
//
// NOTE: Each Worker holds an independent Kafka consumer group member.
// Effective parallelism is bounded by topic partition count, not parallelN.
// This is intentional: partition-level fan-out and within-pod concurrency
// design are deferred to v0.5.0 (#23).
//
// Horizontal scaling (pod count) is managed externally by KEDA.
//
// If a worker terminates with an error, peer workers are stopped via context
// cancellation (ensuring graceful shutdown: flush, commit, leave group),
// and the error is returned to the caller.
func RunWorkers(ctx context.Context, workers []*Worker, parallelN int) error {
	if parallelN <= 0 {
		parallelN = len(workers)
	}
	if parallelN <= 0 {
		return nil
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	sem := make(chan struct{}, parallelN)
	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)

	for _, w := range workers {
		wg.Add(1)
		sem <- struct{}{}
		go func(worker *Worker) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := worker.Run(workerCtx); err != nil {
				worker.Logger.Error(workerCtx, "worker exited with error", zap.Error(err))
				if ctx.Err() == nil {
					errOnce.Do(func() {
						firstErr = err
						cancelWorkers()
					})
				}
			}

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer closeCancel()
			worker.Close(closeCtx)
		}(w)
	}
	wg.Wait()

	if ctx.Err() != nil {
		// Normal shutdown via external signal (SIGTERM/SIGINT) ends with status 0
		return nil
	}
	return firstErr
}
