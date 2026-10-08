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

func (w *Worker) flushAndCommit(ctx context.Context, buffer *[]schema.CovidEvent) error {
	if len(*buffer) > 0 && w.Sink != nil {
		if err := w.Sink.Flush(ctx, buffer); err != nil {
			w.Logger.Error(ctx, "Failed to flush events to sink", zap.Error(err))
			return err
		}
	}
	if w.Source != nil {
		if err := w.Source.Commit(ctx); err != nil {
			w.Logger.Error(ctx, "Failed to commit offsets to source", zap.Error(err))
			return err
		}
	}
	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	buffer := make([]schema.CovidEvent, 0, w.Config.BatchSize)

	// add worker ID to the context for logging
	workerCtx := context.WithValue(ctx, w.Logger.WorkerIDKey, w.ID)

	w.Logger.Info(workerCtx, "Starting worker", zap.String("event", "Worker started"))

	// flush remaining events in buffer on exit (shutdown flush)
	defer func() {
		// WG-2: shutdown flush runs with a non-canceled context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		shutdownCtx = context.WithValue(shutdownCtx, w.Logger.WorkerIDKey, w.ID)

		if len(buffer) > 0 {
			w.Logger.Info(shutdownCtx, "Flushing remaining events before shutdown",
				zap.String("event", "Flushing remaining events"))
			if err := w.flushAndCommit(shutdownCtx, &buffer); err != nil {
				w.Logger.Error(shutdownCtx, "Failed to flush remaining events on shutdown", zap.Error(err))
			}
		} else if w.Source != nil {
			if err := w.Source.Commit(shutdownCtx); err != nil {
				w.Logger.Error(shutdownCtx, "Failed to commit remaining offsets on shutdown", zap.Error(err))
			}
		}
	}()

	for {
		if ctx.Err() != nil {
			w.Logger.Info(workerCtx, "Context canceled, stopping worker",
				zap.String("event", "context canceled"))
			return nil
		}

		readCtx, cancel := context.WithTimeout(ctx, w.Config.FlushInterval)
		event, err := w.Source.GetEvent(readCtx)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// WG-8: timeout reached, flush and handle errors
				if err := w.flushAndCommit(workerCtx, &buffer); err != nil {
					w.Logger.Error(workerCtx, "Failed to flush on timeout", zap.Error(err))
					return err
				}
			} else if errors.Is(err, context.Canceled) {
				w.Logger.Info(workerCtx, "Get context canceled, stopping worker",
					zap.String("event", "context canceled"))
				return nil
			} else {
				w.Logger.Error(workerCtx, "Failed to get event from source", zap.Error(err))
				return err
			}
			continue
		}

		// WG-4 & WK-R8: Per-event logging context carrying TraceID without mutating workerCtx
		logCtx := context.WithValue(workerCtx, w.Logger.TraceIDKey, event.TraceID)

		// If validator is present and event fails validation, skip it (WK-R5)
		if w.Validator != nil && !w.Validator.Validate(logCtx, *event) {
			w.Logger.Warn(logCtx, "An event validation failed, skipping event",
				zap.String("event", "Event validation failed"))
			continue
		}

		w.Logger.Info(logCtx, "Received event from source",
			zap.String("event", "Event received"))
		buffer = append(buffer, *event)

		if len(buffer) >= w.Config.BatchSize {
			if err := w.flushAndCommit(workerCtx, &buffer); err != nil {
				w.Logger.Error(workerCtx, "Failed to flush events", zap.Error(err))
				return err
			}
		}
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
