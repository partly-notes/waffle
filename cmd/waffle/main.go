package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
	"github.com/waffle/waffle/internal/bedrock"
	"github.com/waffle/waffle/internal/config"
	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/iac"
	"github.com/waffle/waffle/internal/logging"
	"github.com/waffle/waffle/internal/report"
	"github.com/waffle/waffle/internal/session"
	"github.com/waffle/waffle/internal/wafr"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Exit codes
const (
	ExitSuccess            = 0
	ExitGeneralError       = 1
	ExitInvalidArguments   = 2
	ExitDirectoryAccess    = 3
	ExitBedrockAPIError    = 4
	ExitAnalysisIncomplete = 5
)

func main() {
	// Parse flags early to get logging configuration
	rootCmd.ParseFlags(os.Args[1:])

	// Initialize logging with command-line overrides
	logConfig := logging.DefaultConfig()
	logConfig.Level = getLogLevel()

	if err := logging.InitGlobalLogger(logConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(ExitGeneralError)
	}
	defer logging.CloseGlobalLogger()

	logger := logging.GetLogger()
	logger.Info("waffle started",
		"version", version,
		"commit", commit,
		"date", date,
	)

	if err := rootCmd.Execute(); err != nil {
		logger.Error("command execution failed", "error", err)
		// Error is already printed by cobra
		// Exit code is set by the command
		os.Exit(ExitGeneralError)
	}

	logger.Info("waffle completed successfully")
}

// getLogLevel returns the log level from command-line flags or environment variable
func getLogLevel() logging.LogLevel {
	// Check command-line flags first (if rootCmd is available)
	if rootCmd != nil {
		if quiet, _ := rootCmd.PersistentFlags().GetBool("quiet"); quiet {
			return logging.LevelError
		}
		if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
			return logging.LevelDebug
		}
		if logLevel, _ := rootCmd.PersistentFlags().GetString("log-level"); logLevel != "" {
			switch strings.ToUpper(logLevel) {
			case "DEBUG":
				return logging.LevelDebug
			case "INFO":
				return logging.LevelInfo
			case "WARNING", "WARN":
				return logging.LevelWarning
			case "ERROR":
				return logging.LevelError
			}
		}
	}

	// Fall back to environment variable
	level := os.Getenv("WAFFLE_LOG_LEVEL")
	switch strings.ToUpper(level) {
	case "DEBUG":
		return logging.LevelDebug
	case "INFO":
		return logging.LevelInfo
	case "WARNING", "WARN":
		return logging.LevelWarning
	case "ERROR":
		return logging.LevelError
	default:
		return logging.LevelInfo
	}
}

// loadConfigWithOverrides loads configuration and applies command-line flag overrides
func loadConfigWithOverrides(cmd *cobra.Command) (*config.Config, error) {
	// Load base configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply command-line flag overrides
	if region, _ := cmd.Flags().GetString("region"); region != "" {
		cfg.AWS.Region = region
		cfg.Bedrock.Region = region
	}

	if profile, _ := cmd.Flags().GetString("profile"); profile != "" {
		cfg.AWS.Profile = profile
	}

	if modelID, _ := cmd.Flags().GetString("model-id"); modelID != "" {
		cfg.Bedrock.ModelID = modelID
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

var rootCmd = &cobra.Command{
	Use:   "waffle",
	Short: "Waffle - Well Architected Framework for Less Effort",
	Long: `Waffle automates AWS Well-Architected Framework Reviews by analyzing 
infrastructure-as-code repositories using Amazon Bedrock foundation models.

Global Flags:
  --quiet, -q       Quiet mode - only show errors
  --verbose, -v     Verbose mode - show debug information  
  --log-level       Set log level: DEBUG, INFO, WARNING, ERROR`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(resultsCmd)
	rootCmd.AddCommand(initCmd)
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Initiate a new WAFR review",
	Long: `Initiate a new Well-Architected Framework Review for the current directory.

The review command analyzes your infrastructure-as-code and evaluates it against
AWS Well-Architected Framework best practices.

By default, Waffle analyzes Terraform configuration files (.tf) and modules.
Alternatively, you can specify a Terraform JSON file for analysis with computed values.

Examples:
  # Review using Terraform configuration files (default)
  waffle review --workload-id my-app

  # Review using Terraform plan JSON (alternative)
  terraform plan -out=plan.tfplan && terraform show -json plan.tfplan > plan.json
  waffle review --workload-id my-app --plan-file plan.json

  # Review using current state JSON (alternative)
  terraform show -json > state.json
  waffle review --workload-id my-app --plan-file state.json

  # Review with quiet output (errors only)
  waffle review --workload-id my-app --quiet

  # Review with specific AWS region
  waffle review --workload-id my-app --region us-west-2

  # Review with specific Bedrock model
  waffle review --workload-id my-app --model-id us.anthropic.claude-sonnet-4-20250514-v1:0

  # Review specific pillar
  waffle review --workload-id my-app --scope pillar --pillar security

  # Review specific question
  waffle review --workload-id my-app --scope question --question-id sec_data_1

Analysis Modes:
  - Default: Analyzes Terraform configuration files (.tf) and modules
  - Alternative: Uses Terraform JSON file (plan or state) for computed values and dependencies
  - Note: Only one mode is used per review - configuration files OR JSON file, not both`,
	RunE: runReview,
}

var statusCmd = &cobra.Command{
	Use:   "status [session-id]",
	Short: "Check review session status",
	Long: `Check the status of a review session.

Returns the current status of the specified review session including:
- Session state (created, in_progress, completed, failed)
- Progress information
- Timestamp information

Examples:
  waffle status abc123-def456-789`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

var resultsCmd = &cobra.Command{
	Use:   "results [session-id]",
	Short: "Retrieve and display review results",
	Long: `Retrieve and display the results of a completed review session.

Results can be exported in multiple formats:
- JSON: Machine-readable format with IaC evidence and confidence scores
- PDF: Professional report generated by AWS Well-Architected Tool

Examples:
  # Get results as JSON to stdout
  waffle results abc123-def456-789

  # Get results as JSON to file
  waffle results abc123-def456-789 --format json --output results.json

  # Get results as PDF
  waffle results abc123-def456-789 --format pdf --output report.pdf`,
	Args: cobra.ExactArgs(1),
	RunE: runResults,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize and validate Waffle setup",
	Long: `Validate AWS credentials, Bedrock access, and WAFR permissions.

This command checks:
- AWS credentials are configured
- Bedrock model access is enabled
- Well-Architected Tool permissions are available

Examples:
  # Validate with default region
  waffle init

  # Validate with specific region
  waffle init --region us-west-2

  # Validate with specific profile and region
  waffle init --profile my-profile --region eu-west-1

  # Validate with specific model
  waffle init --model-id us.anthropic.claude-sonnet-4-20250514-v1:0`,
	RunE: runInit,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("region", "", "AWS region for Bedrock and WAFR (overrides config file and AWS_REGION)")
	rootCmd.PersistentFlags().String("profile", "", "AWS profile to use (overrides config file and AWS_PROFILE)")
	rootCmd.PersistentFlags().String("model-id", "", "Bedrock model ID to use for analysis (overrides config file and environment variables)")
	rootCmd.PersistentFlags().String("log-level", "", "Log level: DEBUG, INFO, WARNING, ERROR (overrides config file and WAFFLE_LOG_LEVEL)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - only show errors (equivalent to --log-level ERROR)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose mode - show debug information (equivalent to --log-level DEBUG)")

	// Review command flags
	reviewCmd.Flags().String("workload-id", "", "Workload identifier (required)")
	reviewCmd.Flags().String("plan-file", "", "Path to Terraform JSON file (plan or state, alternative to HCL analysis)")
	reviewCmd.Flags().String("scope", "workload", "Review scope: workload, pillar, or question")
	reviewCmd.Flags().String("pillar", "", "Specific pillar when scope is pillar (operationalExcellence, security, reliability, performance, costOptimization, sustainability)")
	reviewCmd.Flags().String("question-id", "", "Specific question ID when scope is question")
	reviewCmd.MarkFlagRequired("workload-id")

	// Results command flags
	resultsCmd.Flags().String("format", "json", "Output format: json or pdf")
	resultsCmd.Flags().String("output", "", "Output file path (optional, defaults to stdout for JSON)")
}

// runReview executes the review command
func runReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := logging.GetLogger()

	// Get flags
	workloadID, _ := cmd.Flags().GetString("workload-id")
	planFile, _ := cmd.Flags().GetString("plan-file")
	scopeStr, _ := cmd.Flags().GetString("scope")
	pillarStr, _ := cmd.Flags().GetString("pillar")
	questionID, _ := cmd.Flags().GetString("question-id")

	// Validate workload ID
	if workloadID == "" {
		fmt.Fprintln(os.Stderr, "Error: workload-id is required")
		os.Exit(ExitInvalidArguments)
	}

	// Parse and validate scope
	scope, err := parseReviewScope(scopeStr, pillarStr, questionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitInvalidArguments)
	}

	// Validate scope
	if err := scope.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitInvalidArguments)
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get current directory: %v\n", err)
		os.Exit(ExitDirectoryAccess)
	}

	// Display initial information
	fmt.Fprintf(os.Stderr, "Starting WAFR review...\n")
	fmt.Fprintf(os.Stderr, "Workload ID: %s\n", workloadID)
	fmt.Fprintf(os.Stderr, "Directory: %s\n", currentDir)
	fmt.Fprintf(os.Stderr, "Scope: %s\n", formatScope(scope))
	if planFile != "" {
		fmt.Fprintf(os.Stderr, "Analysis: Terraform JSON file (%s)\n", planFile)
	} else {
		fmt.Fprintf(os.Stderr, "Analysis: Terraform configuration files (.tf)\n")
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Load configuration with command-line overrides
	cfg, err := loadConfigWithOverrides(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	// Initialize dependencies
	logger.Info("initializing dependencies")
	engine, err := initializeEngine(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize engine: %v\n", err)
		logger.Error("failed to initialize engine", "error", err)
		os.Exit(ExitGeneralError)
	}

	// Create progress reporter
	progress := core.NewCLIProgressReporter(os.Stderr)

	// Initiate review
	logger.Info("initiating review", "workload_id", workloadID)
	session, err := engine.InitiateReview(ctx, workloadID, scope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initiate review: %v\n", err)
		logger.Error("failed to initiate review", "error", err)
		handleReviewError(err)
	}

	// Set plan file path from flag or configuration
	if planFile != "" {
		session.PlanFilePath = planFile
	} else if cfg.IaC.PlanFilePath != "" {
		session.PlanFilePath = cfg.IaC.PlanFilePath
	}

	logger.Info("executing review", "session_id", session.SessionID)

	// Execute review with progress reporting
	results, err := engine.ExecuteReviewWithProgress(ctx, session, progress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: review execution failed: %v\n", err)
		logger.Error("review execution failed",
			"session_id", session.SessionID,
			"error", err,
		)
		handleReviewError(err)
	}

	logger.Info("review completed successfully",
		"session_id", session.SessionID,
		"questions_evaluated", len(results.Evaluations),
	)

	// Output JSON for CI/CD integration
	reviewOutput := &core.ReviewOutput{
		SessionID:  session.SessionID,
		WorkloadID: workloadID,
		Status:     string(session.Status),
		CreatedAt:  session.CreatedAt,
		Summary: &core.ReviewSummaryOutput{
			QuestionsEvaluated:  results.Summary.QuestionsEvaluated,
			HighRisks:           results.Summary.HighRisks,
			MediumRisks:         results.Summary.MediumRisks,
			AverageConfidence:   results.Summary.AverageConfidence,
			ImprovementPlanSize: results.Summary.ImprovementPlanSize,
		},
		Metadata: map[string]interface{}{
			"scope":           formatScope(scope),
			"directory":       currentDir,
			"aws_workload_id": session.AWSWorkloadID,
			"milestone_id":    session.MilestoneID,
		},
	}

	if planFile != "" {
		reviewOutput.Metadata["plan_file"] = planFile
	}

	if err := core.WriteJSON(os.Stdout, reviewOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write JSON output: %v\n", err)
		logger.Error("failed to write JSON output", "error", err)
		os.Exit(ExitGeneralError)
	}

	return nil
}

// runStatus executes the status command
func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := logging.GetLogger()
	sessionID := args[0]

	fmt.Fprintf(os.Stderr, "Checking status for session: %s\n\n", sessionID)

	// Load configuration
	cfg, err := loadConfigWithOverrides(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	// Initialize session manager
	sessionManager, err := initializeSessionManager(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize session manager: %v\n", err)
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(ExitGeneralError)
	}

	// Load session
	logger.Info("loading session", "session_id", sessionID)
	session, err := sessionManager.LoadSession(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: session not found: %v\n", err)
		logger.Error("session not found", "session_id", sessionID, "error", err)
		os.Exit(ExitGeneralError)
	}

	// Display status information
	fmt.Fprintf(os.Stderr, "Session Status:\n")
	fmt.Fprintf(os.Stderr, "  Status: %s\n", session.Status)
	fmt.Fprintf(os.Stderr, "  Workload ID: %s\n", session.WorkloadID)
	fmt.Fprintf(os.Stderr, "  AWS Workload ID: %s\n", session.AWSWorkloadID)
	fmt.Fprintf(os.Stderr, "  Created: %s\n", session.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "  Updated: %s\n", session.UpdatedAt.Format(time.RFC3339))
	if session.Checkpoint != "" {
		fmt.Fprintf(os.Stderr, "  Checkpoint: %s\n", session.Checkpoint)
	}
	if session.Results != nil && session.Results.Summary != nil {
		fmt.Fprintf(os.Stderr, "  Questions Evaluated: %d\n", session.Results.Summary.QuestionsEvaluated)
		fmt.Fprintf(os.Stderr, "  High Risks: %d\n", session.Results.Summary.HighRisks)
		fmt.Fprintf(os.Stderr, "  Medium Risks: %d\n", session.Results.Summary.MediumRisks)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Build status output
	statusOutput := &core.StatusOutput{
		SessionID:  sessionID,
		WorkloadID: session.WorkloadID,
		Status:     string(session.Status),
		CreatedAt:  session.CreatedAt,
		UpdatedAt:  session.UpdatedAt,
		Metadata: map[string]interface{}{
			"scope":           formatScope(session.Scope),
			"aws_workload_id": session.AWSWorkloadID,
		},
	}

	if session.Checkpoint != "" {
		statusOutput.Metadata["checkpoint"] = session.Checkpoint
	}

	if session.PlanFilePath != "" {
		statusOutput.Metadata["plan_file"] = session.PlanFilePath
	}

	if session.Results != nil && session.Results.Summary != nil {
		statusOutput.Metadata["summary"] = &core.ReviewSummaryOutput{
			QuestionsEvaluated:  session.Results.Summary.QuestionsEvaluated,
			HighRisks:           session.Results.Summary.HighRisks,
			MediumRisks:         session.Results.Summary.MediumRisks,
			AverageConfidence:   session.Results.Summary.AverageConfidence,
			ImprovementPlanSize: session.Results.Summary.ImprovementPlanSize,
		}
	}

	// Output JSON
	if err := core.WriteJSON(os.Stdout, statusOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write JSON output: %v\n", err)
		logger.Error("failed to write JSON output", "error", err)
		os.Exit(ExitGeneralError)
	}

	logger.Info("status retrieved successfully", "session_id", sessionID)
	return nil
}

// runResults executes the results command
func runResults(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := logging.GetLogger()
	sessionID := args[0]
	format, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")

	// Validate format
	if format != "json" && format != "pdf" {
		fmt.Fprintf(os.Stderr, "Error: invalid format '%s', must be 'json' or 'pdf'\n", format)
		os.Exit(ExitInvalidArguments)
	}

	fmt.Fprintf(os.Stderr, "Retrieving results for session: %s\n", sessionID)
	fmt.Fprintf(os.Stderr, "Format: %s\n", format)
	if outputPath != "" {
		fmt.Fprintf(os.Stderr, "Output: %s\n", outputPath)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Load configuration
	cfg, err := loadConfigWithOverrides(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	// Initialize session manager
	sessionManager, err := initializeSessionManager(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize session manager: %v\n", err)
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(ExitGeneralError)
	}

	// Load session
	logger.Info("loading session", "session_id", sessionID)
	session, err := sessionManager.LoadSession(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: session not found: %v\n", err)
		logger.Error("session not found", "session_id", sessionID, "error", err)
		os.Exit(ExitGeneralError)
	}

	// Check if session is completed
	if session.Status != core.SessionStatusCompleted {
		fmt.Fprintf(os.Stderr, "Warning: session is not completed (status: %s)\n", session.Status)
		if session.Status == core.SessionStatusFailed {
			fmt.Fprintf(os.Stderr, "Error: session failed, cannot retrieve results\n")
			os.Exit(ExitGeneralError)
		}
	}

	// Initialize AWS config
	awsCfg, err := initializeAWSConfig(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize AWS config: %v\n", err)
		logger.Error("failed to initialize AWS config", "error", err)
		os.Exit(ExitGeneralError)
	}

	// Initialize report generator
	reportGen, err := initializeReportGenerator(ctx, awsCfg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize report generator: %v\n", err)
		logger.Error("failed to initialize report generator", "error", err)
		os.Exit(ExitGeneralError)
	}

	if format == "json" {
		// Get enhanced JSON results
		logger.Info("retrieving JSON results", "aws_workload_id", session.AWSWorkloadID)
		resultsData, err := reportGen.GetResultsJSON(ctx, session.AWSWorkloadID, session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to retrieve results: %v\n", err)
			logger.Error("failed to retrieve results", "error", err)
			os.Exit(ExitGeneralError)
		}

		// Write to file or stdout
		if outputPath != "" {
			file, err := os.Create(outputPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to create output file: %v\n", err)
				os.Exit(ExitGeneralError)
			}
			defer file.Close()

			if err := core.WriteJSON(file, resultsData); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to write JSON output: %v\n", err)
				os.Exit(ExitGeneralError)
			}
			fmt.Fprintf(os.Stderr, "Results written to %s\n", outputPath)
		} else {
			if err := core.WriteJSON(os.Stdout, resultsData); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to write JSON output: %v\n", err)
				os.Exit(ExitGeneralError)
			}
		}
	} else {
		// Get PDF report from AWS
		logger.Info("retrieving PDF report", "aws_workload_id", session.AWSWorkloadID)
		pdfData, err := reportGen.GetConsolidatedReport(ctx, session.AWSWorkloadID, core.ReportFormatPDF)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to retrieve PDF report: %v\n", err)
			logger.Error("failed to retrieve PDF report", "error", err)
			os.Exit(ExitGeneralError)
		}

		// PDF must be written to a file
		if outputPath == "" {
			fmt.Fprintf(os.Stderr, "Error: output file path is required for PDF format\n")
			os.Exit(ExitInvalidArguments)
		}

		if err := os.WriteFile(outputPath, pdfData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write PDF file: %v\n", err)
			os.Exit(ExitGeneralError)
		}

		fmt.Fprintf(os.Stderr, "PDF report written to %s\n", outputPath)
	}

	logger.Info("results retrieved successfully", "session_id", sessionID, "format", format)
	return nil
}

// runInit executes the init command
func runInit(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(os.Stderr, "Validating Waffle setup...\n\n")

	// Load configuration with command-line overrides
	cfg, err := loadConfigWithOverrides(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	fmt.Fprintf(os.Stderr, "Configuration loaded successfully\n")
	fmt.Fprintf(os.Stderr, "  Bedrock Region: %s\n", cfg.Bedrock.Region)
	fmt.Fprintf(os.Stderr, "  Bedrock Model: %s\n", cfg.Bedrock.ModelID)
	if cfg.AWS.Profile != "" {
		fmt.Fprintf(os.Stderr, "  AWS Profile: %s\n", cfg.AWS.Profile)
	}
	if cfg.AWS.Region != "" {
		fmt.Fprintf(os.Stderr, "  AWS Region: %s\n", cfg.AWS.Region)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Create validator
	validator := config.NewValidator(cfg)

	// Run validation checks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Running validation checks...\n\n")

	results, err := validator.ValidateAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: validation failed: %v\n", err)
		os.Exit(ExitGeneralError)
	}

	// Display results
	allSuccess := true
	for _, result := range results {
		if result.Success {
			fmt.Fprintf(os.Stderr, "✓ %s\n", result.Name)
			fmt.Fprintf(os.Stderr, "  %s\n", result.Message)
		} else {
			fmt.Fprintf(os.Stderr, "✗ %s\n", result.Name)
			fmt.Fprintf(os.Stderr, "  %s\n", result.Message)
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", result.Error)
			}
			allSuccess = false
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Summary
	if allSuccess {
		fmt.Fprintf(os.Stderr, "✓ All validation checks passed!\n")
		fmt.Fprintf(os.Stderr, "\nYou're ready to run WAFR reviews with Waffle.\n")
		fmt.Fprintf(os.Stderr, "\nNext steps:\n")
		fmt.Fprintf(os.Stderr, "  1. Navigate to your IaC directory\n")
		fmt.Fprintf(os.Stderr, "  2. Run: waffle review --workload-id <your-workload-id>\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "✗ Some validation checks failed\n")
	fmt.Fprintf(os.Stderr, "\nPlease address the issues above before running reviews.\n")
	os.Exit(ExitGeneralError)
	return nil
}

// parseReviewScope parses the scope string and related flags into a ReviewScope
func parseReviewScope(scopeStr, pillarStr, questionID string) (core.ReviewScope, error) {
	scopeStr = strings.ToLower(scopeStr)

	switch scopeStr {
	case "workload":
		return core.ReviewScope{
			Level: core.ScopeLevelWorkload,
		}, nil

	case "pillar":
		if pillarStr == "" {
			return core.ReviewScope{}, fmt.Errorf("pillar flag is required when scope is 'pillar'")
		}
		pillar, err := parsePillar(pillarStr)
		if err != nil {
			return core.ReviewScope{}, err
		}
		return core.ReviewScope{
			Level:  core.ScopeLevelPillar,
			Pillar: &pillar,
		}, nil

	case "question":
		if questionID == "" {
			return core.ReviewScope{}, fmt.Errorf("question-id flag is required when scope is 'question'")
		}
		return core.ReviewScope{
			Level:      core.ScopeLevelQuestion,
			QuestionID: questionID,
		}, nil

	default:
		return core.ReviewScope{}, fmt.Errorf("invalid scope '%s', must be 'workload', 'pillar', or 'question'", scopeStr)
	}
}

// parsePillar parses a pillar string into a Pillar constant
func parsePillar(pillarStr string) (core.Pillar, error) {
	pillarStr = strings.ToLower(pillarStr)

	switch pillarStr {
	case "operationalexcellence", "operational-excellence", "operational_excellence":
		return core.PillarOperationalExcellence, nil
	case "security":
		return core.PillarSecurity, nil
	case "reliability":
		return core.PillarReliability, nil
	case "performance", "performanceefficiency", "performance-efficiency", "performance_efficiency":
		return core.PillarPerformanceEfficiency, nil
	case "cost", "costoptimization", "cost-optimization", "cost_optimization":
		return core.PillarCostOptimization, nil
	case "sustainability":
		return core.PillarSustainability, nil
	default:
		return "", fmt.Errorf("invalid pillar '%s', must be one of: operationalExcellence, security, reliability, performance, costOptimization, sustainability", pillarStr)
	}
}

// formatScope formats a ReviewScope for display
func formatScope(scope core.ReviewScope) string {
	switch scope.Level {
	case core.ScopeLevelWorkload:
		return "workload (all pillars)"
	case core.ScopeLevelPillar:
		if scope.Pillar != nil {
			return fmt.Sprintf("pillar (%s)", *scope.Pillar)
		}
		return "pillar"
	case core.ScopeLevelQuestion:
		return fmt.Sprintf("question (%s)", scope.QuestionID)
	default:
		return "unknown"
	}
}

// initializeEngine initializes the core engine with all dependencies
func initializeEngine(ctx context.Context, cfg *config.Config) (core.CoreEngine, error) {
	logger := logging.GetLogger()

	// Initialize AWS clients
	logger.Debug("initializing AWS clients")
	awsCfg, err := initializeAWSConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS config: %w", err)
	}

	// Initialize Session Manager
	logger.Debug("initializing session manager")
	sessionManager, err := initializeSessionManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session manager: %w", err)
	}

	// Initialize IaC Analyzer
	logger.Debug("initializing IaC analyzer")
	iacAnalyzer, err := initializeIaCAnalyzer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize IaC analyzer: %w", err)
	}

	// Initialize Bedrock Client
	logger.Debug("initializing Bedrock client")
	bedrockClient, err := initializeBedrockClient(ctx, awsCfg, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Bedrock client: %w", err)
	}

	// Initialize WAFR Evaluator
	logger.Debug("initializing WAFR evaluator")
	wafrEvaluator, err := initializeWAFREvaluator(ctx, awsCfg, cfg, bedrockClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize WAFR evaluator: %w", err)
	}

	// Initialize Report Generator
	logger.Debug("initializing report generator")
	reportGen, err := initializeReportGenerator(ctx, awsCfg, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize report generator: %w", err)
	}

	// Create core engine
	logger.Debug("creating core engine")
	engine := core.NewEngine(
		sessionManager,
		iacAnalyzer,
		wafrEvaluator,
		bedrockClient,
		reportGen,
	)

	logger.Info("engine initialized successfully")
	return engine, nil
}

// handleReviewError handles errors during review execution and exits with appropriate code
func handleReviewError(err error) {
	logger := logging.GetLogger()

	// Check for specific error types using errors.As
	var dirErr *core.DirectoryAccessError
	if errors.As(err, &dirErr) {
		logger.Error("directory access error", "error", err)
		os.Exit(ExitDirectoryAccess)
	}

	var bedrockErr *core.BedrockAPIError
	if errors.As(err, &bedrockErr) {
		logger.Error("Bedrock API error", "error", err)
		os.Exit(ExitBedrockAPIError)
	}

	var iacErr *core.IaCParsingError
	if errors.As(err, &iacErr) {
		logger.Error("IaC parsing error", "error", err)
		os.Exit(ExitAnalysisIncomplete)
	}

	// Default to general error
	logger.Error("general error", "error", err)
	os.Exit(ExitGeneralError)
}

// initializeAWSConfig initializes AWS configuration
func initializeAWSConfig(ctx context.Context, cfg *config.Config) (*config.AWSConfig, error) {
	// AWS configuration is already loaded in cfg
	return &cfg.AWS, nil
}

// loadAWSSDKConfig loads AWS SDK configuration
func loadAWSSDKConfig(ctx context.Context, awsCfg *config.AWSConfig) (aws.Config, error) {
	// Import AWS SDK config package
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(awsCfg.Region),
		awsconfig.WithSharedConfigProfile(awsCfg.Profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}
	return awsConfig, nil
}

// initializeSessionManager initializes the session manager
func initializeSessionManager(cfg *config.Config) (core.SessionManager, error) {
	// Import session package
	sessionMgr, err := session.NewManager(cfg.Storage.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	return sessionMgr, nil
}

// initializeIaCAnalyzer initializes the IaC analyzer
func initializeIaCAnalyzer(ctx context.Context, cfg *config.Config) (core.IaCAnalyzer, error) {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create analyzer with current directory
	analyzer := iac.NewAnalyzerWithDir(currentDir)
	return analyzer, nil
}

// initializeBedrockClient initializes the Bedrock client
func initializeBedrockClient(ctx context.Context, awsCfg *config.AWSConfig, cfg *config.Config) (core.BedrockClient, error) {
	// Load AWS SDK config
	sdkCfg, err := loadAWSSDKConfig(ctx, awsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	// Convert config.BedrockConfig to bedrock.Config
	bedrockCfg := &bedrock.Config{
		ModelID:        cfg.Bedrock.ModelID,
		Region:         cfg.Bedrock.Region,
		MaxTokens:      cfg.Bedrock.MaxTokens,
		Temperature:    cfg.Bedrock.Temperature,
		TopP:           0.9, // Default value
		MaxRetries:     cfg.Bedrock.MaxRetries,
		TimeoutSeconds: cfg.Bedrock.Timeout,
		RateLimit:      2.0, // Default rate limit
	}

	client := bedrock.NewClient(sdkCfg, bedrockCfg)
	return client, nil
}

// initializeWAFREvaluator initializes the WAFR evaluator
func initializeWAFREvaluator(ctx context.Context, awsCfg *config.AWSConfig, cfg *config.Config, bedrockClient core.BedrockClient) (core.WAFREvaluator, error) {
	// Create WAFR client configuration
	clientCfg := &wafr.ClientConfig{
		Region:  awsCfg.Region,
		Profile: awsCfg.Profile,
	}

	// Create evaluator configuration
	evalCfg := &wafr.EvaluatorConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}

	// Create evaluator with configuration
	evaluator, err := wafr.NewEvaluatorWithConfig(ctx, clientCfg, evalCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create WAFR evaluator: %w", err)
	}

	// Cast bedrock client to wafr.BedrockClient
	wafrBedrockClient, ok := bedrockClient.(wafr.BedrockClient)
	if !ok {
		return nil, fmt.Errorf("bedrock client does not implement wafr.BedrockClient interface")
	}

	// Wrap evaluator with adapter
	adapter := NewWAFREvaluatorAdapter(evaluator, wafrBedrockClient)
	return adapter, nil
}

// initializeReportGenerator initializes the report generator
func initializeReportGenerator(ctx context.Context, awsCfg *config.AWSConfig, cfg *config.Config) (core.ReportGenerator, error) {
	// Create WAFR client configuration
	clientCfg := &wafr.ClientConfig{
		Region:  awsCfg.Region,
		Profile: awsCfg.Profile,
	}

	// Create evaluator configuration
	evalCfg := &wafr.EvaluatorConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
	}

	// Create evaluator with configuration
	evaluator, err := wafr.NewEvaluatorWithConfig(ctx, clientCfg, evalCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator for report generator: %w", err)
	}

	generator := report.NewGeneratorWithEvaluator(evaluator)
	return generator, nil
}
