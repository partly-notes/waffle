package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProgressReporter_ReportStep(t *testing.T) {
	tests := []struct {
		name            string
		step            string
		message         string
		expectedContain []string
	}{
		{
			name:    "step with message",
			step:    "iac_analysis",
			message: "Analyzing infrastructure-as-code files...",
			expectedContain: []string{
				"Iac Analysis",
				"Analyzing infrastructure-as-code files...",
			},
		},
		{
			name:    "step without message",
			step:    "retrieve_questions",
			message: "",
			expectedContain: []string{
				"Retrieve Questions",
			},
		},
		{
			name:    "multi-word step",
			step:    "submit_answers",
			message: "Submitting to AWS",
			expectedContain: []string{
				"Submit Answers",
				"Submitting to AWS",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			reporter := NewCLIProgressReporter(buf)

			reporter.ReportStep(tt.step, tt.message)

			output := buf.String()
			for _, expected := range tt.expectedContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestCLIProgressReporter_ReportProgress(t *testing.T) {
	tests := []struct {
		name            string
		current         int
		total           int
		message         string
		expectedContain []string
	}{
		{
			name:    "progress at 50%",
			current: 5,
			total:   10,
			message: "Processing item 5 of 10",
			expectedContain: []string{
				"50%",
				"Processing item 5 of 10",
			},
		},
		{
			name:    "progress at 100%",
			current: 10,
			total:   10,
			message: "Complete",
			expectedContain: []string{
				"100%",
				"Complete",
			},
		},
		{
			name:    "progress at start",
			current: 1,
			total:   5,
			message: "Starting",
			expectedContain: []string{
				"20%",
				"Starting",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			reporter := NewCLIProgressReporter(buf)

			reporter.ReportProgress(tt.current, tt.total, tt.message)

			output := buf.String()
			for _, expected := range tt.expectedContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestCLIProgressReporter_ReportCompletion(t *testing.T) {
	buf := &bytes.Buffer{}
	reporter := NewCLIProgressReporter(buf)

	summary := &ResultsSummary{
		TotalQuestions:      10,
		QuestionsEvaluated:  10,
		HighRisks:           2,
		MediumRisks:         3,
		AverageConfidence:   0.85,
		ImprovementPlanSize: 5,
	}

	reporter.ReportCompletion(summary)

	output := buf.String()
	assert.Contains(t, output, "Review completed successfully")
	assert.Contains(t, output, "Questions evaluated: 10")
	assert.Contains(t, output, "High risks: 2")
	assert.Contains(t, output, "Medium risks: 3")
	assert.Contains(t, output, "Average confidence: 0.85")
	assert.Contains(t, output, "Improvement plan items: 5")
}

func TestFormatStepName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "iac_analysis",
			expected: "Iac Analysis",
		},
		{
			input:    "retrieve_questions",
			expected: "Retrieve Questions",
		},
		{
			input:    "submit_answers",
			expected: "Submit Answers",
		},
		{
			input:    "single",
			expected: "Single",
		},
		{
			input:    "multiple_word_step_name",
			expected: "Multiple Word Step Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatStepName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		validate func(t *testing.T, bar string)
	}{
		{
			name:    "empty progress",
			current: 0,
			total:   10,
			width:   10,
			validate: func(t *testing.T, bar string) {
				// Check rune count instead of byte length
				assert.Equal(t, 10, len([]rune(bar)))
				assert.Equal(t, strings.Repeat("░", 10), bar)
			},
		},
		{
			name:    "half progress",
			current: 5,
			total:   10,
			width:   10,
			validate: func(t *testing.T, bar string) {
				// Check rune count instead of byte length
				assert.Equal(t, 10, len([]rune(bar)))
				assert.Contains(t, bar, "█")
				assert.Contains(t, bar, "░")
			},
		},
		{
			name:    "full progress",
			current: 10,
			total:   10,
			width:   10,
			validate: func(t *testing.T, bar string) {
				// Check rune count instead of byte length
				assert.Equal(t, 10, len([]rune(bar)))
				assert.Equal(t, strings.Repeat("█", 10), bar)
			},
		},
		{
			name:    "zero total",
			current: 0,
			total:   0,
			width:   10,
			validate: func(t *testing.T, bar string) {
				assert.Equal(t, 10, len(bar))
				assert.Equal(t, strings.Repeat(" ", 10), bar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := progressBar(tt.current, tt.total, tt.width)
			tt.validate(t, bar)
		})
	}
}

func TestCLIProgressReporter_Integration(t *testing.T) {
	buf := &bytes.Buffer{}
	reporter := NewCLIProgressReporter(buf)

	// Simulate a complete workflow
	reporter.ReportStep("iac_analysis", "Analyzing files...")
	reporter.ReportProgress(1, 1, "Analysis complete")

	reporter.ReportStep("evaluate_questions", "Evaluating questions...")
	for i := 1; i <= 3; i++ {
		reporter.ReportProgress(i, 3, "Evaluating...")
	}

	summary := &ResultsSummary{
		QuestionsEvaluated:  3,
		HighRisks:           1,
		MediumRisks:         1,
		AverageConfidence:   0.90,
		ImprovementPlanSize: 2,
	}
	reporter.ReportCompletion(summary)

	output := buf.String()

	// Verify all steps are present
	require.Contains(t, output, "Iac Analysis")
	require.Contains(t, output, "Evaluate Questions")
	require.Contains(t, output, "Review completed successfully")

	// Verify progress indicators
	require.Contains(t, output, "100%")

	// Verify summary
	require.Contains(t, output, "Questions evaluated: 3")
	require.Contains(t, output, "High risks: 1")
}
