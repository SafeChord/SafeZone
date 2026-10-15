package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestContextLoggerIncludesWorkerIDAndPreservesFieldNames(t *testing.T) {
	var output bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&output), zapcore.DebugLevel)
	cl := &ContextLogger{
		Logger:      zap.New(core).With(zap.String("service", "worker-golang"), zap.String("service_version", "test")),
		WorkerIDKey: contextKey("worker_id"),
		TraceIDKey:  contextKey("trace_id"),
	}
	ctx := context.WithValue(context.Background(), cl.WorkerIDKey, "worker-7")
	ctx = context.WithValue(ctx, cl.TraceIDKey, "trace-123")

	cl.Info(ctx, "worker started")
	cl.Debug(ctx, "debug message")
	cl.Warn(ctx, "warning message")
	cl.Error(ctx, "error message")

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d log lines, want 4: %s", len(lines), output.String())
	}
	for _, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("decoding JSON log line %q: %v", line, err)
		}
		if fields["worker_id"] != "worker-7" {
			t.Errorf("worker_id = %v, want worker-7", fields["worker_id"])
		}
		if fields["trace_id"] != "trace-123" {
			t.Errorf("trace_id = %v, want trace-123", fields["trace_id"])
		}
		for _, key := range []string{"service", "service_version", "timestamp", "worker_id", "trace_id", "msg", "level"} {
			if _, ok := fields[key]; !ok {
				t.Errorf("log field %q missing from %v", key, fields)
			}
		}
	}
}
