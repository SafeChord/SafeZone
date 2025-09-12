package strategy

import (
	"context"

	"go.uber.org/zap"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
)

type MockSink struct {
	Logger *logger.ContextLogger
}

func (m *MockSink) Flush(ctx context.Context, buffer *[]schema.CovidEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		for _, evt := range *buffer {
			m.Logger.Debug(ctx, "Flushing event",
				zap.String("event_type", evt.EventType),
				zap.String("trace_id", evt.TraceID),
				zap.String("date", evt.Payload.Date),
				zap.String("city", evt.Payload.City),
				zap.String("region", evt.Payload.Region),
				zap.Int("cases", evt.Payload.Cases))
		}
		// clear the buffer after flushing
		*buffer = (*buffer)[:0]
		return nil
	}
}

func (m *MockSink) Close(ctx context.Context) error {
	m.Logger.Info(ctx, "MockSink closed")
	return nil
}
