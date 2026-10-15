package worker

import (
	"context"

	"safezone.service.worker-golang/app/event"
)

type EventSink interface {
	Flush(context.Context, *[]event.CovidEvent) error
	Close(ctx context.Context) error
}
