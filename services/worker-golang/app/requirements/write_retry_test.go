package requirements_test

import (
	"errors"
	"testing"
	"time"
)

// TestWorkerRetriesFailedWritesAcrossPartitions covers the fetch rewind when one failed
// write contains records from more than one partition. It is an engineering test, not a
// separate blueprint requirement.
func TestWorkerRetriesFailedWritesAcrossPartitions(t *testing.T) {
	b := newBroker(t, 2)
	b.produceEvent(t, 0, newEvent(101, "region-0"))
	b.produceEvent(t, 1, newEvent(102, "region-1"))

	db := newStore()
	db.fail = errors.New("database unavailable")
	r := startWorker(t, b, workerOpts{sink: db, batch: 10, timeout: dependencyTimeout})

	eventually(t, 20*time.Second, "both partitions to be attempted while writes fail", func() bool {
		return len(db.triedEvents()) == 2
	})
	if got := b.committedAt(t, 0); got != 0 {
		t.Fatalf("partition 0 committed offset moved to %d while writes failed", got)
	}
	if got := b.committedAt(t, 1); got != 0 {
		t.Fatalf("partition 1 committed offset moved to %d while writes failed", got)
	}

	db.setFail(nil)
	eventually(t, 30*time.Second, "both partitions to be persisted and committed after recovery", func() bool {
		return db.has(101) && db.has(102) && b.committedAt(t, 0) == 1 && b.committedAt(t, 1) == 1
	})
	if r.exited() {
		t.Fatal("worker exited after the database recovered")
	}
}
