package report

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/wafr"
)

// MockWAFRClient implements the WAFRClient interface for testing
type MockWAFRClient struct {
	GetMilestoneFunc func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
}

func (m *MockWAFRClient) CreateWorkload(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error) {
	return nil, nil
}

func (m *MockWAFRClient) GetWorkload(ctx context.Context, params *wellarchitected.GetWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error) {
	return nil, nil
}

func (m *MockWAFRClient) ListAnswers(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
	return nil, nil
}

func (m *MockWAFRClient) UpdateAnswer(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error) {
	return nil, nil
}

func (m *MockWAFRClient) CreateMilestone(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error) {
	return nil, nil
}

func (m *MockWAFRClient) GetConsolidatedReport(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
	return nil, nil
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

func TestCompareMilestones(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		milestoneID1  string
		milestoneID2  string
		mockFunc      func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, result *core.MilestoneComparison)
	}{
		{
			name:          "successful comparison with improvements",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "2",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
				milestoneNum := aws.ToInt32(params.MilestoneNumber)

				if milestoneNum == 1 {
					return &wellarchitected.GetMilestoneOutput{
						Milestone: &types.Milestone{
							MilestoneName: aws.String("v1.0"),
							RecordedAt:    aws.Time(time.Now().Add(-24 * time.Hour)),
							Workload: &types.Workload{
								RiskCounts: map[string]int32{
									string(types.RiskHigh):   5,
									string(types.RiskMedium): 10,
								},
							},
						},
					}, nil
				}

				return &wellarchitected.GetMilestoneOutput{
					Milestone: &types.Milestone{
						MilestoneName: aws.String("v2.0"),
						RecordedAt:    aws.Time(time.Now()),
						Workload: &types.Workload{
							RiskCounts: map[string]int32{
								string(types.RiskHigh):   2,
								string(types.RiskMedium): 8,
							},
						},
					},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *core.MilestoneComparison) {
				assert.Equal(t, "1", result.Milestone1)
				assert.Equal(t, "2", result.Milestone2)
				assert.Len(t, result.Improvements, 2) // High and medium risks reduced
				assert.Len(t, result.Regressions, 0)
				assert.Contains(t, result.Improvements[0], "High risks reduced from 5 to 2")
				assert.Contains(t, result.Improvements[1], "Medium risks reduced from 10 to 8")
			},
		},
		{
			name:          "comparison with regressions",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "2",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
				milestoneNum := aws.ToInt32(params.MilestoneNumber)

				if milestoneNum == 1 {
					return &wellarchitected.GetMilestoneOutput{
						Milestone: &types.Milestone{
							MilestoneName: aws.String("v1.0"),
							RecordedAt:    aws.Time(time.Now().Add(-24 * time.Hour)),
							Workload: &types.Workload{
								RiskCounts: map[string]int32{
									string(types.RiskHigh):   2,
									string(types.RiskMedium): 5,
								},
							},
						},
					}, nil
				}

				return &wellarchitected.GetMilestoneOutput{
					Milestone: &types.Milestone{
						MilestoneName: aws.String("v2.0"),
						RecordedAt:    aws.Time(time.Now()),
						Workload: &types.Workload{
							RiskCounts: map[string]int32{
								string(types.RiskHigh):   5,
								string(types.RiskMedium): 8,
							},
						},
					},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *core.MilestoneComparison) {
				assert.Len(t, result.Improvements, 0)
				assert.Len(t, result.Regressions, 2) // High and medium risks increased
				assert.Contains(t, result.Regressions[0], "High risks increased from 2 to 5")
				assert.Contains(t, result.Regressions[1], "Medium risks increased from 5 to 8")
			},
		},
		{
			name:          "no changes between milestones",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "2",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
				return &wellarchitected.GetMilestoneOutput{
					Milestone: &types.Milestone{
						MilestoneName: aws.String("v1.0"),
						RecordedAt:    aws.Time(time.Now()),
						Workload: &types.Workload{
							RiskCounts: map[string]int32{
								string(types.RiskHigh):   3,
								string(types.RiskMedium): 7,
							},
						},
					},
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *core.MilestoneComparison) {
				assert.Len(t, result.Improvements, 0)
				assert.Len(t, result.Regressions, 0)
				assert.Len(t, result.NewRisks, 0)
				assert.Len(t, result.ResolvedRisks, 0)
			},
		},
		{
			name:          "evaluator not initialized",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "2",
			wantErr:       true,
			wantErrMsg:    "evaluator not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var generator *Generator

			if tt.name == "evaluator not initialized" {
				// Create generator without evaluator
				generator = &Generator{}
			} else {
				// Create generator with mock evaluator
				mockClient := &MockWAFRClient{
					GetMilestoneFunc: tt.mockFunc,
				}

				config := &wafr.EvaluatorConfig{
					MaxRetries: 3,
					BaseDelay:  1 * time.Millisecond,
				}
				evaluator := wafr.NewEvaluator(mockClient, config)
				generator = NewGeneratorWithEvaluator(evaluator)
			}

			result, err := generator.CompareMilestones(context.Background(), tt.awsWorkloadID, tt.milestoneID1, tt.milestoneID2)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestCompareMilestones_MissingMilestones(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		milestoneID1  string
		milestoneID2  string
		mockFunc      func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:          "milestone 1 not found",
			awsWorkloadID: "wl-123",
			milestoneID1:  "999",
			milestoneID2:  "2",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
				milestoneNum := aws.ToInt32(params.MilestoneNumber)
				if milestoneNum == 999 {
					return nil, &types.ResourceNotFoundException{
						Message: aws.String("milestone not found"),
					}
				}
				return &wellarchitected.GetMilestoneOutput{
					Milestone: &types.Milestone{
						MilestoneName: aws.String("v2.0"),
						Workload: &types.Workload{
							RiskCounts: map[string]int32{
								string(types.RiskHigh):   0,
								string(types.RiskMedium): 0,
							},
						},
					},
				}, nil
			},
			wantErr:    true,
			wantErrMsg: "failed to get milestone 1",
		},
		{
			name:          "milestone 2 not found",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "999",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error) {
				milestoneNum := aws.ToInt32(params.MilestoneNumber)
				if milestoneNum == 999 {
					return nil, &types.ResourceNotFoundException{
						Message: aws.String("milestone not found"),
					}
				}
				return &wellarchitected.GetMilestoneOutput{
					Milestone: &types.Milestone{
						MilestoneName: aws.String("v1.0"),
						Workload: &types.Workload{
							RiskCounts: map[string]int32{
								string(types.RiskHigh):   0,
								string(types.RiskMedium): 0,
							},
						},
					},
				}, nil
			},
			wantErr:    true,
			wantErrMsg: "failed to get milestone 2",
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			milestoneID1:  "1",
			milestoneID2:  "2",
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "empty milestone ID 1",
			awsWorkloadID: "wl-123",
			milestoneID1:  "",
			milestoneID2:  "2",
			wantErr:       true,
			wantErrMsg:    "both milestone IDs are required",
		},
		{
			name:          "empty milestone ID 2",
			awsWorkloadID: "wl-123",
			milestoneID1:  "1",
			milestoneID2:  "",
			wantErr:       true,
			wantErrMsg:    "both milestone IDs are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				GetMilestoneFunc: tt.mockFunc,
			}

			config := &wafr.EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := wafr.NewEvaluator(mockClient, config)
			generator := NewGeneratorWithEvaluator(evaluator)

			result, err := generator.CompareMilestones(context.Background(), tt.awsWorkloadID, tt.milestoneID1, tt.milestoneID2)

			require.Error(t, err)
			assert.Nil(t, result)
			if tt.wantErrMsg != "" {
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			}
		})
	}
}
