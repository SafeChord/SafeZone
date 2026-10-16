package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/worker"
)

func newOrchestratorTestWorker(id int, src worker.EventSource, sink worker.EventSink) *worker.Worker {
	return &worker.Worker{
		Source: src,
		Sink:   sink,
		Config: &config.Config{
			BatchSize:     3,
			FlushInterval: 50 * time.Millisecond,
		},
		Logger: testLogger(),
		ID:     id,
	}
}

// TestRunWorkers_PollErrorDoesNotStopOthers verifies that an unhealthy dependency
// does not cancel peer workers or end the process.
func TestRunWorkers_PollErrorDoesNotStopOthers(t *testing.T) {
	src0 := NewMockSource()
	sink0 := &MockSink{}
	w0 := newOrchestratorTestWorker(0, src0, sink0)

	src1 := NewMockSource()
	sink1 := &MockSink{}
	w1 := newOrchestratorTestWorker(1, src1, sink1)

	src0.PushError(errors.New("temporary broker connection lost"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- worker.RunWorkers(ctx, []*worker.Worker{w0, w1}, 2)
	}()
	time.Sleep(150 * time.Millisecond)
	src1.Push(makeEvent(9))
	if !waitFor(t, time.Second, func() bool { return sink1.FlushedCount() > 0 }) {
		t.Fatal("peer worker stopped after another worker's poll error")
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunWorkers returned an error after parent cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunWorkers did not stop after parent cancellation")
	}
}

// TestRunWorkers_SignalContextCancellationReturnsNil verifies that when the parent
// context is canceled (simulating SIGTERM/SIGINT), RunWorkers terminates gracefully
// and returns nil (status 0 exit).
func TestRunWorkers_SignalContextCancellationReturnsNil(t *testing.T) {
	src0 := NewMockSource()
	sink0 := &MockSink{}
	w0 := newOrchestratorTestWorker(0, src0, sink0)

	src1 := NewMockSource()
	sink1 := &MockSink{}
	w1 := newOrchestratorTestWorker(1, src1, sink1)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	err := worker.RunWorkers(ctx, []*worker.Worker{w0, w1}, 2)
	if err != nil {
		t.Fatalf("expected nil error on signal cancellation, got %v", err)
	}
}
