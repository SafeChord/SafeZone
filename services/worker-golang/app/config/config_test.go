package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestLoadAllowsMissingDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	for _, key := range []string{
		"ENVIRONMENT", "LOG_LEVEL", "SERVICE_NAME", "SERVICE_VERSION", "BATCH_SIZE", "FLUSH_INTERVAL",
		"LIVENESS_WINDOW", "DEPENDENCY_TIMEOUT", "WORKER_COUNT", "PARALLEL_N", "KAFKA_GROUP_ID", "KAFKA_TOPIC", "DB_URL",
	} {
		t.Setenv(key, "")
	}
	unsetEnvForTest(t, "KAFKA_BROKER")
	unsetEnvForTest(t, "SERVER_IP")
	unsetEnvForTest(t, "SERVER_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("loading config without .env: %v", err)
	}
	if cfg.KafkaBroker != "localhost:9092" {
		t.Fatalf("default KAFKA_BROKER = %q, want localhost:9092", cfg.KafkaBroker)
	}
	if cfg.LivenessWindow != 30*time.Second {
		t.Fatalf("default LIVENESS_WINDOW = %s, want 30s", cfg.LivenessWindow)
	}
	if cfg.DependencyTimeout != 10*time.Second {
		t.Fatalf("default DEPENDENCY_TIMEOUT = %s, want 10s", cfg.DependencyTimeout)
	}
	if cfg.DependencyTimeout >= cfg.LivenessWindow {
		t.Fatalf("default DEPENDENCY_TIMEOUT = %s, must be below LIVENESS_WINDOW %s", cfg.DependencyTimeout, cfg.LivenessWindow)
	}
	if cfg.LivenessWindow <= cfg.FlushInterval {
		t.Fatalf("default LIVENESS_WINDOW = %s, must exceed FLUSH_INTERVAL %s", cfg.LivenessWindow, cfg.FlushInterval)
	}
	if cfg.ServerIP != "0.0.0.0" || cfg.ServerPort != 8080 {
		t.Fatalf("default server address = %s:%d, want 0.0.0.0:8080", cfg.ServerIP, cfg.ServerPort)
	}
}

func TestLoadDependencyTimeoutConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DEPENDENCY_TIMEOUT", "750ms")
	t.Setenv("LIVENESS_WINDOW", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.DependencyTimeout != 750*time.Millisecond {
		t.Fatalf("DEPENDENCY_TIMEOUT = %s, want 750ms", cfg.DependencyTimeout)
	}
}

func TestLoadRejectsDependencyTimeoutAtOrAboveLivenessWindow(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DEPENDENCY_TIMEOUT", "500ms")
	t.Setenv("LIVENESS_WINDOW", "500ms")

	if _, err := Load(); err == nil {
		t.Fatal("expected config to reject a dependency timeout that does not fit within the liveness window")
	}
}

func TestLoadLivenessAndServerConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("LIVENESS_WINDOW", "45s")
	t.Setenv("SERVER_IP", "127.0.0.1")
	t.Setenv("SERVER_PORT", "12345")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.LivenessWindow.String() != "45s" {
		t.Errorf("LIVENESS_WINDOW = %s, want 45s", cfg.LivenessWindow)
	}
	if cfg.ServerIP != "127.0.0.1" || cfg.ServerPort != 12345 {
		t.Errorf("server address = %s:%d, want 127.0.0.1:12345", cfg.ServerIP, cfg.ServerPort)
	}
}

func TestLoadLogLevelAcceptsLevelNamesInEitherCase(t *testing.T) {
	cases := map[string]zapcore.Level{
		"debug": zapcore.DebugLevel,
		"DEBUG": zapcore.DebugLevel,
		"info":  zapcore.InfoLevel,
		"INFO":  zapcore.InfoLevel,
		"Warn":  zapcore.WarnLevel,
		"WARN":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"Error": zapcore.ErrorLevel,
	}
	for value, want := range cases {
		t.Run(value, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("LOG_LEVEL", value)

			var cfg *Config
			stderr := captureStderr(t, func() {
				var err error
				if cfg, err = Load(); err != nil {
					t.Fatalf("loading config: %v", err)
				}
			})
			if cfg.LogLevel != want {
				t.Errorf("LOG_LEVEL %q = %s, want %s", value, cfg.LogLevel, want)
			}
			if strings.Contains(stderr, "LOG_LEVEL") {
				t.Errorf("LOG_LEVEL %q produced a warning: %s", value, stderr)
			}
		})
	}
}

func TestLoadLogLevelFallsBackToInfoWithWarning(t *testing.T) {
	for _, value := range []string{"verbose", "warning", "fatal"} {
		t.Run(value, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("LOG_LEVEL", value)

			var cfg *Config
			stderr := captureStderr(t, func() {
				var err error
				if cfg, err = Load(); err != nil {
					t.Fatalf("loading config: %v", err)
				}
			})
			if cfg.LogLevel != zapcore.InfoLevel {
				t.Errorf("LOG_LEVEL %q = %s, want info", value, cfg.LogLevel)
			}
			if !strings.Contains(stderr, "LOG_LEVEL") || !strings.Contains(stderr, value) {
				t.Errorf("LOG_LEVEL %q produced no warning naming it; stderr: %s", value, stderr)
			}
		})
	}
}

func TestLoadReturnsErrorsOtherThanMissingDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, ".env"), 0o755); err != nil {
		t.Fatalf("creating .env directory: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected loading a non-file .env path to fail")
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	old, wasSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetting %s: %v", key, err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stderr pipe: %v", err)
	}
	defer reader.Close()

	originalStderr := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = originalStderr }()
	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("closing stderr pipe: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	return string(output)
}
