package main

import (
	"context"

	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/wafr"
)

// WAFREvaluatorAdapter adapts wafr.Evaluator to core.WAFREvaluator interface
type WAFREvaluatorAdapter struct {
	evaluator      *wafr.Evaluator
	bedrockClient  wafr.BedrockClient
	workloadModels map[string]*core.WorkloadModel // Map AWS workload ID to workload model
}

// NewWAFREvaluatorAdapter creates a new adapter
func NewWAFREvaluatorAdapter(evaluator *wafr.Evaluator, bedrockClient wafr.BedrockClient) *WAFREvaluatorAdapter {
	return &WAFREvaluatorAdapter{
		evaluator:      evaluator,
		bedrockClient:  bedrockClient,
		workloadModels: make(map[string]*core.WorkloadModel),
	}
}

// SetWorkloadModel stores the workload model for a given AWS workload ID
func (a *WAFREvaluatorAdapter) SetWorkloadModel(awsWorkloadID string, model *core.WorkloadModel) {
	a.workloadModels[awsWorkloadID] = model
}

// CreateWorkload creates a workload in AWS Well-Architected Tool
func (a *WAFREvaluatorAdapter) CreateWorkload(
	ctx context.Context,
	workloadID string,
	description string,
) (string, error) {
	return a.evaluator.CreateWorkload(ctx, workloadID, description)
}

// GetQuestions retrieves WAFR questions based on scope
func (a *WAFREvaluatorAdapter) GetQuestions(
	ctx context.Context,
	awsWorkloadID string,
	scope core.ReviewScope,
) ([]*core.WAFRQuestion, error) {
	return a.evaluator.GetQuestions(ctx, awsWorkloadID, scope)
}

// EvaluateQuestion evaluates a single question against the workload
func (a *WAFREvaluatorAdapter) EvaluateQuestion(
	ctx context.Context,
	question *core.WAFRQuestion,
	workloadModel *core.WorkloadModel,
) (*core.QuestionEvaluation, error) {
	return a.evaluator.EvaluateQuestion(ctx, question, workloadModel, a.bedrockClient)
}

// SubmitAnswer submits an answer to AWS Well-Architected Tool
func (a *WAFREvaluatorAdapter) SubmitAnswer(
	ctx context.Context,
	awsWorkloadID string,
	questionID string,
	evaluation *core.QuestionEvaluation,
) error {
	return a.evaluator.SubmitAnswer(ctx, awsWorkloadID, questionID, evaluation)
}

// GetImprovementPlan retrieves the improvement plan from AWS
func (a *WAFREvaluatorAdapter) GetImprovementPlan(
	ctx context.Context,
	awsWorkloadID string,
) (*core.ImprovementPlan, error) {
	// Get the workload model for this AWS workload ID (may be nil)
	workloadModel := a.workloadModels[awsWorkloadID]
	// Pass the workload model to the evaluator (it can handle nil)
	return a.evaluator.GetImprovementPlan(ctx, awsWorkloadID, workloadModel)
}

// CreateMilestone creates a milestone in AWS
func (a *WAFREvaluatorAdapter) CreateMilestone(
	ctx context.Context,
	awsWorkloadID string,
	milestoneName string,
) (string, error) {
	return a.evaluator.CreateMilestone(ctx, awsWorkloadID, milestoneName)
}
