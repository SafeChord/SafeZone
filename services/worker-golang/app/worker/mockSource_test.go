package worker_test

import (
	"context"
	"sync"

	"safezone.service.worker-golang/app/event"
)

type mockResult struct {
	event *event.CovidEvent
	err   error
}

// MockSource is a controllable EventSource for testing.
// Push events via Push() or inject errors via PushError().
// Poll blocks until an event/error is available or ctx is done.
type MockSource struct {
	ch          chan mockResult
	mu          sync.Mutex
	committed   int
	commitCalls int
	commitErrs  []error
	retry       []event.CovidEvent
	lastPolled  []event.CovidEvent
}

func NewMockSource() *MockSource {
	return &MockSource{ch: make(chan mockResult, 100)}
}

func (m *MockSource) Push(event *event.CovidEvent) {
	m.ch <- mockResult{event: event}
}

func (m *MockSource) PushError(err error) {
	m.ch <- mockResult{err: err}
}

func (m *MockSource) PushCommitError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commitErrs = append(m.commitErrs, err)
}

func (m *MockSource) Poll(ctx context.Context, max int) ([]event.CovidEvent, error) {
	if max <= 0 {
		max = 1
	}

	m.mu.Lock()
	if len(m.retry) > 0 {
		events := m.retry[:min(max, len(m.retry))]
		m.retry = m.retry[len(events):]
		m.mu.Unlock()
		m.rememberPoll(events)
		return events, nil
	}
	m.mu.Unlock()

	var first mockResult
	select {
	case first = <-m.ch:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if first.err != nil {
		return nil, first.err
	}
	events := []event.CovidEvent{*first.event}
	for len(events) < max {
		select {
		case next := <-m.ch:
			if next.err != nil {
				m.rememberPoll(events)
				return events, nil
			}
			events = append(events, *next.event)
		default:
			m.rememberPoll(events)
			return events, nil
		}
	}
	m.rememberPoll(events)
	return events, nil
}

func (m *MockSource) rememberPoll(events []event.CovidEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastPolled = append(m.lastPolled[:0], events...)
}

func (m *MockSource) Commit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commitCalls++
	if len(m.commitErrs) > 0 {
		err := m.commitErrs[0]
		m.commitErrs = m.commitErrs[1:]
		return err
	}
	m.committed++
	m.lastPolled = nil
	return nil
}

func (m *MockSource) Retry() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.retry = append(append([]event.CovidEvent(nil), m.lastPolled...), m.retry...)
	m.lastPolled = nil
}

func (m *MockSource) CommittedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.committed
}

func (m *MockSource) CommitAttempts() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.commitCalls
}

func (m *MockSource) AllowRebalance() {}

func (m *MockSource) Close(ctx context.Context) error {
	return nil
}
