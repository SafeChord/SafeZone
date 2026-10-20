package requirements_test

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

// The outage tests run the worker with a dependency timeout well inside the liveness
// window, as production does, so that a /health that stays 200 shows the loop still turns.
const (
	outageWindow      = 2 * time.Second
	dependencyTimeout = 200 * time.Millisecond
)

// WK-R11: An unreachable dependency does not end the worker.
// Scenario: Kafka unreachable.
func TestWK_R11_KafkaUnreachableDoesNotEndWorker(t *testing.T) {
	for _, tc := range []struct {
		name   string
		silent bool
	}{
		{"connection closed", false},
		{"request never answered", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newBroker(t, 1)
			db := newStore()
			r := startWorker(t, b, workerOpts{sink: db, batch: 10, timeout: dependencyTimeout})
			p := newHealthProbe(t, outageWindow, r)

			b.produceEvent(t, 0, newEvent(1, "region-0"))
			eventually(t, 20*time.Second, "the first event to be persisted and committed", func() bool {
				return db.has(1) && b.committedAt(t, 0) == 1
			})

			end := b.outage(t, -1, tc.silent)
			// Kafka stays away for several liveness windows; the loop keeps turning throughout.
			p.staysHealthy(t, 3*outageWindow)
			if r.exited() {
				t.Fatalf("the worker exited while Kafka was unreachable")
			}
			end()

			b.produceEvent(t, 0, newEvent(2, "region-0"))
			eventually(t, 30*time.Second, "an event produced after Kafka is back to be persisted and committed", func() bool {
				return db.has(2) && b.committedAt(t, 0) == 2
			})
			if r.exited() {
				t.Fatalf("the worker exited after Kafka came back")
			}
		})
	}
}

// WK-R11: An unreachable dependency does not end the worker.
// Scenario: database unreachable.
func TestWK_R11_DatabaseUnreachableDoesNotEndWorker(t *testing.T) {
	for _, tc := range []struct {
		name string
		down func(s *store) (restore func())
	}{
		{"write refused", func(s *store) func() {
			s.fail = errors.New("dial tcp: connection refused")
			return func() { s.setFail(nil) }
		}},
		{"write never answered", func(s *store) func() {
			s.hang = make(chan struct{})
			var once sync.Once
			return func() { once.Do(func() { close(s.hang) }) }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const n = 30
			b := newBroker(t, 1)
			for i := 0; i < n; i++ {
				b.produceEvent(t, 0, newEvent(i, "region-0"))
			}

			db := newStore()
			restore := tc.down(db)
			r := startWorker(t, b, workerOpts{sink: db, batch: 10, timeout: dependencyTimeout})
			t.Cleanup(restore) // runs before the worker is stopped
			p := newHealthProbe(t, outageWindow, r)

			eventually(t, 20*time.Second, "the worker to attempt a write", func() bool { return db.attemptCount() > 0 })
			// The database stays away for several liveness windows. The loop keeps turning, and
			// nothing is committed, since nothing is persisted.
			for deadline := time.Now().Add(3 * outageWindow); time.Now().Before(deadline); {
				if got := p.status(); got != http.StatusOK {
					t.Fatalf("/health answered %d while the database was unreachable, want 200", got)
				}
				if at := b.committedAt(t, 0); at != 0 {
					t.Fatalf("committed offset moved to %d while no event could be persisted", at)
				}
				time.Sleep(20 * time.Millisecond)
			}
			if r.exited() {
				t.Fatalf("the worker exited while the database was unreachable")
			}
			restore()

			eventually(t, 30*time.Second, "every event to be persisted and committed once the database is back", func() bool {
				return db.seenCount() == n && b.committedAt(t, 0) == n
			})
			if r.exited() {
				t.Fatalf("the worker exited after the database came back")
			}
		})
	}
}

// WK-R11: An unreachable dependency does not end the worker.
// Scenario: offset commit fails.
func TestWK_R11_FailedCommitDoesNotEndWorker(t *testing.T) {
	for _, tc := range []struct {
		name   string
		silent bool
	}{
		{"commit refused", false},
		{"commit never answered", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := newBroker(t, 1)
			db := newStore()
			r := startWorker(t, b, workerOpts{sink: db, batch: 10, timeout: dependencyTimeout})
			p := newHealthProbe(t, outageWindow, r)

			b.produceEvent(t, 0, newEvent(1, "region-0"))
			eventually(t, 20*time.Second, "the first event to be persisted and committed", func() bool {
				return db.has(1) && b.committedAt(t, 0) == 1
			})

			end := b.outage(t, offsetCommit, tc.silent)
			b.produceEvent(t, 0, newEvent(2, "region-0"))
			eventually(t, 20*time.Second, "the second event to be persisted", func() bool { return db.has(2) })
			// Commits keep failing for several liveness windows; the loop keeps turning throughout.
			p.staysHealthy(t, 3*outageWindow)
			if r.exited() {
				t.Fatalf("the worker exited while its offset commits failed")
			}
			end()

			eventually(t, 30*time.Second, "the committed offset to reach the end of the log", func() bool {
				return b.committedAt(t, 0) == 2
			})
			if r.exited() {
				t.Fatalf("the worker exited after Kafka answered commits again")
			}
		})
	}
}
