# Bedrock Inference Profiles

## Overview

As of late 2024, AWS Bedrock requires using **inference profiles** instead of direct model IDs for Claude Sonnet 4 models. This change improves availability and routing across AWS regions.

## What Changed

### Old Format (No Longer Works)
```
anthropic.claude-sonnet-4-20250514-v1:0
```

### New Format (Required)
```
us.anthropic.claude-sonnet-4-20250514-v1:0
```

## Why Inference Profiles?

Inference profiles provide:
- **Cross-region routing**: Automatically routes to available regions
- **Better availability**: Reduces throttling and service unavailability
- **Consistent performance**: Load balancing across regions

## Configuration

Waffle uses the cross-region inference profile by default. You can customize it in `~/.waffle/config.yaml`:

```yaml
bedrock:
  region: us-east-1
  model_id: us.anthropic.claude-sonnet-4-20250514-v1:0
```

## Troubleshooting

### Error: "Invocation of model ID ... with on-demand throughput isn't supported"

This error means you're using the old direct model ID format. Solutions:

1. **Update your config** to use the inference profile format (starts with region prefix like `us.`)
2. **Delete old config** at `~/.waffle/config.yaml` to use the new defaults
3. **Rebuild** the CLI: `go build -o waffle ./cmd/waffle`

### Error: "prompt must start with \""

This error indicates the request format is incorrect. Claude Sonnet 4 requires the Messages API format:

```json
{
  "anthropic_version": "bedrock-2023-05-31",
  "max_tokens": 4096,
  "messages": [
    {
      "role": "user",
      "content": "Your prompt here"
    }
  ]
}
```

This is handled automatically by Waffle's Bedrock client. If you see this error, ensure you're using the latest version.

### Verify Model Access

Ensure you have enabled model access in AWS Bedrock console:
1. Go to AWS Bedrock console
2. Navigate to "Model access"
3. Enable "Claude Sonnet 4" model
4. Wait for access to be granted (usually instant)

### Required Permissions

Your AWS credentials need:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel"
      ],
      "Resource": "arn:aws:bedrock:*::foundation-model/*"
    }
  ]
}
```

## Available Inference Profiles

### Cross-Region Profiles (Recommended)
- `us.anthropic.claude-sonnet-4-20250514-v1:0` - US regions
- `eu.anthropic.claude-sonnet-4-20250514-v1:0` - EU regions

### Region-Specific Profiles
- `us-east-1.anthropic.claude-sonnet-4-20250514-v1:0`
- `us-west-2.anthropic.claude-sonnet-4-20250514-v1:0`
- `eu-west-1.anthropic.claude-sonnet-4-20250514-v1:0`

## References

- [AWS Bedrock Inference Profiles Documentation](https://docs.aws.amazon.com/bedrock/latest/userguide/inference-profiles.html)
- [Claude Models on Bedrock](https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html)
