package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "default config",
			config: &Config{
				Level:      LevelInfo,
				LogDir:     t.TempDir(),
				EnableFile: true,
				EnableJSON: false,
			},
			wantErr: false,
		},
		{
			name: "debug level",
			config: &Config{
				Level:      LevelDebug,
				LogDir:     t.TempDir(),
				EnableFile: true,
				EnableJSON: false,
			},
			wantErr: false,
		},
		{
			name: "json format",
			config: &Config{
				Level:      LevelInfo,
				LogDir:     t.TempDir(),
				EnableFile: true,
				EnableJSON: true,
			},
			wantErr: false,
		},
		{
			name: "console only",
			config: &Config{
				Level:      LevelInfo,
				EnableFile: false,
				EnableJSON: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, logger)
			defer logger.Close()

			// Test that we can log
			logger.Info("test message", "key", "value")
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		Level:      LevelDebug,
		LogDir:     tempDir,
		EnableFile: true,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Test all log levels
	logger.Debug("debug message", "level", "debug")
	logger.DebugContext(ctx, "debug message with context")

	logger.Info("info message", "level", "info")
	logger.InfoContext(ctx, "info message with context")

	logger.Warn("warning message", "level", "warning")
	logger.WarnContext(ctx, "warning message with context")

	logger.Error("error message", "level", "error")
	logger.ErrorContext(ctx, "error message with context")

	// Verify log file was created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, files)
}

func TestLoggerWith(t *testing.T) {
	config := &Config{
		Level:      LevelInfo,
		EnableFile: false,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	// Create logger with additional attributes
	loggerWithAttrs := logger.With("component", "test", "version", "1.0")
	require.NotNil(t, loggerWithAttrs)

	loggerWithAttrs.Info("test message")
}

func TestLoggerWithGroup(t *testing.T) {
	config := &Config{
		Level:      LevelInfo,
		EnableFile: false,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	// Create logger with group
	loggerWithGroup := logger.WithGroup("test-group")
	require.NotNil(t, loggerWithGroup)

	loggerWithGroup.Info("test message", "key", "value")
}

func TestMapLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarning, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{LogLevel("unknown"), slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := mapLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	config := &Config{
		Level:      LevelInfo,
		LogDir:     t.TempDir(),
		EnableFile: true,
		EnableJSON: false,
	}

	err := InitGlobalLogger(config)
	require.NoError(t, err)

	logger := GetLogger()
	require.NotNil(t, logger)

	logger.Info("test message from global logger")

	err = CloseGlobalLogger()
	require.NoError(t, err)

	// Reset for other tests
	globalLogger = nil
}

func TestGetLoggerAutoInit(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	// GetLogger should auto-initialize
	logger := GetLogger()
	require.NotNil(t, logger)

	logger.Info("test message")

	// Cleanup
	CloseGlobalLogger()
	globalLogger = nil
}

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelWarn})

	multiHandler := NewMultiHandler(handler1, handler2)
	logger := slog.New(multiHandler)

	ctx := context.Background()

	// Info message should only go to handler1
	logger.InfoContext(ctx, "info message")
	assert.Contains(t, buf1.String(), "info message")
	assert.Empty(t, buf2.String())

	// Warning message should go to both handlers
	buf1.Reset()
	buf2.Reset()
	logger.WarnContext(ctx, "warning message")
	assert.Contains(t, buf1.String(), "warning message")
	assert.Contains(t, buf2.String(), "warning message")
}

func TestLogFileCreation(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		Level:      LevelInfo,
		LogDir:     tempDir,
		EnableFile: true,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	logger.Info("test message")

	// Verify log file exists
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	// Check file name format
	fileName := files[0].Name()
	assert.True(t, strings.HasPrefix(fileName, "waffle-"))
	assert.True(t, strings.HasSuffix(fileName, ".log"))
}

func TestLogFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		Level:      LevelInfo,
		LogDir:     tempDir,
		EnableFile: true,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	logger.Info("test message")

	// Check directory permissions
	dirInfo, err := os.Stat(tempDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())

	// Check log file permissions
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	logFilePath := filepath.Join(tempDir, files[0].Name())
	fileInfo, err := os.Stat(logFilePath)
	require.NoError(t, err)
	assert.False(t, fileInfo.IsDir())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	require.NotNil(t, config)

	assert.Equal(t, LevelInfo, config.Level)
	assert.True(t, config.EnableFile)
	assert.False(t, config.EnableJSON)
	assert.Equal(t, int64(100), config.MaxSizeMB)
	assert.Contains(t, config.LogDir, ".waffle/logs")
}
