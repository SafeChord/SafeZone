package strategy

import (
	"context"
	"sync"

	"safezone.service.worker-golang/app/schema"
)

// MockSink captures Flush calls for test assertions.
// Set FlushError to inject errors.
type MockSink struct {
	mu         sync.Mutex
	Flushed    [][]schema.CovidEvent
	FlushError error
}

func (m *MockSink) Flush(ctx context.Context, buffer *[]schema.CovidEvent) error {
	if m.FlushError != nil {
		return m.FlushError
	}
	snapshot := make([]schema.CovidEvent, len(*buffer))
	copy(snapshot, *buffer)
	m.mu.Lock()
	m.Flushed = append(m.Flushed, snapshot)
	m.mu.Unlock()
	*buffer = (*buffer)[:0]
	return nil
}

func (m *MockSink) FlushedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Flushed)
}

func (m *MockSink) Close(ctx context.Context) error {
	return nil
}
