package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/service"
	"safezone.service.worker-golang/app/strategy"
)

func testLogger() *logger.ContextLogger {
	return logger.NewContextLogger("test", "0.0.0", "test")
}

func newTestWorker(src adapter.EventSource, sink strategy.EventSink) *service.Worker {
	return &service.Worker{
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

func makeEvent(i int) *schema.CovidEvent {
	e := &schema.CovidEvent{
		EventType: "test",
		TraceID:   fmt.Sprintf("trace-%d", i),
	}
	e.Payload.Date = "2024-01-01"
	e.Payload.City = "Taipei"
	e.Payload.Region = "Zhongzheng"
	e.Payload.Cases = i
	return e
}

func runAsync(ctx context.Context, w *service.Worker) <-chan error {
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	return done
}

// TestWorker_BatchFlush: exactly BatchSize events → one flush with all events
func TestWorker_BatchFlush(t *testing.T) {
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
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
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
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
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
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
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
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
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
	w := newTestWorker(src, sink)

	// wire up a validator with a mock cache that knows one valid city/region
	mockCache := &mockCacheReader{
		cities:  map[string]int{"Taipei": 1},
		regions: map[string]int{"1|Zhongzheng": 10},
	}
	w.Validator = schema.NewCovidValidator(testLogger(), mockCache)

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

// TestWorker_SourceError: non-context error from source → worker returns error
func TestWorker_SourceError(t *testing.T) {
	src := adapter.NewMockSource()
	sink := &strategy.MockSink{}
	w := newTestWorker(src, sink)

	done := runAsync(context.Background(), w)
	src.PushError(errors.New("kafka connection lost"))
	err := <-done

	if err == nil {
		t.Fatal("expected error from worker, got nil")
	}
}

// mockCacheReader satisfies schema.CacheReader for validator tests
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
