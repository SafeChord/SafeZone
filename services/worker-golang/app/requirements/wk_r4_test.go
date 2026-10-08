package requirements_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// WK-R4: Committed progress never moves backwards.
// Scenario: membership changes during consumption.
func TestWK_R4_MembershipChangeNeverRewindsCommittedOffset(t *testing.T) {
	const (
		partitions   = 3
		perPartition = 400
		total        = partitions * perPartition
	)
	b := newBroker(t, partitions)
	for p := int32(0); p < partitions; p++ {
		for i := 0; i < perPartition; i++ {
			b.produceEvent(t, p, event(int(p)*perPartition+i, fmt.Sprintf("region-%d", p)))
		}
	}

	// Watch the group's committed offsets for the whole test.
	var (
		mu       sync.Mutex
		high     = map[int32]int64{}
		rewinds  []string
		watchCtx context.Context
		stopW    context.CancelFunc
	)
	watchCtx, stopW = context.WithCancel(context.Background())
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		for watchCtx.Err() == nil {
			if now, err := b.committed(watchCtx); err == nil {
				mu.Lock()
				for p, at := range now {
					if at < high[p] {
						rewinds = append(rewinds, fmt.Sprintf("partition %d: %d -> %d", p, high[p], at))
					} else {
						high[p] = at
					}
				}
				mu.Unlock()
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()
	defer func() { stopW(); <-watched }()

	// One shared store, slow enough that the first worker is still busy when the group changes.
	db := newStore()
	db.delay = 100 * time.Millisecond
	const batch = 10

	startWorker(t, b, workerOpts{id: 0, sink: db, batch: batch})
	eventually(t, 20*time.Second, "the first worker to persist something", func() bool { return db.flushCount() > 0 })

	// Two workers join under load.
	startWorker(t, b, workerOpts{id: 1, sink: db, batch: batch})
	leaver := startWorker(t, b, workerOpts{id: 2, sink: db, batch: batch})

	// One of them leaves again once the group has had time to rebalance.
	time.Sleep(5 * time.Second)
	leaver.stop()

	eventually(t, 120*time.Second, "every event to be persisted and group lag to reach zero", func() bool {
		if db.seenCount() < total {
			return false
		}
		now, err := b.committed(context.Background())
		if err != nil {
			return false
		}
		for p := int32(0); p < partitions; p++ {
			if now[p] != perPartition {
				return false
			}
		}
		return true
	})

	// Keep watching a little longer: a late commit from a former owner would land now.
	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(rewinds) > 0 {
		shown := rewinds
		if len(shown) > 5 {
			shown = shown[:5]
		}
		t.Fatalf("committed offset moved backwards %d time(s), e.g. %v", len(rewinds), shown)
	}
}
