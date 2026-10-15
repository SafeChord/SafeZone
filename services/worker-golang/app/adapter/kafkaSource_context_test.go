package adapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"safezone.service.worker-golang/app/adapter"
	"safezone.service.worker-golang/app/pkg/logger"
)

func TestNewKafkaSourceWithContextLogsWorkerIDOnConnectionFailure(t *testing.T) {
	var output bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&output), zapcore.ErrorLevel)
	log := &logger.ContextLogger{
		Logger:      zap.New(core),
		WorkerIDKey: "worker_id",
		TraceIDKey:  "trace_id",
	}
	ctx := context.WithValue(context.Background(), log.WorkerIDKey, "worker-9")

	source, err := adapter.NewKafkaSourceWithContext(ctx, log, "127.0.0.1:1", "test-group", "test-topic")
	if err == nil {
		t.Fatal("expected unreachable Kafka broker to fail")
	}
	if source != nil {
		t.Fatalf("expected nil source on connection failure, got %v", source)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d Kafka error logs, want 1: %s", len(lines), output.String())
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &fields); err != nil {
		t.Fatalf("decoding Kafka error log %q: %v", lines[0], err)
	}
	if fields["msg"] != "Failed to connect to Kafka" {
		t.Fatalf("log message = %v, want failed Kafka connection", fields["msg"])
	}
	if fields["worker_id"] != "worker-9" {
		t.Fatalf("worker_id = %v, want worker-9", fields["worker_id"])
	}
}
