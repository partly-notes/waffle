package wafr_test

import (
	"context"
	"fmt"
	"log"

	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/wafr"
)

// Example demonstrates the complete workflow of using the WAFR client
func Example() {
	ctx := context.Background()

	// Initialize the WAFR client
	clientCfg := &wafr.ClientConfig{
		Region:  "us-east-1",
		Profile: "default",
	}

	evaluator, err := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Step 1: Create a workload
	awsWorkloadID, err := evaluator.CreateWorkload(
		ctx,
		"my-application",
		"Production workload for my application",
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created workload: %s\n", awsWorkloadID)

	// Step 2: Get questions for security pillar
	pillar := core.PillarSecurity
	scope := core.ReviewScope{
		Level:  core.ScopeLevelPillar,
		Pillar: &pillar,
	}

	questions, err := evaluator.GetQuestions(ctx, awsWorkloadID, scope)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved %d questions\n", len(questions))

	// Step 3: Submit an answer
	if len(questions) > 0 {
		evaluation := &core.QuestionEvaluation{
			SelectedChoices: []core.Choice{
				{ID: questions[0].Choices[0].ID},
			},
			ConfidenceScore: 0.95,
			Notes:           "Automated analysis based on IaC",
		}

		err = evaluator.SubmitAnswer(ctx, awsWorkloadID, questions[0].ID, evaluation)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Answer submitted successfully")
	}

	// Step 4: Create a milestone
	milestoneID, err := evaluator.CreateMilestone(ctx, awsWorkloadID, "v1.0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created milestone: %s\n", milestoneID)

	// Step 5: Get consolidated report
	report, err := evaluator.GetConsolidatedReport(ctx, awsWorkloadID, "json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Retrieved report: %d bytes\n", len(report))
}

// ExampleEvaluator_CreateWorkload demonstrates creating a workload
func ExampleEvaluator_CreateWorkload() {
	ctx := context.Background()
	clientCfg := &wafr.ClientConfig{Region: "us-east-1"}
	
	evaluator, _ := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)
	
	awsWorkloadID, err := evaluator.CreateWorkload(
		ctx,
		"my-app",
		"My application workload",
	)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Workload ID: %s\n", awsWorkloadID)
}

// ExampleEvaluator_GetQuestions demonstrates retrieving questions with different scopes
func ExampleEvaluator_GetQuestions() {
	ctx := context.Background()
	clientCfg := &wafr.ClientConfig{Region: "us-east-1"}
	evaluator, _ := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)
	awsWorkloadID := "wl-123"

	// Get all questions
	scope := core.ReviewScope{Level: core.ScopeLevelWorkload}
	questions, _ := evaluator.GetQuestions(ctx, awsWorkloadID, scope)
	fmt.Printf("Total questions: %d\n", len(questions))

	// Get questions for a specific pillar
	pillar := core.PillarSecurity
	scope = core.ReviewScope{
		Level:  core.ScopeLevelPillar,
		Pillar: &pillar,
	}
	questions, _ = evaluator.GetQuestions(ctx, awsWorkloadID, scope)
	fmt.Printf("Security questions: %d\n", len(questions))
}

// ExampleEvaluator_SubmitAnswer demonstrates submitting an answer
func ExampleEvaluator_SubmitAnswer() {
	ctx := context.Background()
	clientCfg := &wafr.ClientConfig{Region: "us-east-1"}
	evaluator, _ := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)

	evaluation := &core.QuestionEvaluation{
		SelectedChoices: []core.Choice{
			{ID: "sec_1_choice_1"},
			{ID: "sec_1_choice_2"},
		},
		ConfidenceScore: 0.95,
		Notes:           "Based on IaC analysis",
	}

	err := evaluator.SubmitAnswer(ctx, "wl-123", "sec-1", evaluation)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Answer submitted")
}

// ExampleEvaluator_CreateMilestone demonstrates creating a milestone
func ExampleEvaluator_CreateMilestone() {
	ctx := context.Background()
	clientCfg := &wafr.ClientConfig{Region: "us-east-1"}
	evaluator, _ := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)

	// With custom name
	milestoneID, _ := evaluator.CreateMilestone(ctx, "wl-123", "v1.0")
	fmt.Printf("Milestone: %s\n", milestoneID)

	// Auto-generated name
	milestoneID, _ = evaluator.CreateMilestone(ctx, "wl-123", "")
	fmt.Printf("Auto-generated milestone: %s\n", milestoneID)
}

// ExampleEvaluator_GetConsolidatedReport demonstrates retrieving reports
func ExampleEvaluator_GetConsolidatedReport() {
	ctx := context.Background()
	clientCfg := &wafr.ClientConfig{Region: "us-east-1"}
	evaluator, _ := wafr.NewEvaluatorWithConfig(ctx, clientCfg, nil)

	// Get PDF report
	pdfReport, _ := evaluator.GetConsolidatedReport(ctx, "wl-123", "pdf")
	fmt.Printf("PDF report size: %d bytes\n", len(pdfReport))

	// Get JSON report
	jsonReport, _ := evaluator.GetConsolidatedReport(ctx, "wl-123", "json")
	fmt.Printf("JSON report size: %d bytes\n", len(jsonReport))
}
