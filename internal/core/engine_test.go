package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockSessionManager struct {
	createSessionFunc       func(ctx context.Context, workloadID string, scope ReviewScope, awsWorkloadID string) (*ReviewSession, error)
	saveSessionFunc         func(ctx context.Context, session *ReviewSession) error
	loadSessionFunc         func(ctx context.Context, sessionID string) (*ReviewSession, error)
	updateSessionStatusFunc func(ctx context.Context, sessionID string, status SessionStatus) error
	listSessionsFunc        func(ctx context.Context, workloadID string) ([]*ReviewSession, error)
	getAWSWorkloadIDFunc    func(ctx context.Context, sessionID string) (string, error)
}

func (m *mockSessionManager) CreateSession(ctx context.Context, workloadID string, scope ReviewScope, awsWorkloadID string) (*ReviewSession, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, workloadID, scope, awsWorkloadID)
	}
	return &ReviewSession{
		SessionID:     "test-session-id",
		WorkloadID:    workloadID,
		AWSWorkloadID: awsWorkloadID,
		Scope:         scope,
		Status:        SessionStatusCreated,
	}, nil
}

func (m *mockSessionManager) SaveSession(ctx context.Context, session *ReviewSession) error {
	if m.saveSessionFunc != nil {
		return m.saveSessionFunc(ctx, session)
	}
	return nil
}

func (m *mockSessionManager) LoadSession(ctx context.Context, sessionID string) (*ReviewSession, error) {
	if m.loadSessionFunc != nil {
		return m.loadSessionFunc(ctx, sessionID)
	}
	return nil, ErrSessionNotFound
}

func (m *mockSessionManager) UpdateSessionStatus(ctx context.Context, sessionID string, status SessionStatus) error {
	if m.updateSessionStatusFunc != nil {
		return m.updateSessionStatusFunc(ctx, sessionID, status)
	}
	return nil
}

func (m *mockSessionManager) ListSessions(ctx context.Context, workloadID string) ([]*ReviewSession, error) {
	if m.listSessionsFunc != nil {
		return m.listSessionsFunc(ctx, workloadID)
	}
	return nil, nil
}

func (m *mockSessionManager) GetAWSWorkloadID(ctx context.Context, sessionID string) (string, error) {
	if m.getAWSWorkloadIDFunc != nil {
		return m.getAWSWorkloadIDFunc(ctx, sessionID)
	}
	return "", nil
}

type mockIaCAnalyzer struct {
	retrieveIaCFilesFunc      func(ctx context.Context) ([]IaCFile, error)
	validateTerraformFunc     func(ctx context.Context, files []IaCFile) error
	parseTerraformPlanFunc    func(ctx context.Context, planFilePath string) (*WorkloadModel, error)
	parseTerraformFunc        func(ctx context.Context, files []IaCFile) (*WorkloadModel, error)
	mergeWorkloadModelsFunc   func(ctx context.Context, planModel, sourceModel *WorkloadModel) (*WorkloadModel, error)
	extractResourcesFunc      func(ctx context.Context, model *WorkloadModel) ([]Resource, error)
	identifyRelationshipsFunc func(ctx context.Context, resources []Resource) (*ResourceGraph, error)
}

func (m *mockIaCAnalyzer) RetrieveIaCFiles(ctx context.Context) ([]IaCFile, error) {
	if m.retrieveIaCFilesFunc != nil {
		return m.retrieveIaCFilesFunc(ctx)
	}
	return []IaCFile{{Path: "main.tf", Content: "resource \"aws_s3_bucket\" \"test\" {}"}}, nil
}

func (m *mockIaCAnalyzer) ValidateTerraformFiles(ctx context.Context, files []IaCFile) error {
	if m.validateTerraformFunc != nil {
		return m.validateTerraformFunc(ctx, files)
	}
	return nil
}

func (m *mockIaCAnalyzer) ParseTerraformPlan(ctx context.Context, planFilePath string) (*WorkloadModel, error) {
	if m.parseTerraformPlanFunc != nil {
		return m.parseTerraformPlanFunc(ctx, planFilePath)
	}
	return &WorkloadModel{Framework: "terraform", SourceType: "plan"}, nil
}

func (m *mockIaCAnalyzer) ParseTerraform(ctx context.Context, files []IaCFile) (*WorkloadModel, error) {
	if m.parseTerraformFunc != nil {
		return m.parseTerraformFunc(ctx, files)
	}
	return &WorkloadModel{
		Framework:  "terraform",
		SourceType: "hcl",
		Resources:  []Resource{{ID: "test-resource", Type: "aws_s3_bucket"}},
	}, nil
}

func (m *mockIaCAnalyzer) MergeWorkloadModels(ctx context.Context, planModel, sourceModel *WorkloadModel) (*WorkloadModel, error) {
	if m.mergeWorkloadModelsFunc != nil {
		return m.mergeWorkloadModelsFunc(ctx, planModel, sourceModel)
	}
	return planModel, nil
}

func (m *mockIaCAnalyzer) ExtractResources(ctx context.Context, model *WorkloadModel) ([]Resource, error) {
	if m.extractResourcesFunc != nil {
		return m.extractResourcesFunc(ctx, model)
	}
	return []Resource{{ID: "test-resource", Type: "aws_s3_bucket"}}, nil
}

func (m *mockIaCAnalyzer) IdentifyRelationships(ctx context.Context, resources []Resource) (*ResourceGraph, error) {
	if m.identifyRelationshipsFunc != nil {
		return m.identifyRelationshipsFunc(ctx, resources)
	}
	return &ResourceGraph{Nodes: make(map[string]*Resource), Edges: make(map[string][]string)}, nil
}

type mockWAFREvaluator struct {
	createWorkloadFunc      func(ctx context.Context, workloadID string, description string) (string, error)
	getQuestionsFunc        func(ctx context.Context, awsWorkloadID string, scope ReviewScope) ([]*WAFRQuestion, error)
	evaluateQuestionFunc    func(ctx context.Context, question *WAFRQuestion, workloadModel *WorkloadModel) (*QuestionEvaluation, error)
	submitAnswerFunc        func(ctx context.Context, awsWorkloadID string, questionID string, evaluation *QuestionEvaluation) error
	getImprovementPlanFunc  func(ctx context.Context, awsWorkloadID string) (*ImprovementPlan, error)
	createMilestoneFunc     func(ctx context.Context, awsWorkloadID string, milestoneName string) (string, error)
}

func (m *mockWAFREvaluator) CreateWorkload(ctx context.Context, workloadID string, description string) (string, error) {
	if m.createWorkloadFunc != nil {
		return m.createWorkloadFunc(ctx, workloadID, description)
	}
	return "aws-workload-123", nil
}

func (m *mockWAFREvaluator) GetQuestions(ctx context.Context, awsWorkloadID string, scope ReviewScope) ([]*WAFRQuestion, error) {
	if m.getQuestionsFunc != nil {
		return m.getQuestionsFunc(ctx, awsWorkloadID, scope)
	}
	return []*WAFRQuestion{
		{
			ID:     "sec-1",
			Pillar: PillarSecurity,
			Title:  "Test Question",
		},
	}, nil
}

func (m *mockWAFREvaluator) EvaluateQuestion(ctx context.Context, question *WAFRQuestion, workloadModel *WorkloadModel) (*QuestionEvaluation, error) {
	if m.evaluateQuestionFunc != nil {
		return m.evaluateQuestionFunc(ctx, question, workloadModel)
	}
	return &QuestionEvaluation{
		Question:        question,
		SelectedChoices: []Choice{{ID: "choice-1"}},
		ConfidenceScore: 0.9,
	}, nil
}

func (m *mockWAFREvaluator) SubmitAnswer(ctx context.Context, awsWorkloadID string, questionID string, evaluation *QuestionEvaluation) error {
	if m.submitAnswerFunc != nil {
		return m.submitAnswerFunc(ctx, awsWorkloadID, questionID, evaluation)
	}
	return nil
}

func (m *mockWAFREvaluator) GetImprovementPlan(ctx context.Context, awsWorkloadID string) (*ImprovementPlan, error) {
	if m.getImprovementPlanFunc != nil {
		return m.getImprovementPlanFunc(ctx, awsWorkloadID)
	}
	return &ImprovementPlan{Items: []*ImprovementPlanItem{}}, nil
}

func (m *mockWAFREvaluator) CreateMilestone(ctx context.Context, awsWorkloadID string, milestoneName string) (string, error) {
	if m.createMilestoneFunc != nil {
		return m.createMilestoneFunc(ctx, awsWorkloadID, milestoneName)
	}
	return "milestone-123", nil
}

type mockBedrockClient struct{}

func (m *mockBedrockClient) AnalyzeIaCSemantics(ctx context.Context, resources []Resource) (*SemanticAnalysis, error) {
	return &SemanticAnalysis{}, nil
}

func (m *mockBedrockClient) EvaluateWAFRQuestion(ctx context.Context, question *WAFRQuestion, workloadModel *WorkloadModel) (*QuestionEvaluation, error) {
	return &QuestionEvaluation{}, nil
}

func (m *mockBedrockClient) GenerateImprovementGuidance(ctx context.Context, risk *Risk, resources []Resource) (*ImprovementPlanItem, error) {
	return &ImprovementPlanItem{}, nil
}

type mockReportGenerator struct{}

func (m *mockReportGenerator) GetConsolidatedReport(ctx context.Context, awsWorkloadID string, format ReportFormat) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockReportGenerator) GetResultsJSON(ctx context.Context, awsWorkloadID string, session *ReviewSession) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// Tests

func TestInitiateReview_Success(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	scope := ReviewScope{Level: ScopeLevelWorkload}
	session, err := engine.InitiateReview(context.Background(), "test-workload", scope)

	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "test-session-id", session.SessionID)
	assert.Equal(t, "test-workload", session.WorkloadID)
	assert.Equal(t, "aws-workload-123", session.AWSWorkloadID)
}

func TestInitiateReview_InvalidWorkloadID(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	scope := ReviewScope{Level: ScopeLevelWorkload}
	_, err := engine.InitiateReview(context.Background(), "", scope)

	require.Error(t, err)
	assert.Equal(t, ErrInvalidWorkloadID, err)
}

func TestInitiateReview_InvalidScope(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	// Pillar scope without pillar specified
	scope := ReviewScope{Level: ScopeLevelPillar}
	_, err := engine.InitiateReview(context.Background(), "test-workload", scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid scope")
}

func TestInitiateReview_CreateWorkloadFails(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{
		createWorkloadFunc: func(ctx context.Context, workloadID string, description string) (string, error) {
			return "", errors.New("AWS API error")
		},
	}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	scope := ReviewScope{Level: ScopeLevelWorkload}
	_, err := engine.InitiateReview(context.Background(), "test-workload", scope)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create AWS workload")
}

func TestExecuteReview_Success(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	session := &ReviewSession{
		SessionID:     "test-session",
		WorkloadID:    "test-workload",
		AWSWorkloadID: "aws-workload-123",
		Scope:         ReviewScope{Level: ScopeLevelWorkload},
		Status:        SessionStatusCreated,
	}

	results, err := engine.ExecuteReview(context.Background(), session)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.NotNil(t, results.Summary)
	assert.Equal(t, SessionStatusCompleted, session.Status)
}

func TestExecuteReview_IaCAnalysisFails(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{
		retrieveIaCFilesFunc: func(ctx context.Context) ([]IaCFile, error) {
			return nil, errors.New("directory not found")
		},
	}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	session := &ReviewSession{
		SessionID:     "test-session",
		WorkloadID:    "test-workload",
		AWSWorkloadID: "aws-workload-123",
		Scope:         ReviewScope{Level: ScopeLevelWorkload},
		Status:        SessionStatusCreated,
	}

	_, err := engine.ExecuteReview(context.Background(), session)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "IaC analysis failed")
	assert.Equal(t, SessionStatusFailed, session.Status)
}

func TestGetSessionStatus_Success(t *testing.T) {
	sessionMgr := &mockSessionManager{
		loadSessionFunc: func(ctx context.Context, sessionID string) (*ReviewSession, error) {
			return &ReviewSession{
				SessionID: sessionID,
				Status:    SessionStatusCompleted,
			}, nil
		},
	}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	status, err := engine.GetSessionStatus(context.Background(), "test-session")

	require.NoError(t, err)
	assert.Equal(t, SessionStatusCompleted, status)
}

func TestResumeSession_Success(t *testing.T) {
	sessionMgr := &mockSessionManager{
		loadSessionFunc: func(ctx context.Context, sessionID string) (*ReviewSession, error) {
			return &ReviewSession{
				SessionID:     sessionID,
				WorkloadID:    "test-workload",
				AWSWorkloadID: "aws-workload-123",
				Scope:         ReviewScope{Level: ScopeLevelWorkload},
				Status:        SessionStatusFailed,
				Checkpoint:    "iac_analysis_complete",
				WorkloadModel: &WorkloadModel{
					Framework: "terraform",
					Resources: []Resource{{ID: "test-resource"}},
				},
			}, nil
		},
	}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	session, err := engine.ResumeSession(context.Background(), "test-session")

	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, SessionStatusCompleted, session.Status)
}

func TestResumeSession_AlreadyCompleted(t *testing.T) {
	sessionMgr := &mockSessionManager{
		loadSessionFunc: func(ctx context.Context, sessionID string) (*ReviewSession, error) {
			return &ReviewSession{
				SessionID: sessionID,
				Status:    SessionStatusCompleted,
			}, nil
		},
	}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	_, err := engine.ResumeSession(context.Background(), "test-session")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already completed")
}

func TestExecuteWorkflow_WithCheckpoints(t *testing.T) {
	sessionMgr := &mockSessionManager{}
	iacAnalyzer := &mockIaCAnalyzer{}
	wafrEval := &mockWAFREvaluator{}
	bedrock := &mockBedrockClient{}
	reportGen := &mockReportGenerator{}

	engine := NewEngine(sessionMgr, iacAnalyzer, wafrEval, bedrock, reportGen)

	session := &ReviewSession{
		SessionID:     "test-session",
		WorkloadID:    "test-workload",
		AWSWorkloadID: "aws-workload-123",
		Scope:         ReviewScope{Level: ScopeLevelWorkload},
		Status:        SessionStatusCreated,
		Checkpoint:    "",
	}

	results, err := engine.ExecuteReview(context.Background(), session)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, "milestone_created", session.Checkpoint)
	assert.NotEmpty(t, session.MilestoneID)
}
