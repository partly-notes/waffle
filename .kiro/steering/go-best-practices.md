# Go Best Practices for Waffle

## Documentation Guidelines

### Do Not Create Summary Documents

Do not create summary documents or recap files after completing work. This includes:
- Files like `SUMMARY.md`, `CHANGES.md`, `FIX_SUMMARY.md`
- Verbose recaps of what was accomplished
- Duplicate documentation of changes

Only create documentation that serves a specific purpose:
- User-facing guides (setup, configuration, troubleshooting)
- API documentation
- Architecture decisions that need to be preserved

## Error Handling

### Always Wrap Errors with Context

Use `fmt.Errorf` with `%w` to wrap errors and preserve the error chain:

```go
if err != nil {
    return fmt.Errorf("failed to parse terraform file %s: %w", filename, err)
}
```

### Create Custom Error Types for Domain Logic

```go
type RepositoryAccessError struct {
    Path string
    Err  error
}

func (e *RepositoryAccessError) Error() string {
    return fmt.Sprintf("cannot access repository at %s: %v", e.Path, e.Err)
}

func (e *RepositoryAccessError) Unwrap() error {
    return e.Err
}
```

### Use errors.Is and errors.As for Error Checking

```go
if errors.Is(err, os.ErrNotExist) {
    return &RepositoryAccessError{Path: repoPath, Err: err}
}

var repoErr *RepositoryAccessError
if errors.As(err, &repoErr) {
    // Handle repository-specific error
}
```

## Interface Design

### Keep Interfaces Small and Focused

Define interfaces at the point of use, not at the point of implementation:

```go
// Good: Small, focused interface
type SessionStore interface {
    SaveSession(session *ReviewSession) error
    LoadSession(sessionID string) (*ReviewSession, error)
}

// Avoid: Large, monolithic interfaces
```

### Accept Interfaces, Return Structs

```go
// Good
func NewCoreEngine(store SessionStore, analyzer IaCAnalyzer) *CoreEngine {
    return &CoreEngine{
        store:    store,
        analyzer: analyzer,
    }
}

// Avoid returning interfaces unless necessary for testing
```

### Use Dependency Injection

```go
type CoreEngine struct {
    store      SessionStore
    analyzer   IaCAnalyzer
    evaluator  WAFREvaluator
    bedrock    BedrockClient
}

func NewCoreEngine(
    store SessionStore,
    analyzer IaCAnalyzer,
    evaluator WAFREvaluator,
    bedrock BedrockClient,
) *CoreEngine {
    return &CoreEngine{
        store:     store,
        analyzer:  analyzer,
        evaluator: evaluator,
        bedrock:   bedrock,
    }
}
```

## Project Structure

### Standard Layout

```
waffle/
├── cmd/
│   └── waffle/           # CLI entry point
│       └── main.go
├── internal/             # Private application code
│   ├── core/            # Core engine
│   ├── iac/             # IaC analyzer
│   ├── session/         # Session manager
│   ├── wafr/            # WAFR evaluator
│   ├── bedrock/         # Bedrock client
│   └── report/          # Report generator
├── pkg/                 # Public libraries (if any)
├── test/                # Additional test data and helpers
│   ├── fixtures/        # Test IaC files
│   └── mocks/           # Mock implementations
├── go.mod
└── go.sum
```

### Package Naming

- Use short, lowercase package names
- Avoid package names like `util`, `common`, `helpers`
- Name packages after what they provide, not what they contain

## Testing Patterns

### Table-Driven Tests

```go
func TestValidateTerraformFiles(t *testing.T) {
    tests := []struct {
        name    string
        files   []IaCFile
        wantErr bool
        errType error
    }{
        {
            name: "valid terraform files",
            files: []IaCFile{
                {Path: "main.tf", Content: "resource \"aws_s3_bucket\" \"example\" {}"},
            },
            wantErr: false,
        },
        {
            name: "invalid syntax",
            files: []IaCFile{
                {Path: "main.tf", Content: "resource {{{"},
            },
            wantErr: true,
            errType: &TerraformSyntaxError{},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateTerraformFiles(tt.files)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateTerraformFiles() error = %v, wantErr %v", err, tt.wantErr)
            }
            if tt.errType != nil && !errors.As(err, &tt.errType) {
                t.Errorf("expected error type %T, got %T", tt.errType, err)
            }
        })
    }
}
```

### Use testify for Assertions

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
    store := NewMemorySessionStore()
    session, err := store.CreateSession("workload-1", "/path/to/repo", ScopeWorkload)
    
    require.NoError(t, err, "CreateSession should not return error")
    assert.NotEmpty(t, session.SessionID, "SessionID should be generated")
    assert.Equal(t, "workload-1", session.WorkloadID)
    assert.NotZero(t, session.CreatedAt, "CreatedAt should be set")
}
```

### Mock Interfaces for Testing

```go
type MockBedrockClient struct {
    InvokeAgentFunc func(ctx context.Context, input *InvokeAgentInput) (*InvokeAgentOutput, error)
}

func (m *MockBedrockClient) InvokeAgent(ctx context.Context, input *InvokeAgentInput) (*InvokeAgentOutput, error) {
    if m.InvokeAgentFunc != nil {
        return m.InvokeAgentFunc(ctx, input)
    }
    return nil, nil
}

func TestEvaluateQuestion(t *testing.T) {
    mockBedrock := &MockBedrockClient{
        InvokeAgentFunc: func(ctx context.Context, input *InvokeAgentInput) (*InvokeAgentOutput, error) {
            return &InvokeAgentOutput{
                Choices: []string{"choice_1", "choice_2"},
                Evidence: []string{"S3 bucket has encryption enabled"},
            }, nil
        },
    }
    
    evaluator := NewWAFREvaluator(mockBedrock)
    result, err := evaluator.EvaluateQuestion(ctx, question, workloadModel)
    
    require.NoError(t, err)
    assert.Len(t, result.SelectedChoices, 2)
}
```

## Context Usage

### Always Pass Context as First Parameter

```go
func (e *CoreEngine) ExecuteReview(ctx context.Context, session *ReviewSession) (*ReviewResults, error) {
    // Use ctx for cancellation and timeouts
}
```

### Respect Context Cancellation

```go
func (a *IaCAnalyzer) ParseTerraform(ctx context.Context, files []IaCFile) (*WorkloadModel, error) {
    for _, file := range files {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
            // Continue processing
        }
        
        // Parse file...
    }
    return model, nil
}
```

### Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

result, err := evaluator.EvaluateQuestion(ctx, question, model)
```

## Concurrency Patterns

### Use errgroup for Parallel Operations

```go
import "golang.org/x/sync/errgroup"

func (e *WAFREvaluator) EvaluateQuestions(ctx context.Context, questions []*WAFRQuestion, model *WorkloadModel) ([]*QuestionEvaluation, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]*QuestionEvaluation, len(questions))
    
    for i, q := range questions {
        i, q := i, q // Capture loop variables
        g.Go(func() error {
            eval, err := e.EvaluateQuestion(ctx, q, model)
            if err != nil {
                return fmt.Errorf("failed to evaluate question %s: %w", q.ID, err)
            }
            results[i] = eval
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    return results, nil
}
```

### Use Channels for Pipeline Patterns

```go
func (a *IaCAnalyzer) ParseFilesAsync(ctx context.Context, files []IaCFile) <-chan ParseResult {
    results := make(chan ParseResult)
    
    go func() {
        defer close(results)
        for _, file := range files {
            select {
            case <-ctx.Done():
                return
            case results <- a.parseFile(file):
            }
        }
    }()
    
    return results
}
```

## Logging

### Use Structured Logging

```go
import "log/slog"

func (e *CoreEngine) ExecuteReview(ctx context.Context, session *ReviewSession) (*ReviewResults, error) {
    slog.InfoContext(ctx, "starting review execution",
        "session_id", session.SessionID,
        "workload_id", session.WorkloadID,
        "scope", session.Scope.Level,
    )
    
    // ... execution logic
    
    if err != nil {
        slog.ErrorContext(ctx, "review execution failed",
            "session_id", session.SessionID,
            "error", err,
        )
        return nil, err
    }
    
    slog.InfoContext(ctx, "review execution completed",
        "session_id", session.SessionID,
        "questions_evaluated", len(results.Evaluations),
        "risks_identified", len(results.Risks),
    )
    
    return results, nil
}
```

### Add Correlation IDs

```go
type contextKey string

const correlationIDKey contextKey = "correlation_id"

func WithCorrelationID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, correlationIDKey, id)
}

func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(correlationIDKey).(string); ok {
        return id
    }
    return ""
}

// Use in logging
slog.InfoContext(ctx, "processing request",
    "correlation_id", GetCorrelationID(ctx),
)
```

## Configuration Management

### Use Viper for Configuration

```go
import "github.com/spf13/viper"

type Config struct {
    Bedrock BedrockConfig
    Storage StorageConfig
    IaC     IaCConfig
    WAFR    WAFRConfig
    Logging LoggingConfig
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("$HOME/.waffle")
    viper.AddConfigPath(".")
    
    // Set defaults
    viper.SetDefault("bedrock.region", "us-east-1")
    viper.SetDefault("bedrock.max_retries", 3)
    viper.SetDefault("storage.retention_days", 90)
    
    // Environment variables
    viper.SetEnvPrefix("WAFFLE")
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
        // Config file not found; use defaults
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    return &config, nil
}
```

## Resource Management

### Always Use defer for Cleanup

```go
func (s *SessionStore) SaveSession(session *ReviewSession) error {
    f, err := os.Create(s.sessionPath(session.SessionID))
    if err != nil {
        return fmt.Errorf("failed to create session file: %w", err)
    }
    defer f.Close()
    
    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(session); err != nil {
        return fmt.Errorf("failed to encode session: %w", err)
    }
    
    return nil
}
```

### Use sync.Pool for Frequently Allocated Objects

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processLargeFile(file string) error {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // Use buffer...
    return nil
}
```

## Code Organization

### Group Related Functionality

```go
// Good: Related methods grouped together
type SessionStore struct {
    baseDir string
}

func (s *SessionStore) CreateSession(...) (*ReviewSession, error) { }
func (s *SessionStore) SaveSession(...) error { }
func (s *SessionStore) LoadSession(...) (*ReviewSession, error) { }
func (s *SessionStore) UpdateSessionStatus(...) error { }
func (s *SessionStore) ListSessions(...) ([]*ReviewSession, error) { }
```

### Use Constructor Functions

```go
func NewSessionStore(baseDir string) (*SessionStore, error) {
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create session directory: %w", err)
    }
    
    return &SessionStore{
        baseDir: baseDir,
    }, nil
}
```

### Validate Inputs Early

```go
func (e *CoreEngine) InitiateReview(ctx context.Context, repoLocation, workloadID string, scope ReviewScope) (*ReviewSession, error) {
    // Validate inputs first
    if repoLocation == "" {
        return nil, errors.New("repository location is required")
    }
    if workloadID == "" {
        return nil, errors.New("workload ID is required")
    }
    if err := scope.Validate(); err != nil {
        return nil, fmt.Errorf("invalid scope: %w", err)
    }
    
    // Proceed with logic
    // ...
}
```

## Performance Considerations

### Avoid Premature Optimization

Focus on correctness first, then optimize based on profiling data.

### Use Benchmarks for Performance-Critical Code

```go
func BenchmarkParseTerraform(b *testing.B) {
    files := loadTestFiles()
    analyzer := NewIaCAnalyzer()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := analyzer.ParseTerraform(context.Background(), files)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Profile When Needed

```go
import _ "net/http/pprof"

// In main.go for development builds
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## Documentation

### Write Package Documentation

```go
// Package iac provides infrastructure-as-code analysis capabilities.
// It supports parsing Terraform configurations and extracting resource
// definitions, relationships, and metadata for WAFR evaluation.
package iac
```

### Document Exported Functions

```go
// ParseTerraform parses Terraform HCL files and extracts resource definitions.
// It returns a WorkloadModel containing all resources and their relationships.
//
// The parser handles:
//   - Resource blocks (resource, data, module)
//   - Variable and output declarations
//   - Resource dependencies and references
//
// Returns an error if the Terraform syntax is invalid or if required
// resources cannot be extracted.
func ParseTerraform(ctx context.Context, files []IaCFile) (*WorkloadModel, error) {
    // ...
}
```

### Use Examples in Tests

```go
func ExampleIaCAnalyzer_ParseTerraform() {
    analyzer := NewIaCAnalyzer()
    files := []IaCFile{
        {Path: "main.tf", Content: `resource "aws_s3_bucket" "example" { bucket = "my-bucket" }`},
    }
    
    model, err := analyzer.ParseTerraform(context.Background(), files)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(len(model.Resources))
    // Output: 1
}
```
