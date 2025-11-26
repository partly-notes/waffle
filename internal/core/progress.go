package core

import (
	"fmt"
	"io"
	"strings"
)

// CLIProgressReporter implements ProgressReporter for CLI output
type CLIProgressReporter struct {
	writer io.Writer
}

// NewCLIProgressReporter creates a new CLI progress reporter
func NewCLIProgressReporter(writer io.Writer) *CLIProgressReporter {
	return &CLIProgressReporter{
		writer: writer,
	}
}

// ReportStep reports the current step being executed
func (p *CLIProgressReporter) ReportStep(step string, message string) {
	stepName := formatStepName(step)
	fmt.Fprintf(p.writer, "\n▶ %s\n", stepName)
	if message != "" {
		fmt.Fprintf(p.writer, "  %s\n", message)
	}
}

// ReportProgress reports progress within a step
func (p *CLIProgressReporter) ReportProgress(current, total int, message string) {
	if total > 0 {
		percentage := (current * 100) / total
		bar := progressBar(current, total, 30)
		fmt.Fprintf(p.writer, "\r  [%s] %d%% - %s", bar, percentage, message)
		if current == total {
			fmt.Fprintf(p.writer, "\n")
		}
	} else {
		fmt.Fprintf(p.writer, "  %s\n", message)
	}
}

// ReportCompletion reports completion of the review
func (p *CLIProgressReporter) ReportCompletion(summary *ResultsSummary) {
	fmt.Fprintf(p.writer, "\n✓ Review completed successfully!\n\n")
	fmt.Fprintf(p.writer, "Summary:\n")
	fmt.Fprintf(p.writer, "  Questions evaluated: %d\n", summary.QuestionsEvaluated)
	fmt.Fprintf(p.writer, "  High risks: %d\n", summary.HighRisks)
	fmt.Fprintf(p.writer, "  Medium risks: %d\n", summary.MediumRisks)
	fmt.Fprintf(p.writer, "  Average confidence: %.2f\n", summary.AverageConfidence)
	fmt.Fprintf(p.writer, "  Improvement plan items: %d\n", summary.ImprovementPlanSize)
	fmt.Fprintf(p.writer, "\n")
}

// formatStepName formats a step name for display
func formatStepName(step string) string {
	// Convert snake_case to Title Case
	words := strings.Split(step, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// progressBar generates a text-based progress bar
func progressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat(" ", width)
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}
