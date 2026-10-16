package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/event"
	"safezone.service.worker-golang/app/logger"
	"safezone.service.worker-golang/app/worker"
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
		newWorker = func(int, *config.Config, *logger.ContextLogger) (*worker.Worker, error) {
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

func TestHealthServerRunsAlongsideWorkersAndStopsAfterDrain(t *testing.T) {
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve health server port: %v", err)
	}
	port := reservation.Addr().(*net.TCPAddr).Port
	if err := reservation.Close(); err != nil {
		t.Fatalf("release health server port: %v", err)
	}

	source := &healthTestSource{
		committed: make(chan struct{}),
		closed:    make(chan struct{}),
	}
	sink := &healthTestSink{started: make(chan struct{}), release: make(chan struct{})}
	log := logger.NewContextLogger("test", "test", "test", zapcore.DebugLevel)
	log.Logger = zap.NewNop()
	w := &worker.Worker{
		Source: source,
		Sink:   sink,
		Config: &config.Config{BatchSize: 1, FlushInterval: time.Hour},
		Logger: log,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		LivenessWindow: time.Second,
		ServerIP:       "127.0.0.1",
		ServerPort:     port,
		ParallelN:      1,
	}
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- runWorkersWithHealth(ctx, cfg, []*worker.Worker{w})
	}()
	releaseOnce := sync.Once{}
	releaseFlush := func() { releaseOnce.Do(func() { close(sink.release) }) }
	t.Cleanup(func() {
		cancel()
		releaseFlush()
		select {
		case <-finished:
		case <-time.After(2 * time.Second):
			t.Error("worker service did not stop during test cleanup")
		}
	})

	url := "http://" + net.JoinHostPort(cfg.ServerIP, strconv.Itoa(cfg.ServerPort)) + "/health"
	waitForHealthStatus(t, url, http.StatusOK)
	select {
	case <-sink.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not begin its in-flight write")
	}

	cancel()
	if got := getHealthStatus(t, url); got != http.StatusOK {
		t.Fatalf("health endpoint stopped before the in-flight write drained: status %d", got)
	}

	releaseFlush()
	select {
	case <-source.committed:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not commit after the in-flight write drained")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run workers with health server: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("health server or workers did not stop")
	}
	select {
	case <-source.closed:
	default:
		t.Fatal("worker source was not closed before service shutdown")
	}
	if got := getHealthStatus(t, url); got != 0 {
		t.Fatalf("health endpoint remained available after service shutdown: status %d", got)
	}
}

func waitForHealthStatus(t *testing.T, url string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := getHealthStatus(t, url); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("health endpoint at %s did not answer %d", url, want)
}

func getHealthStatus(t *testing.T, url string) int {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

type healthTestSource struct {
	mu        sync.Mutex
	sent      bool
	committed chan struct{}
	closed    chan struct{}
	commit    sync.Once
	close     sync.Once
}

func (s *healthTestSource) Poll(ctx context.Context, _ int) ([]event.CovidEvent, error) {
	s.mu.Lock()
	if !s.sent {
		s.sent = true
		s.mu.Unlock()
		return []event.CovidEvent{{TraceID: "health-test"}}, nil
	}
	s.mu.Unlock()
	<-ctx.Done()
	return nil, ctx.Err()
}

func (s *healthTestSource) Commit(context.Context) error {
	s.commit.Do(func() { close(s.committed) })
	return nil
}

func (*healthTestSource) AllowRebalance() {}

func (*healthTestSource) Retry() {}

func (s *healthTestSource) Close(context.Context) error {
	s.close.Do(func() { close(s.closed) })
	return nil
}

type healthTestSink struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *healthTestSink) Flush(_ context.Context, buffer *[]event.CovidEvent) error {
	s.once.Do(func() { close(s.started) })
	<-s.release
	*buffer = (*buffer)[:0]
	return nil
}

func (*healthTestSink) Close(context.Context) error { return nil }
