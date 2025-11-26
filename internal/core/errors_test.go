package core

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryAccessError(t *testing.T) {
	baseErr := errors.New("permission denied")
	err := &DirectoryAccessError{
		Path: "/test/path",
		Err:  baseErr,
	}

	assert.Contains(t, err.Error(), "/test/path")
	assert.Contains(t, err.Error(), "permission denied")
	assert.Equal(t, baseErr, errors.Unwrap(err))
}

func TestWorkloadNotFoundError(t *testing.T) {
	err := &WorkloadNotFoundError{
		WorkloadID: "test-workload-123",
	}

	assert.Contains(t, err.Error(), "test-workload-123")
	assert.Contains(t, err.Error(), "workload not found")
}

func TestInsufficientPermissionsError(t *testing.T) {
	err := &InsufficientPermissionsError{
		Operation: "CreateWorkload",
	}

	assert.Contains(t, err.Error(), "CreateWorkload")
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{
		Operation: "ListAnswers",
	}

	assert.Contains(t, err.Error(), "ListAnswers")
	assert.Contains(t, err.Error(), "rate limit")
}

func TestTerraformSyntaxError(t *testing.T) {
	err := &TerraformSyntaxError{
		File:    "main.tf",
		Line:    42,
		Message: "unexpected token",
	}

	assert.Contains(t, err.Error(), "main.tf")
	assert.Contains(t, err.Error(), "42")
	assert.Contains(t, err.Error(), "unexpected token")
}

func TestIaCParsingError(t *testing.T) {
	baseErr := errors.New("invalid syntax")
	
	t.Run("with context", func(t *testing.T) {
		err := &IaCParsingError{
			File:    "main.tf",
			Err:     baseErr,
			Context: "resource block",
		}

		assert.Contains(t, err.Error(), "main.tf")
		assert.Contains(t, err.Error(), "resource block")
		assert.Contains(t, err.Error(), "invalid syntax")
		assert.Equal(t, baseErr, errors.Unwrap(err))
	})

	t.Run("without context", func(t *testing.T) {
		err := &IaCParsingError{
			File: "main.tf",
			Err:  baseErr,
		}

		assert.Contains(t, err.Error(), "main.tf")
		assert.Contains(t, err.Error(), "invalid syntax")
		assert.NotContains(t, err.Error(), "()")
	})
}

func TestBedrockAPIError(t *testing.T) {
	t.Run("with error code", func(t *testing.T) {
		err := &BedrockAPIError{
			Operation: "InvokeModel",
			ErrorCode: "ThrottlingException",
			Message:   "Rate exceeded",
		}

		assert.Contains(t, err.Error(), "InvokeModel")
		assert.Contains(t, err.Error(), "ThrottlingException")
		assert.Contains(t, err.Error(), "Rate exceeded")
	})

	t.Run("without error code", func(t *testing.T) {
		baseErr := errors.New("connection timeout")
		err := &BedrockAPIError{
			Operation: "InvokeModel",
			Err:       baseErr,
		}

		assert.Contains(t, err.Error(), "InvokeModel")
		assert.Contains(t, err.Error(), "connection timeout")
		assert.Equal(t, baseErr, errors.Unwrap(err))
	})
}

func TestWAFRAPIError(t *testing.T) {
	t.Run("with error code", func(t *testing.T) {
		err := &WAFRAPIError{
			Operation: "CreateWorkload",
			ErrorCode: "AccessDeniedException",
			Message:   "User not authorized",
		}

		assert.Contains(t, err.Error(), "CreateWorkload")
		assert.Contains(t, err.Error(), "AccessDeniedException")
		assert.Contains(t, err.Error(), "User not authorized")
	})

	t.Run("without error code", func(t *testing.T) {
		baseErr := errors.New("network error")
		err := &WAFRAPIError{
			Operation: "ListAnswers",
			Err:       baseErr,
		}

		assert.Contains(t, err.Error(), "ListAnswers")
		assert.Contains(t, err.Error(), "network error")
		assert.Equal(t, baseErr, errors.Unwrap(err))
	})
}

func TestFileAccessError(t *testing.T) {
	baseErr := errors.New("file not found")
	err := &FileAccessError{
		Path:      "/path/to/file.tf",
		Operation: "read",
		Err:       baseErr,
	}

	assert.Contains(t, err.Error(), "/path/to/file.tf")
	assert.Contains(t, err.Error(), "read")
	assert.Contains(t, err.Error(), "file not found")
	assert.Equal(t, baseErr, errors.Unwrap(err))
}

func TestValidationError(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		err := &ValidationError{
			Field:   "file_count",
			Value:   15000,
			Message: "exceeded maximum",
		}

		assert.Contains(t, err.Error(), "file_count")
		assert.Contains(t, err.Error(), "15000")
		assert.Contains(t, err.Error(), "exceeded maximum")
	})

	t.Run("without value", func(t *testing.T) {
		err := &ValidationError{
			Field:   "workload_id",
			Message: "required field missing",
		}

		assert.Contains(t, err.Error(), "workload_id")
		assert.Contains(t, err.Error(), "required field missing")
		assert.NotContains(t, err.Error(), "value:")
	})
}

func TestErrorChaining(t *testing.T) {
	// Test that errors can be properly chained and unwrapped
	baseErr := errors.New("base error")
	
	iacErr := &IaCParsingError{
		File: "main.tf",
		Err:  baseErr,
	}
	
	wrappedErr := fmt.Errorf("failed to analyze: %w", iacErr)
	
	// Should be able to unwrap to the base error
	assert.True(t, errors.Is(wrappedErr, baseErr))
	
	// Should be able to check for specific error types
	var parsErr *IaCParsingError
	assert.True(t, errors.As(wrappedErr, &parsErr))
	assert.Equal(t, "main.tf", parsErr.File)
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrPillarRequired", ErrPillarRequired},
		{"ErrQuestionIDRequired", ErrQuestionIDRequired},
		{"ErrInvalidWorkloadID", ErrInvalidWorkloadID},
		{"ErrInvalidDirectoryLocation", ErrInvalidDirectoryLocation},
		{"ErrSessionNotFound", ErrSessionNotFound},
		{"ErrWorkloadNotFound", ErrWorkloadNotFound},
		{"ErrEvaluatorNotInitialized", ErrEvaluatorNotInitialized},
		{"ErrNoFilesProvided", ErrNoFilesProvided},
		{"ErrMaxFilesExceeded", ErrMaxFilesExceeded},
		{"ErrInvalidPlanFile", ErrInvalidPlanFile},
		{"ErrBedrockInvocationFailed", ErrBedrockInvocationFailed},
		{"ErrMaxRetriesExceeded", ErrMaxRetriesExceeded},
		{"ErrSessionAlreadyCompleted", ErrSessionAlreadyCompleted},
		{"ErrInvalidSessionStatus", ErrInvalidSessionStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
			
			// Test that sentinel errors can be compared with errors.Is
			wrappedErr := fmt.Errorf("operation failed: %w", tt.err)
			assert.True(t, errors.Is(wrappedErr, tt.err))
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that all custom error types implement the error interface
	var _ error = &DirectoryAccessError{}
	var _ error = &WorkloadNotFoundError{}
	var _ error = &InsufficientPermissionsError{}
	var _ error = &RateLimitError{}
	var _ error = &TerraformSyntaxError{}
	var _ error = &IaCParsingError{}
	var _ error = &BedrockAPIError{}
	var _ error = &WAFRAPIError{}
	var _ error = &FileAccessError{}
	var _ error = &ValidationError{}
}
