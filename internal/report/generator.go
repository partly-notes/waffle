package report

import (
	"context"
	"encoding/json"

	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/wafr"
)

// Generator implements the ReportGenerator interface
type Generator struct {
	evaluator *wafr.Evaluator
}

// NewGeneratorWithEvaluator creates a new report generator with a WAFR evaluator
func NewGeneratorWithEvaluator(evaluator *wafr.Evaluator) *Generator {
	return &Generator{
		evaluator: evaluator,
	}
}

// NewGenerator creates a new report generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GetConsolidatedReport retrieves a consolidated report from AWS
func (g *Generator) GetConsolidatedReport(
	ctx context.Context,
	awsWorkloadID string,
	format core.ReportFormat,
) ([]byte, error) {
	// TODO: Implement in task 19
	return nil, nil
}

// GetResultsJSON retrieves results in JSON format with IaC evidence
func (g *Generator) GetResultsJSON(
	ctx context.Context,
	awsWorkloadID string,
	session *core.ReviewSession,
) (map[string]interface{}, error) {
	if session == nil {
		return nil, core.ErrSessionNotFound
	}
	
	if session.Results == nil {
		return nil, core.ErrInvalidSessionStatus
	}
	
	// Build the results JSON structure
	results := map[string]interface{}{
		"session_id":       session.SessionID,
		"workload_id":      session.WorkloadID,
		"aws_workload_id":  session.AWSWorkloadID,
		"milestone_id":     session.MilestoneID,
		"status":           string(session.Status),
		"created_at":       session.CreatedAt,
		"updated_at":       session.UpdatedAt,
		"summary": map[string]interface{}{
			"questions_evaluated":  session.Results.Summary.QuestionsEvaluated,
			"high_risks":           session.Results.Summary.HighRisks,
			"medium_risks":         session.Results.Summary.MediumRisks,
			"average_confidence":   session.Results.Summary.AverageConfidence,
			"improvement_plan_size": session.Results.Summary.ImprovementPlanSize,
		},
	}
	
	// Add improvement plan if available
	if session.Results.ImprovementPlan != nil {
		improvementItems := make([]map[string]interface{}, 0, len(session.Results.ImprovementPlan.Items))
		
		for _, item := range session.Results.ImprovementPlan.Items {
			improvementItem := map[string]interface{}{
				"id":                  item.ID,
				"description":         item.Description,
				"priority":            item.Priority,
				"estimated_effort":    item.EstimatedEffort,
				"best_practice_refs":  item.BestPracticeRefs,
				"affected_resources":  item.AffectedResources,
			}
			
			// Add risk details
			if item.Risk != nil {
				improvementItem["risk"] = map[string]interface{}{
					"id":          item.Risk.ID,
					"pillar":      item.Risk.Pillar,
					"severity":    item.Risk.Severity,
					"description": item.Risk.Description,
				}
				
				// Add question details
				if item.Risk.Question != nil {
					improvementItem["question"] = map[string]interface{}{
						"id":    item.Risk.Question.ID,
						"title": item.Risk.Question.Title,
					}
				}
				
				// Add missing best practices
				if len(item.Risk.MissingBestPractices) > 0 {
					bestPractices := make([]map[string]interface{}, 0, len(item.Risk.MissingBestPractices))
					for _, bp := range item.Risk.MissingBestPractices {
						bestPractices = append(bestPractices, map[string]interface{}{
							"id":          bp.ID,
							"title":       bp.Title,
							"description": bp.Description,
						})
					}
					improvementItem["missing_best_practices"] = bestPractices
				}
			}
			
			improvementItems = append(improvementItems, improvementItem)
		}
		
		results["improvement_plan"] = map[string]interface{}{
			"items": improvementItems,
			"total": len(improvementItems),
		}
	}
	
	// Add evaluations summary (without full details to keep response manageable)
	if len(session.Results.Evaluations) > 0 {
		evaluationsSummary := make([]map[string]interface{}, 0, len(session.Results.Evaluations))
		
		for _, eval := range session.Results.Evaluations {
			evalSummary := map[string]interface{}{
				"question_id":      eval.Question.ID,
				"question_title":   eval.Question.Title,
				"pillar":           eval.Question.Pillar,
				"choices_count":    len(eval.SelectedChoices),
				"evidence_count":   len(eval.Evidence),
				"confidence_score": eval.ConfidenceScore,
			}
			evaluationsSummary = append(evaluationsSummary, evalSummary)
		}
		
		results["evaluations"] = evaluationsSummary
	}
	
	return results, nil
}

// milestoneChange is a local struct to unmarshal changes from the result map
type milestoneChange struct {
	Type        string `json:"type"`
	QuestionID  string `json:"question_id"`
	Description string `json:"description"`
	Severity    string `json:"severity,omitempty"`
}

// CompareMilestones compares two milestones
func (g *Generator) CompareMilestones(
	ctx context.Context,
	awsWorkloadID string,
	milestoneID1 string,
	milestoneID2 string,
) (*core.MilestoneComparison, error) {
	if g.evaluator == nil {
		return nil, core.ErrEvaluatorNotInitialized
	}

	// Call the evaluator's CompareMilestones method
	result, err := g.evaluator.CompareMilestones(ctx, awsWorkloadID, milestoneID1, milestoneID2)
	if err != nil {
		return nil, err
	}

	// Convert the map result to MilestoneComparison struct
	comparison := &core.MilestoneComparison{
		Milestone1: milestoneID1,
		Milestone2: milestoneID2,
	}

	// Extract changes from the result using JSON marshaling for type safety
	if changesRaw, ok := result["changes"]; ok {
		// Marshal and unmarshal to convert to our local type
		changesJSON, err := json.Marshal(changesRaw)
		if err == nil {
			var changes []milestoneChange
			if err := json.Unmarshal(changesJSON, &changes); err == nil {
				for _, change := range changes {
					switch change.Type {
					case "improvement":
						comparison.Improvements = append(comparison.Improvements, change.Description)
					case "regression":
						comparison.Regressions = append(comparison.Regressions, change.Description)
					case "new_risk":
						comparison.NewRisks = append(comparison.NewRisks, change.Description)
					case "resolved_risk":
						comparison.ResolvedRisks = append(comparison.ResolvedRisks, change.Description)
					}
				}
			}
		}
	}

	return comparison, nil
}
