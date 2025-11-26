package core

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorPropagation tests that errors are properly propagated through the system
func TestErrorPropagation(t *testing.T) {
	t.Run("directory access error propagation", func(t *testing.T) {
		// Simulate a directory access error
		baseErr := os.ErrPermission
		dirErr := &DirectoryAccessError{
			Path: "/restricted/path",
			Err:  baseErr,
		}

		// Wrap it as if it came from IaC analyzer
		wrappedErr := errors.Join(errors.New("IaC analysis failed"), dirErr)

		// Should be able to detect the directory access error
		var accessErr *DirectoryAccessError
		assert.True(t, errors.As(wrappedErr, &accessErr))
		assert.Equal(t, "/restricted/path", accessErr.Path)
		
		// Should be able to detect the base permission error
		assert.True(t, errors.Is(wrappedErr, os.ErrPermission))
	})

	t.Run("IaC parsing error propagation", func(t *testing.T) {
		baseErr := errors.New("invalid HCL syntax")
		parseErr := &IaCParsingError{
			File:    "main.tf",
			Err:     baseErr,
			Context: "resource block",
		}

		// Wrap as if it came from the engine
		engineErr := errors.Join(errors.New("workflow failed"), parseErr)

		// Should be able to detect the parsing error
		var iacErr *IaCParsingError
		assert.True(t, errors.As(engineErr, &iacErr))
		assert.Equal(t, "main.tf", iacErr.File)
		assert.Equal(t, "resource block", iacErr.Context)
	})

	t.Run("Bedrock API error propagation", func(t *testing.T) {
		bedrockErr := &BedrockAPIError{
			Operation: "InvokeModel",
			ErrorCode: "ThrottlingException",
			Message:   "Rate limit exceeded",
		}

		// Wrap as if it came from evaluator
		evalErr := errors.Join(errors.New("question evaluation failed"), bedrockErr)

		// Should be able to detect the Bedrock error
		var apiErr *BedrockAPIError
		assert.True(t, errors.As(evalErr, &apiErr))
		assert.Equal(t, "InvokeModel", apiErr.Operation)
		assert.Equal(t, "ThrottlingException", apiErr.ErrorCode)
	})

	t.Run("WAFR API error propagation", func(t *testing.T) {
		wafrErr := &WAFRAPIError{
			Operation: "CreateWorkload",
			ErrorCode: "AccessDeniedException",
			Message:   "Insufficient permissions",
		}

		// Wrap as if it came from the engine
		engineErr := errors.Join(errors.New("session creation failed"), wafrErr)

		// Should be able to detect the WAFR error
		var apiErr *WAFRAPIError
		assert.True(t, errors.As(engineErr, &apiErr))
		assert.Equal(t, "CreateWorkload", apiErr.Operation)
		assert.Equal(t, "AccessDeniedException", apiErr.ErrorCode)
	})

	t.Run("validation error propagation", func(t *testing.T) {
		valErr := &ValidationError{
			Field:   "file_count",
			Value:   15000,
			Message: "exceeded maximum file limit",
		}

		// Wrap as if it came from IaC analyzer
		analyzerErr := errors.Join(errors.New("file retrieval failed"), valErr)

		// Should be able to detect the validation error
		var validErr *ValidationError
		assert.True(t, errors.As(analyzerErr, &validErr))
		assert.Equal(t, "file_count", validErr.Field)
		assert.Equal(t, 15000, validErr.Value)
	})

	t.Run("sentinel error propagation", func(t *testing.T) {
		// Test that sentinel errors can be detected through wrapping
		wrappedErr := errors.Join(
			errors.New("operation failed"),
			ErrSessionNotFound,
		)

		assert.True(t, errors.Is(wrappedErr, ErrSessionNotFound))
	})
}

// TestErrorContextPreservation tests that error context is preserved through the stack
func TestErrorContextPreservation(t *testing.T) {
	t.Run("file access error preserves path and operation", func(t *testing.T) {
		err := &FileAccessError{
			Path:      "/path/to/plan.json",
			Operation: "read",
			Err:       os.ErrNotExist,
		}

		// Wrap multiple times
		err1 := errors.Join(errors.New("plan parsing failed"), err)
		err2 := errors.Join(errors.New("IaC analysis failed"), err1)

		// Should still be able to extract the original error
		var fileErr *FileAccessError
		require.True(t, errors.As(err2, &fileErr))
		assert.Equal(t, "/path/to/plan.json", fileErr.Path)
		assert.Equal(t, "read", fileErr.Operation)
		assert.True(t, errors.Is(err2, os.ErrNotExist))
	})

	t.Run("API errors preserve operation and error code", func(t *testing.T) {
		err := &WAFRAPIError{
			Operation: "UpdateAnswer",
			ErrorCode: "ThrottlingException",
			Message:   "Too many requests",
		}

		// Wrap multiple times
		err1 := errors.Join(errors.New("answer submission failed"), err)
		err2 := errors.Join(errors.New("workflow execution failed"), err1)

		// Should still be able to extract the original error
		var apiErr *WAFRAPIError
		require.True(t, errors.As(err2, &apiErr))
		assert.Equal(t, "UpdateAnswer", apiErr.Operation)
		assert.Equal(t, "ThrottlingException", apiErr.ErrorCode)
		assert.Equal(t, "Too many requests", apiErr.Message)
	})
}

// TestErrorTypeDetection tests that we can detect specific error types
func TestErrorTypeDetection(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		checkFunc func(error) bool
	}{
		{
			name: "detect directory access error",
			err: &DirectoryAccessError{
				Path: "/test",
				Err:  os.ErrPermission,
			},
			checkFunc: func(err error) bool {
				var dirErr *DirectoryAccessError
				return errors.As(err, &dirErr)
			},
		},
		{
			name: "detect IaC parsing error",
			err: &IaCParsingError{
				File: "main.tf",
				Err:  errors.New("syntax error"),
			},
			checkFunc: func(err error) bool {
				var parseErr *IaCParsingError
				return errors.As(err, &parseErr)
			},
		},
		{
			name: "detect Bedrock API error",
			err: &BedrockAPIError{
				Operation: "InvokeModel",
				ErrorCode: "ServiceUnavailable",
			},
			checkFunc: func(err error) bool {
				var apiErr *BedrockAPIError
				return errors.As(err, &apiErr)
			},
		},
		{
			name: "detect WAFR API error",
			err: &WAFRAPIError{
				Operation: "ListAnswers",
				ErrorCode: "ResourceNotFound",
			},
			checkFunc: func(err error) bool {
				var apiErr *WAFRAPIError
				return errors.As(err, &apiErr)
			},
		},
		{
			name: "detect validation error",
			err: &ValidationError{
				Field:   "scope",
				Message: "invalid scope",
			},
			checkFunc: func(err error) bool {
				var valErr *ValidationError
				return errors.As(err, &valErr)
			},
		},
		{
			name: "detect file access error",
			err: &FileAccessError{
				Path:      "/test/file",
				Operation: "write",
				Err:       os.ErrPermission,
			},
			checkFunc: func(err error) bool {
				var fileErr *FileAccessError
				return errors.As(err, &fileErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct detection
			assert.True(t, tt.checkFunc(tt.err))

			// Test detection through wrapping
			wrappedErr := errors.Join(errors.New("outer error"), tt.err)
			assert.True(t, tt.checkFunc(wrappedErr))

			// Test detection through multiple layers
			doubleWrapped := errors.Join(errors.New("another layer"), wrappedErr)
			assert.True(t, tt.checkFunc(doubleWrapped))
		})
	}
}

// TestSentinelErrorDetection tests that sentinel errors can be detected
func TestSentinelErrorDetection(t *testing.T) {
	sentinelErrors := []error{
		ErrPillarRequired,
		ErrQuestionIDRequired,
		ErrInvalidWorkloadID,
		ErrInvalidDirectoryLocation,
		ErrSessionNotFound,
		ErrWorkloadNotFound,
		ErrEvaluatorNotInitialized,
		ErrNoFilesProvided,
		ErrMaxFilesExceeded,
		ErrInvalidPlanFile,
		ErrBedrockInvocationFailed,
		ErrMaxRetriesExceeded,
		ErrSessionAlreadyCompleted,
		ErrInvalidSessionStatus,
	}

	for _, sentinelErr := range sentinelErrors {
		t.Run(sentinelErr.Error(), func(t *testing.T) {
			// Test direct detection
			assert.True(t, errors.Is(sentinelErr, sentinelErr))

			// Test detection through wrapping
			wrappedErr := errors.Join(errors.New("operation failed"), sentinelErr)
			assert.True(t, errors.Is(wrappedErr, sentinelErr))

			// Test detection through multiple layers
			doubleWrapped := errors.Join(errors.New("another layer"), wrappedErr)
			assert.True(t, errors.Is(doubleWrapped, sentinelErr))
		})
	}
}
