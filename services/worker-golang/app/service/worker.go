package service

import (
	"context"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/cache"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/strategy"
)

type Worker struct {
	DB        *sqlx.DB               // database connection, if needed
	Cache     *cache.Cache           // cache for city and region mappings
	Validator *schema.CovidValidator // validator for events
	Source    adapter.EventSource
	Sink      strategy.EventSink
	Config    *config.Config
	Logger    *logger.ContextLogger // logger for logging events
	ID        int                   // worker ID for logging and identification
}

func (w *Worker) Run(ctx context.Context) error {
	workerCtx := context.WithValue(ctx, w.Logger.WorkerIDKey, w.ID)
	w.Logger.Info(workerCtx, "Starting worker", zap.String("event", "Worker started"))

	var (
		inFlight       []schema.CovidEvent
		inFlightPolled bool
	)

	// On exit / shutdown: flush and commit anything polled but not yet persisted on a fresh context
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownCtx = context.WithValue(shutdownCtx, w.Logger.WorkerIDKey, w.ID)

		if inFlightPolled {
			if len(inFlight) > 0 && w.Sink != nil {
				w.Logger.Info(shutdownCtx, "Flushing in-flight events before shutdown",
					zap.String("event", "Flushing in-flight events"))
				if err := w.Sink.Flush(shutdownCtx, &inFlight); err != nil {
					w.Logger.Error(shutdownCtx, "Failed to flush in-flight events on shutdown", zap.Error(err))
					if w.Source != nil {
						w.Source.AllowRebalance()
					}
					return
				}
			}
			if w.Source != nil {
				if err := w.Source.Commit(shutdownCtx); err != nil {
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

	for {
		if ctx.Err() != nil {
			w.Logger.Info(workerCtx, "Context canceled, stopping worker",
				zap.String("event", "context canceled"))
			return nil
		}

		// 1. Poll up to BatchSize records, waiting at most FlushInterval.
		readCtx, cancel := context.WithTimeout(ctx, w.Config.FlushInterval)
		events, err := w.Source.Poll(readCtx, batchSize)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				continue
			} else if errors.Is(err, context.Canceled) {
				w.Logger.Info(workerCtx, "Get context canceled, stopping worker",
					zap.String("event", "context canceled"))
				return nil
			} else {
				w.Logger.Error(workerCtx, "Failed to get events from source", zap.Error(err))
				return err
			}
		}

		// 2. Parse and validate them. Invalid ones are skipped but still count as read.
		validEvents := make([]schema.CovidEvent, 0, len(events))
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

		inFlight = validEvents
		inFlightPolled = true

		// 3. If any valid events came out of this poll, flush them. If the flush fails, do not commit.
		if len(validEvents) > 0 && w.Sink != nil {
			if err := w.Sink.Flush(workerCtx, &validEvents); err != nil {
				w.Logger.Error(workerCtx, "Failed to flush events", zap.Error(err))
				return err
			}
		}
		inFlight = nil

		// 4. Commit the offsets of everything this poll returned.
		// 5. Call AllowRebalance (executed inside Source.Commit).
		if w.Source != nil {
			if err := w.Source.Commit(workerCtx); err != nil {
				w.Logger.Error(workerCtx, "Failed to commit offsets", zap.Error(err))
				return err
			}
		}

		inFlightPolled = false

		// 6. Only now poll again.
	}
}

func (w *Worker) Close(ctx context.Context) {

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
