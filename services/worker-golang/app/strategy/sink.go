package strategy

import (
	"context"

	"safezone.service.worker-golang/app/schema"
)

type EventSink interface {
	Flush(context.Context, *[]schema.CovidEvent) error
	Close(ctx context.Context) error
}
