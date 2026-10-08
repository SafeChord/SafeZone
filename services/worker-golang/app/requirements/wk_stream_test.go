package requirements_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// WK-R3: Progress is recorded only after persistence.
// Scenario: stop before persistence.
func TestWK_R3_EventsNotPersistedAreDeliveredAgain(t *testing.T) {
	const n = 30
	b := newBroker(t, 1)
	for i := 0; i < n; i++ {
		b.produceEvent(t, 0, event(i, "region-0"))
	}

	down := newStore()
	down.fail = errors.New("database unavailable")
	first := startWorker(t, b, workerOpts{id: 0, sink: down, batch: 10})
	eventually(t, 20*time.Second, "the worker to attempt a write", func() bool { return down.attemptCount() > 0 })
	first.stop()

	if at := b.committedAt(t, 0); at != 0 {
		t.Fatalf("committed offset is %d although no event was persisted", at)
	}

	up := newStore()
	startWorker(t, b, workerOpts{id: 1, sink: up, batch: 10})
	eventually(t, 30*time.Second, "every event to be delivered again and persisted", func() bool {
		return up.seenCount() == n
	})
}

// WK-R5: An invalid event never blocks the stream.
// Scenario: invalid event between valid ones.
func TestWK_R5_InvalidEventIsSkippedAndNotDeliveredAgain(t *testing.T) {
	unknownCity := event(99, "region-0")
	unknownCity.Payload.City = "Atlantis"

	cases := []struct {
		name    string
		produce func(t *testing.T, b *broker)
		valid   []int // case counts that must be persisted
		end     int64 // log-end offset
	}{
		{
			name: "event that cannot be parsed",
			produce: func(t *testing.T, b *broker) {
				b.produceEvent(t, 0, event(1, "region-0"))
				b.produce(t, 0, []byte("{not json"))
				b.produceEvent(t, 0, event(2, "region-0"))
			},
			valid: []int{1, 2},
			end:   3,
		},
		{
			name: "event naming an unknown city",
			produce: func(t *testing.T, b *broker) {
				b.produceEvent(t, 0, event(1, "region-0"))
				b.produceEvent(t, 0, unknownCity)
				b.produceEvent(t, 0, event(2, "region-0"))
			},
			valid: []int{1, 2},
			end:   3,
		},
		{
			name: "invalid event is the last one read",
			produce: func(t *testing.T, b *broker) {
				b.produceEvent(t, 0, event(1, "region-0"))
				b.produceEvent(t, 0, unknownCity)
			},
			valid: []int{1},
			end:   2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBroker(t, 1)
			tc.produce(t, b)

			db := newStore()
			startWorker(t, b, workerOpts{sink: db, batch: 10, validate: true})

			eventually(t, 20*time.Second, "the valid events around the invalid one to be persisted", func() bool {
				for _, c := range tc.valid {
					if !db.has(c) {
						return false
					}
				}
				return true
			})
			eventually(t, 20*time.Second, "progress to be committed past the invalid event", func() bool {
				return b.committedAt(t, 0) == tc.end
			})
			if db.has(99) {
				t.Fatalf("the invalid event was persisted")
			}
		})
	}
}

// WK-R5: An invalid event never blocks the stream.
// Scenario: invalid event between valid ones, where the invalid event arrives on its own.
//
// Nothing valid follows it, so nothing else will carry the committed offset past it. Left
// uncommitted it is delivered again after every restart, and the group reports a lag that
// never drains.
func TestWK_R5_InvalidEventArrivingAloneIsNotDeliveredAgain(t *testing.T) {
	unknownCity := event(99, "region-0")
	unknownCity.Payload.City = "Atlantis"
	unknownCityJSON, err := json.Marshal(unknownCity)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	cases := []struct {
		name    string
		invalid []byte
	}{
		{"event that cannot be parsed", []byte("{not json")},
		{"event naming an unknown city", unknownCityJSON},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newBroker(t, 1)
			db := newStore()
			b.produceEvent(t, 0, event(1, "region-0"))
			startWorker(t, b, workerOpts{sink: db, batch: 10, validate: true})
			eventually(t, 20*time.Second, "the valid event to be persisted and committed", func() bool {
				return db.has(1) && b.committedAt(t, 0) == 1
			})

			b.produce(t, 0, tc.invalid)

			eventually(t, 10*time.Second, "progress to be committed past the invalid event", func() bool {
				return b.committedAt(t, 0) == 2
			})
			if db.has(99) {
				t.Fatalf("the invalid event was persisted")
			}
		})
	}
}

// WK-R6: Shutdown loses nothing.
// Scenario: terminate with events in hand.
//
// "In hand" is produced without assuming how the worker batches: the store is slow, so the
// worker is stopped while a write is still in flight, and the store abandons that write the
// way a database abandons a transaction whose context is canceled. Whatever the worker had
// tried to persist by then must be in the store when it exits.
func TestWK_R6_ShutdownPersistsHeldEventsAndLeavesGroup(t *testing.T) {
	const n = 5
	b := newBroker(t, 1)
	for i := 0; i < n; i++ {
		b.produceEvent(t, 0, event(i, "region-0"))
	}

	db := newStore()
	db.delay = time.Second
	w := startWorker(t, b, workerOpts{sink: db, batch: n})
	eventually(t, 20*time.Second, "the worker to start writing the events it read", func() bool {
		return db.attemptCount() > 0
	})
	if got := db.seenCount(); got != 0 {
		t.Fatalf("precondition: a write takes a second, yet %d events are already persisted", got)
	}

	w.stop()

	held := db.triedEvents()
	if len(held) == 0 {
		t.Fatalf("precondition: the worker never tried to persist anything")
	}
	for _, c := range held {
		if !db.has(c) {
			t.Fatalf("event %d was in the worker's hands at shutdown and is not persisted (%d of %d are)",
				c, db.seenCount(), len(held))
		}
	}
	// The default session timeout is 45s; an empty group within 5s means the worker left.
	eventually(t, 5*time.Second, "the group to be empty without waiting for a session timeout", func() bool {
		return b.members() == 0
	})
}
