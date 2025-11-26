package bedrock

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/waffle/waffle/internal/core"
)

// SemanticAnalysisResponse represents the response from semantic analysis
type SemanticAnalysisResponse struct {
	SecurityFindings []SecurityFindingResponse `json:"security_findings"`
	Relationships    []RelationshipResponse    `json:"relationships"`
}

// SecurityFindingResponse represents a security finding in the response
type SecurityFindingResponse struct {
	Resource string   `json:"resource"`
	Findings []string `json:"findings"`
	Severity string   `json:"severity"`
}

// RelationshipResponse represents a relationship in the response
type RelationshipResponse struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// WAFREvaluationResponse represents the response from WAFR evaluation
type WAFREvaluationResponse struct {
	SelectedChoices   []string               `json:"selected_choices"`
	Evidence          []EvidenceResponse     `json:"evidence"`
	OverallConfidence float64                `json:"overall_confidence"`
	Notes             string                 `json:"notes"`
}

// EvidenceResponse represents evidence in the response
type EvidenceResponse struct {
	ChoiceID    string   `json:"choice_id"`
	Explanation string   `json:"explanation"`
	Resources   []string `json:"resources"`
	Confidence  float64  `json:"confidence"`
}

// ImprovementResponse represents the response from improvement generation
type ImprovementResponse struct {
	Description       string   `json:"description"`
	Rationale         string   `json:"rationale"`
	BestPracticeRefs  []string `json:"best_practice_refs"`
	AffectedResources []string `json:"affected_resources"`
	EstimatedEffort   string   `json:"estimated_effort"`
}

// parseSemanticAnalysisResponse parses the semantic analysis response
func (c *Client) parseSemanticAnalysisResponse(responseBody string) (*core.SemanticAnalysis, error) {
	// Extract JSON from response (may have markdown code blocks)
	jsonStr := extractJSON(responseBody)

	var response SemanticAnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse semantic analysis response: %w", err)
	}

	// Convert to core types
	analysis := &core.SemanticAnalysis{
		SecurityFindings: make([]core.SecurityFinding, len(response.SecurityFindings)),
		Relationships:    make([]core.Relationship, len(response.Relationships)),
	}

	for i, finding := range response.SecurityFindings {
		analysis.SecurityFindings[i] = core.SecurityFinding{
			Resource: finding.Resource,
			Findings: finding.Findings,
			Severity: finding.Severity,
		}
	}

	for i, rel := range response.Relationships {
		analysis.Relationships[i] = core.Relationship{
			From:   rel.From,
			To:     rel.To,
			Type:   rel.Type,
			Status: rel.Status,
		}
	}

	return analysis, nil
}

// parseWAFREvaluationResponse parses the WAFR evaluation response
func (c *Client) parseWAFREvaluationResponse(responseBody string, question *core.WAFRQuestion) (*core.QuestionEvaluation, error) {
	// Extract JSON from response
	jsonStr := extractJSON(responseBody)

	var response WAFREvaluationResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse WAFR evaluation response: %w", err)
	}

	// Validate confidence scores
	if response.OverallConfidence < 0.0 || response.OverallConfidence > 1.0 {
		return nil, fmt.Errorf("invalid overall confidence: %f", response.OverallConfidence)
	}

	for _, ev := range response.Evidence {
		if ev.Confidence < 0.0 || ev.Confidence > 1.0 {
			return nil, fmt.Errorf("invalid confidence for choice %s: %f", ev.ChoiceID, ev.Confidence)
		}
	}

	// Convert to core types
	evaluation := &core.QuestionEvaluation{
		Question:        question,
		SelectedChoices: make([]core.Choice, 0),
		Evidence:        make([]core.Evidence, len(response.Evidence)),
		ConfidenceScore: response.OverallConfidence,
		Notes:           response.Notes,
	}

	// Map selected choice IDs to Choice objects
	choiceMap := make(map[string]core.Choice)
	for _, choice := range question.Choices {
		choiceMap[choice.ID] = choice
	}

	for _, choiceID := range response.SelectedChoices {
		if choice, ok := choiceMap[choiceID]; ok {
			evaluation.SelectedChoices = append(evaluation.SelectedChoices, choice)
		}
	}

	// Convert evidence
	for i, ev := range response.Evidence {
		evaluation.Evidence[i] = core.Evidence{
			ChoiceID:    ev.ChoiceID,
			Explanation: ev.Explanation,
			Resources:   ev.Resources,
			Confidence:  ev.Confidence,
		}
	}

	return evaluation, nil
}

// parseImprovementResponse parses the improvement response
func (c *Client) parseImprovementResponse(responseBody string, risk *core.Risk) (*core.ImprovementPlanItem, error) {
	// Extract JSON from response
	jsonStr := extractJSON(responseBody)

	var response ImprovementResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse improvement response: %w", err)
	}

	// Convert to core type
	item := &core.ImprovementPlanItem{
		ID:                fmt.Sprintf("improvement-%s", risk.ID),
		Risk:              risk,
		Description:       response.Description,
		BestPracticeRefs:  response.BestPracticeRefs,
		AffectedResources: response.AffectedResources,
		EstimatedEffort:   response.EstimatedEffort,
	}

	return item, nil
}

// extractPartialSemanticData attempts to extract partial data from malformed response
func (c *Client) extractPartialSemanticData(responseBody string) (*core.SemanticAnalysis, error) {
	// Try to find any JSON-like structure
	jsonStr := extractJSON(responseBody)

	// Try to parse as much as possible
	var partial struct {
		SecurityFindings []SecurityFindingResponse `json:"security_findings,omitempty"`
		Relationships    []RelationshipResponse    `json:"relationships,omitempty"`
	}

	// Ignore errors, just extract what we can
	_ = json.Unmarshal([]byte(jsonStr), &partial)

	analysis := &core.SemanticAnalysis{
		SecurityFindings: make([]core.SecurityFinding, len(partial.SecurityFindings)),
		Relationships:    make([]core.Relationship, len(partial.Relationships)),
	}

	for i, finding := range partial.SecurityFindings {
		analysis.SecurityFindings[i] = core.SecurityFinding{
			Resource: finding.Resource,
			Findings: finding.Findings,
			Severity: finding.Severity,
		}
	}

	for i, rel := range partial.Relationships {
		analysis.Relationships[i] = core.Relationship{
			From:   rel.From,
			To:     rel.To,
			Type:   rel.Type,
			Status: rel.Status,
		}
	}

	return analysis, nil
}

// extractJSON extracts JSON from response that may contain markdown code blocks
func extractJSON(response string) string {
	// Remove markdown code blocks if present
	response = strings.TrimSpace(response)

	// Check for ```json ... ``` blocks
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if idx := strings.LastIndex(response, "```"); idx != -1 {
			response = response[:idx]
		}
	}

	return strings.TrimSpace(response)
}
