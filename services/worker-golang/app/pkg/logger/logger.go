package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

type contextKey string

type ContextLogger struct {
	Logger      *zap.Logger
	WorkerIDKey contextKey
	TraceIDKey  contextKey
}

// Init initializes the logger with the given service name, version, and environment.
func NewContextLogger(serviceName, serviceVersion, env string) *ContextLogger {
	var err error
	var baseLogger *zap.Logger

	// Set the logger configuration based on the environment
	// For production and staging, use production logger，it will output structured logs with json format.
	// For development, use development logger, it will output human-readable logs.
	if env == "staging" || env == "prod" {
		baseLogger, err = zap.NewProduction()
	} else {
		baseLogger, err = zap.NewDevelopment()
	}

	// This will skip the decorator function wrapping the zap logger, allowing the caller to see the correct log location.
	baseLogger = baseLogger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	baseLogger = baseLogger.With(
		zap.String("service", serviceName),
		zap.String("service_version", serviceVersion),
	)

	return &ContextLogger{
		Logger:      baseLogger,
		TraceIDKey:  contextKey("trace_id"),
		WorkerIDKey: contextKey("worker_id"),
	}
}

// following methods are decorators for zap.Logger methods
// the additional fields like timestamp, worker_id, and trace_id are added to the log entries
// and make zap logger support context-based logging

func (cl *ContextLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	worker_id := cl.getContextField(ctx, cl.WorkerIDKey)
	trace_id := cl.getContextField(ctx, cl.TraceIDKey)

	baseFields := []zap.Field{
		zap.String("timestamp", timestamp),
		zap.String("worker_id", worker_id),
		zap.String("trace_id", trace_id),
	}
	allFields := append(baseFields, fields...)
	cl.Logger.Info(msg, allFields...)
}

func (cl *ContextLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	worker_id := cl.getContextField(ctx, cl.WorkerIDKey)
	trace_id := cl.getContextField(ctx, cl.TraceIDKey)

	baseFields := []zap.Field{
		zap.String("timestamp", timestamp),
		zap.String("worker_id", worker_id),
		zap.String("trace_id", trace_id),
	}
	allFields := append(baseFields, fields...)
	cl.Logger.Debug(msg, allFields...)
}

func (cl *ContextLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	worker_id := cl.getContextField(ctx, cl.WorkerIDKey)
	trace_id := cl.getContextField(ctx, cl.TraceIDKey)

	baseFields := []zap.Field{
		zap.String("timestamp", timestamp),
		zap.String("worker_id", worker_id),
		zap.String("trace_id", trace_id),
	}
	allFields := append(baseFields, fields...)
	cl.Logger.Warn(msg, allFields...)
}

func (cl *ContextLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	worker_id := cl.getContextField(ctx, cl.WorkerIDKey)
	trace_id := cl.getContextField(ctx, cl.TraceIDKey)

	baseFields := []zap.Field{
		zap.String("timestamp", timestamp),
		zap.String("worker_id", worker_id),
		zap.String("trace_id", trace_id),
	}
	allFields := append(baseFields, fields...)
	cl.Logger.Error(msg, allFields...)
}

func (cl *ContextLogger) getContextField(ctx context.Context, field any) string {
	if value := ctx.Value(field); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return "-"
}
