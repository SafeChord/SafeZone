package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

type ContextLogger struct {
	Logger      *zap.Logger
	WorkerIDKey contextKey
	TraceIDKey  contextKey
}

// NewContextLogger creates a logger for the given service name, version, and environment that logs at level and above.
func NewContextLogger(serviceName, serviceVersion, env string, level zapcore.Level) *ContextLogger {
	// Set the logger configuration based on the environment.
	// Production and staging use structured JSON logs; development uses human-readable logs.
	var zapConfig zap.Config
	if env == "staging" || env == "prod" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	baseLogger, err := zapConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Skip the ContextLogger method and shared helper so the caller sees the correct log location.
	baseLogger = baseLogger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))

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

func (cl *ContextLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	cl.log(ctx, zapcore.InfoLevel, msg, fields...)
}

func (cl *ContextLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	cl.log(ctx, zapcore.DebugLevel, msg, fields...)
}

func (cl *ContextLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	cl.log(ctx, zapcore.WarnLevel, msg, fields...)
}

func (cl *ContextLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	cl.log(ctx, zapcore.ErrorLevel, msg, fields...)
}

func (cl *ContextLogger) log(ctx context.Context, level zapcore.Level, msg string, fields ...zap.Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	baseFields := []zap.Field{
		zap.String("timestamp", timestamp),
		zap.String("worker_id", cl.getContextField(ctx, cl.WorkerIDKey)),
		zap.String("trace_id", cl.getContextField(ctx, cl.TraceIDKey)),
	}
	allFields := append(baseFields, fields...)
	cl.Logger.Log(level, msg, allFields...)
}

func (cl *ContextLogger) getContextField(ctx context.Context, field any) string {
	if value := ctx.Value(field); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return "-"
}
