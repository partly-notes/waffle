# Implementation Plan

- [x] 1. Set up project structure and core interfaces
  - Initialize Go module with proper dependencies (AWS SDK v2, cobra, viper, gopter, testify)
  - Create directory structure: cmd/, internal/, pkg/
  - Define core interfaces for all major components
  - Set up testing framework with gopter for property-based testing
  - _Requirements: 10.1_

- [x] 2. Implement CLI interface with Cobra
  - Create main CLI entry point with cobra
  - Implement `review` command with all required flags (--workload-id, --plan-file, --scope, --pillar, --question-id) that operates on current directory
  - Implement `status` command to check session status
  - Implement `results` command with format options (--format, --output)
  - Implement `compare` command for milestone comparison
  - Add proper exit codes for different error conditions
  - _Requirements: 10.1, 10.2, 10.5, 11.1_

- [ ]* 2.1 Write property test for CLI exit codes
  - **Property 24: Error exit codes**
  - **Validates: Requirements 10.5**

- [x] 3. Implement AWS Well-Architected Tool API client
  - Create wrapper for AWS SDK v2 wellarchitected client
  - Implement CreateWorkload operation
  - Implement ListAnswers operation with scope filtering
  - Implement UpdateAnswer operation
  - Implement CreateMilestone operation
  - Implement GetConsolidatedReport operation
  - Add retry logic with exponential backoff
  - _Requirements: 1.1, 3.1, 6.3, 7.1_

- [ ]* 3.1 Write property test for retry with exponential backoff
  - **Property 15: Retry with exponential backoff**
  - **Validates: Requirements 7.1**

- [x] 4. Implement Session Manager
  - Create session data structures matching design
  - Implement CreateSession with unique ID generation
  - Implement SaveSession with JSON serialization to ~/.waffle/sessions/
  - Implement LoadSession with JSON deserialization
  - Implement UpdateSessionStatus
  - Implement ListSessions for a workload
  - Link sessions to AWS workload IDs
  - _Requirements: 1.3, 1.5_

- [ ]* 4.1 Write property test for session persistence round-trip
  - **Property 2: Session persistence round-trip**
  - **Validates: Requirements 1.5**

- [x] 5. Implement IaC Analyzer - File retrieval
  - Implement RetrieveIaCFiles to scan current working directory
  - Add validation for directory access and IaC file presence
  - Handle file system access errors with specific error messages
  - _Requirements: 1.2, 1.4, 2.1_

- [ ]* 5.1 Write property test for directory validation errors
  - **Property 3: Directory validation errors**
  - **Validates: Requirements 1.2, 1.4**

- [x] 6. Implement IaC Analyzer - Terraform validation
  - Implement ValidateTerraformFiles to identify .tf files
  - Validate Terraform HCL syntax
  - Handle invalid or non-Terraform files with clear errors
  - _Requirements: 2.2_

- [ ]* 6.1 Write property test for Terraform file validation
  - **Property 4: IaC framework identification**
  - **Validates: Requirements 2.2**

- [x] 7. Implement IaC Analyzer - Terraform plan parsing
  - Implement ParseTerraformPlan to read and parse Terraform plan JSON
  - Extract planned_values.root_module resources
  - Extract resources from child_modules recursively
  - Capture resource addresses with module paths
  - Extract computed values and resolved configurations
  - Build WorkloadModel from plan data
  - _Requirements: 11.1, 11.2, 11.3, 11.4_

- [ ]* 7.1 Write property test for Terraform plan parsing completeness
  - **Property 25: Terraform plan parsing completeness**
  - **Validates: Requirements 11.1, 11.2, 11.3**

- [x] 7.2 Implement IaC Analyzer - Terraform HCL parsing
  - Implement ParseTerraform for HCL configurations using HCL parser library
  - Parse resource blocks (resource, data, module)
  - Parse variables and outputs
  - Extract resource types, properties, and metadata
  - Build WorkloadModel with all resources
  - Note: Use local Go HCL parser, not Bedrock, for syntax parsing
  - _Requirements: 2.3, 2.5_

- [x] 8. Implement IaC Analyzer - Model merging
  - Implement MergeWorkloadModels to combine plan and HCL data
  - Prioritize plan data when both sources available
  - Supplement plan data with HCL context (comments, source locations)
  - Handle cases where only one source is available
  - _Requirements: 11.5_

- [ ]* 8.1 Write property test for plan prioritization
  - **Property 26: Plan prioritization**
  - **Validates: Requirements 11.5**

- [x] 8.2 Implement IaC Analyzer - Relationship mapping
  - Implement IdentifyRelationships to build dependency graph
  - Parse resource references and dependencies
  - Create ResourceGraph structure
  - Store relationships in WorkloadModel
  - _Requirements: 2.4, 2.5_

- [ ]* 8.3 Write property test for resource extraction completeness
  - **Property 5: Resource extraction completeness**
  - **Validates: Requirements 2.4, 2.5**

- [x] 9. Implement Bedrock Client integration
  - Create Bedrock runtime client using AWS SDK v2
  - Implement InvokeModel for direct model invocation
  - Build structured prompts for IaC semantic analysis
  - Build structured prompts for WAFR question evaluation
  - Parse JSON responses from Bedrock models
  - Add rate limiting to prevent API throttling
  - Add audit logging for all Bedrock operations
  - Handle Bedrock API errors with retries
  - _Requirements: 2.3, 3.2, 7.1, 8.2, 8.3, 8.4_

- [ ]* 9.1 Write property test for rate limiting enforcement
  - **Property 18: Rate limiting enforcement**
  - **Validates: Requirements 8.3**

- [ ]* 9.2 Write property test for audit logging
  - **Property 19: Audit logging**
  - **Validates: Requirements 8.4**

- [x] 10. Implement sensitive data redaction
  - Create redaction rules for AWS keys, secrets, passwords
  - Implement redaction for email addresses and IP addresses
  - Apply redaction during IaC parsing
  - Store only redacted versions
  - Never log sensitive data
  - _Requirements: 8.5_

- [ ]* 10.1 Write property test for sensitive data redaction
  - **Property 20: Sensitive data redaction**
  - **Validates: Requirements 8.5**

- [x] 11. Implement WAFR Evaluator - Workload creation
  - Implement CreateWorkload using AWS API
  - Map user workload ID to AWS workload ID
  - Store AWS workload ID in session
  - Handle workload creation errors
  - _Requirements: 1.1, 1.3_

- [ ]* 11.1 Write property test for session creation with valid inputs
  - **Property 1: Session creation with valid inputs**
  - **Validates: Requirements 1.1, 1.3**

- [x] 12. Implement WAFR Evaluator - Question retrieval
  - Implement GetQuestions using AWS ListAnswers API
  - Filter questions by scope (workload/pillar/question)
  - Parse AWS question structure into internal model
  - Handle all six pillars correctly
  - _Requirements: 3.1, 9.1, 9.2, 9.3, 9.4_

- [ ]* 12.1 Write property test for question retrieval by scope
  - **Property 6: Question retrieval by scope**
  - **Validates: Requirements 3.1, 9.2, 9.3, 9.4**

- [x] 13. Implement WAFR Evaluator - Question evaluation with Bedrock
  - Implement EvaluateQuestion using Bedrock direct model invocation
  - Build structured prompt with question details and workload model
  - Invoke Bedrock model with prompt
  - Parse JSON response for selected choices and evidence
  - Extract evidence from IaC analysis
  - Calculate confidence score (0.0-1.0) based on data completeness
  - Handle partial data with appropriate confidence scores
  - _Requirements: 3.2, 3.3, 3.4, 7.3_

- [ ]* 13.1 Write property test for choice selection completeness
  - **Property 7: Choice selection completeness**
  - **Validates: Requirements 3.3, 3.4, 3.5**

- [ ]* 13.2 Write property test for graceful degradation
  - **Property 16: Graceful degradation**
  - **Validates: Requirements 7.3**

- [x] 14. Implement WAFR Evaluator - Answer submission
  - Implement SubmitAnswer using AWS UpdateAnswer API
  - Submit selected choices to AWS for each question
  - Include notes about automated analysis
  - Handle submission errors gracefully
  - Continue processing remaining questions on failures
  - _Requirements: 3.5, 7.2_

- [x] 15. Implement WAFR Evaluator - Improvement plan retrieval
  - Implement GetImprovementPlan using AWS API
  - Retrieve risks identified by AWS
  - Retrieve improvement plan items from AWS
  - Parse AWS risk structure (High, Medium, None)
  - Enhance with IaC-specific resource references
  - _Requirements: 4.1, 4.2, 5.1_

- [ ]* 15.1 Write property test for risk categorization
  - **Property 8: Risk categorization**
  - **Validates: Requirements 4.2**

- [x] 16. Implement WAFR Evaluator - Milestone creation
  - Implement CreateMilestone using AWS API
  - Generate milestone name with timestamp
  - Store milestone ID in session
  - Handle milestone creation errors
  - _Requirements: 6.4_

- [x] 17. Implement Core Engine workflow orchestration
  - Implement InitiateReview to coordinate all components
  - Create session and AWS workload
  - Invoke IaC Analyzer
  - Invoke WAFR Evaluator for all questions in scope
  - Submit all answers to AWS
  - Create milestone
  - Handle errors at each step
  - Implement checkpoint persistence for resume capability
  - _Requirements: 1.1, 7.5_

- [ ]* 17.1 Write property test for checkpoint and resume
  - **Property 17: Checkpoint and resume**
  - **Validates: Requirements 7.5**

- [x] 18. Implement scope filtering logic
  - Ensure workload scope processes all pillars and questions
  - Ensure pillar scope processes only specified pillar
  - Ensure question scope processes only specified question
  - Validate scope parameters
  - _Requirements: 9.2, 9.3, 9.4, 9.5_

- [ ]* 18.1 Write property test for scope-limited results
  - **Property 21: Scope-limited results**
  - **Validates: Requirements 9.5**

- [x] 19. Implement Report Generator
  - Implement GetConsolidatedReport using AWS API for PDF format
  - Implement GetConsolidatedReport using AWS API for JSON format
  - Implement GetResultsJSON to enhance AWS JSON with IaC evidence
  - Add confidence scores to JSON output
  - Add resource inventory to JSON output
  - Add AWS workload ID and console links
  - _Requirements: 6.1, 6.2, 6.3_

- [ ]* 19.1 Write property test for report completeness
  - **Property 12: Report completeness**
  - **Validates: Requirements 6.1, 6.2**

- [ ]* 19.2 Write property test for multi-format export
  - **Property 13: Multi-format export**
  - **Validates: Requirements 6.3**

- [x] 20. Implement milestone comparison
  - Implement CompareMilestones using AWS API
  - Categorize changes as improvements, regressions, or new risks
  - Format comparison output
  - Handle missing milestones gracefully
  - _Requirements: 6.4, 6.5_

- [ ]* 20.1 Write property test for milestone comparison categorization
  - **Property 14: Milestone comparison categorization**
  - **Validates: Requirements 6.5**

- [x] 21. Implement CLI progress output
  - Add progress updates to stdout during review execution
  - Show current step (IaC analysis, question evaluation, etc.)
  - Display question count and progress
  - Show completion status
  - _Requirements: 10.3_

- [ ]* 21.1 Write property test for CLI progress output
  - **Property 22: CLI progress output**
  - **Validates: Requirements 10.3**

- [x] 22. Implement JSON output for CI/CD
  - Ensure all CLI commands output valid JSON when requested
  - Include session ID in review command output
  - Include all results in results command output
  - Validate JSON structure
  - _Requirements: 10.4_

- [x] 22.1 Write property test for JSON output validity
  - **Property 23: JSON output validity**
  - **Validates: Requirements 10.4**

- [ ] 23. Implement error handling and logging
  - Create structured logging with levels (DEBUG, INFO, WARNING, ERROR)
  - Log all errors with context
  - Provide clear error messages for users
  - Include troubleshooting guidance in error messages
  - Store logs in .waffle/logs/ (current directory by default)
  - _Requirements: 7.4_

- [x] 24. Implement configuration management
  - Create default configuration structure
  - Support configuration file at ~/.waffle/config.yaml
  - Support environment variables for overrides
  - Support AWS profile selection
  - Configure Bedrock region and model ID
  - Configure retry and timeout settings
  - Add `waffle init` command to validate AWS setup (credentials, Bedrock access, WAFR permissions)
  - _Requirements: 8.1, 8.2_

- [x] 25. Add comprehensive error handling
  - Handle all directory access errors
  - Handle all IaC parsing errors
  - Handle all Bedrock API errors
  - Handle all AWS WAFR API errors
  - Ensure proper error propagation
  - Test error scenarios
  - _Requirements: 7.1, 7.2, 7.4_

- [x] 26. Implement review command workflow orchestration
  - Wire Core Engine into review command
  - Initialize all dependencies (IaC Analyzer, Session Manager, WAFR Evaluator, Bedrock Client, Report Generator)
  - Implement complete workflow: create session → analyze IaC → evaluate questions → submit answers → create milestone
  - Handle errors at each step with appropriate exit codes
  - Replace placeholder progress output with real progress reporting
  - _Requirements: 1.1, 10.1, 10.2, 10.3_

- [x] 26.1 Implement status command
  - Load session from Session Manager
  - Display session status and progress information
  - Output JSON format for CI/CD integration
  - Handle session not found errors
  - _Requirements: 10.1_

- [x] 26.2 Implement results command
  - Load session from Session Manager
  - Retrieve consolidated report from AWS API (PDF or JSON format)
  - Enhance JSON output with IaC evidence and confidence scores
  - Write output to file or stdout
  - Handle incomplete or failed sessions
  - _Requirements: 6.1, 6.2, 6.3, 10.1, 10.4_

- [x] 26.3 Implement compare command
  - Load both sessions from Session Manager
  - Retrieve milestone data from AWS API
  - Compare milestones and categorize changes
  - Output comparison results in JSON format
  - Handle missing milestones gracefully
  - _Requirements: 6.4, 6.5, 10.1_

- [ ] 27. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 28. Create documentation
  - Write README with installation instructions
  - Document CLI commands and flags
  - Provide usage examples
  - Document configuration options
  - Add troubleshooting guide
  - Document AWS permissions required
  - _Requirements: All_

- [x] 29. Build and package
  - Set up Go build for multiple platforms (Linux, macOS, Windows)
  - Create release binaries
  - Set up Docker image
  - Test installation on different platforms
  - _Requirements: 10.1_
