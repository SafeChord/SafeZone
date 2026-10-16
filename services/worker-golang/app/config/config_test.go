package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAllowsMissingDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	for _, key := range []string{
		"ENVIRONMENT", "LOG_LEVEL", "SERVICE_NAME", "SERVICE_VERSION", "BATCH_SIZE", "FLUSH_INTERVAL",
		"LIVENESS_WINDOW", "WORKER_COUNT", "PARALLEL_N", "KAFKA_GROUP_ID", "KAFKA_TOPIC", "DB_URL",
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
	if cfg.LivenessWindow <= cfg.FlushInterval {
		t.Fatalf("default LIVENESS_WINDOW = %s, must exceed FLUSH_INTERVAL %s", cfg.LivenessWindow, cfg.FlushInterval)
	}
	if cfg.ServerIP != "0.0.0.0" || cfg.ServerPort != 8080 {
		t.Fatalf("default server address = %s:%d, want 0.0.0.0:8080", cfg.ServerIP, cfg.ServerPort)
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
