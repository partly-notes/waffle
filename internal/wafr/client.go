package wafr

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
)

// ClientConfig holds configuration for AWS client initialization
type ClientConfig struct {
	Region  string
	Profile string
}

// NewWAFRClient creates a new AWS Well-Architected Tool client
func NewWAFRClient(ctx context.Context, cfg *ClientConfig) (*wellarchitected.Client, error) {
	if cfg == nil {
		cfg = &ClientConfig{
			Region: "us-east-1",
		}
	}

	configOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.Profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(cfg.Profile))
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := wellarchitected.NewFromConfig(awsConfig)
	return client, nil
}

// NewEvaluatorWithConfig creates a new WAFR evaluator with AWS client configuration
func NewEvaluatorWithConfig(ctx context.Context, clientCfg *ClientConfig, evalCfg *EvaluatorConfig) (*Evaluator, error) {
	client, err := NewWAFRClient(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAFR client: %w", err)
	}

	return NewEvaluator(client, evalCfg), nil
}
