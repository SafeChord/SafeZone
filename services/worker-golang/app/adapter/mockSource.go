package adapter

import (
	"context"
	"sync"

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
	ch        chan mockResult
	mu        sync.Mutex
	committed int
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

func (m *MockSource) Poll(ctx context.Context, max int) ([]schema.CovidEvent, error) {
	if max <= 0 {
		max = 1
	}
	select {
	case r := <-m.ch:
		if r.err != nil {
			return nil, r.err
		}
		events := []schema.CovidEvent{*r.event}
		for len(events) < max {
			select {
			case next := <-m.ch:
				if next.err != nil {
					return events, nil
				}
				events = append(events, *next.event)
			default:
				return events, nil
			}
		}
		return events, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MockSource) GetEvent(ctx context.Context) (*schema.CovidEvent, error) {
	select {
	case r := <-m.ch:
		return r.event, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *MockSource) Commit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.committed++
	return nil
}

func (m *MockSource) CommittedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.committed
}

func (m *MockSource) FilterAssigned(events []schema.CovidEvent) []schema.CovidEvent {
	return events
}

func (m *MockSource) Close(ctx context.Context) error {
	return nil
}
