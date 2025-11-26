# WAFR Package

This package provides a comprehensive wrapper for the AWS Well-Architected Tool API, implementing all required operations with robust error handling and retry logic.

## Features

### Core Operations

1. **CreateWorkload** - Creates a new workload in AWS Well-Architected Tool
   - Validates input parameters
   - Automatically sets environment to Production
   - Uses "wellarchitected" lens by default
   - Returns AWS workload ID for tracking

2. **ListAnswers (GetQuestions)** - Retrieves WAFR questions with scope filtering
   - Supports three scope levels:
     - Workload: All questions across all 6 pillars
     - Pillar: Questions for a specific pillar
     - Question: A single specific question
   - Handles pagination automatically
   - Converts AWS answer summaries to internal question format

3. **UpdateAnswer (SubmitAnswer)** - Submits answers to AWS
   - Accepts evaluation with selected choices
   - Includes confidence scores in notes
   - Marks answers as applicable by default

4. **CreateMilestone** - Creates snapshots for historical tracking
   - Auto-generates milestone names with timestamps if not provided
   - Returns milestone number for reference

5. **GetConsolidatedReport** - Retrieves reports from AWS
   - Supports both PDF and JSON formats
   - Case-insensitive format handling
   - Returns base64-encoded report data

### Retry Logic with Exponential Backoff

All AWS API operations include automatic retry logic:
- **Retryable errors**: ThrottlingException, ServiceUnavailableException, InternalServerException
- **Non-retryable errors**: ResourceNotFoundException, AccessDeniedException, ValidationException
- **Max retries**: Configurable (default: 3)
- **Backoff strategy**: Exponential with max backoff of 32 seconds
- **Context-aware**: Respects context cancellation

### Error Handling

- Validates all input parameters before making API calls
- Provides clear error messages with context
- Distinguishes between retryable and non-retryable errors
- Logs all operations with structured logging

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/waffle/waffle/internal/wafr"
)

// Create client with default configuration
ctx := context.Background()
clientCfg := &wafr.ClientConfig{
    Region:  "us-east-1",
    Profile: "default",
}

evaluator, err := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)
if err != nil {
    log.Fatal(err)
}
```

### Creating a Workload

```go
awsWorkloadID, err := evaluator.CreateWorkload(
    ctx,
    "my-application",
    "Production workload for my application",
)
if err != nil {
    log.Fatal(err)
}
```

### Getting Questions

```go
// Get all questions for all pillars
scope := core.ReviewScope{
    Level: core.ScopeLevelWorkload,
}

questions, err := evaluator.GetQuestions(ctx, awsWorkloadID, scope)
if err != nil {
    log.Fatal(err)
}

// Get questions for a specific pillar
pillar := core.PillarSecurity
scope = core.ReviewScope{
    Level:  core.ScopeLevelPillar,
    Pillar: &pillar,
}

questions, err = evaluator.GetQuestions(ctx, awsWorkloadID, scope)
```

### Submitting Answers

```go
evaluation := &core.QuestionEvaluation{
    SelectedChoices: []core.Choice{
        {ID: "sec_1_choice_1"},
        {ID: "sec_1_choice_2"},
    },
    ConfidenceScore: 0.95,
    Notes:           "Based on IaC analysis",
}

err = evaluator.SubmitAnswer(ctx, awsWorkloadID, "sec-1", evaluation)
if err != nil {
    log.Fatal(err)
}
```

### Creating Milestones

```go
// With custom name
milestoneID, err := evaluator.CreateMilestone(ctx, awsWorkloadID, "v1.0")

// Auto-generated name
milestoneID, err := evaluator.CreateMilestone(ctx, awsWorkloadID, "")
```

### Getting Reports

```go
// PDF report
pdfReport, err := evaluator.GetConsolidatedReport(ctx, awsWorkloadID, "pdf")

// JSON report
jsonReport, err := evaluator.GetConsolidatedReport(ctx, awsWorkloadID, "json")
```

## Configuration

### Evaluator Configuration

```go
config := &wafr.EvaluatorConfig{
    MaxRetries: 5,                    // Number of retry attempts
    BaseDelay:  2 * time.Second,      // Initial backoff delay
}

evaluator := wafr.NewEvaluator(client, config)
```

### Client Configuration

```go
clientCfg := &wafr.ClientConfig{
    Region:  "us-west-2",
    Profile: "production",
}
```

## Testing

The package includes comprehensive unit tests with mock clients:

```bash
go test ./internal/wafr/... -v
```

### Test Coverage

- CreateWorkload: Success, validation errors, access denied, throttling with retry
- GetQuestions: All scope levels, pagination, error handling
- SubmitAnswer: Success, validation, error handling
- CreateMilestone: Success, auto-generated names, validation
- GetConsolidatedReport: PDF/JSON formats, validation, error handling
- Retry logic: Success after retry, max retries exceeded, non-retryable errors

## Dependencies

- `github.com/aws/aws-sdk-go-v2/service/wellarchitected` - AWS Well-Architected Tool client
- `github.com/aws/aws-sdk-go-v2/config` - AWS configuration
- `github.com/aws/smithy-go` - AWS error handling

## AWS Permissions Required

The following IAM permissions are required:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "wellarchitected:CreateWorkload",
        "wellarchitected:GetWorkload",
        "wellarchitected:ListAnswers",
        "wellarchitected:UpdateAnswer",
        "wellarchitected:CreateMilestone",
        "wellarchitected:GetConsolidatedReport"
      ],
      "Resource": "*"
    }
  ]
}
```

## Logging

All operations are logged using structured logging (slog):

- INFO: Successful operations with key details
- WARN: Retryable errors with backoff information
- ERROR: Max retries exceeded or critical failures

Example log output:
```
INFO workload created workload_id=my-app aws_workload_id=wl-abc123
WARN retryable error, backing off operation=CreateWorkload attempt=1 error_code=ThrottlingException backoff=1s
INFO answer submitted aws_workload_id=wl-abc123 question_id=sec-1 choices_count=2 confidence=0.95
```
