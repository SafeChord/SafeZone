package requirements_test

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"safezone.service.worker-golang/app/worker"
)

// healthProbe asks a worker's health endpoint over HTTP, as a Kubernetes probe does.
type healthProbe struct {
	url string
	cl  *http.Client
}

func newHealthProbe(t *testing.T, window time.Duration, r *running) *healthProbe {
	t.Helper()
	srv := httptest.NewServer(worker.NewHealthHandler(window, r.w))
	t.Cleanup(srv.Close)
	return &healthProbe{url: srv.URL + "/health", cl: &http.Client{Timeout: time.Second}}
}

// status returns the response status, or 0 when no response came back in time.
func (p *healthProbe) status() int {
	resp, err := p.cl.Get(p.url)
	if err != nil {
		return 0
	}
	resp.Body.Close()
	return resp.StatusCode
}

// staysHealthy fails the test unless every request over the given span answers 200.
func (p *healthProbe) staysHealthy(t *testing.T, span time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(span); time.Now().Before(deadline); {
		if got := p.status(); got != http.StatusOK {
			t.Fatalf("/health answered %d, want 200", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func success(status int) bool { return status >= 200 && status < 300 }

// WK-R9: Health reflects the consume loop.
// Scenario: idle topic.
func TestWK_R9_IdleTopicIsHealthy(t *testing.T) {
	const window = 300 * time.Millisecond
	b := newBroker(t, 1)
	r := startWorker(t, b, workerOpts{id: 0, sink: newStore(), batch: 10})
	p := newHealthProbe(t, window, r)

	eventually(t, 20*time.Second, "/health to answer 200 once the worker runs", func() bool {
		return p.status() == http.StatusOK
	})
	// No event ever arrives. Several liveness windows pass, and the worker stays healthy.
	p.staysHealthy(t, 7*window)
}

// WK-R9: Health reflects the consume loop.
// Scenario: loop stalled.
func TestWK_R9_StalledLoopIsUnhealthy(t *testing.T) {
	const window = 500 * time.Millisecond
	b := newBroker(t, 1)
	stuck := newStore()
	stuck.hang = make(chan struct{})
	r := startWorker(t, b, workerOpts{id: 0, sink: stuck, batch: 10})
	t.Cleanup(func() { close(stuck.hang) }) // runs before the worker is stopped
	p := newHealthProbe(t, window, r)

	eventually(t, 20*time.Second, "/health to answer 200 once the worker runs", func() bool {
		return p.status() == http.StatusOK
	})

	// The database takes the write and never answers, so the loop stops turning.
	b.produceEvent(t, 0, newEvent(1, "region-0"))
	eventually(t, 20*time.Second, "the worker to start the write", func() bool { return stuck.attemptCount() > 0 })
	started := time.Now()

	// A write that has lasted less than the window is a turn in progress, not a stall.
	if got := p.status(); got != http.StatusOK {
		t.Fatalf("/health answered %d %s into the write, within the %s window; want 200",
			got, time.Since(started).Round(time.Millisecond), window)
	}

	// Once the loop has not turned for longer than the window, the worker is unhealthy.
	eventually(t, window+3*time.Second, "/health to stop answering success once the loop has stalled", func() bool {
		return !success(p.status())
	})
}

// WK-R9: Health reflects the consume loop.
// Scenario: database unreachable.
func TestWK_R9_UnreachableDatabaseIsHealthy(t *testing.T) {
	const window = 300 * time.Millisecond

	// A database that refuses the connection, and one that accepts it and never answers.
	refused := func(t *testing.T) string {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		addr := l.Addr().String()
		l.Close()
		return addr
	}
	silent := func(t *testing.T) string {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		t.Cleanup(func() { l.Close() })
		go func() {
			var held []net.Conn
			defer func() {
				for _, c := range held {
					c.Close()
				}
			}()
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				held = append(held, c)
			}
		}()
		return l.Addr().String()
	}

	for _, tc := range []struct {
		name string
		addr func(*testing.T) string
	}{
		{"connection refused", refused},
		{"connection never answered", silent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sqlx.Open("pgx", "postgres://postgres:password@"+tc.addr(t)+"/safezone")
			if err != nil {
				t.Fatalf("open database handle: %v", err)
			}
			down := newStore()
			down.fail = errors.New("dial tcp: connection refused")

			b := newBroker(t, 1)
			r := startWorker(t, b, workerOpts{id: 0, sink: down, batch: 10, db: db})
			p := newHealthProbe(t, window, r)

			eventually(t, 20*time.Second, "/health to answer 200 once the worker runs", func() bool {
				return p.status() == http.StatusOK
			})
			// The loop keeps turning on an idle topic; the database it cannot reach does not count.
			p.staysHealthy(t, 7*window)
		})
	}
}
