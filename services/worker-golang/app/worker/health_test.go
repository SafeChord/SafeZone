package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandlerRequiresARecentTurnFromEveryWorker(t *testing.T) {
	window := 100 * time.Millisecond
	first := &Worker{}
	second := &Worker{}
	first.markLoopTurn()
	second.lastLoopTurn.Store(time.Now().Add(-2 * window).UnixNano())

	handler := NewHealthHandler(window, first, second)
	if got := healthStatus(handler); got != http.StatusServiceUnavailable {
		t.Fatalf("health status with one stale worker = %d, want %d", got, http.StatusServiceUnavailable)
	}

	second.markLoopTurn()
	if got := healthStatus(handler); got != http.StatusOK {
		t.Fatalf("health status after every worker turns = %d, want %d", got, http.StatusOK)
	}

	neverStarted := &Worker{}
	if got := healthStatus(NewHealthHandler(window, neverStarted)); got != http.StatusServiceUnavailable {
		t.Fatalf("health status before a worker's first turn = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestHealthHandlerOnlyServesHealthGet(t *testing.T) {
	w := &Worker{}
	w.markLoopTurn()
	handler := NewHealthHandler(time.Second, w)

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodGet, path: "/health", want: http.StatusOK},
		{method: http.MethodGet, path: "/other", want: http.StatusNotFound},
		{method: http.MethodPost, path: "/health", want: http.StatusMethodNotAllowed},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != tc.want {
				t.Fatalf("status = %d, want %d", resp.Code, tc.want)
			}
		})
	}
}

func healthStatus(handler http.Handler) int {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp.Code
}
