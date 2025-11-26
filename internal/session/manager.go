package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/waffle/waffle/internal/core"
)

// Manager implements the SessionManager interface
type Manager struct {
	baseDir string
}

// NewManager creates a new session manager
func NewManager(baseDir string) (*Manager, error) {
	// Expand home directory if needed
	if strings.HasPrefix(baseDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(home, baseDir[2:])
	}

	// Create session directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &Manager{
		baseDir: baseDir,
	}, nil
}

// CreateSession creates a new review session
func (m *Manager) CreateSession(
	ctx context.Context,
	workloadID string,
	scope core.ReviewScope,
	awsWorkloadID string,
) (*core.ReviewSession, error) {
	// Validate inputs
	if workloadID == "" {
		return nil, core.ErrInvalidWorkloadID
	}

	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scope: %w", err)
	}

	// Generate unique session ID
	sessionID := uuid.New().String()

	now := time.Now()
	session := &core.ReviewSession{
		SessionID:     sessionID,
		WorkloadID:    workloadID,
		AWSWorkloadID: awsWorkloadID,
		Scope:         scope,
		Status:        core.SessionStatusCreated,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Save the session
	if err := m.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	slog.InfoContext(ctx, "session created",
		"session_id", sessionID,
		"workload_id", workloadID,
		"aws_workload_id", awsWorkloadID,
	)

	return session, nil
}

// SaveSession persists a session to storage
func (m *Manager) SaveSession(ctx context.Context, session *core.ReviewSession) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.SessionID == "" {
		return fmt.Errorf("session ID is empty")
	}

	// Update the UpdatedAt timestamp
	session.UpdatedAt = time.Now()

	sessionPath := m.sessionPath(session.SessionID)

	// Create file with restricted permissions (owner read/write only)
	f, err := os.OpenFile(sessionPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(session); err != nil {
		return fmt.Errorf("failed to encode session: %w", err)
	}

	slog.DebugContext(ctx, "session saved",
		"session_id", session.SessionID,
		"status", session.Status,
	)

	return nil
}

// LoadSession loads a session from storage
func (m *Manager) LoadSession(ctx context.Context, sessionID string) (*core.ReviewSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is empty")
	}

	sessionPath := m.sessionPath(sessionID)

	// Check if file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, core.ErrSessionNotFound
	}

	f, err := os.Open(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer f.Close()

	var session core.ReviewSession
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	slog.DebugContext(ctx, "session loaded",
		"session_id", sessionID,
		"status", session.Status,
	)

	return &session, nil
}

// UpdateSessionStatus updates the status of a session
func (m *Manager) UpdateSessionStatus(
	ctx context.Context,
	sessionID string,
	status core.SessionStatus,
) error {
	// Load the session
	session, err := m.LoadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Update status
	session.Status = status
	session.UpdatedAt = time.Now()

	// Save the session
	if err := m.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	slog.InfoContext(ctx, "session status updated",
		"session_id", sessionID,
		"status", status,
	)

	return nil
}

// ListSessions lists all sessions for a workload
func (m *Manager) ListSessions(ctx context.Context, workloadID string) ([]*core.ReviewSession, error) {
	if workloadID == "" {
		return nil, core.ErrInvalidWorkloadID
	}

	// Read all session files
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []*core.ReviewSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract session ID from filename
		sessionID := strings.TrimSuffix(entry.Name(), ".json")

		// Load the session
		session, err := m.LoadSession(ctx, sessionID)
		if err != nil {
			slog.WarnContext(ctx, "failed to load session",
				"session_id", sessionID,
				"error", err,
			)
			continue
		}

		// Filter by workload ID
		if session.WorkloadID == workloadID {
			sessions = append(sessions, session)
		}
	}

	slog.DebugContext(ctx, "sessions listed",
		"workload_id", workloadID,
		"count", len(sessions),
	)

	return sessions, nil
}

// GetAWSWorkloadID retrieves the AWS workload ID for a session
func (m *Manager) GetAWSWorkloadID(ctx context.Context, sessionID string) (string, error) {
	session, err := m.LoadSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to load session: %w", err)
	}

	if session.AWSWorkloadID == "" {
		return "", fmt.Errorf("AWS workload ID not set for session %s", sessionID)
	}

	return session.AWSWorkloadID, nil
}

// UpdateMilestoneID updates the milestone ID for a session
func (m *Manager) UpdateMilestoneID(
	ctx context.Context,
	sessionID string,
	milestoneID string,
) error {
	// Load the session
	session, err := m.LoadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Update milestone ID
	session.MilestoneID = milestoneID
	session.UpdatedAt = time.Now()

	// Save the session
	if err := m.SaveSession(ctx, session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	slog.InfoContext(ctx, "milestone ID updated",
		"session_id", sessionID,
		"milestone_id", milestoneID,
	)

	return nil
}

// sessionPath returns the file path for a session
func (m *Manager) sessionPath(sessionID string) string {
	return filepath.Join(m.baseDir, sessionID+".json")
}

// DeleteSession deletes a session from storage
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is empty")
	}

	sessionPath := m.sessionPath(sessionID)

	// Check if file exists
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return core.ErrSessionNotFound
	}

	// Delete the file
	if err := os.Remove(sessionPath); err != nil {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	slog.InfoContext(ctx, "session deleted",
		"session_id", sessionID,
	)

	return nil
}

// ListAllSessions lists all sessions regardless of workload
func (m *Manager) ListAllSessions(ctx context.Context) ([]*core.ReviewSession, error) {
	// Read all session files
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []*core.ReviewSession
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract session ID from filename
		sessionID := strings.TrimSuffix(entry.Name(), ".json")

		// Load the session
		session, err := m.LoadSession(ctx, sessionID)
		if err != nil {
			slog.WarnContext(ctx, "failed to load session",
				"session_id", sessionID,
				"error", err,
			)
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}
