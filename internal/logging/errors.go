package logging

import (
	"context"
	"fmt"
)

// ErrorWithContext represents an error with additional context and troubleshooting guidance
type ErrorWithContext struct {
	Operation       string
	Err             error
	Context         map[string]interface{}
	Troubleshooting string
}

// Error implements the error interface
func (e *ErrorWithContext) Error() string {
	if e.Troubleshooting != "" {
		return fmt.Sprintf("%s failed: %v\n\nTroubleshooting: %s", e.Operation, e.Err, e.Troubleshooting)
	}
	return fmt.Sprintf("%s failed: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *ErrorWithContext) Unwrap() error {
	return e.Err
}

// LogError logs an error with full context
func (e *ErrorWithContext) LogError(ctx context.Context, logger *Logger) {
	args := []any{
		"operation", e.Operation,
		"error", e.Err,
	}

	for k, v := range e.Context {
		args = append(args, k, v)
	}

	if e.Troubleshooting != "" {
		args = append(args, "troubleshooting", e.Troubleshooting)
	}

	logger.ErrorContext(ctx, "operation failed", args...)
}

// NewErrorWithContext creates a new error with context
func NewErrorWithContext(operation string, err error, troubleshooting string) *ErrorWithContext {
	return &ErrorWithContext{
		Operation:       operation,
		Err:             err,
		Context:         make(map[string]interface{}),
		Troubleshooting: troubleshooting,
	}
}

// WithContext adds context to the error
func (e *ErrorWithContext) WithContext(key string, value interface{}) *ErrorWithContext {
	e.Context[key] = value
	return e
}

// Common troubleshooting messages
const (
	TroubleshootingDirectoryAccess = `
Possible causes:
1. The directory does not exist
2. You don't have read permissions for the directory
3. The path is incorrect

Solutions:
- Verify the directory path exists: ls -la <path>
- Check directory permissions: ls -ld <path>
- Ensure you're running from the correct directory
- Try using an absolute path instead of a relative path
`

	TroubleshootingTerraformParsing = `
Possible causes:
1. Invalid Terraform HCL syntax
2. Unsupported Terraform version
3. Missing required Terraform files

Solutions:
- Validate Terraform syntax: terraform validate
- Check Terraform version: terraform version
- Ensure all .tf files are present in the directory
- Review the Terraform documentation for syntax errors
`

	TroubleshootingTerraformPlan = `
Possible causes:
1. The plan file doesn't exist at the specified path
2. The plan file is not in JSON format
3. The plan file is corrupted or incomplete

Solutions:
- Generate a new plan file: terraform plan -out=plan.tfplan && terraform show -json plan.tfplan > plan.json
- Verify the file exists: ls -la <plan-file-path>
- Check the file is valid JSON: cat <plan-file-path> | jq .
- Ensure you're using the correct path to the plan file
`

	TroubleshootingAWSCredentials = `
Possible causes:
1. AWS credentials are not configured
2. The AWS profile doesn't exist
3. Credentials have expired
4. Insufficient IAM permissions

Solutions:
- Configure AWS credentials: aws configure
- Check available profiles: cat ~/.aws/credentials
- Verify credentials: aws sts get-caller-identity
- Ensure your IAM user/role has the required permissions
- Try using a different AWS profile: --profile <profile-name>
`

	TroubleshootingBedrockAccess = `
Possible causes:
1. Bedrock is not enabled in your AWS region
2. The model is not available in your region
3. You don't have permissions to invoke Bedrock models
4. Model access has not been granted

Solutions:
- Enable Bedrock in the AWS Console: https://console.aws.amazon.com/bedrock
- Request model access in the Bedrock console
- Verify your region supports Bedrock: aws bedrock list-foundation-models --region <region>
- Check IAM permissions include: bedrock:InvokeModel
- Try a different AWS region that supports Bedrock
`

	TroubleshootingWAFRAccess = `
Possible causes:
1. You don't have permissions to access AWS Well-Architected Tool
2. The workload doesn't exist
3. The workload is in a different region

Solutions:
- Check IAM permissions include: wellarchitected:*
- Verify the workload exists: aws wellarchitected list-workloads
- Ensure you're using the correct AWS region
- Check the workload ID is correct
`

	TroubleshootingRateLimit = `
Possible causes:
1. Too many API requests in a short time
2. AWS service quotas exceeded

Solutions:
- Wait a few seconds and retry the operation
- Reduce the number of concurrent operations
- Request a service quota increase in AWS Service Quotas console
- Use a different AWS region with higher quotas
`

	TroubleshootingSessionNotFound = `
Possible causes:
1. The session ID is incorrect
2. The session was deleted
3. The session directory is not accessible

Solutions:
- List available sessions: waffle status
- Check the session directory: ls ~/.waffle/sessions/
- Verify the session ID is correct
- Ensure the session file has not been manually deleted
`

	TroubleshootingDiskSpace = `
Possible causes:
1. Insufficient disk space
2. Disk quota exceeded
3. Permission issues with the log directory

Solutions:
- Check available disk space: df -h
- Clean up old log files: rm ~/.waffle/logs/*.log
- Check disk quotas: quota -v
- Verify write permissions: ls -ld ~/.waffle/logs/
`
)

// WrapError wraps an error with operation context and troubleshooting guidance
func WrapError(operation string, err error, troubleshooting string) error {
	if err == nil {
		return nil
	}
	return NewErrorWithContext(operation, err, troubleshooting)
}

// LogAndWrapError logs an error and returns it wrapped with context
func LogAndWrapError(ctx context.Context, logger *Logger, operation string, err error, troubleshooting string) error {
	if err == nil {
		return nil
	}

	errWithCtx := NewErrorWithContext(operation, err, troubleshooting)
	errWithCtx.LogError(ctx, logger)
	return errWithCtx
}
