package service

import (
	"context"
	"errors"

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
	buffer := make([]schema.CovidEvent, 0, w.Config.BatchSize)

	// add worker ID to the context for logging
	ctx = context.WithValue(ctx, w.Logger.WorkerIDKey, w.ID)

	w.Logger.Info(ctx, "Starting worker", zap.String("event", "Worker started"))

	// flush remain events in buffer on exit
	defer func() {
		if len(buffer) > 0 {
			w.Logger.Info(ctx, "Flushing remaining events before shutdown",
				zap.String("event", "Flushing remaining events"))
			// flush the remaining events
			w.Sink.Flush(ctx, &buffer)
		}
	}()

	for {

		readCtx, cancel := context.WithTimeout(ctx, w.Config.FlushInterval)
		defer cancel()

		event, err := w.Source.GetEvent(readCtx)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				if len(buffer) == 0 {
					continue
				}
				// timeout reached, flush if there are events in the buffer
				w.Sink.Flush(ctx, &buffer)

			} else if errors.Is(err, context.Canceled) {
				w.Logger.Info(ctx, "Get context canceled, stopping worker",
					zap.String("event", "context canceled"))
				return nil
			} else {
				w.Logger.Error(ctx, "Failed to get event from source", zap.Error(err))
				return err
			}
		} else {
			// if the event can't pass validation, skip it
			// future: we add the bad event to a dead-letter queue
			if w.Validator != nil && !w.Validator.Validate(ctx, *event) {
				w.Logger.Warn(ctx, "An event validation failed, skipping event",
					zap.String("event", "Event validation failed"))
				continue
			}
			// if the event is valid, add trace to the context for logging
			ctx = context.WithValue(ctx, w.Logger.TraceIDKey, event.TraceID)

			w.Logger.Info(ctx, "Received event from source",
				zap.String("event", "Event received"))

			buffer = append(buffer, *event)
		}
		// stop timer during flushing
		cancel()

		if len(buffer) >= w.Config.BatchSize {

			w.Sink.Flush(ctx, &buffer)

			if err != nil {
				if errors.Is(err, context.Canceled) {
					w.Logger.Info(ctx, "Get context canceled, stopping worker",
						zap.String("event", "context canceled"))
					return nil
				} else {
					w.Logger.Error(ctx, "Failed to flush events", zap.Error(err))
					return err
				}
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
