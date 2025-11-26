package bedrock

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/waffle/waffle/internal/core"
)

// buildSemanticAnalysisPrompt builds a prompt for IaC semantic analysis
func (c *Client) buildSemanticAnalysisPrompt(resources []core.Resource) string {
	resourcesJSON := formatResources(resources)

	return fmt.Sprintf(`Analyze the following AWS resources for semantic understanding and security implications.

Resources:
%s

For each resource, identify:
1. Security-relevant configurations
2. Compliance implications
3. Relationships that affect security posture
4. Missing security controls

Return analysis as JSON with this exact structure:
{
  "security_findings": [
    {
      "resource": "resource_address",
      "findings": ["finding1", "finding2"],
      "severity": "high|medium|low"
    }
  ],
  "relationships": [
    {
      "from": "resource1",
      "to": "resource2",
      "type": "encryption|access|dependency",
      "status": "configured|missing|misconfigured"
    }
  ]
}

Respond ONLY with valid JSON, no additional text.`, resourcesJSON)
}

// buildWAFREvaluationPrompt builds a prompt for WAFR question evaluation
func (c *Client) buildWAFREvaluationPrompt(question *core.WAFRQuestion, model *core.WorkloadModel) string {
	bestPractices := formatBestPractices(question.BestPractices)
	choices := formatChoices(question.Choices)
	workloadJSON := formatWorkloadModel(model)

	return fmt.Sprintf(`You are evaluating an AWS workload against the Well-Architected Framework.

Question: %s
Pillar: %s
Description: %s

Best Practices:
%s

Available Choices:
%s

Workload Resources:
%s

Based on the infrastructure-as-code analysis, determine which choices apply to this workload.

For each applicable choice:
1. Explain why it applies
2. Provide specific evidence from the IaC (resource names, configurations)
3. Assign a confidence score (0.0-1.0) based on data completeness

Return your analysis as JSON with this exact structure:
{
  "selected_choices": ["choice_id_1", "choice_id_2"],
  "evidence": [
    {
      "choice_id": "choice_id_1",
      "explanation": "Explanation text",
      "resources": ["resource1", "resource2"],
      "confidence": 0.95
    }
  ],
  "overall_confidence": 0.90,
  "notes": "Additional context or caveats"
}

Respond ONLY with valid JSON, no additional text.`,
		question.Title,
		question.Pillar,
		question.Description,
		bestPractices,
		choices,
		workloadJSON,
	)
}

// buildImprovementPrompt builds a prompt for improvement plan generation
func (c *Client) buildImprovementPrompt(risk *core.Risk, resources []core.Resource) string {
	bestPractices := formatBestPractices(risk.MissingBestPractices)
	resourcesJSON := formatResources(resources)

	return fmt.Sprintf(`Generate an improvement plan item for the following WAFR risk.

Risk Details:
- Question: %s
- Pillar: %s
- Severity: %s
- Description: %s

Missing Best Practices:
%s

Affected Resources:
%s

Provide a high-level improvement plan that:
1. Describes what changes are needed (no code)
2. Explains why these changes improve the architecture
3. References relevant best practices and AWS documentation
4. Considers relationships between affected resources

Return as JSON with this exact structure:
{
  "description": "High-level description of recommended changes",
  "rationale": "Explanation of why these changes improve the architecture",
  "best_practice_refs": [
    "https://docs.aws.amazon.com/wellarchitected/..."
  ],
  "affected_resources": ["resource1", "resource2"],
  "estimated_effort": "LOW|MEDIUM|HIGH"
}

Respond ONLY with valid JSON, no additional text.`,
		risk.Question.Title,
		risk.Pillar,
		severityToString(risk.Severity),
		risk.Description,
		bestPractices,
		resourcesJSON,
	)
}

// formatResources formats resources for prompt inclusion
func formatResources(resources []core.Resource) string {
	if len(resources) == 0 {
		return "No resources provided"
	}

	var sb strings.Builder
	for i, resource := range resources {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		sb.WriteString(fmt.Sprintf("Resource %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Address: %s\n", resource.Address))
		sb.WriteString(fmt.Sprintf("  Type: %s\n", resource.Type))

		if len(resource.Properties) > 0 {
			propsJSON, _ := json.MarshalIndent(resource.Properties, "  ", "  ")
			sb.WriteString(fmt.Sprintf("  Properties: %s\n", string(propsJSON)))
		}

		if len(resource.Dependencies) > 0 {
			sb.WriteString(fmt.Sprintf("  Dependencies: %v\n", resource.Dependencies))
		}
	}

	return sb.String()
}

// formatBestPractices formats best practices for prompt inclusion
func formatBestPractices(practices []core.BestPractice) string {
	if len(practices) == 0 {
		return "No best practices specified"
	}

	var sb strings.Builder
	for i, practice := range practices {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- %s: %s", practice.Title, practice.Description))
	}

	return sb.String()
}

// formatChoices formats choices for prompt inclusion
func formatChoices(choices []core.Choice) string {
	if len(choices) == 0 {
		return "No choices available"
	}

	var sb strings.Builder
	for i, choice := range choices {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- %s: %s (%s)", choice.ID, choice.Title, choice.Description))
	}

	return sb.String()
}

// formatWorkloadModel formats workload model for prompt inclusion
func formatWorkloadModel(model *core.WorkloadModel) string {
	if model == nil {
		return "No workload model provided"
	}

	return formatResources(model.Resources)
}

// severityToString converts risk severity to string
func severityToString(severity core.RiskLevel) string {
	switch severity {
	case core.RiskLevelHigh:
		return "HIGH"
	case core.RiskLevelMedium:
		return "MEDIUM"
	case core.RiskLevelNone:
		return "NONE"
	default:
		return "UNKNOWN"
	}
}
