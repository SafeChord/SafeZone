package worker

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/event"
	"safezone.service.worker-golang/app/logger"
	"safezone.service.worker-golang/app/postgres"
)

type Worker struct {
	DB           *sqlx.DB              // database connection, if needed
	Cache        *postgres.Cache       // cache for city and region mappings
	Validator    *event.CovidValidator // validator for events
	Source       EventSource
	Sink         EventSink
	Config       *config.Config
	Logger       *logger.ContextLogger // logger for logging events
	ID           int                   // worker ID for logging and identification
	lastLoopTurn atomic.Int64
}

func (w *Worker) Run(ctx context.Context) error {
	workerCtx := context.WithValue(ctx, w.Logger.WorkerIDKey, strconv.Itoa(w.ID))
	w.Logger.Info(workerCtx, "Starting worker", zap.String("event", "Worker started"))

	var (
		inFlight       []event.CovidEvent
		inFlightPolled bool
		failureCount   int
	)

	// On exit / shutdown: flush and commit anything polled but not yet persisted on a fresh context
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownCtx = context.WithValue(shutdownCtx, w.Logger.WorkerIDKey, strconv.Itoa(w.ID))

		if inFlightPolled {
			if len(inFlight) > 0 && w.Sink != nil {
				w.Logger.Info(shutdownCtx, "Flushing in-flight events before shutdown",
					zap.String("event", "Flushing in-flight events"))
				flushCtx, flushCancel := context.WithTimeout(shutdownCtx, w.dependencyTimeout())
				err := w.Sink.Flush(flushCtx, &inFlight)
				flushCancel()
				if err != nil {
					w.Logger.Error(shutdownCtx, "Failed to flush in-flight events on shutdown", zap.Error(err))
					if w.Source != nil {
						w.Source.AllowRebalance()
					}
					return
				}
			}
			if w.Source != nil {
				commitCtx, commitCancel := context.WithTimeout(shutdownCtx, w.dependencyTimeout())
				err := w.Source.Commit(commitCtx)
				commitCancel()
				if err != nil {
					w.Logger.Error(shutdownCtx, "Failed to commit in-flight offsets on shutdown", zap.Error(err))
				}
			}
		} else if w.Source != nil {
			w.Source.AllowRebalance()
		}
	}()

	batchSize := w.Config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	flushInterval := w.Config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	for {
		if ctx.Err() != nil {
			w.Logger.Info(workerCtx, "Context canceled, stopping worker",
				zap.String("event", "context canceled"))
			return nil
		}
		w.markLoopTurn()

		// 1. Poll up to BatchSize records, waiting at most FlushInterval.
		readCtx, cancel := context.WithTimeout(ctx, flushInterval)
		events, err := w.Source.Poll(readCtx, batchSize)
		pollTimedOut := errors.Is(err, context.DeadlineExceeded) && errors.Is(readCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
		cancel()

		if ctx.Err() != nil {
			w.Logger.Info(workerCtx, "Context canceled, stopping worker",
				zap.String("event", "context canceled"))
			return nil
		}
		if err != nil && !pollTimedOut {
			w.Logger.Error(workerCtx, "Failed to get events from source", zap.Error(err))
			failureCount = nextFailureCount(failureCount)
			if !waitForRetry(ctx, retryDelay(failureCount, w.Config.LivenessWindow)) {
				return nil
			}
			continue
		}

		// 2. Parse and validate them. Invalid ones are skipped but still count as read.
		validEvents := make([]event.CovidEvent, 0, len(events))
		for _, event := range events {
			logCtx := context.WithValue(workerCtx, w.Logger.TraceIDKey, event.TraceID)
			if w.Validator != nil && !w.Validator.Validate(logCtx, event) {
				w.Logger.Warn(logCtx, "An event validation failed, skipping event",
					zap.String("event", "Event validation failed"))
				continue
			}
			w.Logger.Info(logCtx, "Received event from source",
				zap.String("event", "Event received"))
			validEvents = append(validEvents, event)
		}

		inFlight = append(inFlight[:0], validEvents...)
		inFlightPolled = true

		// 3. If any valid events came out of this poll, flush them. If the flush fails, do not commit.
		if len(validEvents) > 0 && w.Sink != nil {
			flushCtx, flushCancel := context.WithTimeout(workerCtx, w.dependencyTimeout())
			err := w.Sink.Flush(flushCtx, &validEvents)
			flushCancel()
			if err != nil {
				if ctx.Err() != nil {
					w.Logger.Info(workerCtx, "Context canceled during write; flushing in-flight events on shutdown",
						zap.String("event", "context canceled"))
					return nil
				}
				w.Logger.Error(workerCtx, "Failed to flush events", zap.Error(err))
				w.Source.Retry()
				inFlight = nil
				inFlightPolled = false
				failureCount = nextFailureCount(failureCount)
				if !waitForRetry(ctx, retryDelay(failureCount, w.Config.LivenessWindow)) {
					return nil
				}
				continue
			}
		}
		inFlight = nil

		// 4. Commit the offsets of everything this poll returned.
		// 5. Call AllowRebalance (executed inside Source.Commit).
		if w.Source != nil {
			commitCtx, commitCancel := context.WithTimeout(workerCtx, w.dependencyTimeout())
			err := w.Source.Commit(commitCtx)
			commitCancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				w.Logger.Error(workerCtx, "Failed to commit offsets", zap.Error(err))
				failureCount = nextFailureCount(failureCount)
				if !waitForRetry(ctx, retryDelay(failureCount, w.Config.LivenessWindow)) {
					return nil
				}
				continue
			}
		}

		failureCount = 0
		inFlightPolled = false

		// 6. Only now poll again.
	}
}

const (
	baseRetryDelay = 250 * time.Millisecond
	maxRetryDelay  = time.Second
)

func (w *Worker) dependencyTimeout() time.Duration {
	if w.Config != nil && w.Config.DependencyTimeout > 0 {
		return w.Config.DependencyTimeout
	}
	return config.DefaultDependencyTimeout
}

func retryDelay(failureCount int, livenessWindow time.Duration) time.Duration {
	maxDelay := maxRetryDelay
	if livenessWindow > 0 {
		windowLimit := livenessWindow / 4
		if windowLimit < baseRetryDelay {
			windowLimit = baseRetryDelay
		}
		if windowLimit < maxDelay {
			maxDelay = windowLimit
		}
	}
	delay := baseRetryDelay
	for attempt := 1; attempt < failureCount && delay < maxDelay; attempt++ {
		delay *= 2
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func nextFailureCount(failureCount int) int {
	if failureCount < 5 {
		return failureCount + 1
	}
	return failureCount
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

func (w *Worker) Close(ctx context.Context) {
	ctx = context.WithValue(ctx, w.Logger.WorkerIDKey, strconv.Itoa(w.ID))

	if err := w.Source.Close(ctx); err != nil {
		w.Logger.Error(ctx, "Failed to close event source", zap.Error(err))
	}
	if err := w.Sink.Close(ctx); err != nil {
		w.Logger.Error(ctx, "Failed to close event sink", zap.Error(err))
	}
	if w.DB != nil {
		if err := w.DB.Close(); err != nil {
			w.Logger.Error(ctx, "Failed to close database connection", zap.Error(err))
		}
	}
	w.Logger.Info(ctx, "Closing worker", zap.String("event", "Worker closed"))

}
