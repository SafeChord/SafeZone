// Package requirements_test holds the tests that enforce the worker blueprint's
// Requirements (Docs: safechord.safezone.service.worker.md). They drive the worker
// through its real Kafka adapter against an in-process broker (kfake), so they bind
// to what the worker promises and not to how it is built.
package requirements_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/service"
)

const (
	topic = "covid.case.data"
	group = "safezone-worker-group"
)

func quietLogger() *logger.ContextLogger {
	l := logger.NewContextLogger("test", "0.0.0", "test")
	l.Logger = zap.NewNop()
	return l
}

// broker is an in-process Kafka with one topic, plus a client to produce and inspect.
type broker struct {
	addr string
	cl   *kgo.Client
	adm  *kadm.Client
}

func newBroker(t *testing.T, partitions int32) *broker {
	t.Helper()
	c, err := kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(partitions, topic))
	if err != nil {
		t.Fatalf("start kfake: %v", err)
	}
	t.Cleanup(c.Close)
	addr := c.ListenAddrs()[0]
	cl, err := kgo.NewClient(kgo.SeedBrokers(addr), kgo.RecordPartitioner(kgo.ManualPartitioner()))
	if err != nil {
		t.Fatalf("broker client: %v", err)
	}
	t.Cleanup(cl.Close)
	return &broker{addr: addr, cl: cl, adm: kadm.NewClient(cl)}
}

// event builds a valid event whose case count is seq, so a test can tell events apart.
func event(seq int, region string) schema.CovidEvent {
	e := schema.CovidEvent{
		EventType: "covid.case.reported",
		TraceID:   fmt.Sprintf("trace-%d", seq),
		Version:   "1.0",
	}
	e.Payload.Date = "2024-01-01"
	e.Payload.City = "Taipei"
	e.Payload.Region = region
	e.Payload.Cases = seq
	return e
}

func (b *broker) produce(t *testing.T, partition int32, value []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r := &kgo.Record{Topic: topic, Partition: partition, Value: value}
	if err := b.cl.ProduceSync(ctx, r).FirstErr(); err != nil {
		t.Fatalf("produce: %v", err)
	}
}

func (b *broker) produceEvent(t *testing.T, partition int32, e schema.CovidEvent) {
	t.Helper()
	v, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	b.produce(t, partition, v)
}

// committed returns the group's committed offset per partition (absent = nothing committed).
func (b *broker) committed(ctx context.Context) (map[int32]int64, error) {
	resp, err := b.adm.FetchOffsets(ctx, group)
	if err != nil {
		return nil, err
	}
	out := map[int32]int64{}
	resp.Each(func(o kadm.OffsetResponse) {
		if o.Topic == topic && o.At >= 0 {
			out[o.Partition] = o.At
		}
	})
	return out, nil
}

func (b *broker) committedAt(t *testing.T, partition int32) int64 {
	t.Helper()
	now, err := b.committed(context.Background())
	if err != nil {
		t.Fatalf("fetch committed offsets: %v", err)
	}
	return now[partition]
}

// members returns how many members the consumer group has right now.
func (b *broker) members() int {
	described, err := b.adm.DescribeGroups(context.Background(), group)
	if err != nil {
		return -1
	}
	return len(described[group].Members)
}

// store is a sink that remembers what it was asked to persist, keyed like the case table.
// Several workers may share one store, as several pods share one database.
type store struct {
	mu       sync.Mutex
	delay    time.Duration // time one write takes
	fail     error         // when set, every write fails
	cases    map[string]int
	seen     map[int]bool // every case count ever persisted
	tried    map[int]bool // every case count the worker ever tried to persist
	flushes  int
	attempts int
}

func newStore() *store {
	return &store{cases: map[string]int{}, seen: map[int]bool{}, tried: map[int]bool{}}
}

func (s *store) Flush(ctx context.Context, buffer *[]schema.CovidEvent) error {
	s.mu.Lock()
	s.attempts++
	for _, e := range *buffer {
		s.tried[e.Payload.Cases] = true
	}
	s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err // a real database refuses a write on a context that is already done
	}
	if s.delay > 0 {
		time.Sleep(s.delay)
		if err := ctx.Err(); err != nil {
			return err // and abandons a transaction whose context is canceled part-way
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return s.fail
	}
	for _, e := range *buffer {
		s.cases[e.Payload.Date+"/"+e.Payload.City+"/"+e.Payload.Region] = e.Payload.Cases
		s.seen[e.Payload.Cases] = true
	}
	s.flushes++
	*buffer = (*buffer)[:0]
	return nil
}

func (s *store) Close(context.Context) error { return nil }

func (s *store) seenCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

func (s *store) has(cases int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen[cases]
}

// stored returns the case count the store holds for a region of Taipei on the test date.
func (s *store) stored(region string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cases["2024-01-01/Taipei/"+region]
}

// triedEvents returns the case counts of every event the worker tried to persist.
func (s *store) triedEvents() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int, 0, len(s.tried))
	for c := range s.tried {
		out = append(out, c)
	}
	return out
}

func (s *store) flushCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushes
}

func (s *store) attemptCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts
}

// areas stands in for the administrative-area tables: Taipei and its "region-*" regions exist.
type areas struct{}

func (areas) GetCityID(city string) int {
	if city == "Taipei" {
		return 1
	}
	return -1
}

func (areas) GetRegionID(cityID int, region string) int {
	if cityID == 1 && strings.HasPrefix(region, "region-") {
		return 1
	}
	return -1
}

type workerOpts struct {
	id            int
	sink          *store
	batch         int
	flushInterval time.Duration // default 50ms
	validate      bool          // check events against areas, as production does
}

// running is one worker: its own group member, as each pod is.
type running struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func startWorker(t *testing.T, b *broker, o workerOpts) *running {
	t.Helper()
	log := quietLogger()
	// #64: the constructor reports failure as an error, never as a nil source.
	src, err := adapter.NewKafkaSource(log, b.addr, group, topic)
	if err != nil {
		t.Fatalf("worker %d: kafka source: %v", o.id, err)
	}
	if o.flushInterval == 0 {
		o.flushInterval = 50 * time.Millisecond
	}
	w := &service.Worker{
		Source: src,
		Sink:   o.sink,
		Config: &config.Config{BatchSize: o.batch, FlushInterval: o.flushInterval},
		Logger: log,
		ID:     o.id,
	}
	if o.validate {
		w.Validator = schema.NewCovidValidator(log, areas{})
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &running{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		_ = w.Run(ctx)
		w.Close(context.Background())
	}()
	t.Cleanup(r.stop)
	return r
}

// stop ends the worker the way a stopping pod does, and waits for it to exit.
func (r *running) stop() {
	r.cancel()
	select {
	case <-r.done:
	case <-time.After(30 * time.Second):
	}
}

func eventually(t *testing.T, within time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for: %s", within, what)
}
