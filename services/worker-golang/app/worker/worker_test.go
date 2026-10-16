package worker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap/zapcore"
	"time"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/event"
	"safezone.service.worker-golang/app/logger"
	"safezone.service.worker-golang/app/worker"
)

func testLogger() *logger.ContextLogger {
	return logger.NewContextLogger("test", "0.0.0", "test", zapcore.DebugLevel)
}

func newTestWorker(src worker.EventSource, sink worker.EventSink) *worker.Worker {
	return &worker.Worker{
		Source: src,
		Sink:   sink,
		Config: &config.Config{
			BatchSize:     3,
			FlushInterval: 50 * time.Millisecond,
		},
		Logger: testLogger(),
		ID:     0,
	}
}

func makeEvent(i int) *event.CovidEvent {
	e := &event.CovidEvent{
		EventType: "test",
		TraceID:   fmt.Sprintf("trace-%d", i),
	}
	e.Payload.Date = "2024-01-01"
	e.Payload.City = "Taipei"
	e.Payload.Region = "Zhongzheng"
	e.Payload.Cases = i
	return e
}

func runAsync(ctx context.Context, w *worker.Worker) <-chan error {
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	return done
}

// TestWorker_BatchFlush: exactly BatchSize events → one flush with all events
func TestWorker_BatchFlush(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	for i := 0; i < 3; i++ {
		src.Push(makeEvent(i))
	}
	time.Sleep(20 * time.Millisecond) // let worker consume and flush
	cancel()
	<-done

	if sink.FlushedCount() != 1 {
		t.Fatalf("expected 1 flush, got %d", sink.FlushedCount())
	}
	if len(sink.Flushed[0]) != 3 {
		t.Fatalf("expected 3 events in flush, got %d", len(sink.Flushed[0]))
	}
}

// TestWorker_TimeoutFlush: events < BatchSize → FlushInterval triggers flush
func TestWorker_TimeoutFlush(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	src.Push(makeEvent(0))
	src.Push(makeEvent(1))
	time.Sleep(150 * time.Millisecond) // wait past FlushInterval (50ms)
	cancel()
	<-done

	if sink.FlushedCount() < 1 {
		t.Fatal("expected at least 1 flush from timeout, got 0")
	}

	total := 0
	for _, batch := range sink.Flushed {
		total += len(batch)
	}
	if total != 2 {
		t.Fatalf("expected 2 total events flushed, got %d", total)
	}
}

// TestWorker_EmptyBufferOnTimeout: no events → timeout fires but no flush
func TestWorker_EmptyBufferOnTimeout(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	time.Sleep(150 * time.Millisecond)
	cancel()
	<-done

	if sink.FlushedCount() != 0 {
		t.Fatalf("expected 0 flushes on empty buffer, got %d", sink.FlushedCount())
	}
}

// TestWorker_GracefulShutdown: cancel with buffered events → deferred flush
func TestWorker_GracefulShutdown(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	src.Push(makeEvent(0))
	src.Push(makeEvent(1))
	time.Sleep(20 * time.Millisecond) // let events land in buffer
	cancel()
	<-done

	total := 0
	for _, batch := range sink.Flushed {
		total += len(batch)
	}
	if total != 2 {
		t.Fatalf("expected 2 total events on graceful shutdown, got %d", total)
	}
}

// TestWorker_ValidationReject: invalid events skipped, valid events processed
func TestWorker_ValidationReject(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	// wire up a validator with a mock cache that knows one valid city/region
	mockCache := &mockCacheReader{
		cities:  map[string]int{"Taipei": 1},
		regions: map[string]int{"1|Zhongzheng": 10},
	}
	w.Validator = event.NewCovidValidator(testLogger(), mockCache)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	valid := makeEvent(0) // Taipei/Zhongzheng — valid
	invalid := makeEvent(1)
	invalid.Payload.City = "UnknownCity"

	src.Push(valid)
	src.Push(invalid)
	time.Sleep(150 * time.Millisecond) // wait for timeout flush
	cancel()
	<-done

	total := 0
	for _, batch := range sink.Flushed {
		total += len(batch)
	}
	if total != 1 {
		t.Fatalf("expected 1 valid event flushed, got %d", total)
	}
}

// TestWorker_SourceError: a non-context poll error is logged and the worker keeps polling.
func TestWorker_SourceError(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runAsync(ctx, w)
	src.PushError(errors.New("kafka connection lost"))
	src.Push(makeEvent(1))

	select {
	case err := <-done:
		t.Fatalf("worker exited after poll error: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("worker returned an error on cancellation: %v", err)
	}
	if sink.FlushedCount() != 1 {
		t.Fatalf("expected worker to persist the event after poll recovery, got %d flushes", sink.FlushedCount())
	}
}

// TestWorker_TimeoutFlushError verifies a failed timeout flush is replayed on a later poll.
func TestWorker_TimeoutFlushError(t *testing.T) {
	src := NewMockSource()
	sink := &failOnceSink{firstErr: errors.New("db down on timeout flush")}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runAsync(ctx, w)
	src.Push(makeEvent(0))

	if !waitFor(t, time.Second, func() bool { return sink.successfulFlushes() == 1 }) {
		t.Fatal("worker did not replay and persist the event after the failed flush")
	}
	if sink.attempts() < 2 {
		t.Fatalf("expected failed write followed by replay, got %d attempts", sink.attempts())
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("worker returned an error on cancellation: %v", err)
	}
}

// TestWorker_BatchFlushError verifies that a batch write failure is replayed, not dropped.
func TestWorker_BatchFlushError(t *testing.T) {
	src := NewMockSource()
	sink := &failOnceSink{firstErr: errors.New("db down on batch flush")}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runAsync(ctx, w)
	for i := 0; i < 3; i++ {
		src.Push(makeEvent(i))
	}

	if !waitFor(t, time.Second, func() bool { return sink.successfulFlushes() == 1 }) {
		t.Fatal("worker did not replay and persist the batch after the failed flush")
	}
	if got := sink.persistedEvents(); got != 3 {
		t.Fatalf("persisted %d events after retry, want 3", got)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("worker returned an error on cancellation: %v", err)
	}
}

func TestWorker_FailedCommitRetriesOnIdleTurn(t *testing.T) {
	src := NewMockSource()
	src.PushCommitError(errors.New("broker unavailable"))
	sink := &MockSink{}
	w := newTestWorker(src, sink)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runAsync(ctx, w)
	src.Push(makeEvent(4))

	if !waitFor(t, time.Second, func() bool { return src.CommittedCount() == 1 }) {
		t.Fatal("worker did not retry the failed commit on an idle turn")
	}
	if src.CommitAttempts() < 2 {
		t.Fatalf("expected a failed commit and an idle-turn retry, got %d attempts", src.CommitAttempts())
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("worker returned an error on cancellation: %v", err)
	}
}

func TestWorker_CanceledWriteIsFlushedDuringShutdown(t *testing.T) {
	src := NewMockSource()
	sink := &cancelDuringFlushSink{started: make(chan struct{})}
	w := newTestWorker(src, sink)
	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)
	src.Push(makeEvent(7))

	select {
	case <-sink.started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("worker did not start the write")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker returned an error on cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not finish shutdown flush")
	}
	if got := sink.persistedEvents(); got != 1 {
		t.Fatalf("shutdown persisted %d events, want 1", got)
	}
	if src.CommittedCount() == 0 {
		t.Fatal("shutdown did not commit offsets after persisting the in-flight event")
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return condition()
}

type failOnceSink struct {
	mu        sync.Mutex
	firstErr  error
	attempt   int
	persisted int
}

func (s *failOnceSink) Flush(_ context.Context, events *[]event.CovidEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempt++
	if s.attempt == 1 {
		return s.firstErr
	}
	s.persisted += len(*events)
	*events = (*events)[:0]
	return nil
}

func (*failOnceSink) Close(context.Context) error { return nil }

func (s *failOnceSink) attempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempt
}

func (s *failOnceSink) successfulFlushes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attempt == 0 || s.persisted == 0 {
		return 0
	}
	return 1
}

func (s *failOnceSink) persistedEvents() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persisted
}

type cancelDuringFlushSink struct {
	mu        sync.Mutex
	started   chan struct{}
	calls     int
	persisted int
}

func (s *cancelDuringFlushSink) Flush(ctx context.Context, events *[]event.CovidEvent) error {
	s.mu.Lock()
	s.calls++
	call := s.calls
	if call == 1 {
		close(s.started)
	}
	s.mu.Unlock()
	if call == 1 {
		<-ctx.Done()
		return ctx.Err()
	}
	s.mu.Lock()
	s.persisted += len(*events)
	s.mu.Unlock()
	*events = (*events)[:0]
	return nil
}

func (*cancelDuringFlushSink) Close(context.Context) error { return nil }

func (s *cancelDuringFlushSink) persistedEvents() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persisted
}

// TestWorker_CommitCalledAfterSuccessfulFlush verifies offsets are committed after flush.
func TestWorker_CommitCalledAfterSuccessfulFlush(t *testing.T) {
	src := NewMockSource()
	sink := &MockSink{}
	w := newTestWorker(src, sink)

	ctx, cancel := context.WithCancel(context.Background())
	done := runAsync(ctx, w)

	for i := 0; i < 3; i++ {
		src.Push(makeEvent(i))
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if sink.FlushedCount() != 1 {
		t.Fatalf("expected 1 flush, got %d", sink.FlushedCount())
	}
	if src.CommittedCount() < 1 {
		t.Fatalf("expected source.Commit to be called at least once, got %d", src.CommittedCount())
	}
}

// mockCacheReader satisfies event.CacheReader for validator tests
type mockCacheReader struct {
	cities  map[string]int
	regions map[string]int // key: "cityID|regionName"
}

func (m *mockCacheReader) GetCityID(city string) int {
	if id, ok := m.cities[city]; ok {
		return id
	}
	return -1
}

func (m *mockCacheReader) GetRegionID(cityID int, region string) int {
	key := fmt.Sprintf("%d|%s", cityID, region)
	if id, ok := m.regions[key]; ok {
		return id
	}
	return -1
}
