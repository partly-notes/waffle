package wafr

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConsolidatedReport(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		format        string
		mockFunc      func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error)
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:          "successful PDF report",
			awsWorkloadID: "wl-123",
			format:        "pdf",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				assert.Equal(t, types.ReportFormatPdf, params.Format)
				// Return base64-encoded string
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("dGVzdC1wZGYtZGF0YQ=="), // "test-pdf-data" in base64
				}, nil
			},
			wantErr: false,
		},
		{
			name:          "successful JSON report",
			awsWorkloadID: "wl-123",
			format:        "json",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				assert.Equal(t, types.ReportFormatJson, params.Format)
				// Return base64-encoded JSON
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("eyJ0ZXN0IjoidmFsdWUifQ=="), // {"test":"value"} in base64
				}, nil
			},
			wantErr: false,
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			format:        "pdf",
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "unsupported format",
			awsWorkloadID: "wl-123",
			format:        "xml",
			wantErr:       true,
			wantErrMsg:    "unsupported report format",
		},
		{
			name:          "case insensitive format - PDF",
			awsWorkloadID: "wl-123",
			format:        "PDF",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				assert.Equal(t, types.ReportFormatPdf, params.Format)
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("dGVzdC1kYXRh"), // "test-data" in base64
				}, nil
			},
			wantErr: false,
		},
		{
			name:          "case insensitive format - JSON",
			awsWorkloadID: "wl-123",
			format:        "JSON",
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				assert.Equal(t, types.ReportFormatJson, params.Format)
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("dGVzdC1kYXRh"), // "test-data" in base64
				}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				GetConsolidatedReportFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			report, err := evaluator.GetConsolidatedReport(context.Background(), tt.awsWorkloadID, tt.format)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, report)
			}
		})
	}
}

func TestGetResultsJSON(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		sessionData   map[string]interface{}
		mockFunc      func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, result map[string]interface{})
	}{
		{
			name:          "successful enhanced report with full session data",
			awsWorkloadID: "wl-123",
			sessionData: map[string]interface{}{
				"workload_id": "my-app",
				"region":      "us-west-2",
				"workload_model": map[string]interface{}{
					"framework":   "terraform",
					"source_type": "plan",
					"resources": []interface{}{
						map[string]interface{}{
							"address": "aws_s3_bucket.example",
							"type":    "aws_s3_bucket",
						},
					},
				},
				"results": map[string]interface{}{
					"evaluations": []interface{}{
						map[string]interface{}{
							"question": map[string]interface{}{
								"id":     "sec_data_1",
								"title":  "How do you classify your data?",
								"pillar": "security",
							},
							"selected_choices": []interface{}{
								map[string]interface{}{
									"title": "Data classification defined",
								},
							},
							"confidence_score": 0.95,
							"evidence": []interface{}{
								map[string]interface{}{
									"explanation": "S3 bucket has encryption enabled",
								},
							},
							"notes": "Automated analysis",
						},
					},
					"risks": []interface{}{
						map[string]interface{}{
							"id": "risk-1",
							"question": map[string]interface{}{
								"id":    "sec_data_2",
								"title": "How do you protect your data at rest?",
							},
							"pillar":             "security",
							"severity":           "HIGH",
							"description":        "Encryption not enabled",
							"affected_resources": []interface{}{"aws_s3_bucket.example"},
						},
					},
					"improvement_plan": map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{
								"description": "Enable encryption for S3 buckets",
							},
						},
					},
				},
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("eyJ0ZXN0IjoidmFsdWUifQ=="), // {"test":"value"} in base64
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "wl-123", result["aws_workload_id"])
				assert.Equal(t, "my-app", result["workload_name"])
				assert.Contains(t, result["console_link"], "us-west-2")
				assert.Equal(t, "terraform", result["iac_framework"])
				assert.Equal(t, "plan", result["iac_source_type"])
				assert.Equal(t, float64(1), result["resource_count"])
				
				evaluations := result["evaluations"].([]interface{})
				assert.Len(t, evaluations, 1)
				
				risks := result["risks"].([]interface{})
				assert.Len(t, risks, 1)
				
				improvementPlan := result["improvement_plan"].([]interface{})
				assert.Len(t, improvementPlan, 1)
				
				assert.Equal(t, 0.95, result["average_confidence"])
			},
		},
		{
			name:          "empty workload ID",
			awsWorkloadID: "",
			sessionData:   map[string]interface{}{},
			wantErr:       true,
			wantErrMsg:    "AWS workload ID is required",
		},
		{
			name:          "minimal session data",
			awsWorkloadID: "wl-456",
			sessionData: map[string]interface{}{
				"workload_id": "minimal-app",
			},
			mockFunc: func(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error) {
				return &wellarchitected.GetConsolidatedReportOutput{
					Base64String: aws.String("e30="), // {} in base64
				}, nil
			},
			wantErr: false,
			checkResult: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "wl-456", result["aws_workload_id"])
				assert.Equal(t, "minimal-app", result["workload_name"])
				assert.NotEmpty(t, result["console_link"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				GetConsolidatedReportFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			result, err := evaluator.GetResultsJSON(context.Background(), tt.awsWorkloadID, tt.sessionData)

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

func TestCompareMilestones(t *testing.T) {
	tests := []struct {
		name          string
		awsWorkloadID string
		milestoneID1  string
		milestoneID2  string
		mockFunc      func(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
		wantErr       bool
		wantErrMsg    string
		checkResult   func(t *testing.T, result map[string]interface{})
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
			checkResult: func(t *testing.T, result map[string]interface{}) {
				summary := result["summary"].(map[string]interface{})
				assert.Equal(t, 2, summary["improvements"]) // High and medium risks reduced
				assert.Equal(t, 0, summary["regressions"])
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
			checkResult: func(t *testing.T, result map[string]interface{}) {
				summary := result["summary"].(map[string]interface{})
				assert.Equal(t, 0, summary["improvements"])
				assert.Equal(t, 2, summary["regressions"]) // High and medium risks increased
			},
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
			name:          "missing milestone ID",
			awsWorkloadID: "wl-123",
			milestoneID1:  "",
			milestoneID2:  "2",
			wantErr:       true,
			wantErrMsg:    "both milestone IDs are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				GetMilestoneFunc: tt.mockFunc,
			}

			config := &EvaluatorConfig{
				MaxRetries: 3,
				BaseDelay:  1 * time.Millisecond,
			}
			evaluator := NewEvaluator(mockClient, config)

			result, err := evaluator.CompareMilestones(context.Background(), tt.awsWorkloadID, tt.milestoneID1, tt.milestoneID2)

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
