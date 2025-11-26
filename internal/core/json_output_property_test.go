package core

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: waffle-automated-wafr, Property 23: JSON output validity**
// **Validates: Requirements 10.4**
//
// Property: For any completed CLI review, the JSON output should be valid and parseable
//
// This property ensures that all JSON output from CLI commands is:
// 1. Valid JSON that can be parsed
// 2. Contains required fields (session_id, workload_id, status)
// 3. Has correct data types for all fields
// 4. Can be round-tripped (marshaled and unmarshaled)
func TestProperty_JSONOutputValidity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("all review outputs produce valid JSON", prop.ForAll(
		func(sessionID, workloadID string, status SessionStatus) bool {
			session := &ReviewSession{
				SessionID:  sessionID,
				WorkloadID: workloadID,
				Status:     status,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				Scope:      ReviewScope{Level: ScopeLevelWorkload},
				Results: &ReviewResults{
					Summary: &ResultsSummary{
						QuestionsEvaluated: 5,
						HighRisks:          1,
						MediumRisks:        2,
						AverageConfidence:  0.85,
					},
				},
			}

			// Convert to output
			output := ConvertReviewSessionToOutput(session)

			// Marshal to JSON
			var buf bytes.Buffer
			if err := WriteJSON(&buf, output); err != nil {
				t.Logf("Failed to write JSON: %v", err)
				return false
			}

			// Verify it's valid JSON by unmarshaling
			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Logf("Failed to parse JSON: %v", err)
				return false
			}

			// Verify required fields exist
			if _, ok := result["session_id"]; !ok {
				t.Logf("Missing session_id field")
				return false
			}
			if _, ok := result["workload_id"]; !ok {
				t.Logf("Missing workload_id field")
				return false
			}
			if _, ok := result["status"]; !ok {
				t.Logf("Missing status field")
				return false
			}
			if _, ok := result["created_at"]; !ok {
				t.Logf("Missing created_at field")
				return false
			}

			// Verify session_id matches
			if result["session_id"] != sessionID {
				t.Logf("session_id mismatch: expected %s, got %v", sessionID, result["session_id"])
				return false
			}

			// Verify workload_id matches
			if result["workload_id"] != workloadID {
				t.Logf("workload_id mismatch: expected %s, got %v", workloadID, result["workload_id"])
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		genSessionStatus(),
	))

	properties.Property("all status outputs produce valid JSON", prop.ForAll(
		func(sessionID, workloadID string, status SessionStatus) bool {
			session := &ReviewSession{
				SessionID:  sessionID,
				WorkloadID: workloadID,
				Status:     status,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Convert to status output
			output := ConvertReviewSessionToStatusOutput(session)

			// Marshal to JSON
			var buf bytes.Buffer
			if err := WriteJSON(&buf, output); err != nil {
				t.Logf("Failed to write JSON: %v", err)
				return false
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Logf("Failed to parse JSON: %v", err)
				return false
			}

			// Verify required fields
			requiredFields := []string{"session_id", "workload_id", "status", "created_at", "updated_at"}
			for _, field := range requiredFields {
				if _, ok := result[field]; !ok {
					t.Logf("Missing required field: %s", field)
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		genSessionStatus(),
	))

	properties.Property("all results outputs produce valid JSON", prop.ForAll(
		func(sessionID, workloadID string, scope ReviewScope) bool {
			session := &ReviewSession{
				SessionID:  sessionID,
				WorkloadID: workloadID,
				Status:     SessionStatusCompleted,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				Scope:      scope,
				Results: &ReviewResults{
					Summary: &ResultsSummary{
						QuestionsEvaluated: 5,
						HighRisks:          1,
						MediumRisks:        2,
						AverageConfidence:  0.85,
					},
				},
			}

			// Convert to results output
			output := ConvertReviewSessionToResultsOutput(session)

			// Marshal to JSON
			var buf bytes.Buffer
			if err := WriteJSON(&buf, output); err != nil {
				t.Logf("Failed to write JSON: %v", err)
				return false
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Logf("Failed to parse JSON: %v", err)
				return false
			}

			// Verify required fields
			requiredFields := []string{"session_id", "workload_id", "status", "scope", "summary"}
			for _, field := range requiredFields {
				if _, ok := result[field]; !ok {
					t.Logf("Missing required field: %s", field)
					return false
				}
			}

			// Verify scope structure
			scopeResult, ok := result["scope"].(map[string]interface{})
			if !ok {
				t.Logf("scope is not an object")
				return false
			}
			if _, ok := scopeResult["level"]; !ok {
				t.Logf("scope missing level field")
				return false
			}

			// Verify summary structure
			summary, ok := result["summary"].(map[string]interface{})
			if !ok {
				t.Logf("summary is not an object")
				return false
			}
			summaryFields := []string{"questions_evaluated", "high_risks", "medium_risks", "average_confidence", "improvement_plan_size"}
			for _, field := range summaryFields {
				if _, ok := summary[field]; !ok {
					t.Logf("summary missing field: %s", field)
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		genReviewScope(),
	))

	properties.Property("JSON output can be round-tripped", prop.ForAll(
		func(sessionID, workloadID string, status SessionStatus) bool {
			session := &ReviewSession{
				SessionID:  sessionID,
				WorkloadID: workloadID,
				Status:     status,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Convert to output
			output := ConvertReviewSessionToOutput(session)

			// Marshal to JSON
			data, err := json.Marshal(output)
			if err != nil {
				t.Logf("Failed to marshal: %v", err)
				return false
			}

			// Unmarshal back
			var roundTripped ReviewOutput
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Logf("Failed to unmarshal: %v", err)
				return false
			}

			// Verify key fields match
			if roundTripped.SessionID != output.SessionID {
				t.Logf("session_id mismatch after round-trip")
				return false
			}
			if roundTripped.WorkloadID != output.WorkloadID {
				t.Logf("workload_id mismatch after round-trip")
				return false
			}
			if roundTripped.Status != output.Status {
				t.Logf("status mismatch after round-trip")
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		genSessionStatus(),
	))

	properties.TestingRun(t)
}

// genReviewSession generates random ReviewSession instances for property testing
func genReviewSession() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		genSessionStatus(),
		genReviewScope(),
		genWorkloadModel(),
		genReviewResults(),
	).Map(func(vals []interface{}) *ReviewSession {
		return &ReviewSession{
			SessionID:     vals[0].(string),
			WorkloadID:    vals[1].(string),
			AWSWorkloadID: vals[2].(string),
			Status:        vals[3].(SessionStatus),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Scope:         vals[4].(ReviewScope),
			WorkloadModel: vals[5].(*WorkloadModel),
			Results:       vals[6].(*ReviewResults),
		}
	})
}

func genWorkloadModel() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOf(genResource()),
		gen.Identifier(),
	).Map(func(vals []interface{}) *WorkloadModel {
		return &WorkloadModel{
			Resources:  vals[0].([]Resource),
			Framework:  vals[1].(string),
			SourceType: "terraform",
		}
	})
}

func genResource() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
		gen.Bool(),
	).Map(func(vals []interface{}) Resource {
		return Resource{
			ID:         vals[0].(string),
			Type:       vals[1].(string),
			Address:    vals[2].(string),
			IsFromPlan: vals[3].(bool),
			Properties: map[string]interface{}{
				"name": vals[0].(string),
			},
		}
	})
}

func genReviewResults() gopter.Gen {
	return genResultsSummary().Map(func(summary interface{}) *ReviewResults {
		return &ReviewResults{
			Evaluations:     []*QuestionEvaluation{},
			Risks:           []*Risk{},
			ImprovementPlan: &ImprovementPlan{Items: []*ImprovementPlanItem{}},
			Summary:         summary.(*ResultsSummary),
		}
	})
}

func genQuestionEvaluation() gopter.Gen {
	return gopter.CombineGens(
		genWAFRQuestion(),
		gen.SliceOf(genChoice()),
		gen.SliceOf(genEvidence()),
		gen.Float64Range(0.0, 1.0),
		gen.Identifier(),
	).Map(func(vals []interface{}) *QuestionEvaluation {
		return &QuestionEvaluation{
			Question:        vals[0].(*WAFRQuestion),
			SelectedChoices: vals[1].([]Choice),
			Evidence:        vals[2].([]Evidence),
			ConfidenceScore: vals[3].(float64),
			Notes:           vals[4].(string),
		}
	})
}

func genWAFRQuestion() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		genPillar(),
		gen.Identifier(),
		gen.Identifier(),
	).Map(func(vals []interface{}) *WAFRQuestion {
		return &WAFRQuestion{
			ID:          vals[0].(string),
			Pillar:      vals[1].(Pillar),
			Title:       vals[2].(string),
			Description: vals[3].(string),
		}
	})
}



func genChoice() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.Identifier(),
	).Map(func(vals []interface{}) Choice {
		return Choice{
			ID:          vals[0].(string),
			Title:       vals[1].(string),
			Description: vals[2].(string),
		}
	})
}

func genEvidence() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.SliceOf(gen.Identifier()),
		gen.Float64Range(0.0, 1.0),
	).Map(func(vals []interface{}) Evidence {
		return Evidence{
			ChoiceID:    vals[0].(string),
			Explanation: vals[1].(string),
			Resources:   vals[2].([]string),
			Confidence:  vals[3].(float64),
		}
	})
}

func genRisk() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		genWAFRQuestion(),
		genPillar(),
		genRiskLevel(),
		gen.Identifier(),
		gen.SliceOf(gen.Identifier()),
	).Map(func(vals []interface{}) *Risk {
		return &Risk{
			ID:                vals[0].(string),
			Question:          vals[1].(*WAFRQuestion),
			Pillar:            vals[2].(Pillar),
			Severity:          vals[3].(RiskLevel),
			Description:       vals[4].(string),
			AffectedResources: vals[5].([]string),
		}
	})
}

func genRiskLevel() gopter.Gen {
	return gen.OneConstOf(
		RiskLevelNone,
		RiskLevelMedium,
		RiskLevelHigh,
	)
}

func genImprovementPlan() gopter.Gen {
	return gen.SliceOf(genImprovementPlanItem(), reflect.TypeOf(&ImprovementPlanItem{})).Map(func(items interface{}) *ImprovementPlan {
		return &ImprovementPlan{
			Items: items.([]*ImprovementPlanItem),
		}
	})
}

func genImprovementPlanItem() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.SliceOf(gen.Identifier()),
		gen.SliceOf(gen.Identifier()),
		gen.IntRange(1, 10),
		gen.OneConstOf("LOW", "MEDIUM", "HIGH"),
	).Map(func(vals []interface{}) *ImprovementPlanItem {
		return &ImprovementPlanItem{
			ID:                vals[0].(string),
			Description:       vals[1].(string),
			BestPracticeRefs:  vals[2].([]string),
			AffectedResources: vals[3].([]string),
			Priority:          vals[4].(int),
			EstimatedEffort:   vals[5].(string),
		}
	})
}

func genResultsSummary() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
		gen.IntRange(0, 50),
		gen.IntRange(0, 50),
		gen.Float64Range(0.0, 1.0),
		gen.IntRange(0, 20),
	).Map(func(vals []interface{}) *ResultsSummary {
		return &ResultsSummary{
			TotalQuestions:      vals[0].(int),
			QuestionsEvaluated:  vals[1].(int),
			HighRisks:           vals[2].(int),
			MediumRisks:         vals[3].(int),
			AverageConfidence:   vals[4].(float64),
			ImprovementPlanSize: vals[5].(int),
		}
	})
}

// TestProperty_JSONOutputValidity_MinimalSession tests with minimal session data
func TestProperty_JSONOutputValidity_MinimalSession(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("minimal sessions produce valid JSON", prop.ForAll(
		func(sessionID, workloadID string, status SessionStatus) bool {
			session := &ReviewSession{
				SessionID:  sessionID,
				WorkloadID: workloadID,
				Status:     status,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Test all output types
			outputs := []interface{}{
				ConvertReviewSessionToOutput(session),
				ConvertReviewSessionToStatusOutput(session),
				ConvertReviewSessionToResultsOutput(session),
			}

			for _, output := range outputs {
				data, err := json.Marshal(output)
				if err != nil {
					t.Logf("Failed to marshal: %v", err)
					return false
				}

				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Logf("Failed to unmarshal: %v", err)
					return false
				}

				// Verify basic fields
				if result["session_id"] != sessionID {
					return false
				}
				if result["workload_id"] != workloadID {
					return false
				}
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		genSessionStatus(),
	))

	properties.TestingRun(t)
}
