package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Engine implements the CoreEngine interface
type Engine struct {
	sessionManager SessionManager
	iacAnalyzer    IaCAnalyzer
	wafrEvaluator  WAFREvaluator
	bedrockClient  BedrockClient
	reportGen      ReportGenerator
}

// NewEngine creates a new core engine
func NewEngine(
	sessionManager SessionManager,
	iacAnalyzer IaCAnalyzer,
	wafrEvaluator WAFREvaluator,
	bedrockClient BedrockClient,
	reportGen ReportGenerator,
) *Engine {
	return &Engine{
		sessionManager: sessionManager,
		iacAnalyzer:    iacAnalyzer,
		wafrEvaluator:  wafrEvaluator,
		bedrockClient:  bedrockClient,
		reportGen:      reportGen,
	}
}

// InitiateReview starts a new WAFR review session
func (e *Engine) InitiateReview(
	ctx context.Context,
	workloadID string,
	scope ReviewScope,
) (*ReviewSession, error) {
	slog.InfoContext(ctx, "initiating review",
		"workload_id", workloadID,
		"scope_level", scope.Level,
	)

	// Validate inputs
	if workloadID == "" {
		return nil, ErrInvalidWorkloadID
	}

	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scope: %w", err)
	}

	// Create AWS workload
	slog.InfoContext(ctx, "creating AWS workload")
	awsWorkloadID, err := e.wafrEvaluator.CreateWorkload(ctx, workloadID, "Automated WAFR review")
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS workload: %w", err)
	}

	// Create session
	slog.InfoContext(ctx, "creating review session",
		"aws_workload_id", awsWorkloadID,
	)
	session, err := e.sessionManager.CreateSession(ctx, workloadID, scope, awsWorkloadID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Save initial session state
	if err := e.sessionManager.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	slog.InfoContext(ctx, "review initiated successfully",
		"session_id", session.SessionID,
		"aws_workload_id", awsWorkloadID,
	)

	return session, nil
}

// ExecuteReview executes the review workflow
func (e *Engine) ExecuteReview(ctx context.Context, session *ReviewSession) (*ReviewResults, error) {
	return e.ExecuteReviewWithProgress(ctx, session, nil)
}

// ExecuteReviewWithProgress executes the review workflow with progress reporting
func (e *Engine) ExecuteReviewWithProgress(ctx context.Context, session *ReviewSession, progress ProgressReporter) (*ReviewResults, error) {
	slog.InfoContext(ctx, "starting review execution",
		"session_id", session.SessionID,
		"workload_id", session.WorkloadID,
	)

	// Update session status to in progress
	session.Status = SessionStatusInProgress
	session.UpdatedAt = time.Now()
	if err := e.sessionManager.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session status: %w", err)
	}

	// Execute workflow with checkpoint handling
	results, err := e.executeWorkflowWithProgress(ctx, session, progress)
	if err != nil {
		session.Status = SessionStatusFailed
		session.UpdatedAt = time.Now()
		if saveErr := e.sessionManager.SaveSession(ctx, session); saveErr != nil {
			slog.ErrorContext(ctx, "failed to save failed session state",
				"session_id", session.SessionID,
				"error", saveErr,
			)
		}
		return nil, err
	}

	// Update session with results
	session.Results = results
	session.Status = SessionStatusCompleted
	session.UpdatedAt = time.Now()
	if err := e.sessionManager.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save completed session: %w", err)
	}

	slog.InfoContext(ctx, "review execution completed",
		"session_id", session.SessionID,
		"questions_evaluated", len(results.Evaluations),
		"risks_identified", len(results.Risks),
	)

	// Report completion
	if progress != nil {
		progress.ReportCompletion(results.Summary)
	}

	return results, nil
}

// executeWorkflow executes the main workflow with checkpoint support
func (e *Engine) executeWorkflow(ctx context.Context, session *ReviewSession) (*ReviewResults, error) {
	return e.executeWorkflowWithProgress(ctx, session, nil)
}

// executeWorkflowWithProgress executes the main workflow with checkpoint support and progress reporting
func (e *Engine) executeWorkflowWithProgress(ctx context.Context, session *ReviewSession, progress ProgressReporter) (*ReviewResults, error) {
	// Step 1: IaC Analysis (checkpoint: iac_analysis_complete)
	if session.Checkpoint == "" || session.Checkpoint == "created" {
		slog.InfoContext(ctx, "step 1: analyzing IaC")
		if progress != nil {
			progress.ReportStep("iac_analysis", "Analyzing infrastructure-as-code files...")
		}
		if err := e.analyzeIaC(ctx, session); err != nil {
			return nil, fmt.Errorf("IaC analysis failed: %w", err)
		}
		session.Checkpoint = "iac_analysis_complete"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Step 2: Get WAFR questions (checkpoint: questions_retrieved)
	var questions []*WAFRQuestion
	if session.Checkpoint == "iac_analysis_complete" {
		slog.InfoContext(ctx, "step 2: retrieving WAFR questions")
		if progress != nil {
			progress.ReportStep("retrieve_questions", "Retrieving WAFR questions from AWS...")
		}
		var err error
		questions, err = e.wafrEvaluator.GetQuestions(ctx, session.AWSWorkloadID, session.Scope)
		if err != nil {
			return nil, fmt.Errorf("failed to get questions: %w", err)
		}
		slog.InfoContext(ctx, "retrieved questions", "count", len(questions))
		if progress != nil {
			progress.ReportProgress(len(questions), len(questions), fmt.Sprintf("Retrieved %d questions", len(questions)))
		}
		session.Checkpoint = "questions_retrieved"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Step 3: Evaluate questions (checkpoint: questions_evaluated)
	var evaluations []*QuestionEvaluation
	if session.Checkpoint == "questions_retrieved" {
		slog.InfoContext(ctx, "step 3: evaluating questions")
		if progress != nil {
			progress.ReportStep("evaluate_questions", "Evaluating questions using Bedrock...")
		}
		var err error
		evaluations, err = e.evaluateQuestionsWithProgress(ctx, session, questions, progress)
		if err != nil {
			return nil, fmt.Errorf("question evaluation failed: %w", err)
		}
		session.Checkpoint = "questions_evaluated"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Step 4: Submit answers (checkpoint: answers_submitted)
	if session.Checkpoint == "questions_evaluated" {
		slog.InfoContext(ctx, "step 4: submitting answers to AWS")
		if progress != nil {
			progress.ReportStep("submit_answers", "Submitting answers to AWS Well-Architected Tool...")
		}
		if err := e.submitAnswersWithProgress(ctx, session, evaluations, progress); err != nil {
			return nil, fmt.Errorf("answer submission failed: %w", err)
		}
		session.Checkpoint = "answers_submitted"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Step 5: Get improvement plan (checkpoint: improvement_plan_retrieved)
	var improvementPlan *ImprovementPlan
	if session.Checkpoint == "answers_submitted" {
		slog.InfoContext(ctx, "step 5: retrieving improvement plan")
		if progress != nil {
			progress.ReportStep("improvement_plan", "Retrieving improvement plan from AWS...")
		}
		var err error
		improvementPlan, err = e.wafrEvaluator.GetImprovementPlan(ctx, session.AWSWorkloadID)
		if err != nil {
			slog.WarnContext(ctx, "failed to get improvement plan, continuing",
				"error", err,
			)
			// Continue with empty improvement plan
			improvementPlan = &ImprovementPlan{Items: []*ImprovementPlanItem{}}
		}
		session.Checkpoint = "improvement_plan_retrieved"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Step 6: Create milestone (checkpoint: milestone_created)
	if session.Checkpoint == "improvement_plan_retrieved" {
		slog.InfoContext(ctx, "step 6: creating milestone")
		if progress != nil {
			progress.ReportStep("create_milestone", "Creating milestone in AWS...")
		}
		milestoneName := fmt.Sprintf("waffle-%s", time.Now().Format("2006-01-02-15-04-05"))
		milestoneID, err := e.wafrEvaluator.CreateMilestone(ctx, session.AWSWorkloadID, milestoneName)
		if err != nil {
			slog.WarnContext(ctx, "failed to create milestone, continuing",
				"error", err,
			)
		} else {
			session.MilestoneID = milestoneID
			slog.InfoContext(ctx, "milestone created", "milestone_id", milestoneID)
		}
		session.Checkpoint = "milestone_created"
		if err := e.sessionManager.SaveSession(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to save checkpoint: %w", err)
		}
	}

	// Build results
	results := &ReviewResults{
		Evaluations:     evaluations,
		Risks:           e.extractRisks(evaluations),
		ImprovementPlan: improvementPlan,
		Summary:         e.buildSummary(evaluations, improvementPlan),
	}

	return results, nil
}

// analyzeIaC performs IaC analysis
func (e *Engine) analyzeIaC(ctx context.Context, session *ReviewSession) error {
	// Retrieve IaC files
	files, err := e.iacAnalyzer.RetrieveIaCFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve IaC files: %w", err)
	}

	// Validate Terraform files
	if err := e.iacAnalyzer.ValidateTerraformFiles(ctx, files); err != nil {
		return fmt.Errorf("terraform validation failed: %w", err)
	}

	var workloadModel *WorkloadModel

	// Parse Terraform plan if provided
	if session.PlanFilePath != "" {
		slog.InfoContext(ctx, "parsing terraform plan", "path", session.PlanFilePath)
		planModel, err := e.iacAnalyzer.ParseTerraformPlan(ctx, session.PlanFilePath)
		if err != nil {
			slog.WarnContext(ctx, "failed to parse terraform plan, falling back to HCL",
				"error", err,
			)
		} else {
			workloadModel = planModel
		}
	}

	// Parse Terraform HCL files
	slog.InfoContext(ctx, "parsing terraform HCL files")
	sourceModel, err := e.iacAnalyzer.ParseTerraform(ctx, files)
	if err != nil {
		return fmt.Errorf("failed to parse terraform: %w", err)
	}

	// Merge models if we have both
	if workloadModel != nil {
		slog.InfoContext(ctx, "merging plan and source models")
		mergedModel, err := e.iacAnalyzer.MergeWorkloadModels(ctx, workloadModel, sourceModel)
		if err != nil {
			slog.WarnContext(ctx, "failed to merge models, using plan model",
				"error", err,
			)
		} else {
			workloadModel = mergedModel
		}
	} else {
		workloadModel = sourceModel
	}

	// Extract resources
	resources, err := e.iacAnalyzer.ExtractResources(ctx, workloadModel)
	if err != nil {
		return fmt.Errorf("failed to extract resources: %w", err)
	}

	// Identify relationships
	relationships, err := e.iacAnalyzer.IdentifyRelationships(ctx, resources)
	if err != nil {
		return fmt.Errorf("failed to identify relationships: %w", err)
	}

	workloadModel.Resources = resources
	workloadModel.Relationships = relationships

	session.WorkloadModel = workloadModel
	slog.InfoContext(ctx, "IaC analysis complete",
		"resource_count", len(resources),
	)

	return nil
}

// evaluateQuestions evaluates all questions
func (e *Engine) evaluateQuestions(ctx context.Context, session *ReviewSession, questions []*WAFRQuestion) ([]*QuestionEvaluation, error) {
	return e.evaluateQuestionsWithProgress(ctx, session, questions, nil)
}

// evaluateQuestionsWithProgress evaluates all questions with progress reporting
func (e *Engine) evaluateQuestionsWithProgress(ctx context.Context, session *ReviewSession, questions []*WAFRQuestion, progress ProgressReporter) ([]*QuestionEvaluation, error) {
	evaluations := make([]*QuestionEvaluation, 0, len(questions))

	for i, question := range questions {
		// Only log detailed progress when no progress reporter is active
		if progress == nil {
			slog.InfoContext(ctx, "evaluating question",
				"question_id", question.ID,
				"progress", fmt.Sprintf("%d/%d", i+1, len(questions)),
			)
		}

		if progress != nil {
			progress.ReportProgress(i+1, len(questions), fmt.Sprintf("Evaluating question %d of %d", i+1, len(questions)))
		}

		evaluation, err := e.wafrEvaluator.EvaluateQuestion(ctx, question, session.WorkloadModel)
		if err != nil {
			slog.ErrorContext(ctx, "failed to evaluate question, continuing",
				"question_id", question.ID,
				"error", err,
			)
			// Continue with remaining questions
			continue
		}

		evaluations = append(evaluations, evaluation)

		// Log successful evaluation at debug level
		slog.DebugContext(ctx, "question evaluated",
			"question_id", question.ID,
			"choices_count", len(evaluation.SelectedChoices),
			"confidence", evaluation.ConfidenceScore,
		)
	}

	if len(evaluations) == 0 {
		return nil, fmt.Errorf("no questions were successfully evaluated")
	}

	return evaluations, nil
}

// submitAnswers submits all answers to AWS
func (e *Engine) submitAnswers(ctx context.Context, session *ReviewSession, evaluations []*QuestionEvaluation) error {
	return e.submitAnswersWithProgress(ctx, session, evaluations, nil)
}

// submitAnswersWithProgress submits all answers to AWS with progress reporting
func (e *Engine) submitAnswersWithProgress(ctx context.Context, session *ReviewSession, evaluations []*QuestionEvaluation, progress ProgressReporter) error {
	successCount := 0
	errorCount := 0

	for i, evaluation := range evaluations {
		// Only log detailed progress when no progress reporter is active
		if progress == nil {
			slog.InfoContext(ctx, "submitting answer",
				"question_id", evaluation.Question.ID,
				"progress", fmt.Sprintf("%d/%d", i+1, len(evaluations)),
			)
		}

		if progress != nil {
			progress.ReportProgress(i+1, len(evaluations), fmt.Sprintf("Submitting answer %d of %d", i+1, len(evaluations)))
		}

		err := e.wafrEvaluator.SubmitAnswer(ctx, session.AWSWorkloadID, evaluation.Question.ID, evaluation)
		if err != nil {
			slog.ErrorContext(ctx, "failed to submit answer, continuing",
				"question_id", evaluation.Question.ID,
				"error", err,
			)
			errorCount++
			// Continue with remaining answers
			continue
		}

		successCount++

		// Log successful submission at debug level
		slog.DebugContext(ctx, "answer submitted",
			"aws_workload_id", session.AWSWorkloadID,
			"question_id", evaluation.Question.ID,
			"choices_count", len(evaluation.SelectedChoices),
			"confidence", evaluation.ConfidenceScore,
		)
	}

	slog.InfoContext(ctx, "answer submission complete",
		"success", successCount,
		"errors", errorCount,
	)

	if successCount == 0 {
		return fmt.Errorf("failed to submit any answers")
	}

	return nil
}

// extractRisks extracts risks from evaluations
func (e *Engine) extractRisks(evaluations []*QuestionEvaluation) []*Risk {
	risks := make([]*Risk, 0)

	for _, eval := range evaluations {
		// If confidence is low or no choices selected, it might be a risk
		if eval.ConfidenceScore < 0.5 || len(eval.SelectedChoices) == 0 {
			risk := &Risk{
				ID:                fmt.Sprintf("risk-%s", eval.Question.ID),
				Question:          eval.Question,
				Pillar:            eval.Question.Pillar,
				Severity:          RiskLevelMedium,
				Description:       fmt.Sprintf("Low confidence or incomplete answer for: %s", eval.Question.Title),
				AffectedResources: []string{},
			}
			risks = append(risks, risk)
		}
	}

	return risks
}

// buildSummary builds a summary of the results
func (e *Engine) buildSummary(evaluations []*QuestionEvaluation, improvementPlan *ImprovementPlan) *ResultsSummary {
	totalConfidence := 0.0
	highRisks := 0
	mediumRisks := 0

	for _, eval := range evaluations {
		totalConfidence += eval.ConfidenceScore
		if eval.ConfidenceScore < 0.3 {
			highRisks++
		} else if eval.ConfidenceScore < 0.7 {
			mediumRisks++
		}
	}

	avgConfidence := 0.0
	if len(evaluations) > 0 {
		avgConfidence = totalConfidence / float64(len(evaluations))
	}

	improvementPlanSize := 0
	if improvementPlan != nil {
		improvementPlanSize = len(improvementPlan.Items)
	}

	return &ResultsSummary{
		TotalQuestions:      len(evaluations),
		QuestionsEvaluated:  len(evaluations),
		HighRisks:           highRisks,
		MediumRisks:         mediumRisks,
		AverageConfidence:   avgConfidence,
		ImprovementPlanSize: improvementPlanSize,
	}
}

// GetSessionStatus retrieves the status of a review session
func (e *Engine) GetSessionStatus(ctx context.Context, sessionID string) (SessionStatus, error) {
	session, err := e.sessionManager.LoadSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
	}

	return session.Status, nil
}

// ResumeSession resumes a previously interrupted session
func (e *Engine) ResumeSession(ctx context.Context, sessionID string) (*ReviewSession, error) {
	slog.InfoContext(ctx, "resuming session", "session_id", sessionID)

	// Load the session
	session, err := e.sessionManager.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Check if session can be resumed
	if session.Status == SessionStatusCompleted {
		return nil, ErrSessionAlreadyCompleted
	}

	if session.Status != SessionStatusFailed && session.Status != SessionStatusInProgress {
		return nil, &ValidationError{
			Field:   "session_status",
			Value:   session.Status,
			Message: fmt.Sprintf("session cannot be resumed from status: %s", session.Status),
		}
	}

	slog.InfoContext(ctx, "resuming from checkpoint",
		"checkpoint", session.Checkpoint,
	)

	// Execute the workflow from the checkpoint
	results, err := e.executeWorkflow(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to resume workflow: %w", err)
	}

	session.Results = results
	session.Status = SessionStatusCompleted
	session.UpdatedAt = time.Now()

	if err := e.sessionManager.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save resumed session: %w", err)
	}

	slog.InfoContext(ctx, "session resumed successfully",
		"session_id", sessionID,
	)

	return session, nil
}
