package worker

import (
	"testing"
	"time"
)

func TestRetryDelayGrowsAndFitsWithinLivenessBudget(t *testing.T) {
	for _, tc := range []struct {
		failures int
		window   time.Duration
		want     time.Duration
	}{
		{failures: 1, window: 30 * time.Second, want: 250 * time.Millisecond},
		{failures: 2, window: 30 * time.Second, want: 500 * time.Millisecond},
		{failures: 3, window: 30 * time.Second, want: time.Second},
		{failures: 5, window: 30 * time.Second, want: time.Second},
		{failures: 10, window: 2 * time.Second, want: 500 * time.Millisecond},
	} {
		if got := retryDelay(tc.failures, tc.window); got != tc.want {
			t.Errorf("retryDelay(%d, %s) = %s, want %s", tc.failures, tc.window, got, tc.want)
		}
	}
}
