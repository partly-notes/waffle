package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waffle/waffle/internal/core"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		wantErr bool
	}{
		{
			name:    "valid directory",
			baseDir: t.TempDir(),
			wantErr: false,
		},
		{
			name:    "creates directory if not exists",
			baseDir: filepath.Join(t.TempDir(), "sessions"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.baseDir)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, manager)
				assert.DirExists(t, manager.baseDir)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	tests := []struct {
		name          string
		workloadID    string
		scope         core.ReviewScope
		awsWorkloadID string
		wantErr       bool
		errType       error
	}{
		{
			name:          "valid workload scope",
			workloadID:    "test-workload",
			scope:         core.ReviewScope{Level: core.ScopeLevelWorkload},
			awsWorkloadID: "aws-wl-123",
			wantErr:       false,
		},
		{
			name:       "valid pillar scope",
			workloadID: "test-workload",
			scope: core.ReviewScope{
				Level:  core.ScopeLevelPillar,
				Pillar: ptrTo(core.PillarSecurity),
			},
			awsWorkloadID: "aws-wl-456",
			wantErr:       false,
		},
		{
			name:       "valid question scope",
			workloadID: "test-workload",
			scope: core.ReviewScope{
				Level:      core.ScopeLevelQuestion,
				QuestionID: "sec_data_1",
			},
			awsWorkloadID: "aws-wl-789",
			wantErr:       false,
		},
		{
			name:          "empty workload ID",
			workloadID:    "",
			scope:         core.ReviewScope{Level: core.ScopeLevelWorkload},
			awsWorkloadID: "aws-wl-123",
			wantErr:       true,
			errType:       core.ErrInvalidWorkloadID,
		},
		{
			name:       "invalid pillar scope - missing pillar",
			workloadID: "test-workload",
			scope: core.ReviewScope{
				Level: core.ScopeLevelPillar,
			},
			awsWorkloadID: "aws-wl-123",
			wantErr:       true,
			errType:       core.ErrPillarRequired,
		},
		{
			name:       "invalid question scope - missing question ID",
			workloadID: "test-workload",
			scope: core.ReviewScope{
				Level: core.ScopeLevelQuestion,
			},
			awsWorkloadID: "aws-wl-123",
			wantErr:       true,
			errType:       core.ErrQuestionIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := manager.CreateSession(ctx, tt.workloadID, tt.scope, tt.awsWorkloadID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, session)
				assert.NotEmpty(t, session.SessionID)
				assert.Equal(t, tt.workloadID, session.WorkloadID)
				assert.Equal(t, tt.awsWorkloadID, session.AWSWorkloadID)
				assert.Equal(t, core.SessionStatusCreated, session.Status)
				assert.NotZero(t, session.CreatedAt)
				assert.NotZero(t, session.UpdatedAt)
			}
		})
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Load the session
	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify equivalence
	assert.Equal(t, session.SessionID, loaded.SessionID)
	assert.Equal(t, session.WorkloadID, loaded.WorkloadID)
	assert.Equal(t, session.AWSWorkloadID, loaded.AWSWorkloadID)
	assert.Equal(t, session.Status, loaded.Status)
	assert.Equal(t, session.Scope.Level, loaded.Scope.Level)
	assert.WithinDuration(t, session.CreatedAt, loaded.CreatedAt, time.Second)
	assert.WithinDuration(t, session.UpdatedAt, loaded.UpdatedAt, time.Second)
}

func TestLoadSession_NotFound(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	_, err = manager.LoadSession(ctx, "non-existent-session")
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrSessionNotFound)
}

func TestUpdateSessionStatus(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)
	assert.Equal(t, core.SessionStatusCreated, session.Status)

	// Update status to in progress
	err = manager.UpdateSessionStatus(ctx, session.SessionID, core.SessionStatusInProgress)
	require.NoError(t, err)

	// Load and verify
	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, core.SessionStatusInProgress, loaded.Status)

	// Update status to completed
	err = manager.UpdateSessionStatus(ctx, session.SessionID, core.SessionStatusCompleted)
	require.NoError(t, err)

	// Load and verify
	loaded, err = manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, core.SessionStatusCompleted, loaded.Status)
}

func TestUpdateSessionStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	err = manager.UpdateSessionStatus(ctx, "non-existent-session", core.SessionStatusCompleted)
	require.Error(t, err)
}

func TestListSessions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create sessions for different workloads
	session1, err := manager.CreateSession(
		ctx,
		"workload-1",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-1",
	)
	require.NoError(t, err)

	session2, err := manager.CreateSession(
		ctx,
		"workload-1",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-2",
	)
	require.NoError(t, err)

	session3, err := manager.CreateSession(
		ctx,
		"workload-2",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-3",
	)
	require.NoError(t, err)

	// List sessions for workload-1
	sessions, err := manager.ListSessions(ctx, "workload-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Verify session IDs
	sessionIDs := []string{sessions[0].SessionID, sessions[1].SessionID}
	assert.Contains(t, sessionIDs, session1.SessionID)
	assert.Contains(t, sessionIDs, session2.SessionID)
	assert.NotContains(t, sessionIDs, session3.SessionID)

	// List sessions for workload-2
	sessions, err = manager.ListSessions(ctx, "workload-2")
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session3.SessionID, sessions[0].SessionID)

	// List sessions for non-existent workload
	sessions, err = manager.ListSessions(ctx, "workload-3")
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestListSessions_EmptyWorkloadID(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	_, err = manager.ListSessions(ctx, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrInvalidWorkloadID)
}

func TestGetAWSWorkloadID(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Get AWS workload ID
	awsWorkloadID, err := manager.GetAWSWorkloadID(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, "aws-wl-123", awsWorkloadID)
}

func TestGetAWSWorkloadID_NotFound(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	_, err = manager.GetAWSWorkloadID(ctx, "non-existent-session")
	require.Error(t, err)
}

func TestDeleteSession(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Verify session exists
	_, err = manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)

	// Delete the session
	err = manager.DeleteSession(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify session no longer exists
	_, err = manager.LoadSession(ctx, session.SessionID)
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrSessionNotFound)
}

func TestDeleteSession_NotFound(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	err = manager.DeleteSession(ctx, "non-existent-session")
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrSessionNotFound)
}

func TestListAllSessions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create sessions for different workloads
	session1, err := manager.CreateSession(
		ctx,
		"workload-1",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-1",
	)
	require.NoError(t, err)

	session2, err := manager.CreateSession(
		ctx,
		"workload-2",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-2",
	)
	require.NoError(t, err)

	session3, err := manager.CreateSession(
		ctx,
		"workload-3",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-3",
	)
	require.NoError(t, err)

	// List all sessions
	sessions, err := manager.ListAllSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Verify all session IDs are present
	sessionIDs := make([]string, len(sessions))
	for i, s := range sessions {
		sessionIDs[i] = s.SessionID
	}
	assert.Contains(t, sessionIDs, session1.SessionID)
	assert.Contains(t, sessionIDs, session2.SessionID)
	assert.Contains(t, sessionIDs, session3.SessionID)
}

func TestSessionPersistence_WithComplexData(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session with complex data
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Add workload model
	session.WorkloadModel = &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:      "res-1",
				Type:    "aws_s3_bucket",
				Address: "aws_s3_bucket.example",
				Properties: map[string]interface{}{
					"bucket": "my-bucket",
					"versioning": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		Framework:  "terraform",
		SourceType: "hcl",
		Metadata: map[string]interface{}{
			"version": "1.0",
		},
	}

	// Add results
	session.Results = &core.ReviewResults{
		Evaluations: []*core.QuestionEvaluation{
			{
				Question: &core.WAFRQuestion{
					ID:     "sec_data_1",
					Pillar: core.PillarSecurity,
					Title:  "How do you classify your data?",
				},
				SelectedChoices: []core.Choice{
					{
						ID:    "choice_1",
						Title: "Data classification defined",
					},
				},
				ConfidenceScore: 0.95,
			},
		},
		Summary: &core.ResultsSummary{
			TotalQuestions:     10,
			QuestionsEvaluated: 10,
			HighRisks:          2,
			MediumRisks:        3,
			AverageConfidence:  0.85,
		},
	}

	// Save the session
	err = manager.SaveSession(ctx, session)
	require.NoError(t, err)

	// Load the session
	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify complex data
	assert.NotNil(t, loaded.WorkloadModel)
	assert.Len(t, loaded.WorkloadModel.Resources, 1)
	assert.Equal(t, "aws_s3_bucket", loaded.WorkloadModel.Resources[0].Type)
	assert.Equal(t, "terraform", loaded.WorkloadModel.Framework)

	assert.NotNil(t, loaded.Results)
	assert.Len(t, loaded.Results.Evaluations, 1)
	assert.Equal(t, "sec_data_1", loaded.Results.Evaluations[0].Question.ID)
	assert.Equal(t, 0.95, loaded.Results.Evaluations[0].ConfidenceScore)

	assert.NotNil(t, loaded.Results.Summary)
	assert.Equal(t, 10, loaded.Results.Summary.TotalQuestions)
	assert.Equal(t, 2, loaded.Results.Summary.HighRisks)
}

func TestSessionPath(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	sessionID := "test-session-123"
	path := manager.sessionPath(sessionID)

	assert.Contains(t, path, sessionID)
	assert.Contains(t, path, ".json")
}

func TestFilePermissions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Check file permissions
	sessionPath := manager.sessionPath(session.SessionID)
	info, err := os.Stat(sessionPath)
	require.NoError(t, err)

	// Verify permissions are 0600 (owner read/write only)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestUpdateMilestoneID(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)
	assert.Empty(t, session.MilestoneID)

	// Update milestone ID
	milestoneID := "milestone-1"
	err = manager.UpdateMilestoneID(ctx, session.SessionID, milestoneID)
	require.NoError(t, err)

	// Load and verify
	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, milestoneID, loaded.MilestoneID)
	assert.True(t, loaded.UpdatedAt.After(session.UpdatedAt))
}

func TestUpdateMilestoneID_NotFound(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	err = manager.UpdateMilestoneID(ctx, "non-existent-session", "milestone-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load session")
}

func TestUpdateMilestoneID_Multiple(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Update milestone ID multiple times
	milestoneID1 := "milestone-1"
	err = manager.UpdateMilestoneID(ctx, session.SessionID, milestoneID1)
	require.NoError(t, err)

	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, milestoneID1, loaded.MilestoneID)

	// Update to a new milestone ID
	milestoneID2 := "milestone-2"
	err = manager.UpdateMilestoneID(ctx, session.SessionID, milestoneID2)
	require.NoError(t, err)

	loaded, err = manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, milestoneID2, loaded.MilestoneID)
}

func TestSessionPersistence_WithMilestoneID(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(t.TempDir())
	require.NoError(t, err)

	// Create a session
	session, err := manager.CreateSession(
		ctx,
		"test-workload",
		core.ReviewScope{Level: core.ScopeLevelWorkload},
		"aws-wl-123",
	)
	require.NoError(t, err)

	// Update milestone ID
	milestoneID := "milestone-42"
	err = manager.UpdateMilestoneID(ctx, session.SessionID, milestoneID)
	require.NoError(t, err)

	// Load the session
	loaded, err := manager.LoadSession(ctx, session.SessionID)
	require.NoError(t, err)

	// Verify milestone ID persisted
	assert.Equal(t, milestoneID, loaded.MilestoneID)
	assert.Equal(t, session.SessionID, loaded.SessionID)
	assert.Equal(t, session.WorkloadID, loaded.WorkloadID)
	assert.Equal(t, session.AWSWorkloadID, loaded.AWSWorkloadID)
}

// Helper function to create a pointer to a Pillar
func ptrTo(p core.Pillar) *core.Pillar {
	return &p
}
