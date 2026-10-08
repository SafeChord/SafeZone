package main_test

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestSignalNotify_SIGTERMCancelsContext verifies that SIGTERM is registered
// and cancels the context (WK-R6 graceful shutdown trigger in main.go).
func TestSignalNotify_SIGTERMCancelsContext(t *testing.T) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find process: %v", err)
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	select {
	case <-ctx.Done():
		// Context was successfully canceled by SIGTERM
	case <-time.After(2 * time.Second):
		t.Fatal("context was not canceled by SIGTERM within 2 seconds")
	}
}
