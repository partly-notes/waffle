package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReviewCommandJSONOutput tests that the review command outputs valid JSON
func TestReviewCommandJSONOutput(t *testing.T) {
	// Build the binary
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	// Run the review command
	cmd := exec.Command(binaryPath, "review", "--workload-id", "test-workload")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "Command should succeed")

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err, "Output should be valid JSON")

	// Verify required fields
	assert.Contains(t, result, "session_id")
	assert.Contains(t, result, "workload_id")
	assert.Contains(t, result, "status")
	assert.Contains(t, result, "created_at")
	assert.Contains(t, result, "summary")

	// Verify workload_id matches input
	assert.Equal(t, "test-workload", result["workload_id"])

	// Verify summary structure
	summary, ok := result["summary"].(map[string]interface{})
	require.True(t, ok, "Summary should be an object")
	assert.Contains(t, summary, "questions_evaluated")
	assert.Contains(t, summary, "high_risks")
	assert.Contains(t, summary, "medium_risks")
	assert.Contains(t, summary, "average_confidence")
	assert.Contains(t, summary, "improvement_plan_size")
}

// TestStatusCommandJSONOutput tests that the status command outputs valid JSON
func TestStatusCommandJSONOutput(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	cmd := exec.Command(binaryPath, "status", "test-session-123")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "Command should succeed")

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err, "Output should be valid JSON")

	// Verify required fields
	assert.Contains(t, result, "session_id")
	assert.Contains(t, result, "workload_id")
	assert.Contains(t, result, "status")
	assert.Contains(t, result, "created_at")
	assert.Contains(t, result, "updated_at")

	// Verify session_id matches input
	assert.Equal(t, "test-session-123", result["session_id"])
}

// TestResultsCommandJSONOutput tests that the results command outputs valid JSON
func TestResultsCommandJSONOutput(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	t.Run("stdout output", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "results", "test-session-456")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		require.NoError(t, err, "Command should succeed")

		// Parse JSON output
		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err, "Output should be valid JSON")

		// Verify required fields
		assert.Contains(t, result, "session_id")
		assert.Contains(t, result, "workload_id")
		assert.Contains(t, result, "aws_workload_id")
		assert.Contains(t, result, "status")
		assert.Contains(t, result, "scope")
		assert.Contains(t, result, "summary")

		// Verify session_id matches input
		assert.Equal(t, "test-session-456", result["session_id"])

		// Verify scope structure
		scope, ok := result["scope"].(map[string]interface{})
		require.True(t, ok, "Scope should be an object")
		assert.Contains(t, scope, "level")

		// Verify summary structure
		summary, ok := result["summary"].(map[string]interface{})
		require.True(t, ok, "Summary should be an object")
		assert.Contains(t, summary, "questions_evaluated")
		assert.Contains(t, summary, "high_risks")
		assert.Contains(t, summary, "medium_risks")
		assert.Contains(t, summary, "average_confidence")
		assert.Contains(t, summary, "improvement_plan_size")
	})

	t.Run("file output", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "results.json")

		cmd := exec.Command(binaryPath, "results", "test-session-789", "--format", "json", "--output", tmpFile)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		require.NoError(t, err, "Command should succeed")

		// Verify file was created
		_, err = os.Stat(tmpFile)
		require.NoError(t, err, "Output file should exist")

		// Read and parse file
		data, err := os.ReadFile(tmpFile)
		require.NoError(t, err, "Should be able to read output file")

		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		require.NoError(t, err, "File content should be valid JSON")

		// Verify session_id matches input
		assert.Equal(t, "test-session-789", result["session_id"])
	})
}

// TestCompareCommandJSONOutput tests that the compare command outputs valid JSON
func TestCompareCommandJSONOutput(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	cmd := exec.Command(binaryPath, "compare", "session-1", "session-2")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "Command should succeed")

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err, "Output should be valid JSON")

	// Verify required fields
	assert.Contains(t, result, "session_id_1")
	assert.Contains(t, result, "session_id_2")
	assert.Contains(t, result, "workload_id")
	assert.Contains(t, result, "improvements")
	assert.Contains(t, result, "regressions")
	assert.Contains(t, result, "new_risks")
	assert.Contains(t, result, "summary")

	// Verify session IDs match input
	assert.Equal(t, "session-1", result["session_id_1"])
	assert.Equal(t, "session-2", result["session_id_2"])

	// Verify summary structure
	summary, ok := result["summary"].(map[string]interface{})
	require.True(t, ok, "Summary should be an object")
	assert.Contains(t, summary, "total_improvements")
	assert.Contains(t, summary, "total_regressions")
	assert.Contains(t, summary, "total_new_risks")
}

// TestJSONOutputWithDifferentScopes tests JSON output with different review scopes
func TestJSONOutputWithDifferentScopes(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)

	tests := []struct {
		name  string
		args  []string
		check func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "workload scope",
			args: []string{"review", "--workload-id", "test", "--scope", "workload"},
			check: func(t *testing.T, result map[string]interface{}) {
				metadata, ok := result["metadata"].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, metadata["scope"], "workload")
			},
		},
		{
			name: "pillar scope",
			args: []string{"review", "--workload-id", "test", "--scope", "pillar", "--pillar", "security"},
			check: func(t *testing.T, result map[string]interface{}) {
				metadata, ok := result["metadata"].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, metadata["scope"], "security")
			},
		},
		{
			name: "question scope",
			args: []string{"review", "--workload-id", "test", "--scope", "question", "--question-id", "sec_data_1"},
			check: func(t *testing.T, result map[string]interface{}) {
				metadata, ok := result["metadata"].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, metadata["scope"], "sec_data_1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			require.NoError(t, err, "Command should succeed")

			var result map[string]interface{}
			err = json.Unmarshal(stdout.Bytes(), &result)
			require.NoError(t, err, "Output should be valid JSON")

			tt.check(t, result)
		})
	}
}

// buildTestBinary builds the waffle binary for testing
func buildTestBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "waffle")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(output))

	return binaryPath
}
