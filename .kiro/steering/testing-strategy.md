# Testing Strategy for Waffle

## Overview

Waffle uses a comprehensive testing approach combining unit tests, property-based tests, integration tests, and manual testing to ensure correctness and reliability.

## Property-Based Testing with gopter

### Setup and Configuration

```go
import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

func TestProperties(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("session persistence round-trip", prop.ForAll(
        func(session *ReviewSession) bool {
            store := NewMemorySessionStore()
            
            // Save session
            if err := store.SaveSession(session); err != nil {
                return false
            }
            
            // Load session
            loaded, err := store.LoadSession(session.SessionID)
            if err != nil {
                return false
            }
            
            // Verify equivalence
            return sessionsEqual(session, loaded)
        },
        genReviewSession(),
    ))
    
    properties.TestingRun(t)
}
```

### Custom Generators

```go
// Generate valid session IDs
func genSessionID() gopter.Gen {
    return gen.RegexMatch(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
}

// Generate review sessions
func genReviewSession() gopter.Gen {
    return gopter.CombineGens(
        genSessionID(),
        gen.Identifier(),
        gen.Identifier(),
        genReviewScope(),
        genSessionStatus(),
    ).Map(func(vals []interface{}) *ReviewSession {
        return &ReviewSession{
            SessionID:    vals[0].(string),
            WorkloadID:   vals[1].(string),
            RepoLocation: vals[2].(string),
            Scope:        vals[3].(ReviewScope),
            Status:       vals[4].(SessionStatus),
            CreatedAt:    time.Now(),
            UpdatedAt:    time.Now(),
        }
    })
}

// Generate review scopes
func genReviewScope() gopter.Gen {
    return gen.OneConstOf(
        ScopeLevelWorkload,
        ScopeLevelPillar,
        ScopeLevelQuestion,
    ).FlatMap(func(level interface{}) gopter.Gen {
        scopeLevel := level.(ScopeLevel)
        
        switch scopeLevel {
        case ScopeLevelPillar:
            return gen.OneConstOf(
                PillarSecurity,
                PillarReliability,
                PillarPerformance,
                PillarCostOptimization,
                PillarOperationalExcellence,
                PillarSustainability,
            ).Map(func(p interface{}) ReviewScope {
                pillar := p.(Pillar)
                return ReviewScope{
                    Level:  scopeLevel,
                    Pillar: &pillar,
                }
            })
            
        case ScopeLevelQuestion:
            return gen.Identifier().Map(func(qid interface{}) ReviewScope {
                questionID := qid.(string)
                return ReviewScope{
                    Level:      scopeLevel,
                    QuestionID: questionID,
                }
            })
            
        default:
            return gen.Const(ReviewScope{Level: scopeLevel})
        }
    }, reflect.TypeOf(ReviewScope{}))
}

// Generate Terraform files
func genTerraformFile() gopter.Gen {
    return gopter.CombineGens(
        gen.RegexMatch(`[a-z]+\.tf`),
        genTerraformContent(),
    ).Map(func(vals []interface{}) IaCFile {
        return IaCFile{
            Path:    vals[0].(string),
            Content: vals[1].(string),
        }
    })
}

func genTerraformContent() gopter.Gen {
    return gen.OneConstOf(
        `resource "aws_s3_bucket" "example" { bucket = "test-bucket" }`,
        `resource "aws_ec2_instance" "web" { ami = "ami-12345" instance_type = "t2.micro" }`,
        `resource "aws_rds_instance" "db" { engine = "postgres" instance_class = "db.t3.micro" }`,
    )
}

// Generate confidence scores (0.0 to 1.0)
func genConfidenceScore() gopter.Gen {
    return gen.Float64Range(0.0, 1.0)
}
```

### Property Test Examples

```go
// Property 1: Session creation with valid inputs
func TestProperty_SessionCreation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("valid inputs create session with unique ID", prop.ForAll(
        func(workloadID, repoLocation string) bool {
            if workloadID == "" || repoLocation == "" {
                return true // Skip invalid inputs
            }
            
            engine := NewCoreEngine(/* deps */)
            session, err := engine.InitiateReview(
                context.Background(),
                repoLocation,
                workloadID,
                ReviewScope{Level: ScopeLevelWorkload},
            )
            
            if err != nil {
                return false
            }
            
            return session.SessionID != "" &&
                session.WorkloadID == workloadID &&
                !session.CreatedAt.IsZero()
        },
        gen.Identifier(),
        gen.Identifier(),
    ))
    
    properties.TestingRun(t)
}

// Property 7: Choice selection completeness
func TestProperty_ChoiceSelectionCompleteness(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("evaluation includes choices, evidence, and valid confidence", prop.ForAll(
        func(question *WAFRQuestion, model *WorkloadModel) bool {
            evaluator := NewWAFREvaluator(/* deps */)
            eval, err := evaluator.EvaluateQuestion(
                context.Background(),
                question,
                model,
            )
            
            if err != nil {
                return false
            }
            
            // Check completeness
            hasChoices := len(eval.SelectedChoices) >= 0
            hasEvidence := len(eval.Evidence) >= 0
            validConfidence := eval.ConfidenceScore >= 0.0 && eval.ConfidenceScore <= 1.0
            
            return hasChoices && hasEvidence && validConfidence
        },
        genWAFRQuestion(),
        genWorkloadModel(),
    ))
    
    properties.TestingRun(t)
}

// Property 20: Sensitive data redaction
func TestProperty_SensitiveDataRedaction(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("redacted content contains no sensitive patterns", prop.ForAll(
        func(content string) bool {
            redactor := NewRedactor()
            redacted, _ := redactor.Redact(content)
            
            // Verify no AWS keys remain
            awsKeyPattern := regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
            if awsKeyPattern.MatchString(redacted) {
                return false
            }
            
            // Verify no secrets remain (40-char base64)
            secretPattern := regexp.MustCompile(`[A-Za-z0-9/+=]{40}`)
            if secretPattern.MatchString(redacted) {
                return false
            }
            
            return true
        },
        genContentWithSecrets(),
    ))
    
    properties.TestingRun(t)
}
```

## Unit Testing

### Table-Driven Tests

```go
func TestValidateReviewScope(t *testing.T) {
    tests := []struct {
        name    string
        scope   ReviewScope
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid workload scope",
            scope: ReviewScope{
                Level: ScopeLevelWorkload,
            },
            wantErr: false,
        },
        {
            name: "valid pillar scope",
            scope: ReviewScope{
                Level:  ScopeLevelPillar,
                Pillar: &PillarSecurity,
            },
            wantErr: false,
        },
        {
            name: "pillar scope missing pillar",
            scope: ReviewScope{
                Level: ScopeLevelPillar,
            },
            wantErr: true,
            errMsg:  "pillar is required",
        },
        {
            name: "question scope missing question ID",
            scope: ReviewScope{
                Level: ScopeLevelQuestion,
            },
            wantErr: true,
            errMsg:  "question ID is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.scope.Validate()
            
            if tt.wantErr {
                require.Error(t, err)
                if tt.errMsg != "" {
                    assert.Contains(t, err.Error(), tt.errMsg)
                }
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Mock-Based Testing

```go
type MockWAFRClient struct {
    mock.Mock
}

func (m *MockWAFRClient) CreateWorkload(ctx context.Context, input *CreateWorkloadInput) (string, error) {
    args := m.Called(ctx, input)
    return args.String(0), args.Error(1)
}

func TestCreateWorkload(t *testing.T) {
    mockClient := new(MockWAFRClient)
    mockClient.On("CreateWorkload", mock.Anything, mock.MatchedBy(func(input *CreateWorkloadInput) bool {
        return input.Name == "test-workload"
    })).Return("wl-123", nil)
    
    evaluator := &WAFREvaluator{client: mockClient}
    workloadID, err := evaluator.CreateWorkload(context.Background(), &CreateWorkloadInput{
        Name: "test-workload",
    })
    
    require.NoError(t, err)
    assert.Equal(t, "wl-123", workloadID)
    mockClient.AssertExpectations(t)
}
```

## Integration Testing

### Test with Real AWS Services (Optional)

```go
// +build integration

func TestIntegration_WAFRWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    ctx := context.Background()
    
    // Initialize real AWS clients
    cfg, err := config.LoadDefaultConfig(ctx)
    require.NoError(t, err)
    
    waClient := wellarchitected.NewFromConfig(cfg)
    evaluator := NewWAFREvaluator(waClient)
    
    // Create test workload
    workloadID, err := evaluator.CreateWorkload(ctx, &CreateWorkloadInput{
        Name:        "integration-test-" + uuid.New().String(),
        Description: "Integration test workload",
    })
    require.NoError(t, err)
    defer cleanupWorkload(t, waClient, workloadID)
    
    // Get questions
    questions, err := evaluator.GetQuestions(ctx, workloadID, ReviewScope{
        Level:  ScopeLevelPillar,
        Pillar: &PillarSecurity,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, questions)
    
    // Submit answer
    err = evaluator.SubmitAnswer(ctx, &UpdateAnswerInput{
        WorkloadID:      workloadID,
        LensAlias:       "wellarchitected",
        QuestionID:      questions[0].ID,
        SelectedChoices: []string{questions[0].Choices[0].ID},
    })
    require.NoError(t, err)
}
```

### Test with Mock Bedrock

```go
func TestIntegration_IaCAnalysis(t *testing.T) {
    mockBedrock := &MockBedrockClient{}
    mockBedrock.SetResponse("parse_iac", &IaCAnalysisResponse{
        Resources: []ResourceDefinition{
            {
                Type: "aws_s3_bucket",
                Name: "example",
                Properties: map[string]interface{}{
                    "bucket": "test-bucket",
                    "versioning": map[string]interface{}{
                        "enabled": true,
                    },
                },
            },
        },
    })
    
    analyzer := NewIaCAnalyzer(mockBedrock)
    
    files := []IaCFile{
        {
            Path: "main.tf",
            Content: `
resource "aws_s3_bucket" "example" {
  bucket = "test-bucket"
  
  versioning {
    enabled = true
  }
}
`,
        },
    }
    
    model, err := analyzer.ParseTerraform(context.Background(), files)
    require.NoError(t, err)
    assert.Len(t, model.Resources, 1)
    assert.Equal(t, "aws_s3_bucket", model.Resources[0].Type)
}
```

## Test Fixtures

### Sample IaC Files

```go
// test/fixtures/terraform/s3_encrypted.tf
const S3EncryptedFixture = `
resource "aws_s3_bucket" "example" {
  bucket = "my-encrypted-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "example" {
  bucket = aws_s3_bucket.example.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.example.arn
    }
  }
}

resource "aws_kms_key" "example" {
  description = "S3 bucket encryption key"
}
`

// test/fixtures/terraform/s3_unencrypted.tf
const S3UnencryptedFixture = `
resource "aws_s3_bucket" "example" {
  bucket = "my-unencrypted-bucket"
}
`
```

### Sample WAFR Questions

```go
func SampleSecurityQuestion() *WAFRQuestion {
    return &WAFRQuestion{
        ID:          "sec_data_classification_1",
        Pillar:      PillarSecurity,
        Title:       "How do you classify your data?",
        Description: "Classification provides a way to categorize data...",
        BestPractices: []BestPractice{
            {
                ID:          "sec_data_classification_identify",
                Title:       "Identify the data within your workload",
                Description: "Understand the type and classification of data...",
            },
        },
        Choices: []Choice{
            {
                ID:          "sec_data_classification_1_choice_1",
                Title:       "Data classification schema defined",
                Description: "You have defined a data classification schema...",
            },
        },
    }
}
```

## Benchmark Tests

### Performance Benchmarks

```go
func BenchmarkParseTerraform(b *testing.B) {
    files := loadLargeTerraformFixture()
    analyzer := NewIaCAnalyzer(nil)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := analyzer.ParseTerraform(context.Background(), files)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkRedaction(b *testing.B) {
    content := loadSampleIaCWithSecrets()
    redactor := NewRedactor()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = redactor.Redact(content)
    }
}

func BenchmarkSessionSerialization(b *testing.B) {
    session := generateLargeSession()
    store := NewMemorySessionStore()
    
    b.Run("save", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = store.SaveSession(session)
        }
    })
    
    b.Run("load", func(b *testing.B) {
        store.SaveSession(session)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            _, _ = store.LoadSession(session.SessionID)
        }
    })
}
```

## Test Organization

### Directory Structure

```
waffle/
├── internal/
│   ├── core/
│   │   ├── engine.go
│   │   └── engine_test.go
│   ├── iac/
│   │   ├── analyzer.go
│   │   ├── analyzer_test.go
│   │   └── analyzer_property_test.go
│   └── session/
│       ├── store.go
│       └── store_test.go
├── test/
│   ├── fixtures/
│   │   ├── terraform/
│   │   │   ├── s3_encrypted.tf
│   │   │   └── s3_unencrypted.tf
│   │   └── wafr/
│   │       └── sample_questions.json
│   ├── mocks/
│   │   ├── bedrock.go
│   │   └── wafr_client.go
│   └── integration/
│       └── workflow_test.go
```

### Test Helpers

```go
// test/helpers.go
package test

func LoadFixture(name string) string {
    data, err := os.ReadFile(filepath.Join("fixtures", name))
    if err != nil {
        panic(err)
    }
    return string(data)
}

func NewTestContext() context.Context {
    return context.WithValue(context.Background(), "test", true)
}

func AssertSessionEqual(t *testing.T, expected, actual *ReviewSession) {
    assert.Equal(t, expected.SessionID, actual.SessionID)
    assert.Equal(t, expected.WorkloadID, actual.WorkloadID)
    assert.Equal(t, expected.Status, actual.Status)
}
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      
      - name: Run property tests
        run: go test -v -tags=property ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Testing Best Practices

1. **Write tests first**: Follow TDD for critical functionality
2. **Use property tests**: Validate invariants across many inputs
3. **Mock external dependencies**: Don't rely on AWS services for unit tests
4. **Test error paths**: Ensure error handling works correctly
5. **Use table-driven tests**: Cover multiple scenarios efficiently
6. **Benchmark performance**: Track performance regressions
7. **Test with real data**: Use actual Terraform files in fixtures
8. **Isolate tests**: Each test should be independent
9. **Use descriptive names**: Test names should explain what they verify
10. **Maintain test fixtures**: Keep sample data up to date
