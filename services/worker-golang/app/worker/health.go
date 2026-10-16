package worker

import (
	"net/http"
	"time"
)

func (w *Worker) markLoopTurn() {
	w.lastLoopTurn.Store(time.Now().UnixNano())
}

// NewHealthHandler reports healthy only while every worker's consume loop has
// started a turn within window. It deliberately does not query worker dependencies.
func NewHealthHandler(window time.Duration, workers ...*Worker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if len(workers) == 0 || window <= 0 {
			http.Error(w, "worker is not healthy", http.StatusServiceUnavailable)
			return
		}

		now := time.Now().UnixNano()
		for _, current := range workers {
			if current == nil {
				http.Error(w, "worker is not healthy", http.StatusServiceUnavailable)
				return
			}
			lastTurn := current.lastLoopTurn.Load()
			if lastTurn == 0 || now-lastTurn > int64(window) {
				http.Error(w, "worker is not healthy", http.StatusServiceUnavailable)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	})
	return mux
}
