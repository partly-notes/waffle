package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go"
	"golang.org/x/time/rate"

	"github.com/waffle/waffle/internal/core"
)

// Config holds configuration for the Bedrock client
type Config struct {
	ModelID        string
	Region         string
	MaxTokens      int
	Temperature    float64
	TopP           float64
	MaxRetries     int
	TimeoutSeconds int
	RateLimit      float64 // requests per second
}

// DefaultConfig returns default Bedrock configuration
func DefaultConfig() *Config {
	return &Config{
		ModelID:        "us.anthropic.claude-sonnet-4-20250514-v1:0",
		Region:         "us-east-1",
		MaxTokens:      4096,
		Temperature:    0.7,
		TopP:           0.9,
		MaxRetries:     3,
		TimeoutSeconds: 60,
		RateLimit:      2.0, // 2 requests per second
	}
}

// BedrockRuntimeAPI defines the interface for Bedrock Runtime operations
type BedrockRuntimeAPI interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
}

// Client implements the BedrockClient interface
type Client struct {
	client       BedrockRuntimeAPI
	config       *Config
	limiter      *rate.Limiter
	tokenTracker *TokenUsageTracker
	auditLogger  *AuditLogger
}

// TokenUsageTracker tracks token usage for cost monitoring
type TokenUsageTracker struct {
	mu               sync.Mutex
	inputTokens      int64
	outputTokens     int64
	totalInvocations int64
}

// AuditLogger logs all Bedrock operations for audit purposes
type AuditLogger struct {
	logger *slog.Logger
}

// NewClient creates a new Bedrock client
func NewClient(awsConfig aws.Config, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := bedrockruntime.NewFromConfig(awsConfig)

	return &Client{
		client:       client,
		config:       config,
		limiter:      rate.NewLimiter(rate.Limit(config.RateLimit), int(config.RateLimit)),
		tokenTracker: &TokenUsageTracker{},
		auditLogger:  &AuditLogger{logger: slog.Default()},
	}
}

// ClaudeRequest represents a request to Claude models
type ClaudeRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature,omitempty"`
	TopP             float64         `json:"top_p,omitempty"`
	Messages         []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in the Claude API
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents a response from Claude models
type ClaudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []ClaudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason"`
	Usage        ClaudeUsage          `json:"usage"`
	ErrorMessage string               `json:"error,omitempty"`
}

// ClaudeContentBlock represents a content block in Claude response
type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeUsage represents token usage in Claude response
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// InvokeModel invokes a Bedrock model with retry logic
func (c *Client) InvokeModel(ctx context.Context, prompt string) (string, error) {
	// Apply rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Retry with exponential backoff
	backoff := 1 * time.Second
	maxBackoff := 32 * time.Second

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		response, err := c.invokeModelOnce(ctx, prompt)
		if err == nil {
			return response, nil
		}

		// Check for retryable errors
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "ThrottlingException":
				c.auditLogger.LogThrottling(ctx, attempt+1, backoff)
			case "ServiceUnavailableException":
				c.auditLogger.LogServiceUnavailable(ctx, attempt+1)
			case "ModelTimeoutException":
				c.auditLogger.LogModelTimeout(ctx, attempt+1)
			default:
				// Non-retryable error
				c.auditLogger.LogError(ctx, "non-retryable error", err)
				return "", &core.BedrockAPIError{
					Operation: "InvokeModel",
					ErrorCode: apiErr.ErrorCode(),
					Message:   apiErr.ErrorMessage(),
					Err:       err,
				}
			}
		} else {
			// Unknown error type, don't retry
			c.auditLogger.LogError(ctx, "unknown error", err)
			return "", &core.BedrockAPIError{
				Operation: "InvokeModel",
				Err:       err,
			}
		}

		// Retry with backoff
		if attempt < c.config.MaxRetries-1 {
			select {
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	return "", &core.BedrockAPIError{
		Operation: "InvokeModel",
		Message:   fmt.Sprintf("max retries (%d) exceeded", c.config.MaxRetries),
		Err:       core.ErrMaxRetriesExceeded,
	}
}

// invokeModelOnce performs a single model invocation
func (c *Client) invokeModelOnce(ctx context.Context, prompt string) (string, error) {
	// Build request
	request := ClaudeRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        c.config.MaxTokens,
		Temperature:      c.config.Temperature,
		TopP:             c.config.TopP,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log invocation
	c.auditLogger.LogInvocation(ctx, c.config.ModelID, len(prompt))

	// Invoke model
	output, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.config.ModelID),
		Body:        requestBody,
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return "", fmt.Errorf("failed to invoke model: %w", err)
	}

	// Parse response
	var response ClaudeResponse
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error in response
	if response.ErrorMessage != "" {
		return "", fmt.Errorf("model returned error: %s", response.ErrorMessage)
	}

	// Extract text from content blocks
	if len(response.Content) == 0 {
		return "", fmt.Errorf("model returned empty content")
	}

	var result string
	for _, block := range response.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	// Track token usage
	c.tokenTracker.RecordInvocation(response.Usage.InputTokens, response.Usage.OutputTokens)

	// Log success
	c.auditLogger.LogSuccess(ctx, response.Usage.InputTokens, response.Usage.OutputTokens)

	return result, nil
}

// AnalyzeIaCSemantics analyzes IaC resources for semantic understanding
func (c *Client) AnalyzeIaCSemantics(
	ctx context.Context,
	resources []core.Resource,
) (*core.SemanticAnalysis, error) {
	prompt := c.buildSemanticAnalysisPrompt(resources)

	response, err := c.InvokeModel(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze IaC semantics: %w", err)
	}

	analysis, err := c.parseSemanticAnalysisResponse(response)
	if err != nil {
		// Try to extract partial data
		slog.WarnContext(ctx, "failed to parse semantic analysis, attempting partial extraction",
			"error", err,
		)
		return c.extractPartialSemanticData(response)
	}

	return analysis, nil
}

// EvaluateWAFRQuestion evaluates a WAFR question against workload
func (c *Client) EvaluateWAFRQuestion(
	ctx context.Context,
	question *core.WAFRQuestion,
	workloadModel *core.WorkloadModel,
) (*core.QuestionEvaluation, error) {
	prompt := c.buildWAFREvaluationPrompt(question, workloadModel)

	response, err := c.InvokeModel(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate WAFR question: %w", err)
	}

	evaluation, err := c.parseWAFREvaluationResponse(response, question)
	if err != nil {
		// Return low-confidence result
		slog.WarnContext(ctx, "failed to parse WAFR evaluation, returning low confidence",
			"error", err,
		)
		return &core.QuestionEvaluation{
			Question:        question,
			SelectedChoices: []core.Choice{},
			Evidence:        []core.Evidence{},
			ConfidenceScore: 0.0,
			Notes:           fmt.Sprintf("Failed to parse response: %v", err),
		}, nil
	}

	return evaluation, nil
}

// GenerateImprovementGuidance generates improvement guidance for a risk
func (c *Client) GenerateImprovementGuidance(
	ctx context.Context,
	risk *core.Risk,
	resources []core.Resource,
) (*core.ImprovementPlanItem, error) {
	prompt := c.buildImprovementPrompt(risk, resources)

	response, err := c.InvokeModel(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate improvement guidance: %w", err)
	}

	item, err := c.parseImprovementResponse(response, risk)
	if err != nil {
		return nil, fmt.Errorf("failed to parse improvement response: %w", err)
	}

	return item, nil
}

// GetTokenUsageStats returns token usage statistics
func (c *Client) GetTokenUsageStats() TokenUsageStats {
	return c.tokenTracker.GetStats()
}

// TokenUsageStats represents token usage statistics
type TokenUsageStats struct {
	InputTokens      int64
	OutputTokens     int64
	TotalInvocations int64
	EstimatedCost    float64
}

// RecordInvocation records token usage for an invocation
func (t *TokenUsageTracker) RecordInvocation(inputTokens, outputTokens int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.inputTokens += int64(inputTokens)
	t.outputTokens += int64(outputTokens)
	t.totalInvocations++
}

// GetStats returns current token usage statistics
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

// calculateCost estimates the cost based on token usage
func (t *TokenUsageTracker) calculateCost() float64 {
	// Claude Sonnet 4 pricing (example rates)
	inputCostPer1K := 0.003
	outputCostPer1K := 0.015

	inputCost := float64(t.inputTokens) / 1000.0 * inputCostPer1K
	outputCost := float64(t.outputTokens) / 1000.0 * outputCostPer1K

	return inputCost + outputCost
}

// Audit logging methods
func (a *AuditLogger) LogInvocation(ctx context.Context, modelID string, promptLength int) {
	a.logger.InfoContext(ctx, "bedrock_invocation",
		"event_type", "bedrock_invocation",
		"model_id", modelID,
		"prompt_length", promptLength,
		"timestamp", time.Now().UTC(),
	)
}

func (a *AuditLogger) LogSuccess(ctx context.Context, inputTokens, outputTokens int) {
	a.logger.InfoContext(ctx, "bedrock_success",
		"event_type", "bedrock_success",
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"timestamp", time.Now().UTC(),
	)
}

func (a *AuditLogger) LogError(ctx context.Context, errorType string, err error) {
	a.logger.ErrorContext(ctx, "bedrock_error",
		"event_type", "bedrock_error",
		"error_type", errorType,
		"error", err.Error(),
		"timestamp", time.Now().UTC(),
	)
}

func (a *AuditLogger) LogThrottling(ctx context.Context, attempt int, backoff time.Duration) {
	a.logger.WarnContext(ctx, "bedrock_throttled",
		"event_type", "bedrock_throttled",
		"attempt", attempt,
		"backoff", backoff.String(),
		"timestamp", time.Now().UTC(),
	)
}

func (a *AuditLogger) LogServiceUnavailable(ctx context.Context, attempt int) {
	a.logger.WarnContext(ctx, "bedrock_unavailable",
		"event_type", "bedrock_unavailable",
		"attempt", attempt,
		"timestamp", time.Now().UTC(),
	)
}

func (a *AuditLogger) LogModelTimeout(ctx context.Context, attempt int) {
	a.logger.WarnContext(ctx, "bedrock_timeout",
		"event_type", "bedrock_timeout",
		"attempt", attempt,
		"timestamp", time.Now().UTC(),
	)
}
