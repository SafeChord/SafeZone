package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAllowsMissingDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	for _, key := range []string{
		"ENVIRONMENT", "LOG_LEVEL", "SERVICE_NAME", "SERVICE_VERSION", "BATCH_SIZE", "FLUSH_INTERVAL",
		"WORKER_COUNT", "PARALLEL_N", "KAFKA_GROUP_ID", "KAFKA_TOPIC", "DB_URL",
	} {
		t.Setenv(key, "")
	}
	unsetEnvForTest(t, "KAFKA_BROKER")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("loading config without .env: %v", err)
	}
	if cfg.KafkaBroker != "localhost:9092" {
		t.Fatalf("default KAFKA_BROKER = %q, want localhost:9092", cfg.KafkaBroker)
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
