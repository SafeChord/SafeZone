package adapter

import (
	"context"

	"safezone.service.worker-golang/app/schema"
)

type mockResult struct {
	event *schema.CovidEvent
	err   error
}

// MockSource is a controllable EventSource for testing.
// Push events via Push() or inject errors via PushError().
// GetEvent blocks until an event/error is available or ctx is done.
type MockSource struct {
	ch chan mockResult
}

func NewMockSource() *MockSource {
	return &MockSource{ch: make(chan mockResult, 100)}
}

func (m *MockSource) Push(event *schema.CovidEvent) {
	m.ch <- mockResult{event: event}
}

func (m *MockSource) PushError(err error) {
	m.ch <- mockResult{err: err}
}

func (m *MockSource) GetEvent(ctx context.Context) (*schema.CovidEvent, error) {
	select {
	case r := <-m.ch:
		return r.event, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MockSource) Close(ctx context.Context) error {
	return nil
}
