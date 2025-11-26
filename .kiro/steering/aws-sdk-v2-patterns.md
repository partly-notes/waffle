# AWS SDK v2 Patterns for Waffle

## Client Initialization

### Load AWS Configuration with Context

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/wellarchitected"
    "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

func InitializeAWSClients(ctx context.Context, region, profile string) (*wellarchitected.Client, *bedrockruntime.Client, error) {
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(region),
        config.WithSharedConfigProfile(profile),
    )
    if err != nil {
        return nil, nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    waClient := wellarchitected.NewFromConfig(cfg)
    bedrockClient := bedrockruntime.NewFromConfig(cfg)
    
    return waClient, bedrockClient, nil
}
```

### Support Multiple Credential Sources

```go
func LoadConfigWithOptions(ctx context.Context, opts *AWSOptions) (aws.Config, error) {
    configOpts := []func(*config.LoadOptions) error{
        config.WithRegion(opts.Region),
    }
    
    // Support profile-based credentials
    if opts.Profile != "" {
        configOpts = append(configOpts, config.WithSharedConfigProfile(opts.Profile))
    }
    
    // Support role assumption
    if opts.RoleARN != "" {
        configOpts = append(configOpts, config.WithAssumeRoleCredentialOptions(func(o *stscreds.AssumeRoleOptions) {
            o.RoleARN = opts.RoleARN
            if opts.ExternalID != "" {
                o.ExternalID = aws.String(opts.ExternalID)
            }
        }))
    }
    
    cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    return cfg, nil
}
```

## Error Handling

### Check for Specific AWS Error Types

```go
import (
    "github.com/aws/smithy-go"
    "github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
)

func (e *WAFREvaluator) GetWorkload(ctx context.Context, workloadID string) (*types.Workload, error) {
    output, err := e.client.GetWorkload(ctx, &wellarchitected.GetWorkloadInput{
        WorkloadId: aws.String(workloadID),
    })
    
    if err != nil {
        var apiErr smithy.APIError
        if errors.As(err, &apiErr) {
            switch apiErr.ErrorCode() {
            case "ResourceNotFoundException":
                return nil, &WorkloadNotFoundError{WorkloadID: workloadID}
            case "AccessDeniedException":
                return nil, &InsufficientPermissionsError{Operation: "GetWorkload"}
            case "ThrottlingException":
                return nil, &RateLimitError{Operation: "GetWorkload"}
            }
        }
        return nil, fmt.Errorf("failed to get workload: %w", err)
    }
    
    return output.Workload, nil
}
```

### Handle Validation Errors

```go
import "github.com/aws/smithy-go"

func handleAWSError(err error, operation string) error {
    if err == nil {
        return nil
    }
    
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        return fmt.Errorf("%s failed: %s - %s", operation, apiErr.ErrorCode(), apiErr.ErrorMessage())
    }
    
    return fmt.Errorf("%s failed: %w", operation, err)
}
```

## Retry and Timeout Configuration

### Configure Retry Strategy

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws/retry"
    "time"
)

func NewClientWithRetry(cfg aws.Config, maxRetries int) *wellarchitected.Client {
    return wellarchitected.NewFromConfig(cfg, func(o *wellarchitected.Options) {
        o.Retryer = retry.NewStandard(func(so *retry.StandardOptions) {
            so.MaxAttempts = maxRetries
            so.MaxBackoff = 20 * time.Second
        })
    })
}
```

### Use Context Timeouts for Operations

```go
func (e *WAFREvaluator) CreateWorkloadWithTimeout(ctx context.Context, input *CreateWorkloadInput) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    output, err := e.client.CreateWorkload(ctx, &wellarchitected.CreateWorkloadInput{
        WorkloadName: aws.String(input.Name),
        Description:  aws.String(input.Description),
        Environment:  types.WorkloadEnvironment(input.Environment),
        Lenses:       []string{"wellarchitected"},
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to create workload: %w", err)
    }
    
    return aws.ToString(output.WorkloadId), nil
}
```

## Pagination

### Handle Paginated Responses

```go
import "github.com/aws/aws-sdk-go-v2/service/wellarchitected"

func (e *WAFREvaluator) ListAllWorkloads(ctx context.Context) ([]*types.WorkloadSummary, error) {
    var workloads []*types.WorkloadSummary
    
    paginator := wellarchitected.NewListWorkloadsPaginator(e.client, &wellarchitected.ListWorkloadsInput{
        MaxResults: aws.Int32(50),
    })
    
    for paginator.HasMorePages() {
        output, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, fmt.Errorf("failed to list workloads: %w", err)
        }
        
        for i := range output.WorkloadSummaries {
            workloads = append(workloads, &output.WorkloadSummaries[i])
        }
    }
    
    return workloads, nil
}
```

### Manual Pagination with NextToken

```go
func (e *WAFREvaluator) ListAnswers(ctx context.Context, workloadID, lensAlias, pillarID string) ([]*types.AnswerSummary, error) {
    var answers []*types.AnswerSummary
    var nextToken *string
    
    for {
        output, err := e.client.ListAnswers(ctx, &wellarchitected.ListAnswersInput{
            WorkloadId: aws.String(workloadID),
            LensAlias:  aws.String(lensAlias),
            PillarId:   aws.String(pillarID),
            NextToken:  nextToken,
            MaxResults: aws.Int32(50),
        })
        
        if err != nil {
            return nil, fmt.Errorf("failed to list answers: %w", err)
        }
        
        for i := range output.AnswerSummaries {
            answers = append(answers, &output.AnswerSummaries[i])
        }
        
        if output.NextToken == nil {
            break
        }
        nextToken = output.NextToken
    }
    
    return answers, nil
}
```

## Working with AWS Types

### Use aws.String, aws.Int32, etc. for Pointers

```go
import "github.com/aws/aws-sdk-go-v2/aws"

func (e *WAFREvaluator) UpdateAnswer(ctx context.Context, input *UpdateAnswerInput) error {
    _, err := e.client.UpdateAnswer(ctx, &wellarchitected.UpdateAnswerInput{
        WorkloadId:      aws.String(input.WorkloadID),
        LensAlias:       aws.String(input.LensAlias),
        QuestionId:      aws.String(input.QuestionID),
        SelectedChoices: input.SelectedChoices,
        Notes:           aws.String(input.Notes),
        IsApplicable:    aws.Bool(true),
    })
    
    return err
}
```

### Safely Dereference AWS Pointers

```go
func extractWorkloadInfo(workload *types.Workload) WorkloadInfo {
    return WorkloadInfo{
        ID:          aws.ToString(workload.WorkloadId),
        Name:        aws.ToString(workload.WorkloadName),
        Description: aws.ToString(workload.Description),
        Environment: string(workload.Environment),
        UpdatedAt:   aws.ToTime(workload.UpdatedAt),
        RiskCounts:  extractRiskCounts(workload.RiskCounts),
    }
}

func extractRiskCounts(counts map[string]int32) map[string]int {
    result := make(map[string]int)
    for k, v := range counts {
        result[k] = int(v)
    }
    return result
}
```

## Bedrock Runtime Patterns

### Invoke Bedrock Model

```go
import (
    "encoding/json"
    "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type ClaudeRequest struct {
    Prompt            string  `json:"prompt"`
    MaxTokensToSample int     `json:"max_tokens_to_sample"`
    Temperature       float64 `json:"temperature"`
    TopP              float64 `json:"top_p"`
}

type ClaudeResponse struct {
    Completion string `json:"completion"`
}

func (b *BedrockClient) InvokeModel(ctx context.Context, prompt string) (string, error) {
    request := ClaudeRequest{
        Prompt:            prompt,
        MaxTokensToSample: 2048,
        Temperature:       0.7,
        TopP:              0.9,
    }
    
    requestBody, err := json.Marshal(request)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }
    
    output, err := b.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
        ModelId:     aws.String("us.anthropic.claude-sonnet-4-20250514-v1:0"),
        Body:        requestBody,
        ContentType: aws.String("application/json"),
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to invoke model: %w", err)
    }
    
    var response ClaudeResponse
    if err := json.Unmarshal(output.Body, &response); err != nil {
        return "", fmt.Errorf("failed to unmarshal response: %w", err)
    }
    
    return response.Completion, nil
}
```

### Stream Bedrock Responses

```go
func (b *BedrockClient) InvokeModelWithResponseStream(ctx context.Context, prompt string) (<-chan string, <-chan error) {
    chunks := make(chan string)
    errs := make(chan error, 1)
    
    go func() {
        defer close(chunks)
        defer close(errs)
        
        request := ClaudeRequest{
            Prompt:            prompt,
            MaxTokensToSample: 2048,
        }
        
        requestBody, err := json.Marshal(request)
        if err != nil {
            errs <- fmt.Errorf("failed to marshal request: %w", err)
            return
        }
        
        output, err := b.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
            ModelId:     aws.String(b.modelID),
            Body:        requestBody,
            ContentType: aws.String("application/json"),
        })
        
        if err != nil {
            errs <- fmt.Errorf("failed to invoke model: %w", err)
            return
        }
        
        stream := output.GetStream()
        defer stream.Close()
        
        for event := range stream.Events() {
            switch e := event.(type) {
            case *types.ResponseStreamMemberChunk:
                var chunk ClaudeResponse
                if err := json.Unmarshal(e.Value.Bytes, &chunk); err != nil {
                    errs <- fmt.Errorf("failed to unmarshal chunk: %w", err)
                    return
                }
                chunks <- chunk.Completion
            }
        }
        
        if err := stream.Err(); err != nil {
            errs <- fmt.Errorf("stream error: %w", err)
        }
    }()
    
    return chunks, errs
}
```

## Rate Limiting

### Implement Token Bucket Rate Limiter

```go
import (
    "golang.org/x/time/rate"
    "time"
)

type RateLimitedClient struct {
    client  *wellarchitected.Client
    limiter *rate.Limiter
}

func NewRateLimitedClient(client *wellarchitected.Client, requestsPerSecond float64) *RateLimitedClient {
    return &RateLimitedClient{
        client:  client,
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), int(requestsPerSecond)),
    }
}

func (r *RateLimitedClient) CreateWorkload(ctx context.Context, input *wellarchitected.CreateWorkloadInput) (*wellarchitected.CreateWorkloadOutput, error) {
    if err := r.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit wait failed: %w", err)
    }
    
    return r.client.CreateWorkload(ctx, input)
}
```

### Handle Throttling with Exponential Backoff

```go
func (e *WAFREvaluator) UpdateAnswerWithBackoff(ctx context.Context, input *UpdateAnswerInput) error {
    backoff := 1 * time.Second
    maxBackoff := 32 * time.Second
    maxRetries := 5
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        _, err := e.client.UpdateAnswer(ctx, &wellarchitected.UpdateAnswerInput{
            WorkloadId:      aws.String(input.WorkloadID),
            LensAlias:       aws.String(input.LensAlias),
            QuestionId:      aws.String(input.QuestionID),
            SelectedChoices: input.SelectedChoices,
        })
        
        if err == nil {
            return nil
        }
        
        var apiErr smithy.APIError
        if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ThrottlingException" {
            if attempt < maxRetries-1 {
                slog.WarnContext(ctx, "throttled, retrying",
                    "attempt", attempt+1,
                    "backoff", backoff,
                )
                
                select {
                case <-time.After(backoff):
                    backoff *= 2
                    if backoff > maxBackoff {
                        backoff = maxBackoff
                    }
                case <-ctx.Done():
                    return ctx.Err()
                }
                continue
            }
        }
        
        return fmt.Errorf("failed to update answer: %w", err)
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

## Testing with AWS SDK

### Mock AWS Clients

```go
type MockWAFRClient struct {
    CreateWorkloadFunc func(ctx context.Context, input *wellarchitected.CreateWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error)
    GetWorkloadFunc    func(ctx context.Context, input *wellarchitected.GetWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error)
}

func (m *MockWAFRClient) CreateWorkload(ctx context.Context, input *wellarchitected.CreateWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
    if m.CreateWorkloadFunc != nil {
        return m.CreateWorkloadFunc(ctx, input, opts...)
    }
    return &wellarchitected.CreateWorkloadOutput{
        WorkloadId: aws.String("test-workload-id"),
    }, nil
}

func (m *MockWAFRClient) GetWorkload(ctx context.Context, input *wellarchitected.GetWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error) {
    if m.GetWorkloadFunc != nil {
        return m.GetWorkloadFunc(ctx, input, opts...)
    }
    return nil, &types.ResourceNotFoundException{
        Message: aws.String("workload not found"),
    }
}
```

### Use Table Tests with Mock Responses

```go
func TestCreateWorkload(t *testing.T) {
    tests := []struct {
        name       string
        input      *CreateWorkloadInput
        mockFunc   func(ctx context.Context, input *wellarchitected.CreateWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error)
        wantErr    bool
        wantErrMsg string
    }{
        {
            name: "successful creation",
            input: &CreateWorkloadInput{
                Name:        "test-workload",
                Description: "test description",
            },
            mockFunc: func(ctx context.Context, input *wellarchitected.CreateWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
                return &wellarchitected.CreateWorkloadOutput{
                    WorkloadId: aws.String("wl-123"),
                }, nil
            },
            wantErr: false,
        },
        {
            name: "access denied",
            input: &CreateWorkloadInput{
                Name: "test-workload",
            },
            mockFunc: func(ctx context.Context, input *wellarchitected.CreateWorkloadInput, opts ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
                return nil, &types.AccessDeniedException{
                    Message: aws.String("insufficient permissions"),
                }
            },
            wantErr:    true,
            wantErrMsg: "insufficient permissions",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClient := &MockWAFRClient{
                CreateWorkloadFunc: tt.mockFunc,
            }
            
            evaluator := &WAFREvaluator{client: mockClient}
            workloadID, err := evaluator.CreateWorkload(context.Background(), tt.input)
            
            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErrMsg != "" {
                    assert.Contains(t, err.Error(), tt.wantErrMsg)
                }
            } else {
                require.NoError(t, err)
                assert.NotEmpty(t, workloadID)
            }
        })
    }
}
```

## Best Practices Summary

1. **Always use context**: Pass context to all AWS SDK calls for cancellation and timeouts
2. **Handle errors properly**: Check for specific AWS error types using smithy.APIError
3. **Use pagination**: Always handle paginated responses for list operations
4. **Implement retries**: Use exponential backoff for transient errors
5. **Rate limit**: Implement rate limiting to avoid throttling
6. **Use pointers safely**: Always use aws.ToString, aws.ToInt32, etc. to safely dereference
7. **Configure timeouts**: Set appropriate timeouts for long-running operations
8. **Mock for testing**: Create mock clients that implement the same interface
9. **Log operations**: Use structured logging for all AWS API calls
10. **Handle credentials**: Support multiple credential sources (profiles, roles, environment)
