package service

import (
	"context"
	"sync"
)

// RunWorkers runs workers with at most parallelN goroutines concurrently.
//
// NOTE: Each Worker holds an independent Kafka consumer group member.
// Effective parallelism is bounded by topic partition count, not parallelN.
// This is intentional: partition-level fan-out and within-pod concurrency
// design are deferred to v0.5.0 (#23).
//
// Horizontal scaling (pod count) is managed externally by KEDA.
func RunWorkers(ctx context.Context, workers []*Worker, parallelN int) {
	sem := make(chan struct{}, parallelN)
	var wg sync.WaitGroup

	for _, w := range workers {
		wg.Add(1)
		sem <- struct{}{}
		go func(worker *Worker) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := worker.Run(ctx); err != nil {
				worker.Logger.Error(ctx, "worker exited with error")
			}
			worker.Close(ctx)
		}(w)
	}
	wg.Wait()
}
