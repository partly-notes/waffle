# AWS WAFR API Usage Guidelines

## Overview

When implementing the Waffle system, always prefer using the AWS Well-Architected Tool API rather than duplicating WAFR logic locally. This ensures we stay aligned with AWS's official WAFR implementation and benefit from their updates.

## AWS Well-Architected Tool API

The AWS Well-Architected Tool provides a comprehensive API for managing workloads, lenses, and reviews. Use these APIs wherever possible:

### Key API Operations

**Workload Management**:
- `CreateWorkload` - Create a new workload
- `GetWorkload` - Retrieve workload details
- `UpdateWorkload` - Update workload configuration
- `ListWorkloads` - List all workloads

**Lens Operations**:
- `ListLenses` - Get available lenses (including custom lenses)
- `GetLens` - Retrieve lens details and questions
- `GetLensReview` - Get review status for a lens

**Question and Answer Management**:
- `ListAnswers` - Get all answers for a workload
- `GetAnswer` - Retrieve specific answer details
- `UpdateAnswer` - Update answer choices and notes

**Milestone Management**:
- `CreateMilestone` - Create a snapshot of the workload
- `ListMilestones` - List all milestones for a workload
- `GetMilestone` - Retrieve milestone details

**Review and Improvement**:
- `GetLensReviewReport` - Generate review report
- `ListLensReviewImprovements` - Get improvement plan items

### Implementation Guidelines

1. **Use AWS API for WAFR Data**
   - Don't hardcode WAFR questions locally
   - Fetch questions dynamically from the API
   - Use the official pillar and best practice definitions

2. **Leverage Existing Workload Management**
   - Create workloads in AWS Well-Architected Tool
   - Store workload IDs for tracking
   - Use AWS's milestone feature for comparisons

3. **Integrate with AWS Review Process**
   - Update answers via API after IaC analysis
   - Use AWS's risk calculation logic
   - Leverage AWS's improvement plan generation

4. **Benefits of Using AWS API**
   - Always up-to-date with latest WAFR questions
   - Consistent with AWS console experience
   - Access to AWS's risk algorithms
   - Support for custom lenses
   - Built-in milestone and comparison features

### API Authentication

Use AWS credentials and IAM roles for API access:
- Required permissions: `wellarchitected:*` (or specific operations)
- Support AWS profiles and role assumption
- Use AWS SDK for Go v2 for implementation

### Example Integration Pattern

```go
package main

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/wellarchitected"
    "github.com/aws/aws-sdk-go-v2/service/wellarchitected/types"
    "github.com/aws/aws-sdk-go-v2/aws"
)

func main() {
    // Load AWS config
    cfg, _ := config.LoadDefaultConfig(context.TODO())
    
    // Initialize Well-Architected client
    waClient := wellarchitected.NewFromConfig(cfg)
    
    // Create workload
    workload, _ := waClient.CreateWorkload(context.TODO(), &wellarchitected.CreateWorkloadInput{
        WorkloadName: aws.String("my-app"),
        Description:  aws.String("My application workload"),
        Environment:  types.WorkloadEnvironmentProduction,
        Lenses:       []string{"wellarchitected"},
    })
    
    // Get questions for a pillar
    answers, _ := waClient.ListAnswers(context.TODO(), &wellarchitected.ListAnswersInput{
        WorkloadId: workload.WorkloadId,
        LensAlias:  aws.String("wellarchitected"),
        PillarId:   aws.String("security"),
    })
    
    // Update answer based on IaC analysis
    waClient.UpdateAnswer(context.TODO(), &wellarchitected.UpdateAnswerInput{
        WorkloadId:      workload.WorkloadId,
        LensAlias:       aws.String("wellarchitected"),
        QuestionId:      aws.String("sec-1"),
        SelectedChoices: []string{"sec_1_choice_1", "sec_1_choice_2"},
        Notes:           aws.String("Automatically analyzed from IaC"),
    })
    
    // Create milestone
    milestone, _ := waClient.CreateMilestone(context.TODO(), &wellarchitected.CreateMilestoneInput{
        WorkloadId:    workload.WorkloadId,
        MilestoneName: aws.String("v1.0"),
    })
}
```

## When to Use Local Logic

Only implement local logic when:
1. Parsing IaC files (AWS API doesn't do this)
2. Mapping IaC resources to WAFR choices (requires custom logic)
3. Generating evidence from IaC analysis
4. Calculating confidence scores for automated answers

## Architecture Impact

The Waffle system should:
1. Use Bedrock Agent to analyze IaC and determine appropriate WAFR choices
2. Use AWS Well-Architected Tool API to:
   - Create/manage workloads
   - Fetch questions and best practices
   - Submit answers
   - Generate reports
   - Create milestones
   - Compare milestones
3. Focus local implementation on the IaC analysis and choice selection logic

This approach ensures Waffle acts as an intelligent bridge between IaC repositories and the AWS Well-Architected Tool, rather than reimplementing WAFR functionality.
