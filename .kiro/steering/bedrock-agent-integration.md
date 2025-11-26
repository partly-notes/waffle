# Bedrock Direct Invocation Guidelines for Waffle

## Overview

Waffle uses Amazon Bedrock foundation models via direct API invocation to perform intelligent IaC analysis and WAFR question evaluation. This approach requires no infrastructure deployment and makes the CLI fully portable. This document provides guidelines for designing prompts, handling responses, and managing API interactions.

## Architecture Benefits

### No Infrastructure Required

- **Direct Model Invocation**: Use Bedrock Runtime API's InvokeModel operation
- **No Agents to Deploy**: No Lambda functions, no agent configuration
- **Portable**: Works anywhere AWS credentials work
- **Simple Prerequisites**: Enable Bedrock model access in AWS console (one-time)

### Client Configuration

```go
type BedrockConfig struct {
    ModelID         string
    Region          string
    MaxTokens       int
    Temperature     float64
    TopP            float64
    MaxRetries      int
    TimeoutSeconds  int
}

func DefaultBedrockConfig() *BedrockConfig {
    return &BedrockConfig{
        ModelID:        "us.anthropic.claude-sonnet-4-20250514-v1:0",
        Region:         "us-east-1",
        MaxTokens:      4096,
        Temperature:    0.7,
        TopP:           0.9,
        MaxRetries:     3,
        TimeoutSeconds: 60,
    }
}
```

## Prompt Engineering

### Semantic IaC Analysis Prompts

**Objective**: Understand resource semantics and relationships for WAFR evaluation

**Note**: Basic Terraform parsing is done locally using HCL parser libraries. Bedrock is only used for semantic understanding and WAFR-specific analysis.

```go
func buildSemanticAnalysisPrompt(resources []Resource) string {
    return fmt.Sprintf(`Analyze the following AWS resources for semantic understanding and security implications.

Resources:
%s

For each resource, identify:
1. Security-relevant configurations
2. Compliance implications
3. Relationships that affect security posture
4. Missing security controls

Return analysis as JSON:
{
  "security_findings": [
    {
      "resource": "aws_s3_bucket.example",
      "findings": ["encryption not enabled", "public access not blocked"],
      "severity": "high"
    }
  ],
  "relationships": [
    {
      "from": "aws_s3_bucket.example",
      "to": "aws_kms_key.example",
      "type": "encryption",
      "status": "missing"
    }
  ]
}`, formatResources(resources))
}
```

### WAFR Evaluation Prompts

**Objective**: Determine which WAFR choices apply based on IaC analysis

```go
func buildWAFREvaluationPrompt(question *WAFRQuestion, model *WorkloadModel) string {
    return fmt.Sprintf(`You are evaluating an AWS workload against the Well-Architected Framework.

Question: %s
Pillar: %s
Description: %s

Best Practices:
%s

Available Choices:
%s

Workload Resources:
%s

Based on the infrastructure-as-code analysis, determine which choices apply to this workload.

For each applicable choice:
1. Explain why it applies
2. Provide specific evidence from the IaC (resource names, configurations)
3. Assign a confidence score (0.0-1.0) based on data completeness

Return your analysis as JSON:
{
  "selected_choices": ["choice_id_1", "choice_id_2"],
  "evidence": [
    {
      "choice_id": "choice_id_1",
      "explanation": "S3 bucket has encryption enabled",
      "resources": ["aws_s3_bucket.example"],
      "confidence": 0.95
    }
  ],
  "overall_confidence": 0.90,
  "notes": "Additional context or caveats"
}`,
        question.Title,
        question.Pillar,
        question.Description,
        formatBestPractices(question.BestPractices),
        formatChoices(question.Choices),
        formatWorkloadModel(model),
    )
}
```

### Improvement Plan Generation Prompts

**Objective**: Create human-readable guidance for addressing risks

```go
func buildImprovementPrompt(risk *Risk, resources []Resource) string {
    return fmt.Sprintf(`Generate an improvement plan item for the following WAFR risk.

Risk Details:
- Question: %s
- Pillar: %s
- Severity: %s
- Description: %s

Missing Best Practices:
%s

Affected Resources:
%s

Provide a high-level improvement plan that:
1. Describes what changes are needed (no code)
2. Explains why these changes improve the architecture
3. References relevant AWS best practices and documentation
4. Considers relationships between affected resources

Return as JSON:
{
  "description": "Enable encryption at rest for all S3 buckets using AWS KMS",
  "rationale": "Encryption protects data from unauthorized access...",
  "best_practice_refs": [
    "https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/..."
  ],
  "affected_resources": ["aws_s3_bucket.example", "aws_s3_bucket.logs"],
  "estimated_effort": "LOW"
}`,
        risk.Question.Title,
        risk.Pillar,
        risk.Severity,
        risk.Description,
        formatBestPractices(risk.MissingBestPractices),
        formatResources(resources),
    )
}
```

## Response Parsing

### Parse Semantic Analysis Response

```go
type SemanticAnalysisResponse struct {
    SecurityFindings []SecurityFinding `json:"security_findings"`
    Relationships    []Relationship    `json:"relationships"`
}

type SecurityFinding struct {
    Resource string   `json:"resource"`
    Findings []string `json:"findings"`
    Severity string   `json:"severity"`
}

func parseSemanticAnalysisResponse(responseBody []byte) (*SemanticAnalysisResponse, error) {
    var response SemanticAnalysisResponse
    if err := json.Unmarshal(responseBody, &response); err != nil {
        return nil, fmt.Errorf("failed to parse semantic analysis response: %w", err)
    }
    
    return &response, nil
}
```

### Parse WAFR Evaluation Response

```go
type WAFREvaluationResponse struct {
    SelectedChoices   []string           `json:"selected_choices"`
    Evidence          []ChoiceEvidence   `json:"evidence"`
    OverallConfidence float64            `json:"overall_confidence"`
    Notes             string             `json:"notes"`
}

type ChoiceEvidence struct {
    ChoiceID    string   `json:"choice_id"`
    Explanation string   `json:"explanation"`
    Resources   []string `json:"resources"`
    Confidence  float64  `json:"confidence"`
}

func parseWAFREvaluationResponse(responseBody []byte) (*WAFREvaluationResponse, error) {
    var response WAFREvaluationResponse
    if err := json.Unmarshal(responseBody, &response); err != nil {
        return nil, fmt.Errorf("failed to parse WAFR evaluation response: %w", err)
    }
    
    // Validate confidence scores
    if response.OverallConfidence < 0.0 || response.OverallConfidence > 1.0 {
        return nil, fmt.Errorf("invalid overall confidence: %f", response.OverallConfidence)
    }
    
    for _, ev := range response.Evidence {
        if ev.Confidence < 0.0 || ev.Confidence > 1.0 {
            return nil, fmt.Errorf("invalid confidence for choice %s: %f", ev.ChoiceID, ev.Confidence)
        }
    }
    
    return &response, nil
}
```

## Error Handling

### Handle Bedrock API Errors

```go
func (b *BedrockClient) InvokeModelWithRetry(ctx context.Context, prompt string) (string, error) {
    maxRetries := 3
    backoff := 1 * time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        output, err := b.invokeModel(ctx, prompt)
        if err == nil {
            return output, nil
        }
        
        // Check for retryable errors
        var apiErr smithy.APIError
        if errors.As(err, &apiErr) {
            switch apiErr.ErrorCode() {
            case "ThrottlingException":
                slog.WarnContext(ctx, "bedrock throttled, retrying",
                    "attempt", attempt+1,
                    "backoff", backoff,
                )
            case "ServiceUnavailableException":
                slog.WarnContext(ctx, "bedrock unavailable, retrying",
                    "attempt", attempt+1,
                )
            case "ModelTimeoutException":
                slog.WarnContext(ctx, "model timeout, retrying",
                    "attempt", attempt+1,
                )
            default:
                // Non-retryable error
                return "", fmt.Errorf("bedrock model invocation failed: %w", err)
            }
        }
        
        if attempt < maxRetries-1 {
            select {
            case <-time.After(backoff):
                backoff *= 2
            case <-ctx.Done():
                return "", ctx.Err()
            }
        }
    }
    
    return "", fmt.Errorf("max retries exceeded for bedrock model invocation")
}
```

### Handle Malformed Responses

```go
func (b *BedrockClient) ParseResponseWithFallback(responseBody []byte, responseType string) (interface{}, error) {
    switch responseType {
    case "semantic_analysis":
        response, err := parseSemanticAnalysisResponse(responseBody)
        if err != nil {
            // Try to extract partial data
            slog.Warn("failed to parse semantic analysis, attempting partial extraction",
                "error", err,
            )
            return extractPartialSemanticData(responseBody)
        }
        return response, nil
        
    case "wafr_evaluation":
        response, err := parseWAFREvaluationResponse(responseBody)
        if err != nil {
            // Return low-confidence result
            slog.Warn("failed to parse WAFR evaluation, returning low confidence",
                "error", err,
            )
            return &WAFREvaluationResponse{
                SelectedChoices:   []string{},
                Evidence:          []ChoiceEvidence{},
                OverallConfidence: 0.0,
                Notes:             fmt.Sprintf("Failed to parse response: %v", err),
            }, nil
        }
        return response, nil
        
    default:
        return nil, fmt.Errorf("unknown response type: %s", responseType)
    }
}
```

## Cost Optimization

### Token Usage Tracking

```go
type TokenUsageTracker struct {
    mu                sync.Mutex
    inputTokens       int64
    outputTokens      int64
    totalInvocations  int64
}

func (t *TokenUsageTracker) RecordInvocation(inputTokens, outputTokens int) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    t.inputTokens += int64(inputTokens)
    t.outputTokens += int64(outputTokens)
    t.totalInvocations++
}

func (t *TokenUsageTracker) GetStats() TokenUsageStats {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    return TokenUsageStats{
        InputTokens:      t.inputTokens,
        OutputTokens:     t.outputTokens,
        TotalInvocations: t.totalInvocations,
        EstimatedCost:    t.calculateCost(),
    }
}

func (t *TokenUsageTracker) calculateCost() float64 {
    // Claude Sonnet 4 pricing (example)
    inputCostPer1K := 0.003
    outputCostPer1K := 0.015
    
    inputCost := float64(t.inputTokens) / 1000.0 * inputCostPer1K
    outputCost := float64(t.outputTokens) / 1000.0 * outputCostPer1K
    
    return inputCost + outputCost
}
```

### Optimize Prompt Size

```go
func optimizePromptSize(prompt string, maxTokens int) string {
    // Estimate tokens (rough approximation: 1 token ≈ 4 characters)
    estimatedTokens := len(prompt) / 4
    
    if estimatedTokens <= maxTokens {
        return prompt
    }
    
    // Truncate while preserving structure
    targetChars := maxTokens * 4
    truncated := prompt[:targetChars]
    
    // Find last complete sentence
    lastPeriod := strings.LastIndex(truncated, ".")
    if lastPeriod > 0 {
        truncated = truncated[:lastPeriod+1]
    }
    
    return truncated + "\n\n[Content truncated due to size limits]"
}
```

## Testing Strategies

### Mock Bedrock Responses

```go
type MockBedrockClient struct {
    responses map[string]string
    errors    map[string]error
}

func NewMockBedrockClient() *MockBedrockClient {
    return &MockBedrockClient{
        responses: make(map[string]string),
        errors:    make(map[string]error),
    }
}

func (m *MockBedrockClient) SetResponse(promptKey string, response interface{}) error {
    data, err := json.Marshal(response)
    if err != nil {
        return err
    }
    m.responses[promptKey] = string(data)
    return nil
}

func (m *MockBedrockClient) SetError(promptKey string, err error) {
    m.errors[promptKey] = err
}

func (m *MockBedrockClient) InvokeModel(ctx context.Context, prompt string) (string, error) {
    // Use prompt hash or key to lookup mock response
    promptKey := hashPrompt(prompt)
    
    if err, ok := m.errors[promptKey]; ok {
        return "", err
    }
    
    if response, ok := m.responses[promptKey]; ok {
        return response, nil
    }
    
    return "", fmt.Errorf("no mock response configured for prompt")
}
```

### Test with Sample Resources

```go
func TestSemanticAnalysis(t *testing.T) {
    mockClient := NewMockBedrockClient()
    mockClient.SetResponse("semantic", &SemanticAnalysisResponse{
        SecurityFindings: []SecurityFinding{
            {
                Resource: "aws_s3_bucket.example",
                Findings: []string{"encryption not enabled"},
                Severity: "high",
            },
        },
    })
    
    analyzer := NewIaCAnalyzer(mockClient)
    resources := []Resource{
        {Type: "aws_s3_bucket", Name: "example"},
    }
    
    analysis, err := analyzer.AnalyzeSemantics(context.Background(), resources)
    require.NoError(t, err)
    assert.Len(t, analysis.SecurityFindings, 1)
    assert.Equal(t, "high", analysis.SecurityFindings[0].Severity)
}
```

## Best Practices

1. **Keep prompts focused**: Each prompt should have a single, clear purpose
2. **Request structured output**: Always ask for JSON responses with defined schemas
3. **Validate responses**: Check for required fields and valid values
4. **Handle partial failures**: Extract what you can from malformed responses
5. **Track token usage**: Monitor costs and optimize prompt sizes
6. **Use appropriate temperature**: Lower (0.3-0.5) for structured tasks, higher (0.7-0.9) for creative tasks
7. **Implement retries**: Handle transient errors with exponential backoff
8. **Log all interactions**: Keep audit trail of prompts and responses
9. **Test with mocks**: Don't rely on live Bedrock calls for unit tests
10. **Optimize for cost**: Batch operations and minimize redundant calls
11. **No infrastructure required**: Direct invocation means no deployment, no maintenance
12. **Parse locally first**: Use HCL parser for syntax, Bedrock for semantics
