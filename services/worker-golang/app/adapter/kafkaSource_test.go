package adapter_test

import (
	"testing"

	"go.uber.org/zap"

	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/pkg/logger"
)

func testLogger() *logger.ContextLogger {
	l := logger.NewContextLogger("test", "0.0.0", "test")
	l.Logger = zap.NewNop()
	return l
}

// TestNewKafkaSource_UnreachableBrokerReturnsError verifies that NewKafkaSource
// returns an error (and nil source) when it cannot reach Kafka (WG-1).
func TestNewKafkaSource_UnreachableBrokerReturnsError(t *testing.T) {
	src, err := adapter.NewKafkaSource(testLogger(), "127.0.0.1:1", "test-group", "test-topic")
	if err == nil {
		t.Fatal("expected error connecting to unreachable broker, got nil")
	}
	if src != nil {
		t.Fatalf("expected nil source on error, got %v", src)
	}
}
