package core

import (
	"errors"
	"fmt"
)

var (
	// ErrPillarRequired is returned when pillar scope is selected but no pillar is specified
	ErrPillarRequired = errors.New("pillar is required when scope level is pillar")

	// ErrQuestionIDRequired is returned when question scope is selected but no question ID is specified
	ErrQuestionIDRequired = errors.New("question ID is required when scope level is question")

	// ErrInvalidWorkloadID is returned when the workload ID is invalid
	ErrInvalidWorkloadID = errors.New("invalid workload ID")

	// ErrInvalidDirectoryLocation is returned when the directory location is invalid
	ErrInvalidDirectoryLocation = errors.New("invalid directory location")

	// ErrSessionNotFound is returned when a session cannot be found
	ErrSessionNotFound = errors.New("session not found")

	// ErrWorkloadNotFound is returned when a workload cannot be found
	ErrWorkloadNotFound = errors.New("workload not found")

	// ErrEvaluatorNotInitialized is returned when the evaluator is not initialized
	ErrEvaluatorNotInitialized = errors.New("evaluator not initialized")

	// ErrNoFilesProvided is returned when no files are provided for parsing
	ErrNoFilesProvided = errors.New("no files provided for parsing")

	// ErrMaxFilesExceeded is returned when the maximum file limit is exceeded
	ErrMaxFilesExceeded = errors.New("exceeded maximum file limit")

	// ErrInvalidPlanFile is returned when the plan file is invalid
	ErrInvalidPlanFile = errors.New("invalid plan file")

	// ErrBedrockInvocationFailed is returned when Bedrock model invocation fails
	ErrBedrockInvocationFailed = errors.New("bedrock model invocation failed")

	// ErrMaxRetriesExceeded is returned when maximum retries are exceeded
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")

	// ErrSessionAlreadyCompleted is returned when trying to resume a completed session
	ErrSessionAlreadyCompleted = errors.New("session already completed")

	// ErrInvalidSessionStatus is returned when session status is invalid for the operation
	ErrInvalidSessionStatus = errors.New("invalid session status for operation")
)

// DirectoryAccessError represents an error accessing the directory
type DirectoryAccessError struct {
	Path string
	Err  error
}

func (e *DirectoryAccessError) Error() string {
	return "cannot access directory at " + e.Path + ": " + e.Err.Error()
}

func (e *DirectoryAccessError) Unwrap() error {
	return e.Err
}

// WorkloadNotFoundError represents an error when a workload is not found
type WorkloadNotFoundError struct {
	WorkloadID string
}

func (e *WorkloadNotFoundError) Error() string {
	return "workload not found: " + e.WorkloadID
}

// InsufficientPermissionsError represents an error when permissions are insufficient
type InsufficientPermissionsError struct {
	Operation string
}

func (e *InsufficientPermissionsError) Error() string {
	return "insufficient permissions for operation: " + e.Operation
}

// RateLimitError represents an error when rate limits are exceeded
type RateLimitError struct {
	Operation string
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded for operation: " + e.Operation
}

// TerraformSyntaxError represents an error in Terraform syntax
type TerraformSyntaxError struct {
	File    string
	Line    int
	Message string
}

func (e *TerraformSyntaxError) Error() string {
	return fmt.Sprintf("terraform syntax error in %s at line %d: %s", e.File, e.Line, e.Message)
}

// IaCParsingError represents an error during IaC parsing
type IaCParsingError struct {
	File    string
	Err     error
	Context string
}

func (e *IaCParsingError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("failed to parse IaC file %s (%s): %v", e.File, e.Context, e.Err)
	}
	return fmt.Sprintf("failed to parse IaC file %s: %v", e.File, e.Err)
}

func (e *IaCParsingError) Unwrap() error {
	return e.Err
}

// BedrockAPIError represents an error from Bedrock API
type BedrockAPIError struct {
	Operation string
	ErrorCode string
	Message   string
	Err       error
}

func (e *BedrockAPIError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("bedrock %s failed [%s]: %s", e.Operation, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("bedrock %s failed: %v", e.Operation, e.Err)
}

func (e *BedrockAPIError) Unwrap() error {
	return e.Err
}

// WAFRAPIError represents an error from AWS Well-Architected Tool API
type WAFRAPIError struct {
	Operation string
	ErrorCode string
	Message   string
	Err       error
}

func (e *WAFRAPIError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("WAFR %s failed [%s]: %s", e.Operation, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("WAFR %s failed: %v", e.Operation, e.Err)
}

func (e *WAFRAPIError) Unwrap() error {
	return e.Err
}

// FileAccessError represents an error accessing a file
type FileAccessError struct {
	Path      string
	Operation string
	Err       error
}

func (e *FileAccessError) Error() string {
	return fmt.Sprintf("failed to %s file %s: %v", e.Operation, e.Path, e.Err)
}

func (e *FileAccessError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation failed for %s (value: %v): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}
