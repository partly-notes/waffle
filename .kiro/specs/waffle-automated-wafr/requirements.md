# Requirements Document

## Introduction

Waffle (Well Architected Framework for Less Effort) is a service that automates AWS Well-Architected Framework Review (WAFR) executions. The system analyzes infrastructure-as-code (IaC) to automatically answer WAFR questions, identify gaps, and provide high-level improvement guidance using Amazon Bedrock foundation models via direct API invocation.

## Glossary

- **WAFR**: AWS Well-Architected Framework Review - A structured assessment process that evaluates cloud architectures against AWS best practices
- **Workload**: A collection of resources and code that delivers business value (WAFR terminology for the system being assessed)
- **Pillar**: One of the six foundational areas of the Well-Architected Framework (Operational Excellence, Security, Reliability, Performance Efficiency, Cost Optimization, and Sustainability)
- **Question**: A specific assessment question from the WAFR that evaluates a particular aspect of architecture within a pillar
- **Best Practice**: A recommended approach or design pattern from the WAFR that addresses a specific question
- **Choice**: A specific implementation option or answer to a WAFR question
- **Risk**: An identified issue where the workload does not follow WAFR best practices (categorized as High, Medium, or None)
- **Improvement Plan**: High-level guidance describing how to address identified risks and align with WAFR best practices
- **Lens**: A collection of questions focused on a specific technology or domain (e.g., Serverless Lens, SaaS Lens)
- **Milestone**: A snapshot of a workload review at a specific point in time
- **Waffle System**: The automated WAFR review service being developed
- **Bedrock**: Amazon Bedrock - AWS's managed service for foundation models and generative AI
- **Bedrock Runtime API**: The API for directly invoking Bedrock foundation models without requiring deployed agents or infrastructure
- **IaC Analysis**: The process of examining infrastructure-as-code definitions to understand architecture and resource configurations
- **IaC**: Infrastructure as Code - Code-based definitions of infrastructure using Terraform
- **Review Session**: A complete execution of WAFR assessment for a specific workload
- **Terraform Plan**: A JSON-formatted output from `terraform plan -out=plan.tfplan && terraform show -json plan.tfplan` that contains the complete planned infrastructure including resolved module contents and computed values

## Requirements

### Requirement 1

**User Story:** As a cloud architect, I want to initiate an automated WAFR review for my IaC in the current directory, so that I can assess my architecture without manual effort.

#### Acceptance Criteria

1. WHEN a user provides a workload identifier THEN the Waffle System SHALL initiate a new review session using the current directory as the IaC location
2. WHEN a review session is initiated THEN the Waffle System SHALL validate access to the IaC files in the current directory
3. WHEN validation succeeds THEN the Waffle System SHALL create a review session record with unique identifier and timestamp
4. WHEN validation fails THEN the Waffle System SHALL return an error message indicating the specific access issue
5. WHEN a review session is created THEN the Waffle System SHALL persist the session metadata for future reference

### Requirement 2

**User Story:** As a cloud architect, I want the system to analyze my infrastructure-as-code, so that it can understand my architecture and resource configurations.

#### Acceptance Criteria

1. WHEN a review session begins THEN the Waffle System SHALL retrieve the IaC files from the current directory
2. WHEN IaC files are retrieved THEN the Waffle System SHALL identify Terraform configuration files
3. WHEN Terraform files are identified THEN the Waffle System SHALL parse the Terraform configurations using HCL parser libraries and extract resource definitions
4. WHEN IaC parsing completes THEN the Waffle System SHALL identify relationships and dependencies between defined resources
5. WHEN IaC analysis completes THEN the Waffle System SHALL store the architecture model with all resource metadata and relationships

### Requirement 11

**User Story:** As a cloud architect, I want to provide a Terraform plan JSON file for analysis, so that the system can access complete infrastructure details including resolved modules and computed values.

#### Acceptance Criteria

1. WHEN a user provides a Terraform plan JSON file path THEN the Waffle System SHALL read and parse the plan file
2. WHEN a Terraform plan is provided THEN the Waffle System SHALL use the plan data as the primary source for resource definitions
3. WHEN a Terraform plan contains module resources THEN the Waffle System SHALL extract all module resources with their resolved configurations
4. WHEN a Terraform plan contains computed values THEN the Waffle System SHALL include those values in the architecture model
5. WHEN both Terraform source files and a plan file are available THEN the Waffle System SHALL prioritize the plan file for resource analysis while using source files for additional context

### Requirement 3

**User Story:** As a cloud architect, I want the system to automatically answer WAFR questions based on my IaC analysis, so that I can complete the review efficiently.

#### Acceptance Criteria

1. WHEN IaC analysis completes THEN the Waffle System SHALL retrieve all applicable questions based on the selected scope
2. WHEN questions are retrieved THEN the Waffle System SHALL invoke Bedrock foundation models directly to analyze IaC data against each question
3. WHEN Bedrock analyzes a question THEN the Waffle System SHALL select applicable choices with supporting evidence from the IaC analysis
4. WHEN choices are selected THEN the Waffle System SHALL assign a confidence score based on the completeness of available data
5. WHEN all questions are processed THEN the Waffle System SHALL compile the complete set of selected choices with evidence and confidence scores

### Requirement 4

**User Story:** As a cloud architect, I want the system to identify risks in my workload, so that I can understand where improvements are needed.

#### Acceptance Criteria

1. WHEN questions are answered THEN the Waffle System SHALL identify questions where best practices are not followed
2. WHEN risks are identified THEN the Waffle System SHALL categorize each risk by pillar and severity level (High, Medium, or None)
3. WHEN risks are categorized THEN the Waffle System SHALL prioritize risks based on impact and complexity
4. WHEN prioritization completes THEN the Waffle System SHALL generate a risk summary with all identified issues
5. WHEN the risk summary is generated THEN the Waffle System SHALL include specific resource references and configuration details for each risk

### Requirement 5

**User Story:** As a cloud architect, I want the system to provide an improvement plan based on WAFR best practices, so that I understand what changes would address identified risks.

#### Acceptance Criteria

1. WHEN risks are identified THEN the Waffle System SHALL invoke Bedrock foundation models to generate human-readable improvement plan items for each risk
2. WHEN improvement plan items are generated THEN the Waffle System SHALL describe the recommended changes in plain language
3. WHEN improvement plan items are provided THEN the Waffle System SHALL reference relevant best practices and AWS documentation
4. WHEN improvement plan items involve multiple resources THEN the Waffle System SHALL explain the relationships and dependencies in the recommendation
5. WHEN the improvement plan is complete THEN the Waffle System SHALL present recommendations as descriptive text without code implementation details

### Requirement 6

**User Story:** As a cloud architect, I want to review and export the WAFR assessment results, so that I can share findings with stakeholders and track progress over time.

#### Acceptance Criteria

1. WHEN a review session completes THEN the Waffle System SHALL generate a comprehensive report with all selected choices, identified risks, and improvement plan
2. WHEN a report is generated THEN the Waffle System SHALL include executive summary, detailed findings by pillar, and improvement plan sections
3. WHEN a user requests report export THEN the Waffle System SHALL provide the report in multiple formats (PDF, JSON, HTML)
4. WHEN a user requests milestone comparison THEN the Waffle System SHALL compare current results with previous milestones for the same workload
5. WHEN milestone comparison is performed THEN the Waffle System SHALL highlight improvements, regressions, and new risks since the last milestone

### Requirement 7

**User Story:** As a cloud architect, I want the system to handle errors gracefully during analysis, so that partial failures do not prevent me from getting useful results.

#### Acceptance Criteria

1. WHEN Bedrock API calls encounter errors THEN the Waffle System SHALL retry the operation with exponential backoff up to three attempts
2. WHEN retries are exhausted THEN the Waffle System SHALL log the error details and continue processing remaining items
3. WHEN partial IaC data is available THEN the Waffle System SHALL generate answers with appropriate confidence scores reflecting data completeness
4. WHEN critical errors prevent analysis THEN the Waffle System SHALL provide clear error messages with troubleshooting guidance
5. WHEN a review session encounters errors THEN the Waffle System SHALL persist the session state to allow resumption from the last successful checkpoint

### Requirement 8

**User Story:** As a system administrator, I want to configure Bedrock access with appropriate permissions, so that the system can perform IaC analysis securely without requiring infrastructure deployment.

#### Acceptance Criteria

1. WHEN the Waffle System initializes THEN the system SHALL validate AWS credentials and Bedrock model access in the configured region
2. WHEN Bedrock is invoked THEN the Waffle System SHALL use direct model invocation via Bedrock Runtime API without requiring deployed agents
3. WHEN Bedrock API calls are made THEN the Waffle System SHALL implement rate limiting to prevent API throttling
4. WHEN Bedrock operations execute THEN the Waffle System SHALL log all operations for audit purposes
5. WHEN sensitive data is found in IaC THEN the Waffle System SHALL redact credentials, keys, and personally identifiable information before storage

### Requirement 9

**User Story:** As a cloud architect, I want to choose the scope of my WAFR review execution, so that I can focus on specific areas of interest or conduct targeted assessments.

#### Acceptance Criteria

1. WHEN initiating a review session THEN the Waffle System SHALL allow the user to select scope level (workload, pillar, or question)
2. WHEN workload scope is selected THEN the Waffle System SHALL execute the complete WAFR assessment across all pillars and questions
3. WHEN pillar scope is selected THEN the Waffle System SHALL execute assessment only for questions within the specified pillar (Operational Excellence, Security, Reliability, Performance Efficiency, Cost Optimization, or Sustainability)
4. WHEN question scope is selected THEN the Waffle System SHALL execute assessment only for the specific WAFR question identifier provided
5. WHEN a scoped review completes THEN the Waffle System SHALL generate results and recommendations limited to the selected scope

### Requirement 10

**User Story:** As a developer, I want to interact with Waffle through a command-line interface, so that I can integrate WAFR reviews into my development workflow and CI/CD pipelines.

#### Acceptance Criteria

1. WHEN a user invokes the Waffle CLI THEN the Waffle System SHALL provide commands for initiating reviews, checking status, and retrieving results
2. WHEN a user runs a review command THEN the Waffle System SHALL accept parameters for workload identifier and scope selection, using the current directory as the IaC repository location
3. WHEN a user specifies the --region flag THEN the Waffle System SHALL use that AWS region for both Bedrock and WAFR operations, overriding configuration file and environment variable settings
4. WHEN a user specifies the --profile flag THEN the Waffle System SHALL use that AWS profile for credentials, overriding configuration file and environment variable settings
5. WHEN a user specifies the --model-id flag THEN the Waffle System SHALL use that Bedrock model ID for analysis, overriding configuration file and environment variable settings
6. WHEN a review is executed via CLI THEN the Waffle System SHALL display progress updates and status information to standard output
7. WHEN a review completes via CLI THEN the Waffle System SHALL output results in machine-readable format (JSON) for CI/CD integration
8. WHEN the CLI encounters errors THEN the Waffle System SHALL return appropriate exit codes and error messages for automation compatibility
