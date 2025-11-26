# Configuration Package

This package provides configuration management for Waffle, including loading configuration from files and environment variables, and validating AWS setup.

## Configuration File

Waffle looks for a configuration file at `~/.waffle/config.yaml`. If the file doesn't exist, default values are used.

### Example Configuration

```yaml
bedrock:
  region: us-east-1
  model_id: us.anthropic.claude-sonnet-4-20250514-v1:0
  max_retries: 3
  timeout: 60
  max_tokens: 4096
  temperature: 0.7

storage:
  session_dir: ~/.waffle/sessions
  log_dir: ~/.waffle/logs
  retention_days: 90

iac:
  framework: terraform
  max_file_size_mb: 10
  max_files: 10000

wafr:
  default_scope: workload
  default_lens: wellarchitected

logging:
  level: INFO
  format: json

security:
  redact_sensitive_data: true
  encrypt_sessions: true

aws:
  profile: ""
  region: ""
```

## Command-Line Flags

Configuration values can be overridden using command-line flags:

- `--region`: AWS region for Bedrock and WAFR (overrides config file and environment variables)
- `--profile`: AWS profile to use (overrides config file and environment variables)

These flags are available on all commands as persistent flags.

### Examples

```bash
# Use a specific region
waffle init --region us-west-2

# Use a specific profile
waffle review --workload-id my-app --profile production

# Use both region and profile
waffle review --workload-id my-app --region eu-west-1 --profile prod-eu
```

## Environment Variables

Configuration values can be overridden using environment variables with the `WAFFLE_` prefix:

- `AWS_PROFILE`: AWS profile to use
- `AWS_REGION`: AWS region (overrides both `aws.region` and `bedrock.region`)
- `WAFFLE_LOG_LEVEL`: Log level (DEBUG, INFO, WARNING, ERROR)
- `WAFFLE_BEDROCK_REGION`: Bedrock region
- `WAFFLE_BEDROCK_MODEL_ID`: Bedrock model ID
- `WAFFLE_STORAGE_RETENTION_DAYS`: Session retention days

Environment variables use underscores to represent nested configuration keys. For example:
- `WAFFLE_BEDROCK_REGION` maps to `bedrock.region`
- `WAFFLE_STORAGE_SESSION_DIR` maps to `storage.session_dir`

## Usage

### Loading Configuration

```go
import "github.com/waffle/waffle/internal/config"

// Load configuration from file and environment
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}
```

### Saving Configuration

```go
// Create or modify configuration
cfg := config.DefaultConfig()
cfg.Bedrock.Region = "us-west-2"
cfg.AWS.Profile = "my-profile"

// Save to ~/.waffle/config.yaml
if err := config.Save(cfg); err != nil {
    log.Fatal(err)
}
```

### Validating AWS Setup

The validator checks:
1. AWS credentials are configured
2. Bedrock model access is enabled
3. Well-Architected Tool permissions are available

```go
import (
    "context"
    "github.com/waffle/waffle/internal/config"
)

// Load configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Create validator
validator := config.NewValidator(cfg)

// Run validation checks
ctx := context.Background()
results, err := validator.ValidateAll(ctx)
if err != nil {
    log.Fatal(err)
}

// Check results
for _, result := range results {
    if result.Success {
        fmt.Printf("✓ %s: %s\n", result.Name, result.Message)
    } else {
        fmt.Printf("✗ %s: %s\n", result.Name, result.Message)
        if result.Error != nil {
            fmt.Printf("  Error: %v\n", result.Error)
        }
    }
}
```

## Configuration Precedence

Configuration values are loaded in the following order (later values override earlier ones):

1. Default values (from `DefaultConfig()`)
2. Configuration file (`~/.waffle/config.yaml`)
3. Environment variables (`WAFFLE_*`, `AWS_PROFILE`, `AWS_REGION`)
4. Command-line flags (`--region`, `--profile`)

## Validation

The `Validate()` method checks that all required configuration values are present and valid:

- Bedrock region and model ID are required
- Retry counts, timeouts, and token limits must be positive
- Temperature must be between 0 and 1
- Storage directories must be specified
- Log level must be one of: DEBUG, INFO, WARNING, ERROR
- Log format must be one of: json, text

## AWS Setup Validation

The `waffle init` command uses the validator to check:

### 1. AWS Credentials
- Verifies that AWS credentials are configured
- Checks credential sources (environment, profile, IAM role)
- Reports which credential source is being used

### 2. Bedrock Model Access
- Verifies that the specified Bedrock model is accessible
- Checks if model access is enabled in the specified region
- Reports the model ID and region being used

### 3. Well-Architected Tool Permissions
- Verifies that the AWS account has permissions to use the Well-Architected Tool API
- Checks basic operations like `ListWorkloads`
- Reports the region where WAFR API is available

## Default Values

| Configuration | Default Value |
|--------------|---------------|
| `bedrock.region` | `us-east-1` |
| `bedrock.model_id` | `us.anthropic.claude-sonnet-4-20250514-v1:0` |
| `bedrock.max_retries` | `3` |
| `bedrock.timeout` | `60` |
| `bedrock.max_tokens` | `4096` |
| `bedrock.temperature` | `0.7` |
| `storage.session_dir` | `~/.waffle/sessions` |
| `storage.log_dir` | `~/.waffle/logs` |
| `storage.retention_days` | `90` |
| `iac.framework` | `terraform` |
| `iac.max_file_size_mb` | `10` |
| `iac.max_files` | `10000` |
| `wafr.default_scope` | `workload` |
| `wafr.default_lens` | `wellarchitected` |
| `logging.level` | `INFO` |
| `logging.format` | `json` |
| `security.redact_sensitive_data` | `true` |
| `security.encrypt_sessions` | `true` |
