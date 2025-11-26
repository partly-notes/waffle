package wafr

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffle/waffle/internal/core"
)

// MockWAFRClient implements the WAFRClient interface for testing
type MockWAFRClient struct {
	CreateWorkloadFunc         func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error)
	GetWorkloadFunc            func(ctx context.Context, params *wellarchitected.GetWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error)
	ListAnswersFunc            func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error)
	UpdateAnswerFunc           func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error)
	CreateMilestoneFunc        func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error)
	GetConsolidatedReportFunc  func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error)
	GetMilestoneFunc           func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
}

// MockBedrockClient implements the BedrockClient interface for testing
type MockBedrockClient struct {
	EvaluateWAFRQuestionFunc        func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error)
	GenerateImprovementGuidanceFunc func(ctx context.Context, risk *core.Risk, resources []core.Resource) (*core.ImprovementPlanItem, error)
}

func (m *MockBedrockClient) EvaluateWAFRQuestion(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
	if m.EvaluateWAFRQuestionFunc != nil {
		return m.EvaluateWAFRQuestionFunc(ctx, question, workloadModel)
	}
	return &core.QuestionEvaluation{
		Question:        question,
		SelectedChoices: []core.Choice{},
		Evidence:        []core.Evidence{},
		ConfidenceScore: 0.8,
		Notes:           "Mock evaluation",
	}, nil
}

func (m *MockBedrockClient) GenerateImprovementGuidance(ctx context.Context, risk *core.Risk, resources []core.Resource) (*core.ImprovementPlanItem, error) {
	if m.GenerateImprovementGuidanceFunc != nil {
		return m.GenerateImprovementGuidanceFunc(ctx, risk, resources)
	}
	return &core.ImprovementPlanItem{
		ID:          "test-improvement",
		Risk:        risk,
		Description: "Mock improvement",
	}, nil
}

func (m *MockWAFRClient) CreateWorkload(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
	if m.CreateWorkloadFunc != nil {
		return m.CreateWorkloadFunc(ctx, params, optFns...)
	}
	return &wellarchitected.CreateWorkloadOutput{
		WorkloadId: aws.String("test-workload-id"),
	}, nil
}

func (m *MockWAFRClient) GetWorkload(ctx context.Context, params *wellarchitected.GetWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error) {
	if m.GetWorkloadFunc != nil {
		return m.GetWorkloadFunc(ctx, params, optFns...)
	}
	return nil, &types.ResourceNotFoundException{
		Message: aws.String("workload not found"),
	}
}

func (m *MockWAFRClient) ListAnswers(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
	if m.ListAnswersFunc != nil {
		return m.ListAnswersFunc(ctx, params, optFns...)
	}
	return &wellarchitected.ListAnswersOutput{
		AnswerSummaries: []types.AnswerSummary{},
	}, nil
}

func (m *MockWAFRClient) UpdateAnswer(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
	if m.UpdateAnswerFunc != nil {
		return m.UpdateAnswerFunc(ctx, params, optFns...)
	}
	return &wellarchitected.UpdateAnswerOutput{}, nil
}

func (m *MockWAFRClient) CreateMilestone(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
	if m.CreateMilestoneFunc != nil {
		return m.CreateMilestoneFunc(ctx, params, optFns...)
	}
	return &wellarchitected.CreateMilestoneOutput{
		MilestoneNumber: aws.Int32(1),
	}, nil
}

func (m *MockWAFRClient) GetConsolidatedReport(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
	if m.GetConsolidatedReportFunc != nil {
		return m.GetConsolidatedReportFunc(ctx, params, optFns...)
	}
	return &wellarchitected.GetConsolidatedReportOutput{
		Base64String: aws.String("dGVzdC1kYXRh"), // "test-data" in base64
	}, nil
}

func (m *MockWAFRClient) GetMilestone(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
	if m.GetMilestoneFunc != nil {
		return m.GetMilestoneFunc(ctx, params, optFns...)
	}
	return &wellarchitected.GetMilestoneOutput{
		Milestone: &types.Milestone{
			MilestoneNumber: params.MilestoneNumber,
			MilestoneName:   aws.String("test-milestone"),
			Workload: &types.Workload{
				RiskCounts: map[string]int32{
					string(types.RiskHigh):   0,
					string(types.RiskMedium): 0,
				},
			},
		},
	}, nil
}

// APIError implements smithy.APIError for testing
type APIError struct {
	code    string
	message string
}

func (e *APIError) Error() string {
	return e.message
}

func (e *APIError) ErrorCode() string {
	return e.code
}

func (e *APIError) ErrorMessage() string {
	return e.message
}

func (e *APIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultUnknown
}

func TestCreateWorkload(t *testing.T) {
	tests := []struct {
		name       string
		workloadID string
		desc       string
		mockFunc   func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error)
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "successful creation",
			workloadID: "test-workload",
			desc:       "test description",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
				return &wellarchitected.CreateWorkloadOutput{
					WorkloadId: aws.String("wl-123"),
				}, nil
			},
			wantErr: false,
		},
		{
			name:       "empty workload ID",
			workloadID: "",
			desc:       "test description",
			wantErr:    true,
			wantErrMsg: "workload ID is required",
		},
		{
			name:       "access denied",
			workloadID: "test-workload",
			desc:       "test description",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
				return nil, &types.AccessDeniedException{
					Message: aws.String("insufficient permissions"),
				}
			},
			wantErr:    true,
			wantErrMsg: "WAFR CreateWorkload failed",
		},
		{
			name:       "throttling with retry",
			workloadID: "test-workload",
			desc:       "test description",
			mockFunc: func() func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
				attempts := 0
				return func(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
					attempts++
					if attempts < 2 {
						return nil, &APIError{code: "ThrottlingException", message: "rate exceeded"}
					}
					return &wellarchitected.CreateWorkloadOutput{
						WorkloadId: aws.String("wl-123"),
					}, nil
				}
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				CreateWorkloadFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			awsWorkloadID, err := evaluator.CreateWorkload(context.Background(), tt.workloadID, tt.desc)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, awsWorkloadID)
			}
		})
	}
}

func TestGetQuestions(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		scope         core.ReviewScope
		mockFunc      func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error)
		wantErr       bool
		wantErrMsg    string
		wantCount     int
	}{
		{
			name:          "workload scope - all pillars",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level: core.ScopeLevelWorkload,
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String("q1"),
							QuestionTitle: aws.String("Question 1"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 6, // 6 pillars, 1 question each
		},
		{
			name:          "pillar scope",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level:  core.ScopeLevelPillar,
				Pillar: func() *core.Pillar { p := core.PillarSecurity; return &p }(),
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String("sec-1"),
							QuestionTitle: aws.String("Security Question 1"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			scope: core.ReviewScope{
				Level: core.ScopeLevelWorkload,
			},
			wantErr:    true,
			wantErrMsg: "AWS workload ID is required",
		},
		{
			name:          "invalid scope - pillar missing",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level: core.ScopeLevelPillar,
			},
			wantErr:    true,
			wantErrMsg: "invalid scope",
		},
		{
			name:          "question scope",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level:      core.ScopeLevelQuestion,
				QuestionID: "sec-1",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String("sec-1"),
							QuestionTitle: aws.String("Security Question 1"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:          "invalid scope - question ID missing",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level: core.ScopeLevelQuestion,
			},
			wantErr:    true,
			wantErrMsg: "invalid scope",
		},
		{
			name:          "pagination handling",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level:  core.ScopeLevelPillar,
				Pillar: func() *core.Pillar { p := core.PillarSecurity; return &p }(),
			},
			mockFunc: func() func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				callCount := 0
				return func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
					callCount++
					if callCount == 1 {
						return &wellarchitected.ListAnswersOutput{
							AnswerSummaries: []types.AnswerSummary{
								{
									QuestionId:    aws.String("sec-1"),
									QuestionTitle: aws.String("Security Question 1"),
									Choices: []types.Choice{
										{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
									},
								},
							},
							NextToken: aws.String("token-1"),
						}, nil
					}
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String("sec-2"),
								QuestionTitle: aws.String("Security Question 2"),
								Choices: []types.Choice{
									{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
								},
							},
						},
					}, nil
				}
			}(),
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:          "all six pillars handled",
			awsWorkloadID: "wl-123",
			scope: core.ReviewScope{
				Level: core.ScopeLevelWorkload,
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				// Verify all six pillars are requested
				pillarID := aws.ToString(params.PillarId)
				validPillars := map[string]bool{
					"operationalExcellence": true,
					"security":              true,
					"reliability":           true,
					"performance":           true,
					"costOptimization":      true,
					"sustainability":        true,
				}
				assert.True(t, validPillars[pillarID], "Invalid pillar ID: %s", pillarID)
				
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String(pillarID + "-1"),
							QuestionTitle: aws.String("Question for " + pillarID),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				ListAnswersFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			questions, err := evaluator.GetQuestions(context.Background(), tt.awsWorkloadID, tt.scope)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, questions, tt.wantCount)
			}
		})
	}
}

func TestSubmitAnswer(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		questionID    string
		evaluation    *core.QuestionEvaluation
		mockFunc      func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkInput    func(t *testing.T, params *wellarchitected.UpdateAnswerInput)
	}{
		{
			name:          "successful submission",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
				},
				ConfidenceScore: 0.95,
				Notes:           "Test notes",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return &wellarchitected.UpdateAnswerOutput{}, nil
			},
			wantErr: false,
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			questionID:    "q1",
			evaluation:    &core.QuestionEvaluation{},
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "nil evaluation",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation:    nil,
			wantErr:       true,
			wantErrMsg:    "evaluation is required",
		},
		{
			name:          "empty question ID",
			awsWorkloadID: "wl-123",
			questionID:    "",
			evaluation:    &core.QuestionEvaluation{},
			wantErr:       true,
			wantErrMsg:    "question ID is required",
		},
		{
			name:          "includes automated analysis notes",
			awsWorkloadID: "wl-123",
			questionID:    "sec-1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
					{ID: "c2", Title: "Choice 2"},
				},
				ConfidenceScore: 0.87,
				Notes:           "Evidence from IaC analysis",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return &wellarchitected.UpdateAnswerOutput{}, nil
			},
			wantErr: false,
			checkInput: func(t *testing.T, params *wellarchitected.UpdateAnswerInput) {
				// Verify notes include automated analysis marker
				notes := aws.ToString(params.Notes)
				assert.Contains(t, notes, "Automated analysis by Waffle")
				assert.Contains(t, notes, "confidence: 0.87")
				assert.Contains(t, notes, "Evidence from IaC analysis")
				
				// Verify selected choices are included
				assert.Len(t, params.SelectedChoices, 2)
				assert.Contains(t, params.SelectedChoices, "c1")
				assert.Contains(t, params.SelectedChoices, "c2")
				
				// Verify other fields
				assert.Equal(t, "wl-123", aws.ToString(params.WorkloadId))
				assert.Equal(t, "sec-1", aws.ToString(params.QuestionId))
				assert.Equal(t, "wellarchitected", aws.ToString(params.LensAlias))
				assert.True(t, aws.ToBool(params.IsApplicable))
			},
		},
		{
			name:          "submission error with retry - eventually succeeds",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
				},
				ConfidenceScore: 0.90,
				Notes:           "Test notes",
			},
			mockFunc: func() func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				attempts := 0
				return func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
					attempts++
					if attempts < 2 {
						return nil, &APIError{code: "ThrottlingException", message: "rate exceeded"}
					}
					return &wellarchitected.UpdateAnswerOutput{}, nil
				}
			}(),
			wantErr: false,
		},
		{
			name:          "submission error - max retries exceeded",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
				},
				ConfidenceScore: 0.90,
				Notes:           "Test notes",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return nil, &APIError{code: "ThrottlingException", message: "rate exceeded"}
			},
			wantErr:    true,
			wantErrMsg: "WAFR UpdateAnswer failed",
		},
		{
			name:          "submission with multiple choices",
			awsWorkloadID: "wl-123",
			questionID:    "sec-2",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
					{ID: "c2", Title: "Choice 2"},
					{ID: "c3", Title: "Choice 3"},
				},
				ConfidenceScore: 0.92,
				Notes:           "Multiple best practices identified",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return &wellarchitected.UpdateAnswerOutput{}, nil
			},
			wantErr: false,
			checkInput: func(t *testing.T, params *wellarchitected.UpdateAnswerInput) {
				assert.Len(t, params.SelectedChoices, 3)
			},
		},
		{
			name:          "submission with no choices selected",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{},
				ConfidenceScore: 0.50,
				Notes:           "Unable to determine applicable choices",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return &wellarchitected.UpdateAnswerOutput{}, nil
			},
			wantErr: false,
			checkInput: func(t *testing.T, params *wellarchitected.UpdateAnswerInput) {
				assert.Len(t, params.SelectedChoices, 0)
			},
		},
		{
			name:          "non-retryable error - access denied",
			awsWorkloadID: "wl-123",
			questionID:    "q1",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
				},
				ConfidenceScore: 0.90,
				Notes:           "Test notes",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
				return nil, &types.AccessDeniedException{
					Message: aws.String("insufficient permissions"),
				}
			},
			wantErr:    true,
			wantErrMsg: "WAFR UpdateAnswer failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedInput *wellarchitected.UpdateAnswerInput
			
			mockFunc := tt.mockFunc
			if tt.checkInput != nil && mockFunc != nil {
				// Wrap the mock function to capture input
				originalMock := mockFunc
				mockFunc = func(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
					capturedInput = params
					return originalMock(ctx, params, optFns...)
				}
			}
			
			mockClient := &MockWAFRClient{
				UpdateAnswerFunc: mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			err := evaluator.SubmitAnswer(context.Background(), tt.awsWorkloadID, tt.questionID, tt.evaluation)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.checkInput != nil && capturedInput != nil {
					tt.checkInput(t, capturedInput)
				}
			}
		})
	}
}

func TestCreateMilestone(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		milestoneName string
		mockFunc      func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput)
	}{
		{
			name:          "successful creation",
			awsWorkloadID: "wl-123",
			milestoneName: "v1.0",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return &wellarchitected.CreateMilestoneOutput{
					MilestoneNumber: aws.Int32(1),
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput) {
				assert.Equal(t, "1", milestoneID)
				assert.Equal(t, "wl-123", aws.ToString(params.WorkloadId))
				assert.Equal(t, "v1.0", aws.ToString(params.MilestoneName))
			},
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			milestoneName: "v1.0",
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "auto-generated milestone name",
			awsWorkloadID: "wl-123",
			milestoneName: "",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				// Verify that a milestone name was generated
				assert.NotEmpty(t, aws.ToString(params.MilestoneName))
				assert.Contains(t, aws.ToString(params.MilestoneName), "waffle-")
				return &wellarchitected.CreateMilestoneOutput{
					MilestoneNumber: aws.Int32(2),
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput) {
				assert.Equal(t, "2", milestoneID)
				assert.Contains(t, aws.ToString(params.MilestoneName), "waffle-")
				// Verify timestamp format (YYYY-MM-DD-HH-MM-SS)
				name := aws.ToString(params.MilestoneName)
				assert.Regexp(t, `^waffle-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}$`, name)
			},
		},
		{
			name:          "throttling error with retry - eventually succeeds",
			awsWorkloadID: "wl-123",
			milestoneName: "v2.0",
			mockFunc: func() func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				attempts := 0
				return func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
					attempts++
					if attempts < 2 {
						return nil, &APIError{code: "ThrottlingException", message: "rate exceeded"}
					}
					return &wellarchitected.CreateMilestoneOutput{
						MilestoneNumber: aws.Int32(3),
					}, nil
				}
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput) {
				assert.Equal(t, "3", milestoneID)
			},
		},
		{
			name:          "max retries exceeded",
			awsWorkloadID: "wl-123",
			milestoneName: "v3.0",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return nil, &APIError{code: "ThrottlingException", message: "rate exceeded"}
			},
			wantErr:    true,
			wantErrMsg: "WAFR CreateMilestone failed",
		},
		{
			name:          "access denied error",
			awsWorkloadID: "wl-123",
			milestoneName: "v4.0",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return nil, &types.AccessDeniedException{
					Message: aws.String("insufficient permissions to create milestone"),
				}
			},
			wantErr:    true,
			wantErrMsg: "WAFR CreateMilestone failed",
		},
		{
			name:          "resource not found error",
			awsWorkloadID: "wl-nonexistent",
			milestoneName: "v5.0",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return nil, &types.ResourceNotFoundException{
					Message: aws.String("workload not found"),
				}
			},
			wantErr:    true,
			wantErrMsg: "WAFR CreateMilestone failed",
		},
		{
			name:          "validation error - invalid milestone name",
			awsWorkloadID: "wl-123",
			milestoneName: "invalid name with spaces",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return nil, &types.ValidationException{
					Message: aws.String("milestone name contains invalid characters"),
				}
			},
			wantErr:    true,
			wantErrMsg: "WAFR CreateMilestone failed",
		},
		{
			name:          "service unavailable with retry",
			awsWorkloadID: "wl-123",
			milestoneName: "v6.0",
			mockFunc: func() func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				attempts := 0
				return func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
					attempts++
					if attempts < 3 {
						return nil, &APIError{code: "ServiceUnavailableException", message: "service temporarily unavailable"}
					}
					return &wellarchitected.CreateMilestoneOutput{
						MilestoneNumber: aws.Int32(4),
					}, nil
				}
			}(),
			wantErr: false,
			checkResult: func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput) {
				assert.Equal(t, "4", milestoneID)
			},
		},
		{
			name:          "milestone number returned as string",
			awsWorkloadID: "wl-123",
			milestoneName: "v7.0",
			mockFunc: func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
				return &wellarchitected.CreateMilestoneOutput{
					MilestoneNumber: aws.Int32(42),
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, milestoneID string, params *wellarchitected.CreateMilestoneInput) {
				assert.Equal(t, "42", milestoneID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedParams *wellarchitected.CreateMilestoneInput
			
			mockFunc := tt.mockFunc
			if tt.checkResult != nil && mockFunc != nil {
				// Wrap the mock function to capture input
				originalMock := mockFunc
				mockFunc = func(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
					capturedParams = params
					return originalMock(ctx, params, optFns...)
				}
			}
			
			mockClient := &MockWAFRClient{
				CreateMilestoneFunc: mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			milestoneID, err := evaluator.CreateMilestone(context.Background(), tt.awsWorkloadID, tt.milestoneName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, milestoneID)
				if tt.checkResult != nil && capturedParams != nil {
					tt.checkResult(t, milestoneID, capturedParams)
				}
			}
		})
	}
}

func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		name          string
		mockFunc      func() func() error
		wantErr       bool
		wantAttempts  int
	}{
		{
			name: "success on first attempt",
			mockFunc: func() func() error {
				return func() error {
					return nil
				}
			},
			wantErr:      false,
			wantAttempts: 1,
		},
		{
			name: "success after retry",
			mockFunc: func() func() error {
				attempts := 0
				return func() error {
					attempts++
					if attempts < 2 {
						return &APIError{code: "ThrottlingException", message: "rate exceeded"}
					}
					return nil
				}
			},
			wantErr:      false,
			wantAttempts: 2,
		},
		{
			name: "non-retryable error",
			mockFunc: func() func() error {
				return func() error {
					return &types.AccessDeniedException{
						Message: aws.String("access denied"),
					}
				}
			},
			wantErr:      true,
			wantAttempts: 1,
		},
		{
			name: "max retries exceeded",
			mockFunc: func() func() error {
				return func() error {
					return &APIError{code: "ThrottlingException", message: "rate exceeded"}
				}
			},
			wantErr:      true,
			wantAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(&MockWAFRClient{}, config)

			attempts := 0
			fn := tt.mockFunc()
			err := evaluator.retryWithBackoff(context.Background(), "TestOperation", func() error {
				attempts++
				return fn()
			})

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantAttempts, attempts)
		})
	}
}

func TestMapPillarToAWSID(t *testing.T) {
	tests := []struct {
		pillar core.Pillar
		want   string
	}{
		{core.PillarOperationalExcellence, "operationalExcellence"},
		{core.PillarSecurity, "security"},
		{core.PillarReliability, "reliability"},
		{core.PillarPerformanceEfficiency, "performance"},
		{core.PillarCostOptimization, "costOptimization"},
		{core.PillarSustainability, "sustainability"},
	}

	for _, tt := range tests {
		t.Run(string(tt.pillar), func(t *testing.T) {
			got := mapPillarToAWSID(tt.pillar)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEvaluateQuestion(t *testing.T) {
	tests := []struct {
		name          string
		question      *core.WAFRQuestion
		workloadModel *core.WorkloadModel
		bedrockClient BedrockClient
		mockFunc      func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, eval *core.QuestionEvaluation)
	}{
		{
			name: "successful evaluation with high confidence",
			question: &core.WAFRQuestion{
				ID:     "sec-1",
				Pillar: core.PillarSecurity,
				Title:  "Test Question",
				Choices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
					{ID: "c2", Title: "Choice 2"},
				},
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{
						ID:      "r1",
						Type:    "aws_s3_bucket",
						Address: "aws_s3_bucket.example",
						Properties: map[string]interface{}{
							"bucket": "test-bucket",
						},
					},
				},
				SourceType: "plan",
			},
			mockFunc: func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
				return &core.QuestionEvaluation{
					Question: question,
					SelectedChoices: []core.Choice{
						{ID: "c1", Title: "Choice 1"},
					},
					Evidence: []core.Evidence{
						{
							ChoiceID:    "c1",
							Explanation: "S3 bucket has encryption enabled",
							Resources:   []string{"aws_s3_bucket.example"},
							Confidence:  0.95,
						},
					},
					ConfidenceScore: 0.95,
					Notes:           "Evaluation successful",
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.NotNil(t, eval)
				assert.Equal(t, "sec-1", eval.Question.ID)
				assert.Len(t, eval.SelectedChoices, 1)
				assert.Len(t, eval.Evidence, 1)
				assert.Greater(t, eval.ConfidenceScore, 0.0)
				assert.LessOrEqual(t, eval.ConfidenceScore, 1.0)
			},
		},
		{
			name: "evaluation with partial data - low confidence",
			question: &core.WAFRQuestion{
				ID:     "sec-2",
				Pillar: core.PillarSecurity,
				Title:  "Test Question 2",
				Choices: []core.Choice{
					{ID: "c1", Title: "Choice 1"},
				},
			},
			workloadModel: &core.WorkloadModel{
				Resources:  []core.Resource{},
				SourceType: "hcl",
			},
			mockFunc: func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
				return &core.QuestionEvaluation{
					Question:        question,
					SelectedChoices: []core.Choice{},
					Evidence:        []core.Evidence{},
					ConfidenceScore: 0.5,
					Notes:           "Limited data available",
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.NotNil(t, eval)
				// Confidence should be adjusted down due to no resources
				assert.Less(t, eval.ConfidenceScore, 0.5)
			},
		},
		{
			name:          "nil question",
			question:      nil,
			workloadModel: &core.WorkloadModel{},
			wantErr:       true,
			wantErrMsg:    "question is required",
		},
		{
			name: "nil workload model",
			question: &core.WAFRQuestion{
				ID:     "sec-1",
				Pillar: core.PillarSecurity,
			},
			workloadModel: nil,
			wantErr:       true,
			wantErrMsg:    "workload model is required",
		},
		{
			name: "bedrock evaluation fails - returns low confidence",
			question: &core.WAFRQuestion{
				ID:     "sec-3",
				Pillar: core.PillarSecurity,
				Title:  "Test Question 3",
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
				},
			},
			mockFunc: func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
				return nil, assert.AnError
			},
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.NotNil(t, eval)
				assert.Equal(t, 0.0, eval.ConfidenceScore)
				assert.Contains(t, eval.Notes, "Evaluation failed")
			},
		},
		{
			name: "evidence without resource references - reduced confidence",
			question: &core.WAFRQuestion{
				ID:     "sec-4",
				Pillar: core.PillarSecurity,
				Title:  "Test Question 4",
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
					{ID: "r2", Type: "aws_kms_key"},
				},
				SourceType: "plan",
			},
			mockFunc: func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
				return &core.QuestionEvaluation{
					Question: question,
					SelectedChoices: []core.Choice{
						{ID: "c1", Title: "Choice 1"},
					},
					Evidence: []core.Evidence{
						{
							ChoiceID:    "c1",
							Explanation: "General observation",
							Resources:   []string{}, // No resource references
							Confidence:  0.9,
						},
					},
					ConfidenceScore: 0.9,
					Notes:           "Evaluation with limited evidence",
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.NotNil(t, eval)
				// Confidence should be reduced due to lack of resource references
				assert.Less(t, eval.ConfidenceScore, 0.9)
			},
		},
		{
			name: "terraform plan source - higher confidence",
			question: &core.WAFRQuestion{
				ID:     "sec-5",
				Pillar: core.PillarSecurity,
				Title:  "Test Question 5",
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket", IsFromPlan: true},
					{ID: "r2", Type: "aws_kms_key", IsFromPlan: true},
					{ID: "r3", Type: "aws_vpc", IsFromPlan: true},
					{ID: "r4", Type: "aws_subnet", IsFromPlan: true},
					{ID: "r5", Type: "aws_security_group", IsFromPlan: true},
				},
				SourceType: "plan",
			},
			mockFunc: func(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error) {
				return &core.QuestionEvaluation{
					Question: question,
					SelectedChoices: []core.Choice{
						{ID: "c1", Title: "Choice 1"},
					},
					Evidence: []core.Evidence{
						{
							ChoiceID:    "c1",
							Explanation: "Based on plan data",
							Resources:   []string{"r1", "r2"},
							Confidence:  0.95,
						},
					},
					ConfidenceScore: 0.95,
					Notes:           "High confidence from plan data",
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, eval *core.QuestionEvaluation) {
				assert.NotNil(t, eval)
				// Plan source with sufficient resources should maintain high confidence
				assert.GreaterOrEqual(t, eval.ConfidenceScore, 0.9)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bedrockClient BedrockClient
			if tt.mockFunc != nil {
				bedrockClient = &MockBedrockClient{
					EvaluateWAFRQuestionFunc: tt.mockFunc,
				}
			} else if tt.bedrockClient != nil {
				bedrockClient = tt.bedrockClient
			} else {
				bedrockClient = &MockBedrockClient{}
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(&MockWAFRClient{}, config)

			eval, err := evaluator.EvaluateQuestion(context.Background(), tt.question, tt.workloadModel, bedrockClient)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, eval)
				}
			}
		})
	}
}

func TestCalculateConfidenceScore(t *testing.T) {
	tests := []struct {
		name          string
		evaluation    *core.QuestionEvaluation
		workloadModel *core.WorkloadModel
		want          float64
		checkRange    func(t *testing.T, score float64)
	}{
		{
			name: "high confidence with complete data",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence: []core.Evidence{
					{
						ChoiceID:   "c1",
						Resources:  []string{"r1", "r2"},
						Confidence: 0.95,
					},
				},
				ConfidenceScore: 0.95,
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
					{ID: "r2", Type: "aws_kms_key"},
					{ID: "r3", Type: "aws_vpc"},
					{ID: "r4", Type: "aws_subnet"},
					{ID: "r5", Type: "aws_security_group"},
				},
				SourceType: "plan",
			},
			checkRange: func(t *testing.T, score float64) {
				assert.GreaterOrEqual(t, score, 0.9)
				assert.LessOrEqual(t, score, 1.0)
			},
		},
		{
			name: "reduced confidence with no resources",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence:        []core.Evidence{},
				ConfidenceScore: 0.8,
			},
			workloadModel: &core.WorkloadModel{
				Resources:  []core.Resource{},
				SourceType: "hcl",
			},
			checkRange: func(t *testing.T, score float64) {
				// No resources significantly reduces confidence
				assert.Less(t, score, 0.5)
				assert.GreaterOrEqual(t, score, 0.0)
			},
		},
		{
			name: "reduced confidence with few resources",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence: []core.Evidence{
					{ChoiceID: "c1", Resources: []string{"r1"}},
				},
				ConfidenceScore: 0.9,
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
				},
				SourceType: "plan",
			},
			checkRange: func(t *testing.T, score float64) {
				assert.Less(t, score, 0.9) // Few resources reduce confidence
				assert.Greater(t, score, 0.0)
			},
		},
		{
			name: "reduced confidence without evidence resources",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence: []core.Evidence{
					{ChoiceID: "c1", Resources: []string{}}, // No resource refs
				},
				ConfidenceScore: 0.85,
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
					{ID: "r2", Type: "aws_kms_key"},
					{ID: "r3", Type: "aws_vpc"},
					{ID: "r4", Type: "aws_subnet"},
					{ID: "r5", Type: "aws_security_group"},
				},
				SourceType: "plan",
			},
			checkRange: func(t *testing.T, score float64) {
				assert.Less(t, score, 0.85) // No resource refs reduce confidence
			},
		},
		{
			name: "hcl source reduces confidence",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence: []core.Evidence{
					{ChoiceID: "c1", Resources: []string{"r1"}},
				},
				ConfidenceScore: 0.9,
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
					{ID: "r2", Type: "aws_kms_key"},
					{ID: "r3", Type: "aws_vpc"},
					{ID: "r4", Type: "aws_subnet"},
					{ID: "r5", Type: "aws_security_group"},
				},
				SourceType: "hcl",
			},
			checkRange: func(t *testing.T, score float64) {
				assert.Less(t, score, 0.9) // HCL source reduces confidence
			},
		},
		{
			name:          "nil evaluation returns 0",
			evaluation:    nil,
			workloadModel: &core.WorkloadModel{},
			want:          0.0,
		},
		{
			name:          "nil workload model returns 0",
			evaluation:    &core.QuestionEvaluation{ConfidenceScore: 0.9},
			workloadModel: nil,
			want:          0.0,
		},
		{
			name: "confidence clamped to valid range",
			evaluation: &core.QuestionEvaluation{
				SelectedChoices: []core.Choice{{ID: "c1"}},
				Evidence: []core.Evidence{
					{ChoiceID: "c1", Resources: []string{"r1"}},
				},
				ConfidenceScore: 1.5, // Invalid, should be clamped
			},
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
					{ID: "r2", Type: "aws_kms_key"},
					{ID: "r3", Type: "aws_vpc"},
					{ID: "r4", Type: "aws_subnet"},
					{ID: "r5", Type: "aws_security_group"},
				},
				SourceType: "plan",
			},
			checkRange: func(t *testing.T, score float64) {
				assert.LessOrEqual(t, score, 1.0)
				assert.GreaterOrEqual(t, score, 0.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateConfidenceScore(tt.evaluation, tt.workloadModel)

			if tt.want != 0.0 {
				assert.Equal(t, tt.want, got)
			}

			if tt.checkRange != nil {
				tt.checkRange(t, got)
			}

			// Always verify confidence is in valid range
			assert.GreaterOrEqual(t, got, 0.0)
			assert.LessOrEqual(t, got, 1.0)
		})
	}
}

func TestGetImprovementPlan(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		workloadModel *core.WorkloadModel
		mockFunc      func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, plan *core.ImprovementPlan)
	}{
		{
			name:          "successful retrieval with high risks",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{
						ID:      "r1",
						Type:    "aws_s3_bucket",
						Address: "aws_s3_bucket.example",
					},
					{
						ID:      "r2",
						Type:    "aws_kms_key",
						Address: "aws_kms_key.example",
					},
				},
				SourceType: "plan",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return risks for security pillar
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String("sec-1"),
								QuestionTitle: aws.String("How do you protect your data at rest?"),
								Risk:          types.RiskHigh,
								Choices: []types.Choice{
									{ChoiceId: aws.String("sec_data_1"), Title: aws.String("Encrypt data at rest")},
									{ChoiceId: aws.String("sec_data_2"), Title: aws.String("Use AWS KMS")},
								},
								SelectedChoices: []string{"sec_data_1"}, // Only one selected, one missing
							},
						},
					}, nil
				}
				
				// No risks for other pillars
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 1)
				
				item := plan.Items[0]
				assert.NotNil(t, item.Risk)
				assert.Equal(t, core.RiskLevelHigh, item.Risk.Severity)
				assert.Equal(t, core.PillarSecurity, item.Risk.Pillar)
				assert.NotEmpty(t, item.Description)
				assert.NotEmpty(t, item.BestPracticeRefs)
				assert.Greater(t, item.Priority, 0)
				assert.NotEmpty(t, item.EstimatedEffort)
				
				// Check that resources were enhanced
				assert.NotEmpty(t, item.AffectedResources)
			},
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			workloadModel: &core.WorkloadModel{},
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "no risks identified",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String("sec-1"),
							QuestionTitle: aws.String("Security Question"),
							Risk:          types.RiskNone, // No risk
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
							SelectedChoices: []string{"c1"},
						},
					},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 0) // No risks, no improvement items
			},
		},
		{
			name:          "multiple risks across pillars",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data"},
					{ID: "r2", Type: "aws_autoscaling_group", Address: "aws_autoscaling_group.web"},
					{ID: "r3", Type: "aws_instance", Address: "aws_instance.app"},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				switch pillarID {
				case "security":
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("sec-1"),
								QuestionTitle:   aws.String("Data protection"),
								Risk:            types.RiskHigh,
								Choices:         []types.Choice{{ChoiceId: aws.String("c1"), Title: aws.String("Encrypt")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				case "reliability":
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("rel-1"),
								QuestionTitle:   aws.String("Auto scaling"),
								Risk:            types.RiskMedium,
								Choices:         []types.Choice{{ChoiceId: aws.String("c2"), Title: aws.String("Scale")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				default:
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{},
					}, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.GreaterOrEqual(t, len(plan.Items), 2) // At least 2 risks
				
				// Verify risks from different pillars
				pillars := make(map[core.Pillar]bool)
				for _, item := range plan.Items {
					pillars[item.Risk.Pillar] = true
				}
				assert.True(t, len(pillars) >= 2, "Should have risks from multiple pillars")
			},
		},
		{
			name:          "risk with missing best practices",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket", Address: "aws_s3_bucket.example"},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return risks for security pillar
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String("sec-1"),
								QuestionTitle: aws.String("Data protection"),
								Risk:          types.RiskHigh,
								Choices: []types.Choice{
									{ChoiceId: aws.String("c1"), Title: aws.String("Encrypt at rest"), Description: aws.String("Use encryption")},
									{ChoiceId: aws.String("c2"), Title: aws.String("Use KMS"), Description: aws.String("Use AWS KMS")},
									{ChoiceId: aws.String("c3"), Title: aws.String("Rotate keys"), Description: aws.String("Rotate encryption keys")},
								},
								SelectedChoices: []string{"c1"}, // 2 missing best practices
							},
						},
					}, nil
				}
				
				// No risks for other pillars
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 1)
				
				item := plan.Items[0]
				assert.Len(t, item.Risk.MissingBestPractices, 2) // c2 and c3 not selected
				assert.Contains(t, item.Description, "2 best practice(s) not implemented")
			},
		},
		{
			name:          "enhancement with IaC resources",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data"},
					{ID: "r2", Type: "aws_kms_key", Address: "aws_kms_key.encryption"},
					{ID: "r3", Type: "aws_iam_role", Address: "aws_iam_role.app"},
					{ID: "r4", Type: "aws_instance", Address: "aws_instance.web"},
				},
				SourceType: "plan",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return risks for security pillar
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("sec-1"),
								QuestionTitle:   aws.String("Security question"),
								Risk:            types.RiskHigh,
								Choices:         []types.Choice{{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				}
				
				// No risks for other pillars
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 1)
				
				item := plan.Items[0]
				// Security pillar should match S3, KMS, IAM resources
				assert.NotEmpty(t, item.AffectedResources)
				assert.Contains(t, item.AffectedResources, "aws_s3_bucket.data")
			},
		},
		{
			name:          "priority calculation based on severity",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket"},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("sec-1"),
								QuestionTitle:   aws.String("High risk question"),
								Risk:            types.RiskHigh,
								Choices:         []types.Choice{{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				} else if pillarID == "reliability" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("rel-1"),
								QuestionTitle:   aws.String("Medium risk question"),
								Risk:            types.RiskMedium,
								Choices:         []types.Choice{{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				}
				
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.GreaterOrEqual(t, len(plan.Items), 2)
				
				// Find high and medium risk items
				var highPriority, mediumPriority int
				for _, item := range plan.Items {
					if item.Risk.Severity == core.RiskLevelHigh {
						highPriority = item.Priority
					} else if item.Risk.Severity == core.RiskLevelMedium {
						mediumPriority = item.Priority
					}
				}
				
				// High risk should have higher priority
				assert.Greater(t, highPriority, mediumPriority)
			},
		},
		{
			name:          "effort estimation based on complexity",
			awsWorkloadID: "wl-123",
			workloadModel: &core.WorkloadModel{
				Resources: []core.Resource{
					{ID: "r1", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data1"},
					{ID: "r2", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data2"},
					{ID: "r3", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data3"},
					{ID: "r4", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data4"},
					{ID: "r5", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data5"},
					{ID: "r6", Type: "aws_s3_bucket", Address: "aws_s3_bucket.data6"},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return risks for security pillar
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String("sec-1"),
								QuestionTitle: aws.String("Complex security issue"),
								Risk:          types.RiskHigh,
								Choices: []types.Choice{
									{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
									{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
									{ChoiceId: aws.String("c3"), Title: aws.String("Choice 3")},
									{ChoiceId: aws.String("c4"), Title: aws.String("Choice 4")},
								},
								SelectedChoices: []string{}, // All 4 missing
							},
						},
					}, nil
				}
				
				// No risks for other pillars
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 1)
				
				item := plan.Items[0]
				// Many missing best practices + many affected resources = HIGH effort
				assert.Equal(t, "HIGH", item.EstimatedEffort)
			},
		},
		{
			name:          "nil workload model - no resource enhancement",
			awsWorkloadID: "wl-123",
			workloadModel: nil,
			mockFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return risks for security pillar
				if pillarID == "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:      aws.String("sec-1"),
								QuestionTitle:   aws.String("Security question"),
								Risk:            types.RiskHigh,
								Choices:         []types.Choice{{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")}},
								SelectedChoices: []string{},
							},
						},
					}, nil
				}
				
				// No risks for other pillars
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, plan *core.ImprovementPlan) {
				require.NotNil(t, plan)
				assert.Len(t, plan.Items, 1)
				
				item := plan.Items[0]
				// No workload model means no resource enhancement
				assert.Empty(t, item.AffectedResources)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				ListAnswersFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			plan, err := evaluator.GetImprovementPlan(context.Background(), tt.awsWorkloadID, tt.workloadModel)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, plan)
				}
			}
		})
	}
}

func TestMapAWSRiskToRiskLevel(t *testing.T) {
	tests := []struct {
		awsRisk types.Risk
		want    core.RiskLevel
	}{
		{types.RiskHigh, core.RiskLevelHigh},
		{types.RiskMedium, core.RiskLevelMedium},
		{types.RiskNone, core.RiskLevelNone},
		{types.RiskNotApplicable, core.RiskLevelNone},
		{types.RiskUnanswered, core.RiskLevelNone},
	}

	for _, tt := range tests {
		t.Run(string(tt.awsRisk), func(t *testing.T) {
			got := mapAWSRiskToRiskLevel(tt.awsRisk)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRelevantResourceTypes(t *testing.T) {
	tests := []struct {
		name       string
		questionID string
		pillar     core.Pillar
		wantTypes  []string
	}{
		{
			name:       "security pillar",
			questionID: "sec-1",
			pillar:     core.PillarSecurity,
			wantTypes:  []string{"aws_s3_bucket", "aws_kms_key", "aws_iam_role", "aws_iam_policy", "aws_security_group", "aws_vpc", "aws_subnet"},
		},
		{
			name:       "reliability pillar",
			questionID: "rel-1",
			pillar:     core.PillarReliability,
			wantTypes:  []string{"aws_autoscaling_group", "aws_elb", "aws_lb", "aws_rds_instance", "aws_dynamodb_table", "aws_backup_plan"},
		},
		{
			name:       "performance pillar",
			questionID: "perf-1",
			pillar:     core.PillarPerformanceEfficiency,
			wantTypes:  []string{"aws_instance", "aws_lambda_function", "aws_cloudfront_distribution", "aws_elasticache_cluster"},
		},
		{
			name:       "cost optimization pillar",
			questionID: "cost-1",
			pillar:     core.PillarCostOptimization,
			wantTypes:  []string{"aws_instance", "aws_rds_instance", "aws_s3_bucket", "aws_ebs_volume"},
		},
		{
			name:       "operational excellence pillar",
			questionID: "ops-1",
			pillar:     core.PillarOperationalExcellence,
			wantTypes:  []string{"aws_cloudwatch_log_group", "aws_cloudwatch_metric_alarm", "aws_sns_topic", "aws_lambda_function"},
		},
		{
			name:       "sustainability pillar",
			questionID: "sus-1",
			pillar:     core.PillarSustainability,
			wantTypes:  []string{"aws_instance", "aws_autoscaling_group", "aws_lambda_function"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRelevantResourceTypes(tt.questionID, tt.pillar)
			assert.ElementsMatch(t, tt.wantTypes, got)
		})
	}
}

func TestMatchesResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		pattern      string
		want         bool
	}{
		{
			name:         "exact match",
			resourceType: "aws_s3_bucket",
			pattern:      "aws_s3_bucket",
			want:         true,
		},
		{
			name:         "prefix match",
			resourceType: "aws_s3_bucket_policy",
			pattern:      "aws_s3_bucket",
			want:         true,
		},
		{
			name:         "no match",
			resourceType: "aws_dynamodb_table",
			pattern:      "aws_s3_bucket",
			want:         false,
		},
		{
			name:         "partial match not at start",
			resourceType: "module.aws_s3_bucket",
			pattern:      "aws_s3_bucket",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesResourceType(tt.resourceType, tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculatePriority(t *testing.T) {
	tests := []struct {
		name     string
		risk     *core.Risk
		wantMin  int
		wantMax  int
	}{
		{
			name: "high severity with many missing best practices",
			risk: &core.Risk{
				Severity: core.RiskLevelHigh,
				MissingBestPractices: []core.BestPractice{
					{ID: "bp1"},
					{ID: "bp2"},
					{ID: "bp3"},
				},
			},
			wantMin: 100,
			wantMax: 200,
		},
		{
			name: "medium severity with few missing best practices",
			risk: &core.Risk{
				Severity: core.RiskLevelMedium,
				MissingBestPractices: []core.BestPractice{
					{ID: "bp1"},
				},
			},
			wantMin: 50,
			wantMax: 60,
		},
		{
			name: "low severity",
			risk: &core.Risk{
				Severity:             core.RiskLevelNone,
				MissingBestPractices: []core.BestPractice{},
			},
			wantMin: 10,
			wantMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePriority(tt.risk)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

func TestEstimateEffort(t *testing.T) {
	tests := []struct {
		name string
		risk *core.Risk
		want string
	}{
		{
			name: "low effort - few items",
			risk: &core.Risk{
				MissingBestPractices: []core.BestPractice{
					{ID: "bp1"},
				},
				AffectedResources: []string{"r1"},
			},
			want: "LOW",
		},
		{
			name: "medium effort - moderate items",
			risk: &core.Risk{
				MissingBestPractices: []core.BestPractice{
					{ID: "bp1"},
					{ID: "bp2"},
				},
				AffectedResources: []string{"r1", "r2"},
			},
			want: "MEDIUM",
		},
		{
			name: "high effort - many items",
			risk: &core.Risk{
				MissingBestPractices: []core.BestPractice{
					{ID: "bp1"},
					{ID: "bp2"},
					{ID: "bp3"},
				},
				AffectedResources: []string{"r1", "r2", "r3", "r4"},
			},
			want: "HIGH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateEffort(tt.risk)
			assert.Equal(t, tt.want, got)
		})
	}
}
