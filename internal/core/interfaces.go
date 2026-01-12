package core

import (
	"context"
)

// CoreEngine orchestrates the review workflow
type CoreEngine interface {
	// InitiateReview starts a new WAFR review session
	InitiateReview(
		ctx context.Context,
		workloadID string,
		scope ReviewScope,
	) (*ReviewSession, error)

	// ExecuteReview executes the review workflow
	ExecuteReview(ctx context.Context, session *ReviewSession) (*ReviewResults, error)

	// ExecuteReviewWithProgress executes the review workflow with progress reporting
	ExecuteReviewWithProgress(ctx context.Context, session *ReviewSession, progress ProgressReporter) (*ReviewResults, error)

	// GetSessionStatus retrieves the status of a review session
	GetSessionStatus(ctx context.Context, sessionID string) (SessionStatus, error)

	// ResumeSession resumes a previously interrupted session
	ResumeSession(ctx context.Context, sessionID string) (*ReviewSession, error)
}

// ProgressReporter reports progress during review execution
type ProgressReporter interface {
	// ReportStep reports the current step being executed
	ReportStep(step string, message string)

	// ReportProgress reports progress within a step
	ReportProgress(current, total int, message string)

	// ReportCompletion reports completion of the review
	ReportCompletion(summary *ResultsSummary)
}

// IaCAnalyzer parses and analyzes infrastructure-as-code files
type IaCAnalyzer interface {
	// RetrieveIaCFiles retrieves IaC files from the current directory
	RetrieveIaCFiles(ctx context.Context) ([]IaCFile, error)

	// ValidateTerraformFiles validates Terraform file syntax
	ValidateTerraformFiles(ctx context.Context, files []IaCFile) error

	// ParseTerraformPlan parses a Terraform plan JSON file
	ParseTerraformPlan(ctx context.Context, planFilePath string) (*WorkloadModel, error)

	// ParseTerraform parses Terraform HCL files
	ParseTerraform(ctx context.Context, files []IaCFile) (*WorkloadModel, error)

	// MergeWorkloadModels merges plan and source models
	MergeWorkloadModels(ctx context.Context, planModel, sourceModel *WorkloadModel) (*WorkloadModel, error)

	// ExtractResources extracts resources from a workload model
	ExtractResources(ctx context.Context, model *WorkloadModel) ([]Resource, error)

	// IdentifyRelationships identifies relationships between resources
	IdentifyRelationships(ctx context.Context, resources []Resource) (*ResourceGraph, error)
}

// SessionManager manages review session lifecycle and persistence
type SessionManager interface {
	// CreateSession creates a new review session
	CreateSession(
		ctx context.Context,
		workloadID string,
		scope ReviewScope,
		awsWorkloadID string,
	) (*ReviewSession, error)

	// SaveSession persists a session to storage
	SaveSession(ctx context.Context, session *ReviewSession) error

	// LoadSession loads a session from storage
	LoadSession(ctx context.Context, sessionID string) (*ReviewSession, error)

	// UpdateSessionStatus updates the status of a session
	UpdateSessionStatus(
		ctx context.Context,
		sessionID string,
		status SessionStatus,
	) error

	// ListSessions lists all sessions for a workload
	ListSessions(ctx context.Context, workloadID string) ([]*ReviewSession, error)

	// GetAWSWorkloadID retrieves the AWS workload ID for a session
	GetAWSWorkloadID(ctx context.Context, sessionID string) (string, error)
}

// WAFREvaluator evaluates workload against WAFR questions
type WAFREvaluator interface {
	// CreateWorkload creates a workload in AWS Well-Architected Tool
	CreateWorkload(
		ctx context.Context,
		workloadID string,
		description string,
	) (string, error)

	// GetQuestions retrieves WAFR questions based on scope
	GetQuestions(
		ctx context.Context,
		awsWorkloadID string,
		scope ReviewScope,
	) ([]*WAFRQuestion, error)

	// EvaluateQuestion evaluates a single question against the workload
	EvaluateQuestion(
		ctx context.Context,
		question *WAFRQuestion,
		workloadModel *WorkloadModel,
	) (*QuestionEvaluation, error)

	// SubmitAnswer submits an answer to AWS Well-Architected Tool
	SubmitAnswer(
		ctx context.Context,
		awsWorkloadID string,
		questionID string,
		evaluation *QuestionEvaluation,
	) error

	// GetImprovementPlan retrieves the improvement plan from AWS
	GetImprovementPlan(
		ctx context.Context,
		awsWorkloadID string,
	) (*ImprovementPlan, error)

	// CreateMilestone creates a milestone in AWS
	CreateMilestone(
		ctx context.Context,
		awsWorkloadID string,
		milestoneName string,
	) (string, error)
}

// BedrockClient provides access to Amazon Bedrock foundation models
type BedrockClient interface {
	// AnalyzeIaCSemantics analyzes IaC resources for semantic understanding
	AnalyzeIaCSemantics(
		ctx context.Context,
		resources []Resource,
	) (*SemanticAnalysis, error)

	// EvaluateWAFRQuestion evaluates a WAFR question against workload
	EvaluateWAFRQuestion(
		ctx context.Context,
		question *WAFRQuestion,
		workloadModel *WorkloadModel,
	) (*QuestionEvaluation, error)

	// GenerateImprovementGuidance generates improvement guidance for a risk
	GenerateImprovementGuidance(
		ctx context.Context,
		risk *Risk,
		resources []Resource,
	) (*ImprovementPlanItem, error)
}

// SemanticAnalysis represents semantic analysis results from Bedrock
type SemanticAnalysis struct {
	SecurityFindings []SecurityFinding
	Relationships    []Relationship
}

// SecurityFinding represents a security finding from semantic analysis
type SecurityFinding struct {
	Resource string
	Findings []string
	Severity string
}

// Relationship represents a relationship between resources
type Relationship struct {
	From   string
	To     string
	Type   string
	Status string
}

// ReportGenerator generates and formats reports
type ReportGenerator interface {
	// GetConsolidatedReport retrieves a consolidated report from AWS
	GetConsolidatedReport(
		ctx context.Context,
		awsWorkloadID string,
		format ReportFormat,
	) ([]byte, error)

	// GetResultsJSON retrieves results in JSON format with IaC evidence
	GetResultsJSON(
		ctx context.Context,
		awsWorkloadID string,
		session *ReviewSession,
	) (map[string]interface{}, error)
}

// ReportFormat represents the format of a report
type ReportFormat string

const (
	ReportFormatPDF  ReportFormat = "pdf"
	ReportFormatJSON ReportFormat = "json"
)
