package wafr

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waffle/waffle/internal/core"
)

// TestScopeFiltering_WorkloadScope tests that workload scope processes all pillars and questions
// Validates: Requirements 9.2
func TestScopeFiltering_WorkloadScope(t *testing.T) {
	// Track which pillars were requested
	requestedPillars := make(map[string]bool)
	
	mockClient := &MockWAFRClient{
		ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
			pillarID := aws.ToString(params.PillarId)
			requestedPillars[pillarID] = true
			
			// Return one question per pillar
			return &wellarchitected.ListAnswersOutput{
				AnswerSummaries: []types.AnswerSummary{
					{
						QuestionId:    aws.String(pillarID + "-q1"),
						QuestionTitle: aws.String("Question for " + pillarID),
						Choices: []types.Choice{
							{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
						},
					},
				},
			}, nil
		},
	}

	evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

	scope := core.ReviewScope{
		Level: core.ScopeLevelWorkload,
	}

	questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

	require.NoError(t, err)
	
	// Verify all 6 pillars were requested
	expectedPillars := []string{
		"operationalExcellence",
		"security",
		"reliability",
		"performance",
		"costOptimization",
		"sustainability",
	}
	
	for _, pillar := range expectedPillars {
		assert.True(t, requestedPillars[pillar], "Pillar %s should have been requested", pillar)
	}
	
	// Verify we got questions from all pillars
	assert.Len(t, questions, 6, "Should have questions from all 6 pillars")
	
	// Verify each question has the correct pillar
	pillarCounts := make(map[core.Pillar]int)
	for _, q := range questions {
		pillarCounts[q.Pillar]++
	}
	
	assert.Equal(t, 1, pillarCounts[core.PillarOperationalExcellence])
	assert.Equal(t, 1, pillarCounts[core.PillarSecurity])
	assert.Equal(t, 1, pillarCounts[core.PillarReliability])
	assert.Equal(t, 1, pillarCounts[core.PillarPerformanceEfficiency])
	assert.Equal(t, 1, pillarCounts[core.PillarCostOptimization])
	assert.Equal(t, 1, pillarCounts[core.PillarSustainability])
}

// TestScopeFiltering_PillarScope tests that pillar scope processes only the specified pillar
// Validates: Requirements 9.3
func TestScopeFiltering_PillarScope(t *testing.T) {
	tests := []struct {
		name           string
		pillar         core.Pillar
		expectedPillar string
	}{
		{
			name:           "security pillar",
			pillar:         core.PillarSecurity,
			expectedPillar: "security",
		},
		{
			name:           "reliability pillar",
			pillar:         core.PillarReliability,
			expectedPillar: "reliability",
		},
		{
			name:           "operational excellence pillar",
			pillar:         core.PillarOperationalExcellence,
			expectedPillar: "operationalExcellence",
		},
		{
			name:           "performance efficiency pillar",
			pillar:         core.PillarPerformanceEfficiency,
			expectedPillar: "performance",
		},
		{
			name:           "cost optimization pillar",
			pillar:         core.PillarCostOptimization,
			expectedPillar: "costOptimization",
		},
		{
			name:           "sustainability pillar",
			pillar:         core.PillarSustainability,
			expectedPillar: "sustainability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestedPillars := make([]string, 0)
			
			mockClient := &MockWAFRClient{
				ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
					pillarID := aws.ToString(params.PillarId)
					requestedPillars = append(requestedPillars, pillarID)
					
					// Return multiple questions for the pillar
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String(pillarID + "-q1"),
								QuestionTitle: aws.String("Question 1 for " + pillarID),
								Choices: []types.Choice{
									{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
								},
							},
							{
								QuestionId:    aws.String(pillarID + "-q2"),
								QuestionTitle: aws.String("Question 2 for " + pillarID),
								Choices: []types.Choice{
									{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
								},
							},
						},
					}, nil
				},
			}

			evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

			scope := core.ReviewScope{
				Level:  core.ScopeLevelPillar,
				Pillar: &tt.pillar,
			}

			questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

			require.NoError(t, err)
			
			// Verify only the specified pillar was requested
			assert.Len(t, requestedPillars, 1, "Should only request one pillar")
			assert.Equal(t, tt.expectedPillar, requestedPillars[0], "Should request the correct pillar")
			
			// Verify all questions are from the specified pillar
			assert.Len(t, questions, 2, "Should have 2 questions from the pillar")
			for _, q := range questions {
				assert.Equal(t, tt.pillar, q.Pillar, "All questions should be from the specified pillar")
			}
		})
	}
}

// TestScopeFiltering_QuestionScope tests that question scope processes only the specified question
// Validates: Requirements 9.4
func TestScopeFiltering_QuestionScope(t *testing.T) {
	tests := []struct {
		name       string
		questionID string
		pillar     core.Pillar
		pillarID   string
	}{
		{
			name:       "security question",
			questionID: "sec-data-1",
			pillar:     core.PillarSecurity,
			pillarID:   "security",
		},
		{
			name:       "reliability question",
			questionID: "rel-backup-1",
			pillar:     core.PillarReliability,
			pillarID:   "reliability",
		},
		{
			name:       "operational excellence question",
			questionID: "ops-monitor-1",
			pillar:     core.PillarOperationalExcellence,
			pillarID:   "operationalExcellence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
					pillarID := aws.ToString(params.PillarId)
					
					// Only return the target question in the correct pillar
					if pillarID == tt.pillarID {
						return &wellarchitected.ListAnswersOutput{
							AnswerSummaries: []types.AnswerSummary{
								{
									QuestionId:    aws.String(tt.questionID),
									QuestionTitle: aws.String("Target Question"),
									Choices: []types.Choice{
										{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
									},
								},
								{
									QuestionId:    aws.String(pillarID + "-other-1"),
									QuestionTitle: aws.String("Other Question"),
									Choices: []types.Choice{
										{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
									},
								},
							},
						}, nil
					}
					
					// Return other questions for other pillars
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String(pillarID + "-other-1"),
								QuestionTitle: aws.String("Other Question"),
								Choices: []types.Choice{
									{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
								},
							},
						},
					}, nil
				},
			}

			evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

			scope := core.ReviewScope{
				Level:      core.ScopeLevelQuestion,
				QuestionID: tt.questionID,
			}

			questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

			require.NoError(t, err)
			
			// Verify only one question was returned
			assert.Len(t, questions, 1, "Should only return the specified question")
			
			// Verify it's the correct question
			assert.Equal(t, tt.questionID, questions[0].ID, "Should return the correct question ID")
			assert.Equal(t, tt.pillar, questions[0].Pillar, "Should have the correct pillar")
		})
	}
}

// TestScopeFiltering_QuestionNotFound tests that question scope handles missing questions
func TestScopeFiltering_QuestionNotFound(t *testing.T) {
	mockClient := &MockWAFRClient{
		ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
			// Return questions that don't match the target
			return &wellarchitected.ListAnswersOutput{
				AnswerSummaries: []types.AnswerSummary{
					{
						QuestionId:    aws.String("other-question"),
						QuestionTitle: aws.String("Other Question"),
						Choices: []types.Choice{
							{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
						},
					},
				},
			}, nil
		},
	}

	evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

	scope := core.ReviewScope{
		Level:      core.ScopeLevelQuestion,
		QuestionID: "nonexistent-question",
	}

	questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, questions)
}

// TestScopeValidation tests that scope parameters are validated
// Validates: Requirements 9.5
func TestScopeValidation(t *testing.T) {
	tests := []struct {
		name       string
		scope      core.ReviewScope
		wantErr    bool
		errMessage string
	}{
		{
			name: "valid workload scope",
			scope: core.ReviewScope{
				Level: core.ScopeLevelWorkload,
			},
			wantErr: false,
		},
		{
			name: "valid pillar scope",
			scope: core.ReviewScope{
				Level:  core.ScopeLevelPillar,
				Pillar: func() *core.Pillar { p := core.PillarSecurity; return &p }(),
			},
			wantErr: false,
		},
		{
			name: "valid question scope",
			scope: core.ReviewScope{
				Level:      core.ScopeLevelQuestion,
				QuestionID: "sec-1",
			},
			wantErr: false,
		},
		{
			name: "invalid pillar scope - missing pillar",
			scope: core.ReviewScope{
				Level: core.ScopeLevelPillar,
			},
			wantErr:    true,
			errMessage: "invalid scope",
		},
		{
			name: "invalid question scope - missing question ID",
			scope: core.ReviewScope{
				Level: core.ScopeLevelQuestion,
			},
			wantErr:    true,
			errMessage: "invalid scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockWAFRClient{
				ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
					// For question scope, return the question if it matches
					if tt.scope.Level == core.ScopeLevelQuestion && tt.scope.QuestionID != "" {
						return &wellarchitected.ListAnswersOutput{
							AnswerSummaries: []types.AnswerSummary{
								{
									QuestionId:    aws.String(tt.scope.QuestionID),
									QuestionTitle: aws.String("Test Question"),
									Choices: []types.Choice{
										{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
									},
								},
							},
						}, nil
					}
					
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{},
					}, nil
				},
			}

			evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

			_, err := evaluator.GetQuestions(context.Background(), "wl-123", tt.scope)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestScopeFiltering_ResultsLimitedToScope tests that results are limited to the selected scope
// Validates: Requirements 9.5
func TestScopeFiltering_ResultsLimitedToScope(t *testing.T) {
	t.Run("pillar scope returns only pillar questions", func(t *testing.T) {
		mockClient := &MockWAFRClient{
			ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				
				// Only return questions if the correct pillar is requested
				if pillarID != "security" {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{},
					}, nil
				}
				
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String("sec-1"),
							QuestionTitle: aws.String("Security Question 1"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
						{
							QuestionId:    aws.String("sec-2"),
							QuestionTitle: aws.String("Security Question 2"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
							},
						},
					},
				}, nil
			},
		}

		evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

		pillar := core.PillarSecurity
		scope := core.ReviewScope{
			Level:  core.ScopeLevelPillar,
			Pillar: &pillar,
		}

		questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

		require.NoError(t, err)
		assert.Len(t, questions, 2)
		
		// Verify all questions are from security pillar
		for _, q := range questions {
			assert.Equal(t, core.PillarSecurity, q.Pillar)
			assert.Contains(t, q.ID, "sec-")
		}
	})

	t.Run("question scope returns only specified question", func(t *testing.T) {
		targetQuestionID := "sec-data-1"
		
		mockClient := &MockWAFRClient{
			ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				return &wellarchitected.ListAnswersOutput{
					AnswerSummaries: []types.AnswerSummary{
						{
							QuestionId:    aws.String(targetQuestionID),
							QuestionTitle: aws.String("Target Question"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c1"), Title: aws.String("Choice 1")},
							},
						},
						{
							QuestionId:    aws.String("sec-other"),
							QuestionTitle: aws.String("Other Question"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
							},
						},
					},
				}, nil
			},
		}

		evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

		scope := core.ReviewScope{
			Level:      core.ScopeLevelQuestion,
			QuestionID: targetQuestionID,
		}

		questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

		require.NoError(t, err)
		assert.Len(t, questions, 1)
		assert.Equal(t, targetQuestionID, questions[0].ID)
	})
}

// TestScopeFiltering_EmptyWorkloadID tests validation of workload ID
func TestScopeFiltering_EmptyWorkloadID(t *testing.T) {
	mockClient := &MockWAFRClient{}
	evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

	scope := core.ReviewScope{
		Level: core.ScopeLevelWorkload,
	}

	questions, err := evaluator.GetQuestions(context.Background(), "", scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "AWS workload ID is required")
	assert.Nil(t, questions)
}

// TestScopeFiltering_PaginationHandling tests that pagination works correctly for all scope levels
func TestScopeFiltering_PaginationHandling(t *testing.T) {
	t.Run("workload scope with pagination", func(t *testing.T) {
		callCounts := make(map[string]int)
		
		mockClient := &MockWAFRClient{
			ListAnswersFunc: func(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error) {
				pillarID := aws.ToString(params.PillarId)
				callCounts[pillarID]++
				
				// Return paginated results for each pillar
				if params.NextToken == nil {
					return &wellarchitected.ListAnswersOutput{
						AnswerSummaries: []types.AnswerSummary{
							{
								QuestionId:    aws.String(pillarID + "-q1"),
								QuestionTitle: aws.String("Question 1"),
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
							QuestionId:    aws.String(pillarID + "-q2"),
							QuestionTitle: aws.String("Question 2"),
							Choices: []types.Choice{
								{ChoiceId: aws.String("c2"), Title: aws.String("Choice 2")},
							},
						},
					},
				}, nil
			},
		}

		evaluator := NewEvaluator(mockClient, DefaultEvaluatorConfig())

		scope := core.ReviewScope{
			Level: core.ScopeLevelWorkload,
		}

		questions, err := evaluator.GetQuestions(context.Background(), "wl-123", scope)

		require.NoError(t, err)
		
		// Should have 2 questions per pillar (6 pillars * 2 questions = 12 total)
		assert.Len(t, questions, 12)
		
		// Verify pagination was handled for all pillars
		assert.Equal(t, 2, callCounts["operationalExcellence"])
		assert.Equal(t, 2, callCounts["security"])
		assert.Equal(t, 2, callCounts["reliability"])
		assert.Equal(t, 2, callCounts["performance"])
		assert.Equal(t, 2, callCounts["costOptimization"])
		assert.Equal(t, 2, callCounts["sustainability"])
	})
}
