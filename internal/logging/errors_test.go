package logging

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorWithContext(t *testing.T) {
	baseErr := errors.New("base error")
	operation := "test operation"
	troubleshooting := "test troubleshooting"

	err := NewErrorWithContext(operation, baseErr, troubleshooting)

	require.NotNil(t, err)
	assert.Equal(t, operation, err.Operation)
	assert.Equal(t, baseErr, err.Err)
	assert.Equal(t, troubleshooting, err.Troubleshooting)
	assert.NotNil(t, err.Context)
}

func TestErrorWithContext_Error(t *testing.T) {
	tests := []struct {
		name            string
		operation       string
		err             error
		troubleshooting string
		wantContains    []string
	}{
		{
			name:            "with troubleshooting",
			operation:       "parse file",
			err:             errors.New("syntax error"),
			troubleshooting: "check syntax",
			wantContains:    []string{"parse file", "syntax error", "Troubleshooting", "check syntax"},
		},
		{
			name:            "without troubleshooting",
			operation:       "read file",
			err:             errors.New("file not found"),
			troubleshooting: "",
			wantContains:    []string{"read file", "file not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewErrorWithContext(tt.operation, tt.err, tt.troubleshooting)
			errMsg := err.Error()

			for _, want := range tt.wantContains {
				assert.Contains(t, errMsg, want)
			}
		})
	}
}

func TestErrorWithContext_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := NewErrorWithContext("operation", baseErr, "troubleshooting")

	unwrapped := err.Unwrap()
	assert.Equal(t, baseErr, unwrapped)
}

func TestErrorWithContext_WithContext(t *testing.T) {
	err := NewErrorWithContext("operation", errors.New("error"), "troubleshooting")

	err.WithContext("key1", "value1")
	err.WithContext("key2", 42)

	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, 42, err.Context["key2"])
}

func TestErrorWithContext_LogError(t *testing.T) {
	config := &Config{
		Level:      LevelInfo,
		EnableFile: false,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	baseErr := errors.New("test error")
	errWithCtx := NewErrorWithContext("test operation", baseErr, "test troubleshooting")
	errWithCtx.WithContext("file", "test.tf")
	errWithCtx.WithContext("line", 42)

	// Should not panic
	errWithCtx.LogError(ctx, logger)
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name            string
		operation       string
		err             error
		troubleshooting string
		wantNil         bool
	}{
		{
			name:            "wrap error",
			operation:       "test op",
			err:             errors.New("test error"),
			troubleshooting: "test troubleshooting",
			wantNil:         false,
		},
		{
			name:            "nil error",
			operation:       "test op",
			err:             nil,
			troubleshooting: "test troubleshooting",
			wantNil:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.operation, tt.err, tt.troubleshooting)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Contains(t, result.Error(), tt.operation)
			}
		})
	}
}

func TestLogAndWrapError(t *testing.T) {
	config := &Config{
		Level:      LevelInfo,
		EnableFile: false,
		EnableJSON: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	tests := []struct {
		name            string
		operation       string
		err             error
		troubleshooting string
		wantNil         bool
	}{
		{
			name:            "log and wrap error",
			operation:       "test op",
			err:             errors.New("test error"),
			troubleshooting: "test troubleshooting",
			wantNil:         false,
		},
		{
			name:            "nil error",
			operation:       "test op",
			err:             nil,
			troubleshooting: "test troubleshooting",
			wantNil:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LogAndWrapError(ctx, logger, tt.operation, tt.err, tt.troubleshooting)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Contains(t, result.Error(), tt.operation)
			}
		})
	}
}

func TestTroubleshootingMessages(t *testing.T) {
	// Verify troubleshooting messages are not empty and contain useful information
	messages := []struct {
		name    string
		message string
	}{
		{"directory access", TroubleshootingDirectoryAccess},
		{"terraform parsing", TroubleshootingTerraformParsing},
		{"terraform plan", TroubleshootingTerraformPlan},
		{"aws credentials", TroubleshootingAWSCredentials},
		{"bedrock access", TroubleshootingBedrockAccess},
		{"wafr access", TroubleshootingWAFRAccess},
		{"rate limit", TroubleshootingRateLimit},
		{"session not found", TroubleshootingSessionNotFound},
		{"disk space", TroubleshootingDiskSpace},
	}

	for _, msg := range messages {
		t.Run(msg.name, func(t *testing.T) {
			assert.NotEmpty(t, msg.message)
			assert.True(t, strings.Contains(msg.message, "Possible causes") || 
				strings.Contains(msg.message, "Solutions"),
				"message should contain troubleshooting guidance")
		})
	}
}

func TestErrorChaining(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := NewErrorWithContext("operation", baseErr, "troubleshooting")

	// Test that errors.Is works
	assert.True(t, errors.Is(wrappedErr, baseErr))

	// Test that errors.As works
	var errWithCtx *ErrorWithContext
	assert.True(t, errors.As(wrappedErr, &errWithCtx))
	assert.Equal(t, "operation", errWithCtx.Operation)
}
