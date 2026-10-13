package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/service"
)

const (
	rebalanceTopic = "covid.case.data.rebalance"
	rebalanceGroup = "safezone-rebalance-group"
)

type steadyStore struct {
	mu      sync.Mutex
	cases   map[string]int
	seen    map[int]bool
	flushes int
}

func newSteadyStore() *steadyStore {
	return &steadyStore{cases: map[string]int{}, seen: map[int]bool{}}
}

func (s *steadyStore) Flush(ctx context.Context, buffer *[]schema.CovidEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range *buffer {
		s.cases[e.Payload.Date+"/"+e.Payload.City+"/"+e.Payload.Region] = e.Payload.Cases
		s.seen[e.Payload.Cases] = true
	}
	s.flushes++
	*buffer = (*buffer)[:0]
	return nil
}

func (s *steadyStore) Close(context.Context) error { return nil }

func (s *steadyStore) seenCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

func makeRebalanceEvent(seq int, region string) schema.CovidEvent {
	var e schema.CovidEvent
	e.EventType = "covid.case.reported"
	e.TraceID = fmt.Sprintf("trace-%d", seq)
	e.Version = "1.0"
	e.Payload.Date = "2024-01-01"
	e.Payload.City = "Taipei"
	e.Payload.Region = region
	e.Payload.Cases = seq
	return e
}

// TestRebalanceUnderSteadyTraffic verifies that when a second worker joins under
// light, steady traffic, neither worker hangs and all events are persisted (Finding 1).
func TestRebalanceUnderSteadyTraffic(t *testing.T) {
	c, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(2, rebalanceTopic))
	if err != nil {
		t.Fatalf("start kfake: %v", err)
	}
	defer c.Close()

	addr := c.ListenAddrs()[0]
	cl, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.RecordPartitioner(kgo.ManualPartitioner()))
	if err != nil {
		t.Fatalf("broker client: %v", err)
	}
	defer cl.Close()
	adm := kadm.NewClient(cl)

	store := newSteadyStore()
	log := testLogger()
	log.Logger = zap.NewNop()

	startWorker := func(id int) (context.CancelFunc, <-chan struct{}) {
		src, err := adapter.NewKafkaSource(log, addr, rebalanceGroup, rebalanceTopic)
		if err != nil {
			t.Fatalf("start worker %d: %v", id, err)
		}
		w := &service.Worker{
			Source: src,
			Sink:   store,
			Config: &config.Config{
				BatchSize:     5,
				FlushInterval: 500 * time.Millisecond,
			},
			Logger: log,
			ID:     id,
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = w.Run(ctx)
			w.Close(context.Background())
		}()
		return cancel, done
	}

	cancel1, done1 := startWorker(1)
	defer func() {
		cancel1()
		select {
		case <-done1:
		case <-time.After(5 * time.Second):
			t.Log("worker 1 did not exit promptly")
		}
	}()

	var (
		cancel2 context.CancelFunc
		done2   <-chan struct{}
	)
	defer func() {
		if cancel2 != nil {
			cancel2()
			select {
			case <-done2:
			case <-time.After(5 * time.Second):
				t.Log("worker 2 did not exit promptly")
			}
		}
	}()

	const totalEvents = 89
	stopProduce := make(chan struct{})
	producedCount := 0
	var produceMu sync.Mutex

	// Produce 1 event every 100ms alternating partitions
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < totalEvents; i++ {
			select {
			case <-stopProduce:
				return
			case <-ticker.C:
				p := int32(i % 2)
				ev := makeRebalanceEvent(i, fmt.Sprintf("region-%d", p))
				val, _ := json.Marshal(ev)
				ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
				_ = cl.ProduceSync(ctx, &kgo.Record{Topic: rebalanceTopic, Partition: p, Value: val}).FirstErr()
				c()
				produceMu.Lock()
				producedCount++
				produceMu.Unlock()

				// Start worker 2 after 3 seconds (30 events)
				if i == 29 {
					cancel2, done2 = startWorker(2)
				}
			}
		}
	}()

	// Wait for producer to finish producing all events
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		produceMu.Lock()
		count := producedCount
		produceMu.Unlock()
		if count >= totalEvents {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	close(stopProduce)

	// Wait for all events to be persisted
	persisted := false
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if store.seenCount() == totalEvents {
			persisted = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !persisted {
		t.Fatalf("expected %d events persisted, got %d (worker likely hung during rebalance)", totalEvents, store.seenCount())
	}

	// Verify committed offsets reached log end (15 per partition)
	offsets, err := adm.FetchOffsets(context.Background(), rebalanceGroup)
	if err != nil {
		t.Fatalf("fetch offsets: %v", err)
	}
	offsets.Each(func(o kadm.OffsetResponse) {
		if o.Topic == rebalanceTopic {
			var expected int64
			if o.Partition == 0 {
				expected = int64((totalEvents + 1) / 2)
			} else {
				expected = int64(totalEvents / 2)
			}
			if o.At != expected {
				t.Errorf("partition %d: committed %d, want %d", o.Partition, o.At, expected)
			}
		}
	})
}
