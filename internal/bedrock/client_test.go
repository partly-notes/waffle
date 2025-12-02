package bedrock

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffle/waffle/internal/core"
)

// MockBedrockRuntimeClient is a mock implementation of the Bedrock Runtime client
type MockBedrockRuntimeClient struct {
	InvokeModelFunc func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
}

func (m *MockBedrockRuntimeClient) InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	if m.InvokeModelFunc != nil {
		return m.InvokeModelFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}

	client := NewClient(awsConfig, config)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.limiter)
	assert.NotNil(t, client.tokenTracker)
	assert.NotNil(t, client.auditLogger)
	assert.Equal(t, config, client.config)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "eu.anthropic.claude-sonnet-4-20250514-v1:0", config.ModelID)
	assert.Equal(t, "eu-west-1", config.Region)
	assert.Equal(t, 4096, config.MaxTokens)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Equal(t, 0.9, config.TopP)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 60, config.TimeoutSeconds)
	assert.Equal(t, 2.0, config.RateLimit)
}

func TestTokenUsageTracker(t *testing.T) {
	tracker := &TokenUsageTracker{}

	// Record some invocations
	tracker.RecordInvocation(100, 50)
	tracker.RecordInvocation(200, 100)
	tracker.RecordInvocation(150, 75)

	stats := tracker.GetStats()

	assert.Equal(t, int64(450), stats.InputTokens)
	assert.Equal(t, int64(225), stats.OutputTokens)
	assert.Equal(t, int64(3), stats.TotalInvocations)
	assert.Greater(t, stats.EstimatedCost, 0.0)
}

func TestTokenUsageTracker_CalculateCost(t *testing.T) {
	tracker := &TokenUsageTracker{
		inputTokens:  1000,
		outputTokens: 1000,
	}

	cost := tracker.calculateCost()

	// Expected: (1000/1000 * 0.003) + (1000/1000 * 0.015) = 0.003 + 0.015 = 0.018
	assert.InDelta(t, 0.018, cost, 0.001)
}

func TestAuditLogger(t *testing.T) {
	logger := &AuditLogger{logger: slog.Default()}
	ctx := context.Background()

	// These should not panic
	logger.LogInvocation(ctx, "test-model", 100)
	logger.LogSuccess(ctx, 100, 50)
	logger.LogError(ctx, "test_error", assert.AnError)
	logger.LogThrottling(ctx, 1, time.Second)
	logger.LogServiceUnavailable(ctx, 1)
	logger.LogModelTimeout(ctx, 1)
}

func TestBuildSemanticAnalysisPrompt(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	resources := []core.Resource{
		{
			Address: "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
			Properties: map[string]interface{}{
				"bucket": "test-bucket",
			},
		},
	}

	prompt := client.buildSemanticAnalysisPrompt(resources)

	assert.Contains(t, prompt, "Analyze the following AWS resources")
	assert.Contains(t, prompt, "aws_s3_bucket.example")
	assert.Contains(t, prompt, "security_findings")
	assert.Contains(t, prompt, "relationships")
}

func TestBuildWAFREvaluationPrompt(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	question := &core.WAFRQuestion{
		ID:          "sec-1",
		Pillar:      core.PillarSecurity,
		Title:       "How do you protect your data at rest?",
		Description: "Protect data at rest using encryption",
		BestPractices: []core.BestPractice{
			{
				ID:          "bp-1",
				Title:       "Encrypt data at rest",
				Description: "Use encryption for all data at rest",
			},
		},
		Choices: []core.Choice{
			{
				ID:          "choice-1",
				Title:       "Encryption enabled",
				Description: "Data is encrypted at rest",
			},
		},
	}

	model := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				Address: "aws_s3_bucket.example",
				Type:    "aws_s3_bucket",
			},
		},
	}

	prompt := client.buildWAFREvaluationPrompt(question, model)

	assert.Contains(t, prompt, "Well-Architected Framework")
	assert.Contains(t, prompt, "How do you protect your data at rest?")
	assert.Contains(t, prompt, "security")
	assert.Contains(t, prompt, "selected_choices")
	assert.Contains(t, prompt, "evidence")
}

func TestBuildImprovementPrompt(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	risk := &core.Risk{
		ID:       "risk-1",
		Pillar:   core.PillarSecurity,
		Severity: core.RiskLevelHigh,
		Question: &core.WAFRQuestion{
			Title: "Data encryption",
		},
		Description: "Data is not encrypted",
		MissingBestPractices: []core.BestPractice{
			{
				Title:       "Encrypt data",
				Description: "Use encryption",
			},
		},
	}

	resources := []core.Resource{
		{
			Address: "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
		},
	}

	prompt := client.buildImprovementPrompt(risk, resources)

	assert.Contains(t, prompt, "improvement plan")
	assert.Contains(t, prompt, "Data encryption")
	assert.Contains(t, prompt, "HIGH")
	assert.Contains(t, prompt, "description")
	assert.Contains(t, prompt, "rationale")
}

func TestParseSemanticAnalysisResponse(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	tests := []struct {
		name        string
		response    string
		wantErr     bool
		checkResult func(*testing.T, *core.SemanticAnalysis)
	}{
		{
			name: "valid JSON response",
			response: `{
				"security_findings": [
					{
						"resource": "aws_s3_bucket.example",
						"findings": ["encryption not enabled"],
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
			}`,
			wantErr: false,
			checkResult: func(t *testing.T, analysis *core.SemanticAnalysis) {
				assert.Len(t, analysis.SecurityFindings, 1)
				assert.Equal(t, "aws_s3_bucket.example", analysis.SecurityFindings[0].Resource)
				assert.Equal(t, "high", analysis.SecurityFindings[0].Severity)
				assert.Len(t, analysis.Relationships, 1)
			},
		},
		{
			name: "JSON with markdown code blocks",
			response: "```json\n" + `{
				"security_findings": [],
				"relationships": []
			}` + "\n```",
			wantErr: false,
			checkResult: func(t *testing.T, analysis *core.SemanticAnalysis) {
				assert.NotNil(t, analysis)
				assert.Len(t, analysis.SecurityFindings, 0)
			},
		},
		{
			name:     "invalid JSON",
			response: "not json",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseSemanticAnalysisResponse(tt.response)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestParseWAFREvaluationResponse(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	question := &core.WAFRQuestion{
		ID:     "sec-1",
		Pillar: core.PillarSecurity,
		Title:  "Test question",
		Choices: []core.Choice{
			{ID: "choice-1", Title: "Choice 1"},
			{ID: "choice-2", Title: "Choice 2"},
		},
	}

	tests := []struct {
		name        string
		response    string
		wantErr     bool
		checkResult func(*testing.T, *core.QuestionEvaluation)
	}{
		{
			name: "valid evaluation response",
			response: `{
				"selected_choices": ["choice-1"],
				"evidence": [
					{
						"choice_id": "choice-1",
						"explanation": "S3 bucket has encryption",
						"resources": ["aws_s3_bucket.example"],
						"confidence": 0.95
					}
				],
				"overall_confidence": 0.90,
				"notes": "Analysis complete"
			}`,
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.Len(t, eval.SelectedChoices, 1)
				assert.Equal(t, "choice-1", eval.SelectedChoices[0].ID)
				assert.Len(t, eval.Evidence, 1)
				assert.Equal(t, 0.90, eval.ConfidenceScore)
				assert.Equal(t, "Analysis complete", eval.Notes)
			},
		},
		{
			name: "invalid confidence score",
			response: `{
				"selected_choices": [],
				"evidence": [],
				"overall_confidence": 1.5,
				"notes": ""
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseWAFREvaluationResponse(tt.response, question)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestParseImprovementResponse(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	risk := &core.Risk{
		ID: "risk-1",
	}

	tests := []struct {
		name        string
		response    string
		wantErr     bool
		checkResult func(*testing.T, *core.ImprovementPlanItem)
	}{
		{
			name: "valid improvement response",
			response: `{
				"description": "Enable S3 bucket encryption",
				"rationale": "Protects data at rest",
				"best_practice_refs": ["https://docs.aws.amazon.com/..."],
				"affected_resources": ["aws_s3_bucket.example"],
				"estimated_effort": "LOW"
			}`,
			wantErr: false,
			checkResult: func(t *testing.T, item *core.ImprovementPlanItem) {
				assert.Equal(t, "Enable S3 bucket encryption", item.Description)
				assert.Equal(t, "LOW", item.EstimatedEffort)
				assert.Len(t, item.AffectedResources, 1)
			},
		},
		{
			name:     "invalid JSON",
			response: "not json",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseImprovementResponse(tt.response, risk)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with markdown",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with generic markdown",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with whitespace",
			input:    "  \n  {\"key\": \"value\"}  \n  ",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []core.Resource
		contains  []string
	}{
		{
			name:      "empty resources",
			resources: []core.Resource{},
			contains:  []string{"No resources provided"},
		},
		{
			name: "single resource",
			resources: []core.Resource{
				{
					Address: "aws_s3_bucket.example",
					Type:    "aws_s3_bucket",
					Properties: map[string]interface{}{
						"bucket": "test-bucket",
					},
				},
			},
			contains: []string{"aws_s3_bucket.example", "aws_s3_bucket", "test-bucket"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResources(tt.resources)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestInvokeModelOnce_Success(t *testing.T) {
	config := DefaultConfig()
	awsConfig := aws.Config{Region: "us-east-1"}
	client := NewClient(awsConfig, config)

	// Create a mock response
	mockResponse := ClaudeResponse{
		ID:   "test-id",
		Type: "message",
		Role: "assistant",
		Content: []ClaudeContentBlock{
			{
				Type: "text",
				Text: "Test response",
			},
		},
		Usage: ClaudeUsage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}

	responseBody, _ := json.Marshal(mockResponse)

	// Create mock client
	mockClient := &MockBedrockRuntimeClient{
		InvokeModelFunc: func(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
			return &bedrockruntime.InvokeModelOutput{
				Body: responseBody,
			}, nil
		},
	}

	// Replace the client
	client.client = mockClient

	ctx := context.Background()
	result, err := client.invokeModelOnce(ctx, "test prompt")

	require.NoError(t, err)
	assert.Equal(t, "Test response", result)

	// Check token tracking
	stats := client.GetTokenUsageStats()
	assert.Equal(t, int64(100), stats.InputTokens)
	assert.Equal(t, int64(50), stats.OutputTokens)
	assert.Equal(t, int64(1), stats.TotalInvocations)
}
