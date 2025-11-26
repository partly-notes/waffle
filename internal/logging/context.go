package logging

import (
	"context"
)

type contextKey string

const (
	loggerKey      contextKey = "logger"
	correlationKey contextKey = "correlation_id"
	sessionKey     contextKey = "session_id"
	workloadKey    contextKey = "workload_id"
)

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context, or returns the global logger
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return GetLogger()
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationKey, correlationID)
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationKey).(string); ok {
		return id
	}
	return ""
}

// WithSessionID adds a session ID to the context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionKey, sessionID)
}

// GetSessionID retrieves the session ID from context
func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(sessionKey).(string); ok {
		return id
	}
	return ""
}

// WithWorkloadID adds a workload ID to the context
func WithWorkloadID(ctx context.Context, workloadID string) context.Context {
	return context.WithValue(ctx, workloadKey, workloadID)
}

// GetWorkloadID retrieves the workload ID from context
func GetWorkloadID(ctx context.Context) string {
	if id, ok := ctx.Value(workloadKey).(string); ok {
		return id
	}
	return ""
}

// EnrichContext adds common attributes to the logger based on context values
func EnrichContext(ctx context.Context, logger *Logger) *Logger {
	attrs := []any{}

	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		attrs = append(attrs, "correlation_id", correlationID)
	}

	if sessionID := GetSessionID(ctx); sessionID != "" {
		attrs = append(attrs, "session_id", sessionID)
	}

	if workloadID := GetWorkloadID(ctx); workloadID != "" {
		attrs = append(attrs, "workload_id", workloadID)
	}

	if len(attrs) > 0 {
		return logger.With(attrs...)
	}

	return logger
}
