package config

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/smithy-go"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Name    string
	Success bool
	Message string
	Error   error
}

// Validator validates AWS setup and permissions
type Validator struct {
	cfg *Config
}

// NewValidator creates a new Validator
func NewValidator(cfg *Config) *Validator {
	return &Validator{cfg: cfg}
}

// ValidateAll validates all AWS setup requirements
func (v *Validator) ValidateAll(ctx context.Context) ([]ValidationResult, error) {
	results := []ValidationResult{}

	// 1. Validate AWS credentials
	credResult := v.validateCredentials(ctx)
	results = append(results, credResult)
	if !credResult.Success {
		// If credentials fail, no point checking other things
		return results, nil
	}

	// 2. Validate Bedrock access
	bedrockResult := v.validateBedrockAccess(ctx)
	results = append(results, bedrockResult)

	// 3. Validate WAFR permissions
	wafrResult := v.validateWAFRPermissions(ctx)
	results = append(results, wafrResult)

	return results, nil
}

// validateCredentials checks if AWS credentials are configured
func (v *Validator) validateCredentials(ctx context.Context) ValidationResult {
	result := ValidationResult{
		Name: "AWS Credentials",
	}

	// Try to load AWS config
	opts := []func(*config.LoadOptions) error{}

	if v.cfg.AWS.Region != "" {
		opts = append(opts, config.WithRegion(v.cfg.AWS.Region))
	} else if v.cfg.Bedrock.Region != "" {
		opts = append(opts, config.WithRegion(v.cfg.Bedrock.Region))
	}

	if v.cfg.AWS.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(v.cfg.AWS.Profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		result.Success = false
		result.Message = "Failed to load AWS credentials"
		result.Error = err
		return result
	}

	// Try to retrieve credentials to verify they're valid
	creds, err := awsCfg.Credentials.Retrieve(ctx)
	if err != nil {
		result.Success = false
		result.Message = "Failed to retrieve AWS credentials"
		result.Error = err
		return result
	}

	result.Success = true
	if v.cfg.AWS.Profile != "" {
		result.Message = fmt.Sprintf("AWS credentials configured (profile: %s)", v.cfg.AWS.Profile)
	} else {
		result.Message = fmt.Sprintf("AWS credentials configured (source: %s)", creds.Source)
	}

	return result
}

// validateBedrockAccess checks if Bedrock model access is enabled
func (v *Validator) validateBedrockAccess(ctx context.Context) ValidationResult {
	result := ValidationResult{
		Name: "Bedrock Model Access",
	}

	// Load AWS config
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(v.cfg.Bedrock.Region),
	}

	if v.cfg.AWS.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(v.cfg.AWS.Profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		result.Success = false
		result.Message = "Failed to load AWS config for Bedrock"
		result.Error = err
		return result
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(awsCfg)

	// Try a minimal invocation to check access
	// We'll use a very small prompt to minimize cost
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create a minimal test request using Messages API format (required for Claude Sonnet 4)
	testPrompt := `{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens": 1,
		"messages": [
			{
				"role": "user",
				"content": "Hi"
			}
		]
	}`

	_, err = client.InvokeModel(testCtx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(v.cfg.Bedrock.ModelID),
		Body:        []byte(testPrompt),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		var apiErr smithy.APIError
		if err, ok := err.(*smithy.OperationError); ok {
			apiErr, _ = err.Err.(smithy.APIError)
		}

		// Check for specific error types
		if apiErr != nil {
			switch apiErr.ErrorCode() {
			case "AccessDeniedException":
				result.Success = false
				result.Message = fmt.Sprintf("Bedrock model access not enabled in region %s. Please enable model access in AWS Console: Bedrock → Model access", v.cfg.Bedrock.Region)
				result.Error = err
				return result
			case "ResourceNotFoundException":
				result.Success = false
				result.Message = fmt.Sprintf("Bedrock model %s not found in region %s", v.cfg.Bedrock.ModelID, v.cfg.Bedrock.Region)
				result.Error = err
				return result
			case "ValidationException":
				// This might be due to our test prompt format, but it means we have access
				result.Success = true
				result.Message = fmt.Sprintf("Bedrock model access enabled (model: %s, region: %s)", v.cfg.Bedrock.ModelID, v.cfg.Bedrock.Region)
				return result
			}
		}

		// For other errors, we can't be sure
		result.Success = false
		result.Message = "Failed to verify Bedrock access"
		result.Error = err
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Bedrock model access enabled (model: %s, region: %s)", v.cfg.Bedrock.ModelID, v.cfg.Bedrock.Region)
	return result
}

// validateWAFRPermissions checks if WAFR API permissions are available
func (v *Validator) validateWAFRPermissions(ctx context.Context) ValidationResult {
	result := ValidationResult{
		Name: "Well-Architected Tool Permissions",
	}

	// Load AWS config
	opts := []func(*config.LoadOptions) error{}

	region := v.cfg.AWS.Region
	if region == "" {
		region = v.cfg.Bedrock.Region
	}
	opts = append(opts, config.WithRegion(region))

	if v.cfg.AWS.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(v.cfg.AWS.Profile))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		result.Success = false
		result.Message = "Failed to load AWS config for WAFR"
		result.Error = err
		return result
	}

	// Create Well-Architected client
	client := wellarchitected.NewFromConfig(awsCfg)

	// Try to list workloads to check permissions
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = client.ListWorkloads(testCtx, &wellarchitected.ListWorkloadsInput{
		MaxResults: aws.Int32(1),
	})

	if err != nil {
		var apiErr smithy.APIError
		if err, ok := err.(*smithy.OperationError); ok {
			apiErr, _ = err.Err.(smithy.APIError)
		}

		if apiErr != nil {
			switch apiErr.ErrorCode() {
			case "AccessDeniedException":
				result.Success = false
				result.Message = "Insufficient permissions for Well-Architected Tool. Required: wellarchitected:ListWorkloads (and other wellarchitected:* permissions)"
				result.Error = err
				return result
			}
		}

		result.Success = false
		result.Message = "Failed to verify Well-Architected Tool permissions"
		result.Error = err
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Well-Architected Tool permissions verified (region: %s)", region)
	return result
}

// AllSuccess returns true if all validation results are successful
func AllSuccess(results []ValidationResult) bool {
	for _, r := range results {
		if !r.Success {
			return false
		}
	}
	return true
}
