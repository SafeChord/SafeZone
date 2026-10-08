package adapter

import (
	"context"

	"safezone.service.worker-golang/app/schema"
)

type EventSource interface {
	Poll(ctx context.Context, max int) ([]schema.CovidEvent, error) // Poll retrieves up to max events from the source, blocking until available or context expires
	GetEvent(ctx context.Context) (*schema.CovidEvent, error)       // GetEvent retrieves the next single event from the source
	Commit(ctx context.Context) error                                // Commit commits progress for events processed so far
	Close(ctx context.Context) error                                // Close the event source gracefully, releasing any resources
}
