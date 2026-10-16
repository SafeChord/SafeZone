package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

func TestNewContextLoggerCallerPointsToCallingFileAndLine(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating output pipe: %v", err)
	}
	defer reader.Close()

	originalStderr := os.Stderr
	os.Stderr = writer
	cl := NewContextLogger("worker-golang", "test", "prod", zapcore.InfoLevel)
	os.Stderr = originalStderr

	infoFile, infoLine := logInfoFromCallSite(cl)
	errorFile, errorLine := logErrorFromCallSite(cl)
	_ = cl.Logger.Sync()
	if err := writer.Close(); err != nil {
		t.Fatalf("closing output pipe: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading captured logger output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2: %s", len(lines), output)
	}

	for i, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("decoding JSON log line %q: %v", line, err)
		}
		file, lineNumber := infoFile, infoLine
		if i == 1 {
			file, lineNumber = errorFile, errorLine
		}
		wantCaller := file + ":" + strconv.Itoa(lineNumber)
		caller, ok := fields["caller"].(string)
		if !ok || !strings.HasSuffix(caller, wantCaller) {
			t.Errorf("caller = %v, want suffix %q", fields["caller"], wantCaller)
		}
		if i == 1 {
			stacktrace, _ := fields["stacktrace"].(string)
			if !strings.Contains(stacktrace, wantCaller) {
				t.Errorf("error stacktrace does not include call site %q: %s", wantCaller, stacktrace)
			}
		}
	}
}

func TestNewContextLoggerLogsAtLevelAndAbove(t *testing.T) {
	messages := []struct {
		level zapcore.Level
		msg   string
	}{
		{zapcore.DebugLevel, "level check debug"},
		{zapcore.InfoLevel, "level check info"},
		{zapcore.WarnLevel, "level check warn"},
		{zapcore.ErrorLevel, "level check error"},
	}
	// preview runs the development encoder, which logged at debug whatever level was configured.
	for _, env := range []string{"prod", "preview"} {
		for _, level := range []zapcore.Level{zapcore.DebugLevel, zapcore.InfoLevel, zapcore.WarnLevel, zapcore.ErrorLevel} {
			t.Run(env+"/"+level.String(), func(t *testing.T) {
				output := captureStderr(t, func() {
					cl := NewContextLogger("worker-golang", "test", env, level)
					ctx := context.Background()
					cl.Debug(ctx, messages[0].msg)
					cl.Info(ctx, messages[1].msg)
					cl.Warn(ctx, messages[2].msg)
					cl.Error(ctx, messages[3].msg)
					_ = cl.Logger.Sync()
				})
				for _, m := range messages {
					logged := strings.Contains(output, m.msg)
					if want := m.level >= level; logged != want {
						t.Errorf("%s message logged = %v, want %v; output:\n%s", m.level, logged, want, output)
					}
				}
			})
		}
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating output pipe: %v", err)
	}
	defer reader.Close()

	originalStderr := os.Stderr
	os.Stderr = writer
	defer func() { os.Stderr = originalStderr }()
	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("closing output pipe: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading captured output: %v", err)
	}
	return string(output)
}

func logInfoFromCallSite(cl *ContextLogger) (string, int) {
	_, file, line, _ := runtime.Caller(0)
	cl.Info(context.Background(), "caller check info")
	return filepath.Base(file), line + 1
}

func logErrorFromCallSite(cl *ContextLogger) (string, int) {
	_, file, line, _ := runtime.Caller(0)
	cl.Error(context.Background(), "caller check error")
	return filepath.Base(file), line + 1
}
