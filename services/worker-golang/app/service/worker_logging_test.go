package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"safezone.service.worker-golang/app/config"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
	"safezone.service.worker-golang/app/service"
	"safezone.service.worker-golang/app/strategy"
)

type failPollSource struct{ err error }

func (s failPollSource) Poll(context.Context, int) ([]schema.CovidEvent, error) { return nil, s.err }
func (failPollSource) Commit(context.Context) error                             { return nil }
func (failPollSource) AllowRebalance()                                          {}
func (failPollSource) Close(context.Context) error                              { return nil }

func TestWorkerLogsWorkerIDAsString(t *testing.T) {
	var output bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(encoder, zapcore.AddSync(&output), zapcore.DebugLevel)
	log := &logger.ContextLogger{
		Logger:      zap.New(core),
		WorkerIDKey: "worker_id",
		TraceIDKey:  "trace_id",
	}
	sourceErr := errors.New("test source failure")
	w := &service.Worker{
		Source: failPollSource{err: sourceErr},
		Sink:   &strategy.MockSink{},
		Config: &config.Config{BatchSize: 1},
		Logger: log,
		ID:     27,
	}

	if err := service.RunWorkers(context.Background(), []*service.Worker{w}, 1); !errors.Is(err, sourceErr) {
		t.Fatalf("RunWorkers error = %v, want %v", err, sourceErr)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected worker, orchestrator, and close log lines; got %d: %s", len(lines), output.String())
	}
	for _, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("decoding log line %q: %v", line, err)
		}
		if fields["worker_id"] != "27" {
			t.Errorf("worker_id = %v (%T), want string \"27\" in log %q", fields["worker_id"], fields["worker_id"], line)
		}
	}
}
