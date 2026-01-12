package wafr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
)

// GetConsolidatedReport retrieves a consolidated report from AWS in the specified format
func (e *Evaluator) GetConsolidatedReport(
	ctx context.Context,
	awsWorkloadID string,
	format string,
) ([]byte, error) {
	if awsWorkloadID == "" {
		return nil, errors.New("AWS workload ID is required")
	}

	var reportFormat types.ReportFormat
	switch format {
	case "pdf", "PDF":
		reportFormat = types.ReportFormatPdf
	case "json", "JSON":
		reportFormat = types.ReportFormatJson
	default:
		return nil, fmt.Errorf("unsupported report format: %s (supported: pdf, json)", format)
	}

	input := &wellarchitected.GetConsolidatedReportInput{
		Format:                 reportFormat,
		IncludeSharedResources: aws.Bool(false),
	}

	var output *wellarchitected.GetConsolidatedReportOutput
	err := e.retryWithBackoff(ctx, "GetConsolidatedReport", func() error {
		var err error
		output, err = e.client.GetConsolidatedReport(ctx, input)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get consolidated report: %w", err)
	}

	reportData := aws.ToString(output.Base64String)

	// Decode base64 string to bytes
	decodedData, err := base64.StdEncoding.DecodeString(reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode report data: %w", err)
	}

	slog.InfoContext(ctx, "consolidated report retrieved",
		"aws_workload_id", awsWorkloadID,
		"format", format,
		"size_bytes", len(decodedData),
	)

	return decodedData, nil
}

// EnhancedReportData represents the enhanced JSON report with IaC evidence
type EnhancedReportData struct {
	// AWS workload information
	AWSWorkloadID string    `json:"aws_workload_id"`
	WorkloadName  string    `json:"workload_name"`
	ConsoleLink   string    `json:"console_link"`
	GeneratedAt   time.Time `json:"generated_at"`

	// IaC information
	IaCFramework string   `json:"iac_framework"`
	IaCSourceType string  `json:"iac_source_type"`
	ResourceCount int      `json:"resource_count"`
	Resources     []string `json:"resources"`

	// Review results
	Evaluations       []EvaluationSummary `json:"evaluations"`
	Risks             []RiskSummary       `json:"risks"`
	ImprovementPlan   []string            `json:"improvement_plan"`
	AverageConfidence float64             `json:"average_confidence"`

	// Base AWS report (if available)
	AWSReport map[string]interface{} `json:"aws_report,omitempty"`
}

// EvaluationSummary represents a summary of a question evaluation
type EvaluationSummary struct {
	QuestionID      string   `json:"question_id"`
	QuestionTitle   string   `json:"question_title"`
	Pillar          string   `json:"pillar"`
	SelectedChoices []string `json:"selected_choices"`
	ConfidenceScore float64  `json:"confidence_score"`
	Evidence        []string `json:"evidence"`
	Notes           string   `json:"notes,omitempty"`
}

// RiskSummary represents a summary of an identified risk
type RiskSummary struct {
	ID                string   `json:"id"`
	QuestionID        string   `json:"question_id"`
	QuestionTitle     string   `json:"question_title"`
	Pillar            string   `json:"pillar"`
	Severity          string   `json:"severity"`
	Description       string   `json:"description"`
	AffectedResources []string `json:"affected_resources"`
}

// GetResultsJSON retrieves results in JSON format enhanced with IaC evidence
func (e *Evaluator) GetResultsJSON(
	ctx context.Context,
	awsWorkloadID string,
	sessionData map[string]interface{},
) (map[string]interface{}, error) {
	if awsWorkloadID == "" {
		return nil, errors.New("AWS workload ID is required")
	}

	// Get base AWS report in JSON format
	baseReport, err := e.GetConsolidatedReport(ctx, awsWorkloadID, "json")
	if err != nil {
		slog.WarnContext(ctx, "failed to get base AWS report, continuing with session data only",
			"error", err,
		)
	}

	// Parse base report if available
	var awsReportData map[string]interface{}
	if baseReport != nil {
		if err := json.Unmarshal(baseReport, &awsReportData); err != nil {
			slog.WarnContext(ctx, "failed to parse AWS report JSON",
				"error", err,
			)
		}
	}

	// Build enhanced report
	enhancedReport := &EnhancedReportData{
		AWSWorkloadID: awsWorkloadID,
		GeneratedAt:   time.Now().UTC(),
		AWSReport:     awsReportData,
	}

	// Extract workload name from session data
	if workloadName, ok := sessionData["workload_id"].(string); ok {
		enhancedReport.WorkloadName = workloadName
	}

	// Build console link
	region := "us-east-1" // Default region
	if r, ok := sessionData["region"].(string); ok {
		region = r
	}
	enhancedReport.ConsoleLink = fmt.Sprintf(
		"https://%s.console.aws.amazon.com/wellarchitected/home?region=%s#/workload/%s",
		region, region, awsWorkloadID,
	)

	// Extract IaC information from session data
	if workloadModel, ok := sessionData["workload_model"].(map[string]interface{}); ok {
		if framework, ok := workloadModel["framework"].(string); ok {
			enhancedReport.IaCFramework = framework
		}
		if sourceType, ok := workloadModel["source_type"].(string); ok {
			enhancedReport.IaCSourceType = sourceType
		}
		if resources, ok := workloadModel["resources"].([]interface{}); ok {
			enhancedReport.ResourceCount = len(resources)
			for _, r := range resources {
				if resource, ok := r.(map[string]interface{}); ok {
					if addr, ok := resource["address"].(string); ok {
						enhancedReport.Resources = append(enhancedReport.Resources, addr)
					}
				}
			}
		}
	}

	// Extract evaluation results from session data
	if results, ok := sessionData["results"].(map[string]interface{}); ok {
		// Extract evaluations
		if evaluations, ok := results["evaluations"].([]interface{}); ok {
			totalConfidence := 0.0
			for _, e := range evaluations {
				if eval, ok := e.(map[string]interface{}); ok {
					summary := EvaluationSummary{}

					if question, ok := eval["question"].(map[string]interface{}); ok {
						if id, ok := question["id"].(string); ok {
							summary.QuestionID = id
						}
						if title, ok := question["title"].(string); ok {
							summary.QuestionTitle = title
						}
						if pillar, ok := question["pillar"].(string); ok {
							summary.Pillar = pillar
						}
					}

					if choices, ok := eval["selected_choices"].([]interface{}); ok {
						for _, c := range choices {
							if choice, ok := c.(map[string]interface{}); ok {
								if title, ok := choice["title"].(string); ok {
									summary.SelectedChoices = append(summary.SelectedChoices, title)
								}
							}
						}
					}

					if confidence, ok := eval["confidence_score"].(float64); ok {
						summary.ConfidenceScore = confidence
						totalConfidence += confidence
					}

					if evidence, ok := eval["evidence"].([]interface{}); ok {
						for _, ev := range evidence {
							if e, ok := ev.(map[string]interface{}); ok {
								if explanation, ok := e["explanation"].(string); ok {
									summary.Evidence = append(summary.Evidence, explanation)
								}
							}
						}
					}

					if notes, ok := eval["notes"].(string); ok {
						summary.Notes = notes
					}

					enhancedReport.Evaluations = append(enhancedReport.Evaluations, summary)
				}
			}

			if len(evaluations) > 0 {
				enhancedReport.AverageConfidence = totalConfidence / float64(len(evaluations))
			}
		}

		// Extract risks
		if risks, ok := results["risks"].([]interface{}); ok {
			for _, r := range risks {
				if risk, ok := r.(map[string]interface{}); ok {
					summary := RiskSummary{}

					if id, ok := risk["id"].(string); ok {
						summary.ID = id
					}

					if question, ok := risk["question"].(map[string]interface{}); ok {
						if id, ok := question["id"].(string); ok {
							summary.QuestionID = id
						}
						if title, ok := question["title"].(string); ok {
							summary.QuestionTitle = title
						}
					}

					if pillar, ok := risk["pillar"].(string); ok {
						summary.Pillar = pillar
					}

					if severity, ok := risk["severity"].(string); ok {
						summary.Severity = severity
					}

					if description, ok := risk["description"].(string); ok {
						summary.Description = description
					}

					if resources, ok := risk["affected_resources"].([]interface{}); ok {
						for _, res := range resources {
							if r, ok := res.(string); ok {
								summary.AffectedResources = append(summary.AffectedResources, r)
							}
						}
					}

					enhancedReport.Risks = append(enhancedReport.Risks, summary)
				}
			}
		}

		// Extract improvement plan
		if improvementPlan, ok := results["improvement_plan"].(map[string]interface{}); ok {
			if items, ok := improvementPlan["items"].([]interface{}); ok {
				for _, item := range items {
					if i, ok := item.(map[string]interface{}); ok {
						if description, ok := i["description"].(string); ok {
							enhancedReport.ImprovementPlan = append(enhancedReport.ImprovementPlan, description)
						}
					}
				}
			}
		}
	}

	// Convert enhanced report to map
	reportBytes, err := json.Marshal(enhancedReport)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enhanced report: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(reportBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enhanced report: %w", err)
	}

	slog.InfoContext(ctx, "enhanced JSON report generated",
		"aws_workload_id", awsWorkloadID,
		"evaluations", len(enhancedReport.Evaluations),
		"risks", len(enhancedReport.Risks),
		"resources", enhancedReport.ResourceCount,
	)

	return result, nil
}
