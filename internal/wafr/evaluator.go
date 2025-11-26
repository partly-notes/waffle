package wafr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected"
	"github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
	"github.com/aws/smithy-go"

	"github.com/waffle/waffle/internal/core"
)

// WAFRClient defines the interface for AWS Well-Architected Tool operations
type WAFRClient interface {
	CreateWorkload(ctx context.Context, params *wellarchitected.CreateWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateWorkloadOutput, error)
	GetWorkload(ctx context.Context, params *wellarchitected.GetWorkloadInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetWorkloadOutput, error)
	ListWorkloads(ctx context.Context, params *wellarchitected.ListWorkloadsInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListWorkloadsOutput, error)
	ListAnswers(ctx context.Context, params *wellarchitected.ListAnswersInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.ListAnswersOutput, error)
	UpdateAnswer(ctx context.Context, params *wellarchitected.UpdateAnswerInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.UpdateAnswerOutput, error)
	CreateMilestone(ctx context.Context, params *wellarchitected.CreateMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.CreateMilestoneOutput, error)
	GetConsolidatedReport(ctx context.Context, params *wellarchitected.GetConsolidatedReportInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetConsolidatedReportOutput, error)
	GetMilestone(ctx context.Context, params *wellarchitected.GetMilestoneInput, optFns ...func(*wellarchitected.Options)) (*wellarchitected.GetMilestoneOutput, error)
}

// wrapWAFRError wraps a WAFR API error with additional context
func wrapWAFRError(operation string, err error) error {
	if err == nil {
		return nil
	}
	
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return &core.WAFRAPIError{
			Operation: operation,
			ErrorCode: apiErr.ErrorCode(),
			Message:   apiErr.ErrorMessage(),
			Err:       err,
		}
	}
	
	return &core.WAFRAPIError{
		Operation: operation,
		Err:       err,
	}
}

// BedrockClient defines the interface for Bedrock operations
type BedrockClient interface {
	EvaluateWAFRQuestion(ctx context.Context, question *core.WAFRQuestion, workloadModel *core.WorkloadModel) (*core.QuestionEvaluation, error)
	GenerateImprovementGuidance(ctx context.Context, risk *core.Risk, resources []core.Resource) (*core.ImprovementPlanItem, error)
}

// Evaluator implements the WAFREvaluator interface
type Evaluator struct {
	client     WAFRClient
	maxRetries int
	baseDelay  time.Duration
}

// EvaluatorConfig holds configuration for the WAFR evaluator
type EvaluatorConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// DefaultEvaluatorConfig returns default configuration
func DefaultEvaluatorConfig() *EvaluatorConfig {
	return &EvaluatorConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}
}

// NewEvaluator creates a new WAFR evaluator with the given client
func NewEvaluator(client WAFRClient, config *EvaluatorConfig) *Evaluator {
	if config == nil {
		config = DefaultEvaluatorConfig()
	}
	return &Evaluator{
		client:     client,
		maxRetries: config.MaxRetries,
		baseDelay:  config.BaseDelay,
	}
}

// CreateWorkload creates a workload in AWS Well-Architected Tool or returns existing one
func (e *Evaluator) CreateWorkload(
	ctx context.Context,
	workloadID string,
	description string,
) (string, error) {
	if workloadID == "" {
		return "", errors.New("workload ID is required")
	}

	// First, check if a workload with this name already exists
	existingWorkloadID, err := e.findWorkloadByName(ctx, workloadID)
	if err == nil && existingWorkloadID != "" {
		slog.InfoContext(ctx, "workload already exists, reusing",
			"workload_id", workloadID,
			"aws_workload_id", existingWorkloadID,
		)
		return existingWorkloadID, nil
	}

	// Create new workload if it doesn't exist
	input := &wellarchitected.CreateWorkloadInput{
		WorkloadName: aws.String(workloadID),
		Description:  aws.String(description),
		Environment:  types.WorkloadEnvironmentProduction,
		Lenses:       []string{"wellarchitected"},
		ReviewOwner:  aws.String("waffle-automated"),
		AwsRegions:   []string{"us-east-1"}, // Required field
	}

	var output *wellarchitected.CreateWorkloadOutput
	err = e.retryWithBackoff(ctx, "CreateWorkload", func() error {
		var err error
		output, err = e.client.CreateWorkload(ctx, input)
		return err
	})

	if err != nil {
		// Check if it's a conflict error (workload already exists)
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ConflictException" {
			// Try to find the existing workload
			existingWorkloadID, findErr := e.findWorkloadByName(ctx, workloadID)
			if findErr == nil && existingWorkloadID != "" {
				slog.InfoContext(ctx, "workload exists (conflict), reusing",
					"workload_id", workloadID,
					"aws_workload_id", existingWorkloadID,
				)
				return existingWorkloadID, nil
			}
		}
		return "", wrapWAFRError("CreateWorkload", err)
	}

	awsWorkloadID := aws.ToString(output.WorkloadId)
	slog.InfoContext(ctx, "workload created",
		"workload_id", workloadID,
		"aws_workload_id", awsWorkloadID,
	)

	return awsWorkloadID, nil
}

// findWorkloadByName searches for a workload by name and returns its ID
func (e *Evaluator) findWorkloadByName(ctx context.Context, workloadName string) (string, error) {
	var nextToken *string

	for {
		input := &wellarchitected.ListWorkloadsInput{
			WorkloadNamePrefix: aws.String(workloadName),
			NextToken:          nextToken,
			MaxResults:         aws.Int32(50),
		}

		var output *wellarchitected.ListWorkloadsOutput
		err := e.retryWithBackoff(ctx, "ListWorkloads", func() error {
			var err error
			output, err = e.client.ListWorkloads(ctx, input)
			return err
		})

		if err != nil {
			return "", fmt.Errorf("failed to list workloads: %w", err)
		}

		// Look for exact name match
		for _, workload := range output.WorkloadSummaries {
			if aws.ToString(workload.WorkloadName) == workloadName {
				return aws.ToString(workload.WorkloadId), nil
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return "", fmt.Errorf("workload %s not found", workloadName)
}

// GetQuestions retrieves WAFR questions based on scope
func (e *Evaluator) GetQuestions(
	ctx context.Context,
	awsWorkloadID string,
	scope core.ReviewScope,
) ([]*core.WAFRQuestion, error) {
	if awsWorkloadID == "" {
		return nil, errors.New("AWS workload ID is required")
	}

	if err := scope.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scope: %w", err)
	}

	var questions []*core.WAFRQuestion

	switch scope.Level {
	case core.ScopeLevelWorkload:
		// Get questions for all pillars
		for _, pillar := range []core.Pillar{
			core.PillarOperationalExcellence,
			core.PillarSecurity,
			core.PillarReliability,
			core.PillarPerformanceEfficiency,
			core.PillarCostOptimization,
			core.PillarSustainability,
		} {
			pillarQuestions, err := e.getQuestionsForPillar(ctx, awsWorkloadID, pillar)
			if err != nil {
				return nil, fmt.Errorf("failed to get questions for pillar %s: %w", pillar, err)
			}
			questions = append(questions, pillarQuestions...)
		}

	case core.ScopeLevelPillar:
		if scope.Pillar == nil {
			return nil, errors.New("pillar is required for pillar scope")
		}
		var err error
		questions, err = e.getQuestionsForPillar(ctx, awsWorkloadID, *scope.Pillar)
		if err != nil {
			return nil, fmt.Errorf("failed to get questions for pillar: %w", err)
		}

	case core.ScopeLevelQuestion:
		if scope.QuestionID == "" {
			return nil, errors.New("question ID is required for question scope")
		}
		question, err := e.getSpecificQuestion(ctx, awsWorkloadID, scope.QuestionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get specific question: %w", err)
		}
		questions = []*core.WAFRQuestion{question}
	}

	slog.InfoContext(ctx, "retrieved questions",
		"aws_workload_id", awsWorkloadID,
		"scope_level", scope.Level,
		"question_count", len(questions),
	)

	return questions, nil
}

// getQuestionsForPillar retrieves all questions for a specific pillar
func (e *Evaluator) getQuestionsForPillar(
	ctx context.Context,
	awsWorkloadID string,
	pillar core.Pillar,
) ([]*core.WAFRQuestion, error) {
	var questions []*core.WAFRQuestion
	var nextToken *string

	pillarID := mapPillarToAWSID(pillar)

	for {
		input := &wellarchitected.ListAnswersInput{
			WorkloadId: aws.String(awsWorkloadID),
			LensAlias:  aws.String("wellarchitected"),
			PillarId:   aws.String(pillarID),
			NextToken:  nextToken,
			MaxResults: aws.Int32(50),
		}

		var output *wellarchitected.ListAnswersOutput
		err := e.retryWithBackoff(ctx, "ListAnswers", func() error {
			var err error
			output, err = e.client.ListAnswers(ctx, input)
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list answers: %w", err)
		}

		for _, answer := range output.AnswerSummaries {
			question := convertAnswerToQuestion(answer, pillar)
			questions = append(questions, question)
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return questions, nil
}

// getSpecificQuestion retrieves a specific question by ID
func (e *Evaluator) getSpecificQuestion(
	ctx context.Context,
	awsWorkloadID string,
	questionID string,
) (*core.WAFRQuestion, error) {
	// For a specific question, we need to determine which pillar it belongs to
	// We'll search through all pillars to find it
	for _, pillar := range []core.Pillar{
		core.PillarOperationalExcellence,
		core.PillarSecurity,
		core.PillarReliability,
		core.PillarPerformanceEfficiency,
		core.PillarCostOptimization,
		core.PillarSustainability,
	} {
		questions, err := e.getQuestionsForPillar(ctx, awsWorkloadID, pillar)
		if err != nil {
			continue
		}

		for _, q := range questions {
			if q.ID == questionID {
				return q, nil
			}
		}
	}

	return nil, fmt.Errorf("question %s not found", questionID)
}

// EvaluateQuestion evaluates a single question against the workload
func (e *Evaluator) EvaluateQuestion(
	ctx context.Context,
	question *core.WAFRQuestion,
	workloadModel *core.WorkloadModel,
	bedrockClient BedrockClient,
) (*core.QuestionEvaluation, error) {
	if question == nil {
		return nil, errors.New("question is required")
	}
	if workloadModel == nil {
		return nil, errors.New("workload model is required")
	}
	if bedrockClient == nil {
		return nil, errors.New("bedrock client is required")
	}

	slog.InfoContext(ctx, "evaluating question",
		"question_id", question.ID,
		"pillar", question.Pillar,
		"resource_count", len(workloadModel.Resources),
	)

	// Use Bedrock to evaluate the question
	evaluation, err := bedrockClient.EvaluateWAFRQuestion(ctx, question, workloadModel)
	if err != nil {
		// Handle partial data with low confidence
		slog.WarnContext(ctx, "bedrock evaluation failed, returning low confidence",
			"question_id", question.ID,
			"error", err,
		)
		return &core.QuestionEvaluation{
			Question:        question,
			SelectedChoices: []core.Choice{},
			Evidence:        []core.Evidence{},
			ConfidenceScore: 0.0,
			Notes:           fmt.Sprintf("Evaluation failed: %v", err),
		}, nil
	}

	// Calculate confidence score based on data completeness
	finalConfidence := calculateConfidenceScore(evaluation, workloadModel)
	evaluation.ConfidenceScore = finalConfidence

	slog.InfoContext(ctx, "question evaluated",
		"question_id", question.ID,
		"selected_choices", len(evaluation.SelectedChoices),
		"evidence_count", len(evaluation.Evidence),
		"confidence", evaluation.ConfidenceScore,
	)

	return evaluation, nil
}

// SubmitAnswer submits an answer to AWS Well-Architected Tool
func (e *Evaluator) SubmitAnswer(
	ctx context.Context,
	awsWorkloadID string,
	questionID string,
	evaluation *core.QuestionEvaluation,
) error {
	if awsWorkloadID == "" {
		return errors.New("AWS workload ID is required")
	}
	if questionID == "" {
		return errors.New("question ID is required")
	}
	if evaluation == nil {
		return errors.New("evaluation is required")
	}

	// Extract choice IDs from evaluation
	selectedChoices := make([]string, 0, len(evaluation.SelectedChoices))
	for _, choice := range evaluation.SelectedChoices {
		selectedChoices = append(selectedChoices, choice.ID)
	}

	// Build notes with confidence score and evidence
	notes := fmt.Sprintf("Automated analysis by Waffle (confidence: %.2f)\n\n%s",
		evaluation.ConfidenceScore,
		evaluation.Notes,
	)

	input := &wellarchitected.UpdateAnswerInput{
		WorkloadId:      aws.String(awsWorkloadID),
		LensAlias:       aws.String("wellarchitected"),
		QuestionId:      aws.String(questionID),
		SelectedChoices: selectedChoices,
		Notes:           aws.String(notes),
		IsApplicable:    aws.Bool(true),
	}

	err := e.retryWithBackoff(ctx, "UpdateAnswer", func() error {
		_, err := e.client.UpdateAnswer(ctx, input)
		return err
	})

	if err != nil {
		return wrapWAFRError("UpdateAnswer", err)
	}

	slog.InfoContext(ctx, "answer submitted",
		"aws_workload_id", awsWorkloadID,
		"question_id", questionID,
		"choices_count", len(selectedChoices),
		"confidence", evaluation.ConfidenceScore,
	)

	return nil
}

// GetImprovementPlan retrieves the improvement plan from AWS
func (e *Evaluator) GetImprovementPlan(
	ctx context.Context,
	awsWorkloadID string,
	workloadModel *core.WorkloadModel,
) (*core.ImprovementPlan, error) {
	if awsWorkloadID == "" {
		return nil, errors.New("AWS workload ID is required")
	}

	slog.InfoContext(ctx, "retrieving improvement plan",
		"aws_workload_id", awsWorkloadID,
	)

	// Retrieve risks and improvement items from AWS
	risks, err := e.getRisksFromAWS(ctx, awsWorkloadID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve risks: %w", err)
	}

	// Enhance risks with IaC-specific resource references
	if workloadModel != nil {
		risks = e.enhanceRisksWithResources(ctx, risks, workloadModel)
	}

	// Create improvement plan items from risks
	items := make([]*core.ImprovementPlanItem, 0, len(risks))
	for i, risk := range risks {
		item := &core.ImprovementPlanItem{
			ID:                fmt.Sprintf("improvement-%d", i+1),
			Risk:              risk,
			Description:       risk.Description,
			BestPracticeRefs:  extractBestPracticeRefs(risk),
			AffectedResources: risk.AffectedResources,
			Priority:          calculatePriority(risk),
			EstimatedEffort:   estimateEffort(risk),
		}
		items = append(items, item)
	}

	plan := &core.ImprovementPlan{
		Items: items,
	}

	slog.InfoContext(ctx, "improvement plan retrieved",
		"aws_workload_id", awsWorkloadID,
		"risk_count", len(risks),
		"improvement_items", len(items),
	)

	return plan, nil
}

// getRisksFromAWS retrieves risks identified by AWS Well-Architected Tool
func (e *Evaluator) getRisksFromAWS(
	ctx context.Context,
	awsWorkloadID string,
) ([]*core.Risk, error) {
	var risks []*core.Risk

	// Get risks for all pillars
	for _, pillar := range []core.Pillar{
		core.PillarOperationalExcellence,
		core.PillarSecurity,
		core.PillarReliability,
		core.PillarPerformanceEfficiency,
		core.PillarCostOptimization,
		core.PillarSustainability,
	} {
		pillarRisks, err := e.getRisksForPillar(ctx, awsWorkloadID, pillar)
		if err != nil {
			slog.WarnContext(ctx, "failed to get risks for pillar",
				"pillar", pillar,
				"error", err,
			)
			continue
		}
		risks = append(risks, pillarRisks...)
	}

	return risks, nil
}

// getRisksForPillar retrieves risks for a specific pillar
func (e *Evaluator) getRisksForPillar(
	ctx context.Context,
	awsWorkloadID string,
	pillar core.Pillar,
) ([]*core.Risk, error) {
	var risks []*core.Risk
	var nextToken *string

	pillarID := mapPillarToAWSID(pillar)

	for {
		input := &wellarchitected.ListAnswersInput{
			WorkloadId: aws.String(awsWorkloadID),
			LensAlias:  aws.String("wellarchitected"),
			PillarId:   aws.String(pillarID),
			NextToken:  nextToken,
			MaxResults: aws.Int32(50),
		}

		var output *wellarchitected.ListAnswersOutput
		err := e.retryWithBackoff(ctx, "ListAnswers", func() error {
			var err error
			output, err = e.client.ListAnswers(ctx, input)
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list answers: %w", err)
		}

		// Extract risks from answers
		for _, answer := range output.AnswerSummaries {
			// Only process answers with identified risks
			if answer.Risk == types.RiskNone || answer.Risk == types.RiskNotApplicable {
				continue
			}

			risk := convertAnswerToRisk(answer, pillar)
			risks = append(risks, risk)
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return risks, nil
}

// convertAnswerToRisk converts AWS answer summary to internal risk format
func convertAnswerToRisk(answer types.AnswerSummary, pillar core.Pillar) *core.Risk {
	risk := &core.Risk{
		ID:                   aws.ToString(answer.QuestionId),
		Pillar:               pillar,
		Severity:             mapAWSRiskToRiskLevel(answer.Risk),
		Description:          buildRiskDescription(answer),
		AffectedResources:    []string{}, // Will be enhanced later
		MissingBestPractices: []core.BestPractice{},
	}

	// Create a minimal question reference
	risk.Question = &core.WAFRQuestion{
		ID:     aws.ToString(answer.QuestionId),
		Pillar: pillar,
		Title:  aws.ToString(answer.QuestionTitle),
	}

	// Extract missing best practices from choices
	for _, choice := range answer.Choices {
		// Choices that are not selected represent missing best practices
		isSelected := false
		for _, selectedChoice := range answer.SelectedChoices {
			if aws.ToString(choice.ChoiceId) == selectedChoice {
				isSelected = true
				break
			}
		}

		if !isSelected {
			risk.MissingBestPractices = append(risk.MissingBestPractices, core.BestPractice{
				ID:          aws.ToString(choice.ChoiceId),
				Title:       aws.ToString(choice.Title),
				Description: aws.ToString(choice.Description),
			})
		}
	}

	return risk
}

// mapAWSRiskToRiskLevel maps AWS risk type to internal risk level
func mapAWSRiskToRiskLevel(awsRisk types.Risk) core.RiskLevel {
	switch awsRisk {
	case types.RiskHigh:
		return core.RiskLevelHigh
	case types.RiskMedium:
		return core.RiskLevelMedium
	case types.RiskNone, types.RiskNotApplicable, types.RiskUnanswered:
		return core.RiskLevelNone
	default:
		return core.RiskLevelNone
	}
}

// buildRiskDescription builds a description for the risk
func buildRiskDescription(answer types.AnswerSummary) string {
	questionTitle := aws.ToString(answer.QuestionTitle)
	riskLevel := string(answer.Risk)

	description := fmt.Sprintf("Risk identified for question: %s (Risk Level: %s)", questionTitle, riskLevel)

	// Add information about missing best practices
	unselectedCount := len(answer.Choices) - len(answer.SelectedChoices)
	if unselectedCount > 0 {
		description += fmt.Sprintf("\n%d best practice(s) not implemented.", unselectedCount)
	}

	// Notes are not available in AnswerSummary, only in full Answer
	// They would need to be fetched separately if needed

	return description
}

// enhanceRisksWithResources enhances risks with IaC-specific resource references
func (e *Evaluator) enhanceRisksWithResources(
	ctx context.Context,
	risks []*core.Risk,
	workloadModel *core.WorkloadModel,
) []*core.Risk {
	if workloadModel == nil || len(workloadModel.Resources) == 0 {
		return risks
	}

	slog.InfoContext(ctx, "enhancing risks with resource references",
		"risk_count", len(risks),
		"resource_count", len(workloadModel.Resources),
	)

	for _, risk := range risks {
		// Match resources to risks based on question context
		affectedResources := e.findAffectedResources(risk, workloadModel)
		risk.AffectedResources = affectedResources
	}

	return risks
}

// findAffectedResources identifies resources affected by a risk
func (e *Evaluator) findAffectedResources(risk *core.Risk, workloadModel *core.WorkloadModel) []string {
	var affectedResources []string

	// Map question/pillar to relevant resource types
	relevantTypes := getRelevantResourceTypes(risk.Question.ID, risk.Pillar)

	for _, resource := range workloadModel.Resources {
		// Check if resource type is relevant to this risk
		for _, relevantType := range relevantTypes {
			if matchesResourceType(resource.Type, relevantType) {
				affectedResources = append(affectedResources, resource.Address)
				break
			}
		}
	}

	return affectedResources
}

// getRelevantResourceTypes returns resource types relevant to a question/pillar
func getRelevantResourceTypes(questionID string, pillar core.Pillar) []string {
	// This is a simplified mapping - in production, this would be more comprehensive
	switch pillar {
	case core.PillarSecurity:
		return []string{
			"aws_s3_bucket",
			"aws_kms_key",
			"aws_iam_role",
			"aws_iam_policy",
			"aws_security_group",
			"aws_vpc",
			"aws_subnet",
		}
	case core.PillarReliability:
		return []string{
			"aws_autoscaling_group",
			"aws_elb",
			"aws_lb",
			"aws_rds_instance",
			"aws_dynamodb_table",
			"aws_backup_plan",
		}
	case core.PillarPerformanceEfficiency:
		return []string{
			"aws_instance",
			"aws_lambda_function",
			"aws_cloudfront_distribution",
			"aws_elasticache_cluster",
		}
	case core.PillarCostOptimization:
		return []string{
			"aws_instance",
			"aws_rds_instance",
			"aws_s3_bucket",
			"aws_ebs_volume",
		}
	case core.PillarOperationalExcellence:
		return []string{
			"aws_cloudwatch_log_group",
			"aws_cloudwatch_metric_alarm",
			"aws_sns_topic",
			"aws_lambda_function",
		}
	case core.PillarSustainability:
		return []string{
			"aws_instance",
			"aws_autoscaling_group",
			"aws_lambda_function",
		}
	default:
		return []string{}
	}
}

// matchesResourceType checks if a resource type matches a pattern
func matchesResourceType(resourceType, pattern string) bool {
	// Simple prefix matching - could be enhanced with regex
	return resourceType == pattern || 
		   (len(resourceType) > len(pattern) && resourceType[:len(pattern)] == pattern)
}

// extractBestPracticeRefs extracts best practice references from a risk
func extractBestPracticeRefs(risk *core.Risk) []string {
	refs := make([]string, 0, len(risk.MissingBestPractices))
	
	// Generate AWS documentation links for best practices
	baseURL := "https://docs.aws.amazon.com/wellarchitected/latest/framework"
	
	for _, bp := range risk.MissingBestPractices {
		// Create a reference URL based on pillar and best practice ID
		pillarPath := getPillarPath(risk.Pillar)
		ref := fmt.Sprintf("%s/%s.html#%s", baseURL, pillarPath, bp.ID)
		refs = append(refs, ref)
	}
	
	return refs
}

// getPillarPath returns the URL path component for a pillar
func getPillarPath(pillar core.Pillar) string {
	switch pillar {
	case core.PillarOperationalExcellence:
		return "operational-excellence"
	case core.PillarSecurity:
		return "security"
	case core.PillarReliability:
		return "reliability"
	case core.PillarPerformanceEfficiency:
		return "performance-efficiency"
	case core.PillarCostOptimization:
		return "cost-optimization"
	case core.PillarSustainability:
		return "sustainability"
	default:
		return "framework"
	}
}

// calculatePriority calculates priority for an improvement item
func calculatePriority(risk *core.Risk) int {
	// Priority based on severity and number of missing best practices
	basePriority := 0
	
	switch risk.Severity {
	case core.RiskLevelHigh:
		basePriority = 100
	case core.RiskLevelMedium:
		basePriority = 50
	case core.RiskLevelNone:
		basePriority = 10
	}
	
	// Adjust based on number of missing best practices
	missingCount := len(risk.MissingBestPractices)
	if missingCount > 0 {
		basePriority += missingCount * 5
	}
	
	return basePriority
}

// estimateEffort estimates the effort required to address a risk
func estimateEffort(risk *core.Risk) string {
	missingCount := len(risk.MissingBestPractices)
	resourceCount := len(risk.AffectedResources)
	
	// Simple heuristic based on complexity
	totalComplexity := missingCount + resourceCount
	
	if totalComplexity <= 2 {
		return "LOW"
	} else if totalComplexity <= 5 {
		return "MEDIUM"
	} else {
		return "HIGH"
	}
}

// CreateMilestone creates a milestone in AWS
func (e *Evaluator) CreateMilestone(
	ctx context.Context,
	awsWorkloadID string,
	milestoneName string,
) (string, error) {
	if awsWorkloadID == "" {
		return "", errors.New("AWS workload ID is required")
	}
	if milestoneName == "" {
		// Generate default milestone name with timestamp
		milestoneName = fmt.Sprintf("waffle-%s", time.Now().Format("2006-01-02-15-04-05"))
	}

	input := &wellarchitected.CreateMilestoneInput{
		WorkloadId:    aws.String(awsWorkloadID),
		MilestoneName: aws.String(milestoneName),
	}

	var output *wellarchitected.CreateMilestoneOutput
	err := e.retryWithBackoff(ctx, "CreateMilestone", func() error {
		var err error
		output, err = e.client.CreateMilestone(ctx, input)
		return err
	})

	if err != nil {
		return "", wrapWAFRError("CreateMilestone", err)
	}

	milestoneNumber := aws.ToInt32(output.MilestoneNumber)
	slog.InfoContext(ctx, "milestone created",
		"aws_workload_id", awsWorkloadID,
		"milestone_name", milestoneName,
		"milestone_number", milestoneNumber,
	)

	return fmt.Sprintf("%d", milestoneNumber), nil
}

// retryWithBackoff executes an operation with exponential backoff retry logic
func (e *Evaluator) retryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	backoff := e.baseDelay
	maxBackoff := 32 * time.Second

	for attempt := 0; attempt < e.maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "ThrottlingException", "ServiceUnavailableException", "InternalServerException":
				if attempt < e.maxRetries-1 {
					slog.WarnContext(ctx, "retryable error, backing off",
						"operation", operation,
						"attempt", attempt+1,
						"error_code", apiErr.ErrorCode(),
						"backoff", backoff,
					)

					select {
					case <-time.After(backoff):
						backoff *= 2
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
					case <-ctx.Done():
						return ctx.Err()
					}
					continue
				}
			case "ResourceNotFoundException", "AccessDeniedException", "ValidationException":
				// Non-retryable errors
				return err
			}
		}

		// If we've exhausted retries or hit a non-retryable error
		if attempt == e.maxRetries-1 {
			slog.ErrorContext(ctx, "max retries exceeded",
				"operation", operation,
				"attempts", e.maxRetries,
			)
			return fmt.Errorf("max retries exceeded for %s: %w", operation, err)
		}

		return err
	}

	return fmt.Errorf("operation %s failed after %d attempts", operation, e.maxRetries)
}

// mapPillarToAWSID maps internal pillar representation to AWS pillar ID
func mapPillarToAWSID(pillar core.Pillar) string {
	switch pillar {
	case core.PillarOperationalExcellence:
		return "operationalExcellence"
	case core.PillarSecurity:
		return "security"
	case core.PillarReliability:
		return "reliability"
	case core.PillarPerformanceEfficiency:
		return "performance"
	case core.PillarCostOptimization:
		return "costOptimization"
	case core.PillarSustainability:
		return "sustainability"
	default:
		return string(pillar)
	}
}

// convertAnswerToQuestion converts AWS answer summary to internal question format
func convertAnswerToQuestion(answer types.AnswerSummary, pillar core.Pillar) *core.WAFRQuestion {
	question := &core.WAFRQuestion{
		ID:            aws.ToString(answer.QuestionId),
		Pillar:        pillar,
		Title:         aws.ToString(answer.QuestionTitle),
		Description:   "", // Not available in summary
		BestPractices: []core.BestPractice{},
		Choices:       convertChoices(answer.Choices),
		RiskRules:     make(map[string]interface{}),
	}

	// Add risk information if available
	if answer.Risk != "" {
		question.RiskRules["current_risk"] = string(answer.Risk)
	}

	return question
}

// convertChoices converts AWS choices to internal choice format
func convertChoices(awsChoices []types.Choice) []core.Choice {
	choices := make([]core.Choice, 0, len(awsChoices))
	for _, c := range awsChoices {
		choices = append(choices, core.Choice{
			ID:          aws.ToString(c.ChoiceId),
			Title:       aws.ToString(c.Title),
			Description: aws.ToString(c.Description),
		})
	}
	return choices
}

// calculateConfidenceScore calculates the final confidence score based on data completeness
func calculateConfidenceScore(evaluation *core.QuestionEvaluation, workloadModel *core.WorkloadModel) float64 {
	if evaluation == nil || workloadModel == nil {
		return 0.0
	}

	// Start with the Bedrock-provided confidence
	baseConfidence := evaluation.ConfidenceScore

	// Adjust based on data completeness factors
	var adjustments []float64

	// Factor 1: Resource availability (0.0 to 1.0)
	resourceFactor := 1.0
	if len(workloadModel.Resources) == 0 {
		resourceFactor = 0.0
	} else if len(workloadModel.Resources) < 5 {
		// Limited resources may indicate incomplete data
		resourceFactor = 0.7
	}
	adjustments = append(adjustments, resourceFactor)

	// Factor 2: Evidence quality (0.0 to 1.0)
	evidenceFactor := 1.0
	if len(evaluation.Evidence) == 0 {
		evidenceFactor = 0.5 // No evidence reduces confidence
	} else {
		// Check if evidence has resource references
		hasResourceRefs := false
		for _, ev := range evaluation.Evidence {
			if len(ev.Resources) > 0 {
				hasResourceRefs = true
				break
			}
		}
		if !hasResourceRefs {
			evidenceFactor = 0.7 // Evidence without resource refs is less reliable
		}
	}
	adjustments = append(adjustments, evidenceFactor)

	// Factor 3: Source type quality (0.0 to 1.0)
	sourceFactor := 1.0
	if workloadModel.SourceType == "plan" {
		// Terraform plan has complete data
		sourceFactor = 1.0
	} else if workloadModel.SourceType == "hcl" {
		// HCL source may have incomplete computed values
		sourceFactor = 0.85
	} else {
		// Unknown source type
		sourceFactor = 0.7
	}
	adjustments = append(adjustments, sourceFactor)

	// Calculate weighted average of adjustments
	totalAdjustment := 0.0
	for _, adj := range adjustments {
		totalAdjustment += adj
	}
	avgAdjustment := totalAdjustment / float64(len(adjustments))

	// Combine base confidence with adjustments
	// Use geometric mean to ensure both factors matter
	finalConfidence := baseConfidence * avgAdjustment

	// Ensure confidence is in valid range [0.0, 1.0]
	if finalConfidence < 0.0 {
		finalConfidence = 0.0
	}
	if finalConfidence > 1.0 {
		finalConfidence = 1.0
	}

	return finalConfidence
}
