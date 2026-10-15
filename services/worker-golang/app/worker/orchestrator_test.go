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

// TestRunWorkers_WorkerErrorStopsOthersAndReturnsError verifies that when one worker
// exits with an error, the remaining workers are stopped via context cancellation and
// the error is returned to the caller.
func TestRunWorkers_WorkerErrorStopsOthersAndReturnsError(t *testing.T) {
	src0 := NewMockSource()
	sink0 := &MockSink{}
	w0 := newOrchestratorTestWorker(0, src0, sink0)

	src1 := NewMockSource()
	sink1 := &MockSink{}
	w1 := newOrchestratorTestWorker(1, src1, sink1)

	expectedErr := errors.New("fatal database connection lost")
	// Push error to worker 0 so it fails immediately upon running
	src0.PushError(expectedErr)

	done := make(chan error, 1)
	go func() {
		done <- worker.RunWorkers(context.Background(), []*worker.Worker{w0, w1}, 2)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected non-nil error from RunWorkers, got nil")
		}
		if !errors.Is(err, expectedErr) && err.Error() != expectedErr.Error() {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunWorkers timed out; peer workers were not canceled")
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
