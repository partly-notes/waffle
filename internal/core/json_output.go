package core

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// JSONOutput represents the standard JSON output structure for all commands
type JSONOutput struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *JSONError  `json:"error,omitempty"`
}

// JSONError represents an error in JSON output
type JSONError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ReviewOutput represents the JSON output for the review command
type ReviewOutput struct {
	SessionID  string                 `json:"session_id"`
	WorkloadID string                 `json:"workload_id"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	Summary    *ReviewSummaryOutput   `json:"summary,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ReviewSummaryOutput represents a summary of the review for JSON output
type ReviewSummaryOutput struct {
	QuestionsEvaluated  int     `json:"questions_evaluated"`
	HighRisks           int     `json:"high_risks"`
	MediumRisks         int     `json:"medium_risks"`
	AverageConfidence   float64 `json:"average_confidence"`
	ImprovementPlanSize int     `json:"improvement_plan_size"`
}

// StatusOutput represents the JSON output for the status command
type StatusOutput struct {
	SessionID  string                 `json:"session_id"`
	WorkloadID string                 `json:"workload_id"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Progress   *ProgressOutput        `json:"progress,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ProgressOutput represents progress information for JSON output
type ProgressOutput struct {
	CurrentStep       string `json:"current_step"`
	TotalSteps        int    `json:"total_steps"`
	CompletedSteps    int    `json:"completed_steps"`
	CurrentStepDetail string `json:"current_step_detail,omitempty"`
}

// ResultsOutput represents the JSON output for the results command
type ResultsOutput struct {
	SessionID     string                  `json:"session_id"`
	WorkloadID    string                  `json:"workload_id"`
	AWSWorkloadID string                  `json:"aws_workload_id,omitempty"`
	MilestoneID   string                  `json:"milestone_id,omitempty"`
	Status        string                  `json:"status"`
	CreatedAt     time.Time               `json:"created_at"`
	CompletedAt   time.Time               `json:"completed_at,omitempty"`
	Scope         *ScopeOutput            `json:"scope"`
	Summary       *ReviewSummaryOutput    `json:"summary"`
	Evaluations   []*EvaluationOutput     `json:"evaluations,omitempty"`
	Risks         []*RiskOutput           `json:"risks,omitempty"`
	Improvements  []*ImprovementOutput    `json:"improvements,omitempty"`
	Resources     []*ResourceOutput       `json:"resources,omitempty"`
	Links         map[string]string       `json:"links,omitempty"`
}

// ScopeOutput represents the review scope for JSON output
type ScopeOutput struct {
	Level      string  `json:"level"`
	Pillar     *string `json:"pillar,omitempty"`
	QuestionID string  `json:"question_id,omitempty"`
}

// EvaluationOutput represents a question evaluation for JSON output
type EvaluationOutput struct {
	QuestionID      string            `json:"question_id"`
	Pillar          string            `json:"pillar"`
	Title           string            `json:"title"`
	SelectedChoices []string          `json:"selected_choices"`
	Evidence        []*EvidenceOutput `json:"evidence,omitempty"`
	ConfidenceScore float64           `json:"confidence_score"`
	Notes           string            `json:"notes,omitempty"`
}

// EvidenceOutput represents evidence for JSON output
type EvidenceOutput struct {
	ChoiceID    string   `json:"choice_id"`
	Explanation string   `json:"explanation"`
	Resources   []string `json:"resources"`
	Confidence  float64  `json:"confidence"`
}

// RiskOutput represents a risk for JSON output
type RiskOutput struct {
	ID                   string   `json:"id"`
	QuestionID           string   `json:"question_id"`
	Pillar               string   `json:"pillar"`
	Severity             string   `json:"severity"`
	Description          string   `json:"description"`
	AffectedResources    []string `json:"affected_resources"`
	MissingBestPractices []string `json:"missing_best_practices"`
}

// ImprovementOutput represents an improvement plan item for JSON output
type ImprovementOutput struct {
	ID                string   `json:"id"`
	RiskID            string   `json:"risk_id"`
	Description       string   `json:"description"`
	BestPracticeRefs  []string `json:"best_practice_refs"`
	AffectedResources []string `json:"affected_resources"`
	Priority          int      `json:"priority"`
	EstimatedEffort   string   `json:"estimated_effort"`
}

// ResourceOutput represents a resource for JSON output
type ResourceOutput struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Address    string                 `json:"address"`
	SourceFile string                 `json:"source_file,omitempty"`
	IsFromPlan bool                   `json:"is_from_plan"`
	ModulePath string                 `json:"module_path,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// WriteJSON writes a JSON output to the specified writer
func WriteJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// WriteJSONSuccess writes a successful JSON response
func WriteJSONSuccess(w io.Writer, data interface{}) error {
	output := JSONOutput{
		Success: true,
		Data:    data,
	}
	return WriteJSON(w, output)
}

// WriteJSONError writes an error JSON response
func WriteJSONError(w io.Writer, code, message string) error {
	output := JSONOutput{
		Success: false,
		Error: &JSONError{
			Code:    code,
			Message: message,
		},
	}
	return WriteJSON(w, output)
}

// ConvertReviewSessionToOutput converts a ReviewSession to ReviewOutput
func ConvertReviewSessionToOutput(session *ReviewSession) *ReviewOutput {
	output := &ReviewOutput{
		SessionID:  session.SessionID,
		WorkloadID: session.WorkloadID,
		Status:     string(session.Status),
		CreatedAt:  session.CreatedAt,
		Metadata: map[string]interface{}{
			"aws_workload_id": session.AWSWorkloadID,
			"milestone_id":    session.MilestoneID,
		},
	}

	if session.Results != nil && session.Results.Summary != nil {
		output.Summary = &ReviewSummaryOutput{
			QuestionsEvaluated:  session.Results.Summary.QuestionsEvaluated,
			HighRisks:           session.Results.Summary.HighRisks,
			MediumRisks:         session.Results.Summary.MediumRisks,
			AverageConfidence:   session.Results.Summary.AverageConfidence,
			ImprovementPlanSize: session.Results.Summary.ImprovementPlanSize,
		}
	}

	return output
}

// ConvertReviewSessionToStatusOutput converts a ReviewSession to StatusOutput
func ConvertReviewSessionToStatusOutput(session *ReviewSession) *StatusOutput {
	output := &StatusOutput{
		SessionID:  session.SessionID,
		WorkloadID: session.WorkloadID,
		Status:     string(session.Status),
		CreatedAt:  session.CreatedAt,
		UpdatedAt:  session.UpdatedAt,
		Metadata: map[string]interface{}{
			"aws_workload_id": session.AWSWorkloadID,
			"checkpoint":      session.Checkpoint,
		},
	}

	return output
}

// ConvertReviewSessionToResultsOutput converts a ReviewSession to ResultsOutput
func ConvertReviewSessionToResultsOutput(session *ReviewSession) *ResultsOutput {
	output := &ResultsOutput{
		SessionID:     session.SessionID,
		WorkloadID:    session.WorkloadID,
		AWSWorkloadID: session.AWSWorkloadID,
		MilestoneID:   session.MilestoneID,
		Status:        string(session.Status),
		CreatedAt:     session.CreatedAt,
		CompletedAt:   session.UpdatedAt,
		Scope:         convertScopeToOutput(&session.Scope),
	}

	// Add summary
	if session.Results != nil && session.Results.Summary != nil {
		output.Summary = &ReviewSummaryOutput{
			QuestionsEvaluated:  session.Results.Summary.QuestionsEvaluated,
			HighRisks:           session.Results.Summary.HighRisks,
			MediumRisks:         session.Results.Summary.MediumRisks,
			AverageConfidence:   session.Results.Summary.AverageConfidence,
			ImprovementPlanSize: session.Results.Summary.ImprovementPlanSize,
		}
	}

	// Add evaluations
	if session.Results != nil && len(session.Results.Evaluations) > 0 {
		output.Evaluations = make([]*EvaluationOutput, 0, len(session.Results.Evaluations))
		for _, eval := range session.Results.Evaluations {
			output.Evaluations = append(output.Evaluations, convertEvaluationToOutput(eval))
		}
	}

	// Add risks
	if session.Results != nil && len(session.Results.Risks) > 0 {
		output.Risks = make([]*RiskOutput, 0, len(session.Results.Risks))
		for _, risk := range session.Results.Risks {
			output.Risks = append(output.Risks, convertRiskToOutput(risk))
		}
	}

	// Add improvements
	if session.Results != nil && session.Results.ImprovementPlan != nil && len(session.Results.ImprovementPlan.Items) > 0 {
		output.Improvements = make([]*ImprovementOutput, 0, len(session.Results.ImprovementPlan.Items))
		for _, item := range session.Results.ImprovementPlan.Items {
			output.Improvements = append(output.Improvements, convertImprovementToOutput(item))
		}
	}

	// Add resources
	if session.WorkloadModel != nil && len(session.WorkloadModel.Resources) > 0 {
		output.Resources = make([]*ResourceOutput, 0, len(session.WorkloadModel.Resources))
		for _, resource := range session.WorkloadModel.Resources {
			output.Resources = append(output.Resources, convertResourceToOutput(&resource))
		}
	}

	// Add AWS console links
	if session.AWSWorkloadID != "" {
		output.Links = map[string]string{
			"aws_console": fmt.Sprintf("https://console.aws.amazon.com/wellarchitected/home#/workload/%s", session.AWSWorkloadID),
		}
	}

	return output
}

func convertScopeToOutput(scope *ReviewScope) *ScopeOutput {
	output := &ScopeOutput{}

	switch scope.Level {
	case ScopeLevelWorkload:
		output.Level = "workload"
	case ScopeLevelPillar:
		output.Level = "pillar"
		if scope.Pillar != nil {
			pillarStr := string(*scope.Pillar)
			output.Pillar = &pillarStr
		}
	case ScopeLevelQuestion:
		output.Level = "question"
		output.QuestionID = scope.QuestionID
	}

	return output
}

func convertEvaluationToOutput(eval *QuestionEvaluation) *EvaluationOutput {
	output := &EvaluationOutput{
		QuestionID:      eval.Question.ID,
		Pillar:          string(eval.Question.Pillar),
		Title:           eval.Question.Title,
		ConfidenceScore: eval.ConfidenceScore,
		Notes:           eval.Notes,
	}

	// Convert selected choices
	output.SelectedChoices = make([]string, 0, len(eval.SelectedChoices))
	for _, choice := range eval.SelectedChoices {
		output.SelectedChoices = append(output.SelectedChoices, choice.ID)
	}

	// Convert evidence
	if len(eval.Evidence) > 0 {
		output.Evidence = make([]*EvidenceOutput, 0, len(eval.Evidence))
		for _, evidence := range eval.Evidence {
			output.Evidence = append(output.Evidence, &EvidenceOutput{
				ChoiceID:    evidence.ChoiceID,
				Explanation: evidence.Explanation,
				Resources:   evidence.Resources,
				Confidence:  evidence.Confidence,
			})
		}
	}

	return output
}

func convertRiskToOutput(risk *Risk) *RiskOutput {
	output := &RiskOutput{
		ID:                risk.ID,
		QuestionID:        risk.Question.ID,
		Pillar:            string(risk.Pillar),
		Description:       risk.Description,
		AffectedResources: risk.AffectedResources,
	}

	// Convert severity
	switch risk.Severity {
	case RiskLevelHigh:
		output.Severity = "high"
	case RiskLevelMedium:
		output.Severity = "medium"
	case RiskLevelNone:
		output.Severity = "none"
	}

	// Convert missing best practices
	output.MissingBestPractices = make([]string, 0, len(risk.MissingBestPractices))
	for _, bp := range risk.MissingBestPractices {
		output.MissingBestPractices = append(output.MissingBestPractices, bp.ID)
	}

	return output
}

func convertImprovementToOutput(item *ImprovementPlanItem) *ImprovementOutput {
	output := &ImprovementOutput{
		ID:                item.ID,
		Description:       item.Description,
		BestPracticeRefs:  item.BestPracticeRefs,
		AffectedResources: item.AffectedResources,
		Priority:          item.Priority,
		EstimatedEffort:   item.EstimatedEffort,
	}

	if item.Risk != nil {
		output.RiskID = item.Risk.ID
	}

	return output
}

func convertResourceToOutput(resource *Resource) *ResourceOutput {
	return &ResourceOutput{
		ID:         resource.ID,
		Type:       resource.Type,
		Address:    resource.Address,
		SourceFile: resource.SourceFile,
		IsFromPlan: resource.IsFromPlan,
		ModulePath: resource.ModulePath,
		Properties: resource.Properties,
	}
}
