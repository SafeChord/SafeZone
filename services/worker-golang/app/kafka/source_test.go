package kafka_test

import (
	"testing"

	"go.uber.org/zap"

	"safezone.service.worker-golang/app/kafka"
	"safezone.service.worker-golang/app/logger"
)

func testLogger() *logger.ContextLogger {
	l := logger.NewContextLogger("test", "0.0.0", "test")
	l.Logger = zap.NewNop()
	return l
}

// TestNewSource_UnreachableBrokerReturnsError verifies that NewSource
// returns an error (and nil source) when it cannot reach Kafka (WG-1).
func TestNewSource_UnreachableBrokerReturnsError(t *testing.T) {
	src, err := kafka.NewSource(testLogger(), "127.0.0.1:1", "test-group", "test-topic")
	if err == nil {
		t.Fatal("expected error connecting to unreachable broker, got nil")
	}
	if src != nil {
		t.Fatalf("expected nil source on error, got %v", src)
	}
}
