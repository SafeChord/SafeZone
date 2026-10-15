package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/service"
)

// TestSignalNotify_SIGTERMCancelsContext verifies that SIGTERM is registered
// and cancels the context (WK-R6 graceful shutdown trigger in main.go).
func TestSignalNotify_SIGTERMCancelsContext(t *testing.T) {
	ctx, stop := setupSignalContext()
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

func TestMainExitsAndLogsAreaCacheStartupFailure(t *testing.T) {
	const childFlag = "SAFEZONE_TEST_CACHE_STARTUP_FAILURE"
	const cause = "failed to load area tables: city query: connection reset"
	if os.Getenv(childFlag) == "1" {
		newWorker = func(int, *config.Config, *logger.ContextLogger) (*service.Worker, error) {
			return nil, errors.New(cause)
		}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitsAndLogsAreaCacheStartupFailure$")
	cmd.Env = append(os.Environ(),
		childFlag+"=1",
		"ENVIRONMENT=TEST",
		"SERVICE_NAME=worker-golang",
		"SERVICE_VERSION=test",
		"WORKER_COUNT=1",
		"PARALLEL_N=1",
	)
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected worker process to exit non-zero, error=%v, output=%s", err, output)
	}
	if exitErr.ExitCode() == 0 {
		t.Fatalf("worker exited successfully after startup failure: %s", output)
	}
	if !strings.Contains(string(output), "Failed to initialize worker") || !strings.Contains(string(output), cause) {
		t.Fatalf("startup failure log does not include the cause: %s", output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "Failed to initialize worker") && strings.Count(line, "worker_id") != 1 {
			t.Fatalf("startup failure line should contain worker_id exactly once: %s", line)
		}
	}
}
