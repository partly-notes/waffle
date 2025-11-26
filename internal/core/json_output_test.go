package core

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "simple map",
			data: map[string]interface{}{
				"key": "value",
			},
			wantErr: false,
		},
		{
			name: "struct",
			data: ReviewOutput{
				SessionID:  "test-session",
				WorkloadID: "test-workload",
				Status:     "completed",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteJSON(&buf, tt.data)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, buf.String())

				// Verify it's valid JSON
				var result map[string]interface{}
				err = json.Unmarshal(buf.Bytes(), &result)
				require.NoError(t, err)
			}
		})
	}
}

func TestWriteJSONSuccess(t *testing.T) {
	data := map[string]string{
		"session_id": "test-123",
		"status":     "completed",
	}

	var buf bytes.Buffer
	err := WriteJSONSuccess(&buf, data)
	require.NoError(t, err)

	var output JSONOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.NotNil(t, output.Data)
	assert.Nil(t, output.Error)
}

func TestWriteJSONError(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSONError(&buf, "INVALID_INPUT", "The input was invalid")
	require.NoError(t, err)

	var output JSONOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	assert.False(t, output.Success)
	assert.Nil(t, output.Data)
	require.NotNil(t, output.Error)
	assert.Equal(t, "INVALID_INPUT", output.Error.Code)
	assert.Equal(t, "The input was invalid", output.Error.Message)
}

func TestConvertReviewSessionToOutput(t *testing.T) {
	now := time.Now()
	session := &ReviewSession{
		SessionID:     "session-123",
		WorkloadID:    "workload-456",
		AWSWorkloadID: "aws-wl-789",
		MilestoneID:   "milestone-001",
		Status:        SessionStatusCompleted,
		CreatedAt:     now,
		Results: &ReviewResults{
			Summary: &ResultsSummary{
				QuestionsEvaluated:  10,
				HighRisks:           2,
				MediumRisks:         3,
				AverageConfidence:   0.85,
				ImprovementPlanSize: 5,
			},
		},
	}

	output := ConvertReviewSessionToOutput(session)

	assert.Equal(t, "session-123", output.SessionID)
	assert.Equal(t, "workload-456", output.WorkloadID)
	assert.Equal(t, "completed", output.Status)
	assert.Equal(t, now, output.CreatedAt)

	require.NotNil(t, output.Summary)
	assert.Equal(t, 10, output.Summary.QuestionsEvaluated)
	assert.Equal(t, 2, output.Summary.HighRisks)
	assert.Equal(t, 3, output.Summary.MediumRisks)
	assert.Equal(t, 0.85, output.Summary.AverageConfidence)
	assert.Equal(t, 5, output.Summary.ImprovementPlanSize)

	require.NotNil(t, output.Metadata)
	assert.Equal(t, "aws-wl-789", output.Metadata["aws_workload_id"])
	assert.Equal(t, "milestone-001", output.Metadata["milestone_id"])
}

func TestConvertReviewSessionToStatusOutput(t *testing.T) {
	now := time.Now()
	session := &ReviewSession{
		SessionID:     "session-123",
		WorkloadID:    "workload-456",
		AWSWorkloadID: "aws-wl-789",
		Status:        SessionStatusInProgress,
		CreatedAt:     now,
		UpdatedAt:     now.Add(5 * time.Minute),
		Checkpoint:    "evaluate_questions",
	}

	output := ConvertReviewSessionToStatusOutput(session)

	assert.Equal(t, "session-123", output.SessionID)
	assert.Equal(t, "workload-456", output.WorkloadID)
	assert.Equal(t, "in_progress", output.Status)
	assert.Equal(t, now, output.CreatedAt)
	assert.Equal(t, now.Add(5*time.Minute), output.UpdatedAt)

	require.NotNil(t, output.Metadata)
	assert.Equal(t, "aws-wl-789", output.Metadata["aws_workload_id"])
	assert.Equal(t, "evaluate_questions", output.Metadata["checkpoint"])
}

func TestConvertReviewSessionToResultsOutput(t *testing.T) {
	now := time.Now()
	pillar := PillarSecurity

	session := &ReviewSession{
		SessionID:     "session-123",
		WorkloadID:    "workload-456",
		AWSWorkloadID: "aws-wl-789",
		MilestoneID:   "milestone-001",
		Status:        SessionStatusCompleted,
		CreatedAt:     now,
		UpdatedAt:     now.Add(10 * time.Minute),
		Scope: ReviewScope{
			Level:  ScopeLevelPillar,
			Pillar: &pillar,
		},
		WorkloadModel: &WorkloadModel{
			Resources: []Resource{
				{
					ID:         "res-1",
					Type:       "aws_s3_bucket",
					Address:    "aws_s3_bucket.example",
					SourceFile: "main.tf",
					IsFromPlan: true,
				},
			},
		},
		Results: &ReviewResults{
			Summary: &ResultsSummary{
				QuestionsEvaluated:  5,
				HighRisks:           1,
				MediumRisks:         2,
				AverageConfidence:   0.90,
				ImprovementPlanSize: 3,
			},
			Evaluations: []*QuestionEvaluation{
				{
					Question: &WAFRQuestion{
						ID:     "sec_data_1",
						Pillar: PillarSecurity,
						Title:  "How do you classify your data?",
					},
					SelectedChoices: []Choice{
						{ID: "sec_data_1_choice_1", Title: "Data classification defined"},
					},
					Evidence: []Evidence{
						{
							ChoiceID:    "sec_data_1_choice_1",
							Explanation: "S3 bucket has encryption",
							Resources:   []string{"aws_s3_bucket.example"},
							Confidence:  0.95,
						},
					},
					ConfidenceScore: 0.95,
					Notes:           "Automated analysis",
				},
			},
			Risks: []*Risk{
				{
					ID: "risk-1",
					Question: &WAFRQuestion{
						ID:     "sec_data_2",
						Pillar: PillarSecurity,
					},
					Pillar:            PillarSecurity,
					Severity:          RiskLevelHigh,
					Description:       "Data not encrypted",
					AffectedResources: []string{"aws_s3_bucket.example"},
					MissingBestPractices: []BestPractice{
						{ID: "sec_data_encryption", Title: "Encrypt data at rest"},
					},
				},
			},
			ImprovementPlan: &ImprovementPlan{
				Items: []*ImprovementPlanItem{
					{
						ID:                "imp-1",
						Description:       "Enable encryption for S3 buckets",
						BestPracticeRefs:  []string{"sec_data_encryption"},
						AffectedResources: []string{"aws_s3_bucket.example"},
						Priority:          1,
						EstimatedEffort:   "LOW",
					},
				},
			},
		},
	}

	output := ConvertReviewSessionToResultsOutput(session)

	// Verify basic fields
	assert.Equal(t, "session-123", output.SessionID)
	assert.Equal(t, "workload-456", output.WorkloadID)
	assert.Equal(t, "aws-wl-789", output.AWSWorkloadID)
	assert.Equal(t, "milestone-001", output.MilestoneID)
	assert.Equal(t, "completed", output.Status)

	// Verify scope
	require.NotNil(t, output.Scope)
	assert.Equal(t, "pillar", output.Scope.Level)
	require.NotNil(t, output.Scope.Pillar)
	assert.Equal(t, "security", *output.Scope.Pillar)

	// Verify summary
	require.NotNil(t, output.Summary)
	assert.Equal(t, 5, output.Summary.QuestionsEvaluated)
	assert.Equal(t, 1, output.Summary.HighRisks)
	assert.Equal(t, 2, output.Summary.MediumRisks)

	// Verify evaluations
	require.Len(t, output.Evaluations, 1)
	eval := output.Evaluations[0]
	assert.Equal(t, "sec_data_1", eval.QuestionID)
	assert.Equal(t, "security", eval.Pillar)
	assert.Len(t, eval.SelectedChoices, 1)
	assert.Equal(t, "sec_data_1_choice_1", eval.SelectedChoices[0])
	assert.Len(t, eval.Evidence, 1)
	assert.Equal(t, 0.95, eval.ConfidenceScore)

	// Verify risks
	require.Len(t, output.Risks, 1)
	risk := output.Risks[0]
	assert.Equal(t, "risk-1", risk.ID)
	assert.Equal(t, "sec_data_2", risk.QuestionID)
	assert.Equal(t, "high", risk.Severity)
	assert.Len(t, risk.AffectedResources, 1)
	assert.Len(t, risk.MissingBestPractices, 1)

	// Verify improvements
	require.Len(t, output.Improvements, 1)
	imp := output.Improvements[0]
	assert.Equal(t, "imp-1", imp.ID)
	assert.Equal(t, "Enable encryption for S3 buckets", imp.Description)
	assert.Equal(t, 1, imp.Priority)
	assert.Equal(t, "LOW", imp.EstimatedEffort)

	// Verify resources
	require.Len(t, output.Resources, 1)
	res := output.Resources[0]
	assert.Equal(t, "res-1", res.ID)
	assert.Equal(t, "aws_s3_bucket", res.Type)
	assert.True(t, res.IsFromPlan)

	// Verify links
	require.NotNil(t, output.Links)
	assert.Contains(t, output.Links["aws_console"], "aws-wl-789")
}

func TestConvertScopeToOutput(t *testing.T) {
	tests := []struct {
		name     string
		scope    ReviewScope
		expected ScopeOutput
	}{
		{
			name: "workload scope",
			scope: ReviewScope{
				Level: ScopeLevelWorkload,
			},
			expected: ScopeOutput{
				Level: "workload",
			},
		},
		{
			name: "pillar scope",
			scope: ReviewScope{
				Level:  ScopeLevelPillar,
				Pillar: func() *Pillar { p := PillarSecurity; return &p }(),
			},
			expected: ScopeOutput{
				Level:  "pillar",
				Pillar: func() *string { s := "security"; return &s }(),
			},
		},
		{
			name: "question scope",
			scope: ReviewScope{
				Level:      ScopeLevelQuestion,
				QuestionID: "sec_data_1",
			},
			expected: ScopeOutput{
				Level:      "question",
				QuestionID: "sec_data_1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := convertScopeToOutput(&tt.scope)

			assert.Equal(t, tt.expected.Level, output.Level)
			if tt.expected.Pillar != nil {
				require.NotNil(t, output.Pillar)
				assert.Equal(t, *tt.expected.Pillar, *output.Pillar)
			} else {
				assert.Nil(t, output.Pillar)
			}
			assert.Equal(t, tt.expected.QuestionID, output.QuestionID)
		})
	}
}

func TestJSONOutputValidity(t *testing.T) {
	// Test that all output structures can be marshaled to valid JSON
	now := time.Now()
	pillar := PillarSecurity

	session := &ReviewSession{
		SessionID:     "session-123",
		WorkloadID:    "workload-456",
		AWSWorkloadID: "aws-wl-789",
		Status:        SessionStatusCompleted,
		CreatedAt:     now,
		Scope: ReviewScope{
			Level:  ScopeLevelPillar,
			Pillar: &pillar,
		},
		Results: &ReviewResults{
			Summary: &ResultsSummary{
				QuestionsEvaluated: 5,
				HighRisks:          1,
			},
		},
	}

	tests := []struct {
		name   string
		output interface{}
	}{
		{
			name:   "ReviewOutput",
			output: ConvertReviewSessionToOutput(session),
		},
		{
			name:   "StatusOutput",
			output: ConvertReviewSessionToStatusOutput(session),
		},
		{
			name:   "ResultsOutput",
			output: ConvertReviewSessionToResultsOutput(session),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.output)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Verify it's valid JSON by unmarshaling
			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			// Verify required fields exist
			assert.Contains(t, result, "session_id")
			assert.Contains(t, result, "workload_id")
			assert.Contains(t, result, "status")
		})
	}
}
