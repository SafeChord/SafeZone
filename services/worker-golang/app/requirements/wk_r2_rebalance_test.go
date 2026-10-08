package requirements_test

import (
	"testing"
	"time"
)

// WK-R2: The latest event for a key wins.
// Scenario: redelivery, when a partition changes hands.
//
// A worker that is holding events loses one of its partitions. The new owner reads those
// events again and then persists a newer event for the same key. Whatever the first worker
// still holds for that partition is now older than what is stored, and must not be written
// over it.
func TestWK_R2_HeldEventsDoNotOverwriteNewerOnesAfterRebalance(t *testing.T) {
	const (
		oldP0, oldP1 = 1, 2
		newP0, newP1 = 101, 102
	)
	b := newBroker(t, 2)
	db := newStore()

	// One event per partition. The first worker is slow to persist: it holds what it reads.
	b.produceEvent(t, 0, event(oldP0, "region-0"))
	b.produceEvent(t, 1, event(oldP1, "region-1"))
	startWorker(t, b, workerOpts{id: 0, sink: db, batch: 1000, flushInterval: 8 * time.Second})
	eventually(t, 20*time.Second, "the first worker to join the group", func() bool { return b.members() == 1 })
	time.Sleep(time.Second) // let it read both events

	// A second worker joins and persists promptly; the group gives it one of the partitions.
	startWorker(t, b, workerOpts{id: 1, sink: db, batch: 1000, flushInterval: 300 * time.Millisecond})
	eventually(t, 30*time.Second, "one of the first events to be persisted", func() bool {
		return db.has(oldP0) || db.has(oldP1)
	})

	// A newer event arrives for each key.
	b.produceEvent(t, 0, event(newP0, "region-0"))
	b.produceEvent(t, 1, event(newP1, "region-1"))

	// Everything read, persisted and committed: both workers have written all they held.
	eventually(t, 60*time.Second, "both newer events to be persisted and both partitions committed to log-end", func() bool {
		return db.has(newP0) && db.has(newP1) && b.committedAt(t, 0) == 2 && b.committedAt(t, 1) == 2
	})
	time.Sleep(500 * time.Millisecond)

	if got := db.stored("region-0"); got != newP0 {
		t.Errorf("region-0 holds %d, want the latest event's %d: an older held event was written over it", got, newP0)
	}
	if got := db.stored("region-1"); got != newP1 {
		t.Errorf("region-1 holds %d, want the latest event's %d: an older held event was written over it", got, newP1)
	}
}
