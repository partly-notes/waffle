package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug   LogLevel = "DEBUG"
	LevelInfo    LogLevel = "INFO"
	LevelWarning LogLevel = "WARNING"
	LevelError   LogLevel = "ERROR"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	logger *slog.Logger
	file   *os.File
}

// Config holds logger configuration
type Config struct {
	Level      LogLevel
	LogDir     string
	MaxSizeMB  int64
	EnableFile bool
	EnableJSON bool
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	// Use current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// Fallback to current directory if getwd fails
		cwd = "."
	}
	return &Config{
		Level:      LevelInfo,
		LogDir:     filepath.Join(cwd, ".waffle", "logs"),
		MaxSizeMB:  100,
		EnableFile: true,
		EnableJSON: false,
	}
}

// NewLogger creates a new logger instance
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var handlers []slog.Handler
	var logFile *os.File

	// Create log directory if file logging is enabled
	if config.EnableFile {
		if config.LogDir == "" {
			return nil, fmt.Errorf("log directory is required when file logging is enabled")
		}
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create log file with timestamp
		logFileName := fmt.Sprintf("waffle-%s.log", time.Now().Format("2006-01-02"))
		logFilePath := filepath.Join(config.LogDir, logFileName)

		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Create file handler
		fileHandler := createHandler(logFile, config)
		handlers = append(handlers, fileHandler)
	}

	// Create console handler (always enabled)
	consoleHandler := createHandler(os.Stderr, config)
	handlers = append(handlers, consoleHandler)

	// Use multi-handler if we have both file and console
	var handler slog.Handler
	if len(handlers) > 1 {
		handler = NewMultiHandler(handlers...)
	} else {
		handler = handlers[0]
	}

	logger := slog.New(handler)

	return &Logger{
		logger: logger,
		file:   logFile,
	}, nil
}

// createHandler creates a slog handler based on configuration
func createHandler(w io.Writer, config *Config) slog.Handler {
	level := mapLogLevel(config.Level)

	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: level == slog.LevelDebug,
	}

	if config.EnableJSON {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

// mapLogLevel converts LogLevel to slog.Level
func mapLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarning:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Close closes the logger and any open file handles
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// DebugContext logs a debug message with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// InfoContext logs an info message with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// WarnContext logs a warning message with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// ErrorContext logs an error message with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// With returns a new logger with additional attributes
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
		file:   l.file,
	}
}

// WithGroup returns a new logger with a group name
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		logger: l.logger.WithGroup(name),
		file:   l.file,
	}
}

// MultiHandler handles logging to multiple destinations
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a new multi-handler
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

// Enabled reports whether the handler handles records at the given level
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle handles the Record
func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new Handler whose attributes consist of both the receiver's attributes and the arguments
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

// WithGroup returns a new Handler with the given group appended to the receiver's existing groups
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not already initialized
		_ = InitGlobalLogger(DefaultConfig())
	}
	return globalLogger
}

// CloseGlobalLogger closes the global logger
func CloseGlobalLogger() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}
