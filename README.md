# Waffle - Well Architected Framework for Less Effort

Waffle automates AWS Well-Architected Framework Reviews by analyzing infrastructure-as-code repositories using Amazon Bedrock foundation models via direct API invocation.

## Project Structure

```
waffle/
├── cmd/
│   └── waffle/           # CLI entry point
│       └── main.go
├── internal/             # Private application code
│   ├── core/            # Core engine and types
│   │   ├── types.go     # Core data types
│   │   ├── errors.go    # Error definitions
│   │   ├── interfaces.go # Component interfaces
│   │   ├── engine.go    # Core orchestration engine
│   │   ├── types_test.go
│   │   └── properties_test.go
│   ├── iac/             # IaC analyzer
│   │   └── analyzer.go
│   ├── session/         # Session manager
│   │   └── manager.go
│   ├── wafr/            # WAFR evaluator
│   │   └── evaluator.go
│   ├── bedrock/         # Bedrock client
│   │   └── client.go
│   └── report/          # Report generator
│       └── generator.go
├── test/                # Additional test data and helpers
│   ├── fixtures/        # Test IaC files
│   └── mocks/           # Mock implementations
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- **Go 1.21+**
- **AWS SDK for Go v2**
  - `github.com/aws/aws-sdk-go-v2` - Core SDK
  - `github.com/aws/aws-sdk-go-v2/service/wellarchitected` - WAFR API client
  - `github.com/aws/aws-sdk-go-v2/service/bedrockruntime` - Bedrock Runtime API client
- **CLI Framework**
  - `github.com/spf13/cobra` - Command-line interface
  - `github.com/spf13/viper` - Configuration management
- **Testing**
  - `github.com/leanovate/gopter` - Property-based testing
  - `github.com/stretchr/testify` - Test assertions
- **Utilities**
  - `golang.org/x/sync/errgroup` - Concurrent error handling
  - `golang.org/x/time/rate` - Rate limiting

## Installation

### Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/waffle/waffle/releases):

```bash
# Linux (amd64)
wget https://github.com/waffle/waffle/releases/download/v1.0.0/waffle-v1.0.0-linux-amd64.tar.gz
tar -xzf waffle-v1.0.0-linux-amd64.tar.gz
sudo mv waffle-v1.0.0-linux-amd64 /usr/local/bin/waffle

# macOS (Apple Silicon)
curl -LO https://github.com/waffle/waffle/releases/download/v1.0.0/waffle-v1.0.0-darwin-arm64.tar.gz
tar -xzf waffle-v1.0.0-darwin-arm64.tar.gz
sudo mv waffle-v1.0.0-darwin-arm64 /usr/local/bin/waffle
```

### Using Go Install

If you have Go 1.21+ installed:

```bash
go install github.com/waffle/waffle/cmd/waffle@latest
```

This installs the binary to `$GOPATH/bin` (or `~/go/bin` by default). Make sure this directory is in your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc if needed
export PATH=$PATH:$(go env GOPATH)/bin
```

### Using Docker

```bash
docker pull ghcr.io/waffle/waffle:latest

# Run with AWS credentials
docker run --rm -it \
  -v ~/.aws:/root/.aws:ro \
  -v $(pwd):/workspace \
  -w /workspace \
  ghcr.io/waffle/waffle:latest \
  review --workload-id my-app
```

### Build from Source

```bash
git clone https://github.com/waffle/waffle.git
cd waffle
make build
sudo mv build/waffle /usr/local/bin/
```

## Building

### Quick Build

```bash
make build          # Build for current platform
make test           # Run tests
make docker-build   # Build Docker image
```

### Cross-Platform Builds

```bash
make build-all      # Build for all platforms
make release        # Create release archives with checksums
```

### Development

```bash
make dev            # Quick dev build and version check
make test-coverage  # Run tests with coverage report
make lint           # Run linter (requires golangci-lint)
```

Run `make help` to see all available targets.

## Usage

### Configuration

Waffle can be configured through:
1. Configuration file at `~/.waffle/config.yaml`
2. Environment variables (prefixed with `WAFFLE_`)
3. Command-line flags (highest precedence)

See `config.example.yaml` for a complete configuration example.

### Global Flags

All commands support these global flags:

- `--region`: AWS region for Bedrock and WAFR (overrides config and environment)
- `--profile`: AWS profile to use (overrides config and environment)

### Commands

#### Initialize and Validate Setup

```bash
# Validate with default configuration
waffle init

# Validate with specific region
waffle init --region us-west-2

# Validate with specific profile and region
waffle init --profile my-profile --region eu-west-1
```

#### Run a WAFR Review

```bash
# Review entire workload
waffle review --workload-id my-app

# Review with Terraform plan for complete analysis
waffle review --workload-id my-app --plan-file plan.json

# Review with specific AWS region
waffle review --workload-id my-app --region us-west-2

# Review specific pillar
waffle review --workload-id my-app --scope pillar --pillar security

# Review specific question
waffle review --workload-id my-app --scope question --question-id sec_data_1
```

#### Check Review Status

```bash
waffle status <session-id>
```

#### Get Review Results

```bash
# Get results as JSON to stdout
waffle results <session-id>

# Get results as JSON to file
waffle results <session-id> --format json --output results.json

# Get results as PDF
waffle results <session-id> --format pdf --output report.pdf
```

#### Compare Milestones

```bash
waffle compare <session-id-1> <session-id-2>
```

## Contributing

### Prerequisites

- Go 1.21 or later
- AWS credentials configured
- Make (optional but recommended)

### Development Workflow

```bash
# Clone the repository
git clone https://github.com/waffle/waffle.git
cd waffle

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run locally
./build/waffle --version
```

### Running Tests

```bash
make test                # All tests
make test-coverage       # With coverage report
make test-property       # Property-based tests only
```

### Code Quality

```bash
make fmt                 # Format code
make vet                 # Run go vet
make lint                # Run golangci-lint
```

## Development Status

This project is currently under development. The core interfaces and project structure have been established. Implementation of individual components is in progress according to the task list in `.kiro/specs/waffle-automated-wafr/tasks.md`.

## Architecture

Waffle follows a modular architecture with clear separation of concerns:

- **Core Engine**: Orchestrates the review workflow
- **IaC Analyzer**: Parses Terraform configurations and plans
- **Session Manager**: Manages review session lifecycle
- **WAFR Evaluator**: Interfaces with AWS Well-Architected Tool API
- **Bedrock Client**: Invokes foundation models for semantic analysis
- **Report Generator**: Formats and exports review results

## Testing Strategy

The project uses a comprehensive testing approach:

- **Unit Tests**: Test individual functions and components
- **Property-Based Tests**: Verify invariants across many inputs using gopter
- **Integration Tests**: Test component interactions
- **Mock-Based Tests**: Isolate components for testing

## License

TBD
