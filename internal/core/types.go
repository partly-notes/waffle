package core

import (
	"time"
)

// ScopeLevel defines the level of WAFR review scope
type ScopeLevel int

const (
	ScopeLevelWorkload ScopeLevel = iota
	ScopeLevelPillar
	ScopeLevelQuestion
)

// Pillar represents a Well-Architected Framework pillar
type Pillar string

const (
	PillarOperationalExcellence Pillar = "operationalExcellence"
	PillarSecurity              Pillar = "security"
	PillarReliability           Pillar = "reliability"
	PillarPerformanceEfficiency Pillar = "performance"
	PillarCostOptimization      Pillar = "costOptimization"
	PillarSustainability        Pillar = "sustainability"
)

// SessionStatus represents the status of a review session
type SessionStatus string

const (
	SessionStatusCreated    SessionStatus = "created"
	SessionStatusInProgress SessionStatus = "in_progress"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusFailed     SessionStatus = "failed"
)

// RiskLevel represents the severity of a risk
type RiskLevel int

const (
	RiskLevelNone RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
)

// ReviewScope defines the scope of a WAFR review
type ReviewScope struct {
	Level      ScopeLevel
	Pillar     *Pillar
	QuestionID string
}

// Validate checks if the review scope is valid
func (r *ReviewScope) Validate() error {
	switch r.Level {
	case ScopeLevelPillar:
		if r.Pillar == nil {
			return ErrPillarRequired
		}
	case ScopeLevelQuestion:
		if r.QuestionID == "" {
			return ErrQuestionIDRequired
		}
	}
	return nil
}

// ReviewSession represents a WAFR review session
type ReviewSession struct {
	SessionID     string
	WorkloadID    string
	AWSWorkloadID string
	MilestoneID   string
	PlanFilePath  string
	Scope         ReviewScope
	Status        SessionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	WorkloadModel *WorkloadModel
	Results       *ReviewResults
	Checkpoint    string
}

// WorkloadModel represents the parsed IaC workload
type WorkloadModel struct {
	Resources     []Resource
	Relationships *ResourceGraph
	Framework     string
	SourceType    string
	Metadata      map[string]interface{}
}

// Resource represents an infrastructure resource
type Resource struct {
	ID           string
	Type         string
	Address      string
	Properties   map[string]interface{}
	Dependencies []string
	SourceFile   string
	SourceLine   int
	IsFromPlan   bool
	ModulePath   string
}

// ResourceGraph represents relationships between resources
type ResourceGraph struct {
	Nodes map[string]*Resource
	Edges map[string][]string
}

// WAFRQuestion represents a Well-Architected Framework question
type WAFRQuestion struct {
	ID            string
	Pillar        Pillar
	Title         string
	Description   string
	BestPractices []BestPractice
	Choices       []Choice
	RiskRules     map[string]interface{}
}

// BestPractice represents a WAFR best practice
type BestPractice struct {
	ID          string
	Title       string
	Description string
}

// Choice represents a WAFR question choice
type Choice struct {
	ID          string
	Title       string
	Description string
}

// QuestionEvaluation represents the evaluation of a WAFR question
type QuestionEvaluation struct {
	Question        *WAFRQuestion
	SelectedChoices []Choice
	Evidence        []Evidence
	ConfidenceScore float64
	Notes           string
}

// Evidence represents evidence for a choice selection
type Evidence struct {
	ChoiceID    string
	Explanation string
	Resources   []string
	Confidence  float64
}

// Risk represents an identified risk
type Risk struct {
	ID                   string
	Question             *WAFRQuestion
	Pillar               Pillar
	Severity             RiskLevel
	Description          string
	AffectedResources    []string
	MissingBestPractices []BestPractice
}

// ImprovementPlanItem represents an improvement recommendation
type ImprovementPlanItem struct {
	ID                string
	Risk              *Risk
	Description       string
	BestPracticeRefs  []string
	AffectedResources []string
	Priority          int
	EstimatedEffort   string
}

// ImprovementPlan represents the complete improvement plan
type ImprovementPlan struct {
	Items []*ImprovementPlanItem
}

// ReviewResults represents the complete review results
type ReviewResults struct {
	Evaluations     []*QuestionEvaluation
	Risks           []*Risk
	ImprovementPlan *ImprovementPlan
	Summary         *ResultsSummary
}

// ResultsSummary provides a summary of review results
type ResultsSummary struct {
	TotalQuestions      int
	QuestionsEvaluated  int
	HighRisks           int
	MediumRisks         int
	AverageConfidence   float64
	ImprovementPlanSize int
}

// IaCFile represents an infrastructure-as-code file
type IaCFile struct {
	Path    string
	Content string
}
