package worker

import (
	"context"

	"safezone.service.worker-golang/app/event"
)

type EventSource interface {
	Poll(ctx context.Context, max int) ([]event.CovidEvent, error) // Poll retrieves up to max events from the source, blocking until available or context expires
	Commit(ctx context.Context) error                              // Commit commits progress for events processed so far
	Retry()                                                        // Retry restores pre-poll commit progress and replays the most recent poll after a failed write
	AllowRebalance()                                               // AllowRebalance allows consumer group rebalances to proceed
	Close(ctx context.Context) error                               // Close the event source gracefully, releasing any resources
}
