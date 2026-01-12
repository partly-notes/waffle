package iac

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/waffle/waffle/internal/core"
	"github.com/waffle/waffle/internal/redaction"
	"github.com/zclconf/go-cty/cty"
)

const (
	// MaxFileSize is the maximum size of a single IaC file (10MB)
	MaxFileSize = 10 * 1024 * 1024
	// MaxFiles is the maximum number of IaC files to process
	MaxFiles = 10000
)

// Analyzer implements the IaCAnalyzer interface
type Analyzer struct {
	workingDir string
	redactor   *redaction.Redactor
}

// NewAnalyzer creates a new IaC analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		workingDir: ".",
		redactor:   redaction.NewRedactor(),
	}
}

// NewAnalyzerWithDir creates a new IaC analyzer with a specific working directory
func NewAnalyzerWithDir(workingDir string) *Analyzer {
	return &Analyzer{
		workingDir: workingDir,
		redactor:   redaction.NewRedactor(),
	}
}

// RetrieveIaCFiles retrieves IaC files from the current directory
func (a *Analyzer) RetrieveIaCFiles(ctx context.Context) ([]core.IaCFile, error) {
	// Validate directory access
	dirInfo, err := os.Stat(a.workingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &core.DirectoryAccessError{
				Path: a.workingDir,
				Err:  fmt.Errorf("directory does not exist"),
			}
		}
		if os.IsPermission(err) {
			return nil, &core.DirectoryAccessError{
				Path: a.workingDir,
				Err:  fmt.Errorf("permission denied"),
			}
		}
		return nil, &core.DirectoryAccessError{
			Path: a.workingDir,
			Err:  err,
		}
	}

	if !dirInfo.IsDir() {
		return nil, &core.DirectoryAccessError{
			Path: a.workingDir,
			Err:  fmt.Errorf("path is not a directory"),
		}
	}

	slog.InfoContext(ctx, "scanning directory for IaC files",
		"directory", a.workingDir,
	)

	var files []core.IaCFile
	fileCount := 0

	// Walk the directory tree
	err = filepath.WalkDir(a.workingDir, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			slog.WarnContext(ctx, "error accessing path",
				"path", path,
				"error", err,
			)
			return nil // Continue walking
		}

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories and common non-IaC directories
			name := d.Name()
			if strings.HasPrefix(name, ".") || 
			   name == "node_modules" || 
			   name == "vendor" ||
			   name == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a Terraform file
		if !isTerraformFile(path) {
			return nil
		}

		// Check file count limit
		fileCount++
		if fileCount > MaxFiles {
			return &core.ValidationError{
				Field:   "file_count",
				Value:   fileCount,
				Message: fmt.Sprintf("exceeded maximum file limit of %d", MaxFiles),
			}
		}

		// Check file size
		info, err := d.Info()
		if err != nil {
			slog.WarnContext(ctx, "error getting file info",
				"path", path,
				"error", err,
			)
			return nil
		}

		if info.Size() > MaxFileSize {
			slog.WarnContext(ctx, "skipping file exceeding size limit",
				"path", path,
				"size", info.Size(),
				"limit", MaxFileSize,
			)
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsPermission(err) {
				return &core.DirectoryAccessError{
					Path: path,
					Err:  fmt.Errorf("permission denied reading file"),
				}
			}
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Get relative path
		relPath, err := filepath.Rel(a.workingDir, path)
		if err != nil {
			relPath = path
		}

		// Redact sensitive data from file content
		redactedContent, findings := a.redactor.Redact(string(content))
		
		if len(findings) > 0 {
			slog.WarnContext(ctx, "sensitive data redacted from IaC file",
				"file", relPath,
				"findings", findings,
			)
		}

		files = append(files, core.IaCFile{
			Path:    relPath,
			Content: redactedContent,
		})

		slog.DebugContext(ctx, "retrieved IaC file",
			"path", relPath,
			"size", len(content),
			"redacted", len(findings) > 0,
		)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Validate that we found at least one IaC file
	if len(files) == 0 {
		return nil, &core.DirectoryAccessError{
			Path: a.workingDir,
			Err:  fmt.Errorf("no IaC files found in directory"),
		}
	}

	slog.InfoContext(ctx, "IaC file retrieval complete",
		"directory", a.workingDir,
		"files_found", len(files),
	)

	return files, nil
}

// isTerraformFile checks if a file is a Terraform file
func isTerraformFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tf" || ext == ".tfvars"
}

// ValidateTerraformFiles validates Terraform file syntax
func (a *Analyzer) ValidateTerraformFiles(ctx context.Context, files []core.IaCFile) error {
	if len(files) == 0 {
		return core.ErrNoFilesProvided
	}

	slog.InfoContext(ctx, "validating terraform files",
		"file_count", len(files),
	)

	parser := hclparse.NewParser()
	var allDiags hcl.Diagnostics

	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip non-Terraform files
		if !isTerraformFile(file.Path) {
			slog.WarnContext(ctx, "skipping non-terraform file",
				"path", file.Path,
			)
			continue
		}

		// Parse the HCL file
		_, diags := parser.ParseHCL([]byte(file.Content), file.Path)
		
		// Collect diagnostics
		if diags.HasErrors() {
			allDiags = append(allDiags, diags...)
			
			// Log each error for debugging
			for _, diag := range diags {
				if diag.Severity == hcl.DiagError {
					slog.ErrorContext(ctx, "terraform syntax error",
						"file", file.Path,
						"summary", diag.Summary,
						"detail", diag.Detail,
					)
				}
			}
		} else {
			slog.DebugContext(ctx, "terraform file validated successfully",
				"path", file.Path,
			)
		}
	}

	// If we have any errors, return a TerraformSyntaxError
	if allDiags.HasErrors() {
		// Get the first error for the error message
		firstError := allDiags.Errs()[0].(*hcl.Diagnostic)
		
		line := 0
		if firstError.Subject != nil {
			line = firstError.Subject.Start.Line
		}
		
		filename := "unknown"
		if firstError.Subject != nil {
			filename = firstError.Subject.Filename
		}
		
		return &core.TerraformSyntaxError{
			File:    filename,
			Line:    line,
			Message: fmt.Sprintf("%s: %s", firstError.Summary, firstError.Detail),
		}
	}

	slog.InfoContext(ctx, "terraform validation complete",
		"files_validated", len(files),
	)

	return nil
}

// ParseTerraformPlan parses a Terraform JSON file (plan or state)
func (a *Analyzer) ParseTerraformPlan(ctx context.Context, jsonFilePath string) (*core.WorkloadModel, error) {
	slog.InfoContext(ctx, "parsing terraform JSON file",
		"json_file", jsonFilePath,
	)

	// Read the JSON file
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &core.FileAccessError{
				Path:      jsonFilePath,
				Operation: "read",
				Err:       err,
			}
		}
		if os.IsPermission(err) {
			return nil, &core.FileAccessError{
				Path:      jsonFilePath,
				Operation: "access",
				Err:       err,
			}
		}
		return nil, &core.FileAccessError{
			Path:      jsonFilePath,
			Operation: "read",
			Err:       err,
		}
	}

	// Parse the JSON
	var plan TerraformPlan
	if err := json.Unmarshal(jsonData, &plan); err != nil {
		return nil, &core.IaCParsingError{
			File:    jsonFilePath,
			Err:     err,
			Context: "invalid JSON format",
		}
	}

	slog.DebugContext(ctx, "terraform JSON parsed",
		"format_version", plan.FormatVersion,
		"terraform_version", plan.TerraformVersion,
	)

	// Extract resources from the plan
	resources := []core.Resource{}
	
	// Extract resources from root module
	if plan.PlannedValues.RootModule != nil {
		rootResources := a.extractResourcesFromModuleWithRedaction(ctx, plan.PlannedValues.RootModule, "")
		resources = append(resources, rootResources...)
	}

	slog.InfoContext(ctx, "terraform JSON parsing complete",
		"total_resources", len(resources),
	)

	// Build workload model
	model := &core.WorkloadModel{
		Resources:  resources,
		Framework:  "terraform",
		SourceType: "plan",
		Metadata: map[string]interface{}{
			"format_version":    plan.FormatVersion,
			"terraform_version": plan.TerraformVersion,
			"json_file":         jsonFilePath,
		},
	}

	return model, nil
}

// ParseTerraform parses Terraform HCL files
func (a *Analyzer) ParseTerraform(ctx context.Context, files []core.IaCFile) (*core.WorkloadModel, error) {
	slog.InfoContext(ctx, "parsing terraform HCL files",
		"file_count", len(files),
	)

	if len(files) == 0 {
		return nil, core.ErrNoFilesProvided
	}

	parser := hclparse.NewParser()
	var resources []core.Resource
	var allDiags hcl.Diagnostics

	// Parse each file
	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Skip non-Terraform files
		if !isTerraformFile(file.Path) {
			slog.WarnContext(ctx, "skipping non-terraform file",
				"path", file.Path,
			)
			continue
		}

		// Parse the HCL file
		hclFile, diags := parser.ParseHCL([]byte(file.Content), file.Path)
		allDiags = append(allDiags, diags...)

		if diags.HasErrors() {
			slog.ErrorContext(ctx, "failed to parse HCL file",
				"file", file.Path,
				"errors", diags.Error(),
			)
			continue
		}

		// Extract resources from the parsed file
		fileResources, err := a.extractResourcesFromHCLWithRedaction(ctx, hclFile, file.Path)
		if err != nil {
			slog.WarnContext(ctx, "failed to extract resources from HCL",
				"file", file.Path,
				"error", err,
			)
			continue
		}

		resources = append(resources, fileResources...)
	}

	// If we have critical parsing errors, return them
	if allDiags.HasErrors() {
		firstError := allDiags.Errs()[0].(*hcl.Diagnostic)
		
		line := 0
		if firstError.Subject != nil {
			line = firstError.Subject.Start.Line
		}
		
		filename := "unknown"
		if firstError.Subject != nil {
			filename = firstError.Subject.Filename
		}
		
		return nil, &core.TerraformSyntaxError{
			File:    filename,
			Line:    line,
			Message: fmt.Sprintf("%s: %s", firstError.Summary, firstError.Detail),
		}
	}

	slog.InfoContext(ctx, "terraform HCL parsing complete",
		"total_resources", len(resources),
	)

	// Build workload model
	model := &core.WorkloadModel{
		Resources:  resources,
		Framework:  "terraform",
		SourceType: "hcl",
		Metadata: map[string]interface{}{
			"file_count": len(files),
		},
	}

	return model, nil
}

// extractResourcesFromHCLWithRedaction extracts resources from a parsed HCL file with redaction
func (a *Analyzer) extractResourcesFromHCLWithRedaction(ctx context.Context, file *hcl.File, filePath string) ([]core.Resource, error) {
	var resources []core.Resource

	// Get the body content
	content, diags := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
			{Type: "module", LabelNames: []string{"name"}},
		},
	})

	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to get body content: %s", diags.Error())
	}

	// Extract resource blocks
	for _, block := range content.Blocks {
		switch block.Type {
		case "resource":
			if len(block.Labels) >= 2 {
				resourceType := block.Labels[0]
				resourceName := block.Labels[1]
				address := fmt.Sprintf("%s.%s", resourceType, resourceName)

				// Extract properties from the block
				properties, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"resource", address,
						"error", err,
					)
					properties = make(map[string]interface{})
				}

				// Redact sensitive data from properties
				redactedProperties, findings := a.redactor.RedactProperties(properties)
				
				if len(findings) > 0 {
					slog.WarnContext(ctx, "sensitive data redacted from HCL resource",
						"resource", address,
						"file", filePath,
						"findings", findings,
					)
				}

				resource := core.Resource{
					ID:           address,
					Type:         resourceType,
					Address:      address,
					Properties:   redactedProperties,
					Dependencies: []string{},
					SourceFile:   filePath,
					SourceLine:   block.DefRange.Start.Line,
					IsFromPlan:   false,
					ModulePath:   "",
				}

				resources = append(resources, resource)

				slog.DebugContext(ctx, "extracted resource from HCL",
					"address", address,
					"type", resourceType,
					"file", filePath,
					"line", block.DefRange.Start.Line,
					"redacted", len(findings) > 0,
				)
			}

		case "data":
			if len(block.Labels) >= 2 {
				dataType := block.Labels[0]
				dataName := block.Labels[1]
				address := fmt.Sprintf("data.%s.%s", dataType, dataName)

				properties, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"data", address,
						"error", err,
					)
					properties = make(map[string]interface{})
				}

				// Redact sensitive data from properties
				redactedProperties, findings := a.redactor.RedactProperties(properties)
				
				if len(findings) > 0 {
					slog.WarnContext(ctx, "sensitive data redacted from HCL data source",
						"data", address,
						"file", filePath,
						"findings", findings,
					)
				}

				resource := core.Resource{
					ID:           address,
					Type:         dataType,
					Address:      address,
					Properties:   redactedProperties,
					Dependencies: []string{},
					SourceFile:   filePath,
					SourceLine:   block.DefRange.Start.Line,
					IsFromPlan:   false,
					ModulePath:   "",
				}

				resources = append(resources, resource)

				slog.DebugContext(ctx, "extracted data source from HCL",
					"address", address,
					"type", dataType,
					"file", filePath,
					"line", block.DefRange.Start.Line,
					"redacted", len(findings) > 0,
				)
			}

		case "module":
			if len(block.Labels) >= 1 {
				moduleName := block.Labels[0]
				address := fmt.Sprintf("module.%s", moduleName)

				_, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"module", address,
						"error", err,
					)
				}

				// Module calls are tracked but not treated as resources
				// They will be resolved in the plan
				slog.DebugContext(ctx, "found module call in HCL",
					"address", address,
					"file", filePath,
					"line", block.DefRange.Start.Line,
				)
			}
		}
	}

	return resources, nil
}

// extractResourcesFromHCL extracts resources from a parsed HCL file (kept for backward compatibility)
func extractResourcesFromHCL(ctx context.Context, file *hcl.File, filePath string) ([]core.Resource, error) {
	var resources []core.Resource

	// Get the body content
	content, diags := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
			{Type: "module", LabelNames: []string{"name"}},
		},
	})

	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to get body content: %s", diags.Error())
	}

	// Extract resource blocks
	for _, block := range content.Blocks {
		switch block.Type {
		case "resource":
			if len(block.Labels) >= 2 {
				resourceType := block.Labels[0]
				resourceName := block.Labels[1]
				address := fmt.Sprintf("%s.%s", resourceType, resourceName)

				// Extract properties from the block
				properties, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"resource", address,
						"error", err,
					)
					properties = make(map[string]interface{})
				}

				resource := core.Resource{
					ID:           address,
					Type:         resourceType,
					Address:      address,
					Properties:   properties,
					Dependencies: []string{},
					SourceFile:   filePath,
					SourceLine:   block.DefRange.Start.Line,
					IsFromPlan:   false,
					ModulePath:   "",
				}

				resources = append(resources, resource)

				slog.DebugContext(ctx, "extracted resource from HCL",
					"address", address,
					"type", resourceType,
					"file", filePath,
					"line", block.DefRange.Start.Line,
				)
			}

		case "data":
			if len(block.Labels) >= 2 {
				dataType := block.Labels[0]
				dataName := block.Labels[1]
				address := fmt.Sprintf("data.%s.%s", dataType, dataName)

				properties, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"data", address,
						"error", err,
					)
					properties = make(map[string]interface{})
				}

				resource := core.Resource{
					ID:           address,
					Type:         dataType,
					Address:      address,
					Properties:   properties,
					Dependencies: []string{},
					SourceFile:   filePath,
					SourceLine:   block.DefRange.Start.Line,
					IsFromPlan:   false,
					ModulePath:   "",
				}

				resources = append(resources, resource)

				slog.DebugContext(ctx, "extracted data source from HCL",
					"address", address,
					"type", dataType,
					"file", filePath,
					"line", block.DefRange.Start.Line,
				)
			}

		case "module":
			if len(block.Labels) >= 1 {
				moduleName := block.Labels[0]
				address := fmt.Sprintf("module.%s", moduleName)

				_, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					slog.WarnContext(ctx, "failed to extract properties",
						"module", address,
						"error", err,
					)
				}

				// Module calls are tracked but not treated as resources
				// They will be resolved in the plan
				slog.DebugContext(ctx, "found module call in HCL",
					"address", address,
					"file", filePath,
					"line", block.DefRange.Start.Line,
				)
			}
		}
	}

	return resources, nil
}

// extractPropertiesFromBlock extracts properties from an HCL block body
func extractPropertiesFromBlock(body hcl.Body) (map[string]interface{}, error) {
	properties := make(map[string]interface{})

	// Get all attributes
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		// Try to get partial attributes
		slog.Debug("partial attribute extraction",
			"errors", diags.Error(),
		)
	}

	// Extract attribute values
	for name, attr := range attrs {
		// Try to evaluate the attribute
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			// If we can't evaluate, store the expression as a string
			properties[name] = fmt.Sprintf("${%s}", string(attr.Expr.Range().SliceBytes(attr.Expr.Range().SliceBytes([]byte{}))))
			continue
		}

		// Convert cty.Value to Go value
		goVal, err := ctyToGo(val)
		if err != nil {
			slog.Debug("failed to convert cty value",
				"attribute", name,
				"error", err,
			)
			properties[name] = val.GoString()
			continue
		}

		properties[name] = goVal
	}

	// Get nested blocks using a more flexible approach
	// We need to use the body's Blocks() method if available
	// For now, we'll try to extract blocks by attempting common block types
	commonBlockTypes := []string{
		"ebs_block_device", "root_block_device", "lifecycle", "timeouts",
		"versioning_configuration", "logging", "cors_rule", "website",
		"filter", "tags", "ingress", "egress", "rule",
	}

	for _, blockType := range commonBlockTypes {
		content, _, diags := body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: blockType},
			},
		})

		if !diags.HasErrors() && content != nil && len(content.Blocks) > 0 {
			for _, block := range content.Blocks {
				// Recursively extract nested block properties
				nestedProps, err := extractPropertiesFromBlock(block.Body)
				if err != nil {
					continue
				}

				// Store nested blocks as arrays if multiple blocks with same type
				if existing, ok := properties[block.Type]; ok {
					// Convert to array if not already
					if arr, isArray := existing.([]interface{}); isArray {
						properties[block.Type] = append(arr, nestedProps)
					} else {
						properties[block.Type] = []interface{}{existing, nestedProps}
					}
				} else {
					properties[block.Type] = nestedProps
				}
			}
		}
	}

	return properties, nil
}

// ctyToGo converts a cty.Value to a Go value
func ctyToGo(val cty.Value) (interface{}, error) {
	if val.IsNull() {
		return nil, nil
	}

	valType := val.Type()

	switch {
	case valType == cty.String:
		return val.AsString(), nil
	case valType == cty.Number:
		bf := val.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return i, nil
		}
		f, _ := bf.Float64()
		return f, nil
	case valType == cty.Bool:
		return val.True(), nil
	case valType.IsListType() || valType.IsSetType() || valType.IsTupleType():
		var result []interface{}
		it := val.ElementIterator()
		for it.Next() {
			_, elemVal := it.Element()
			goVal, err := ctyToGo(elemVal)
			if err != nil {
				return nil, err
			}
			result = append(result, goVal)
		}
		return result, nil
	case valType.IsMapType() || valType.IsObjectType():
		result := make(map[string]interface{})
		it := val.ElementIterator()
		for it.Next() {
			keyVal, elemVal := it.Element()
			key := keyVal.AsString()
			goVal, err := ctyToGo(elemVal)
			if err != nil {
				return nil, err
			}
			result[key] = goVal
		}
		return result, nil
	default:
		return val.GoString(), nil
	}
}

// MergeWorkloadModels merges plan and configuration models
// Prioritizes configuration data as the foundation and enhances with plan data
// Configuration provides source context (comments, source locations)
// Plan provides computed values and dependencies
func (a *Analyzer) MergeWorkloadModels(ctx context.Context, planModel, configModel *core.WorkloadModel) (*core.WorkloadModel, error) {
	// Handle cases where only one source is available
	if planModel == nil && configModel == nil {
		return nil, fmt.Errorf("both plan and configuration models are nil")
	}
	if planModel == nil {
		slog.InfoContext(ctx, "no plan model provided, using configuration model only")
		return configModel, nil
	}
	if configModel == nil {
		slog.InfoContext(ctx, "no configuration model provided, using plan model only")
		return planModel, nil
	}

	slog.InfoContext(ctx, "merging workload models (HCL-based approach)",
		"hcl_resources", len(configModel.Resources),
		"plan_resources", len(planModel.Resources),
	)

	// Create a map of plan resources by address for quick lookup
	planResourceMap := make(map[string]*core.Resource)
	for i := range planModel.Resources {
		planResourceMap[planModel.Resources[i].Address] = &planModel.Resources[i]
	}

	// Start with configuration resources (they provide the foundation)
	mergedResources := make([]core.Resource, 0, len(configModel.Resources))

	for _, configRes := range configModel.Resources {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		mergedRes := configRes

		// If we have a corresponding plan resource, enhance with plan data
		if planRes, exists := planResourceMap[configRes.Address]; exists {
			// Enhance with computed values from plan
			if len(planRes.Properties) > len(configRes.Properties) {
				// Merge properties, keeping configuration properties and adding plan-computed ones
				enhancedProperties := make(map[string]interface{})
				
				// Start with configuration properties (source of truth for declared values)
				for k, v := range configRes.Properties {
					enhancedProperties[k] = v
				}
				
				// Add plan-computed properties that aren't in configuration
				for k, v := range planRes.Properties {
					if _, exists := enhancedProperties[k]; !exists {
						enhancedProperties[k] = v
					}
				}
				
				mergedRes.Properties = enhancedProperties
			}

			// Enhance with plan dependencies if more complete
			if len(planRes.Dependencies) > len(configRes.Dependencies) {
				mergedRes.Dependencies = planRes.Dependencies
			}

			// Mark as enhanced with plan data
			mergedRes.IsFromPlan = true

			slog.DebugContext(ctx, "enhanced HCL resource with plan data",
				"address", configRes.Address,
				"hcl_properties", len(configRes.Properties),
				"plan_properties", len(planRes.Properties),
				"merged_properties", len(mergedRes.Properties),
			)

			// Mark that we've processed this plan resource
			delete(planResourceMap, configRes.Address)
		}

		mergedResources = append(mergedResources, mergedRes)
	}

	// Add any plan resources that weren't in the configuration
	// This can happen with computed resources or modules that expand at plan time
	for _, planRes := range planResourceMap {
		slog.DebugContext(ctx, "adding plan-only resource (computed or module-expanded)",
			"address", planRes.Address,
			"type", planRes.Type,
		)
		mergedResources = append(mergedResources, *planRes)
	}

	// Merge metadata, prioritizing configuration metadata
	mergedMetadata := make(map[string]interface{})
	for k, v := range configModel.Metadata {
		mergedMetadata[k] = v
	}
	for k, v := range planModel.Metadata {
		if _, exists := mergedMetadata[k]; !exists {
			mergedMetadata[k] = v
		}
	}
	mergedMetadata["merged"] = true
	mergedMetadata["merge_strategy"] = "configuration_first"
	mergedMetadata["config_resource_count"] = len(configModel.Resources)
	mergedMetadata["plan_resource_count"] = len(planModel.Resources)
	mergedMetadata["plan_only_resources"] = len(planResourceMap)

	slog.InfoContext(ctx, "workload model merge complete (configuration-first)",
		"total_resources", len(mergedResources),
		"config_base", len(configModel.Resources),
		"plan_enhanced", len(configModel.Resources)-len(planResourceMap),
		"plan_only", len(planResourceMap),
	)

	// Build merged model with configuration as the foundation
	mergedModel := &core.WorkloadModel{
		Resources:  mergedResources,
		Framework:  configModel.Framework,
		SourceType: "hcl_enhanced", // Indicates HCL with plan enhancement
		Metadata:   mergedMetadata,
	}

	return mergedModel, nil
}

// ExtractResources extracts resources from a workload model
func (a *Analyzer) ExtractResources(ctx context.Context, model *core.WorkloadModel) ([]core.Resource, error) {
	if model == nil {
		return nil, fmt.Errorf("workload model is nil")
	}

	slog.InfoContext(ctx, "extracting resources from workload model",
		"resource_count", len(model.Resources),
		"framework", model.Framework,
		"source_type", model.SourceType,
	)

	// Simply return the resources from the model
	// This method exists to provide a consistent interface
	// and could be extended in the future to filter or transform resources
	return model.Resources, nil
}

// IdentifyRelationships identifies relationships between resources
// Builds a dependency graph by parsing resource references
func (a *Analyzer) IdentifyRelationships(ctx context.Context, resources []core.Resource) (*core.ResourceGraph, error) {
	slog.InfoContext(ctx, "identifying resource relationships",
		"resource_count", len(resources),
	)

	// Create the resource graph
	graph := &core.ResourceGraph{
		Nodes: make(map[string]*core.Resource),
		Edges: make(map[string][]string),
	}

	// Build nodes map
	for i := range resources {
		graph.Nodes[resources[i].Address] = &resources[i]
	}

	// Identify relationships by analyzing resource properties
	for i := range resources {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resource := &resources[i]
		dependencies := []string{}

		// Parse properties to find references to other resources
		deps := findResourceReferences(resource.Properties, graph.Nodes)
		dependencies = append(dependencies, deps...)

		// Also check explicit dependencies if they exist
		if len(resource.Dependencies) > 0 {
			for _, dep := range resource.Dependencies {
				// Verify the dependency exists in our graph
				if _, exists := graph.Nodes[dep]; exists {
					if !contains(dependencies, dep) {
						dependencies = append(dependencies, dep)
					}
				}
			}
		}

		// Update the resource's dependencies
		resource.Dependencies = dependencies

		// Add edges to the graph
		if len(dependencies) > 0 {
			graph.Edges[resource.Address] = dependencies

			slog.DebugContext(ctx, "identified resource dependencies",
				"resource", resource.Address,
				"dependencies", dependencies,
			)
		}
	}

	slog.InfoContext(ctx, "relationship identification complete",
		"total_nodes", len(graph.Nodes),
		"total_edges", len(graph.Edges),
	)

	return graph, nil
}

// findResourceReferences recursively searches for resource references in properties
func findResourceReferences(properties map[string]interface{}, nodes map[string]*core.Resource) []string {
	var references []string

	for _, value := range properties {
		refs := extractReferencesFromValue(value, nodes)
		for _, ref := range refs {
			if !contains(references, ref) {
				references = append(references, ref)
			}
		}
	}

	return references
}

// extractReferencesFromValue extracts resource references from a value
func extractReferencesFromValue(value interface{}, nodes map[string]*core.Resource) []string {
	var references []string

	switch v := value.(type) {
	case string:
		// Look for Terraform-style references
		// Examples: aws_s3_bucket.example.id, module.vpc.vpc_id, data.aws_ami.ubuntu.id
		refs := parseStringForReferences(v, nodes)
		references = append(references, refs...)

	case map[string]interface{}:
		// Recursively search nested maps
		refs := findResourceReferences(v, nodes)
		references = append(references, refs...)

	case []interface{}:
		// Recursively search arrays
		for _, item := range v {
			refs := extractReferencesFromValue(item, nodes)
			references = append(references, refs...)
		}
	}

	return references
}

// parseStringForReferences parses a string for Terraform resource references
func parseStringForReferences(s string, nodes map[string]*core.Resource) []string {
	var references []string

	// Common Terraform reference patterns:
	// - resource_type.resource_name
	// - module.module_name.resource_type.resource_name
	// - data.resource_type.resource_name

	// Split by common delimiters to find potential references
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '"' || r == '\'' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}'
	})

	for _, word := range words {
		// Check if this looks like a resource reference
		if strings.Contains(word, ".") {
			// Try to match against known resources
			for address := range nodes {
				// Check if the word contains or matches the resource address
				if strings.Contains(word, address) {
					if !contains(references, address) {
						references = append(references, address)
					}
					break
				}

				// Also check for partial matches (e.g., "aws_s3_bucket.example" in "aws_s3_bucket.example.id")
				parts := strings.Split(word, ".")
				if len(parts) >= 2 {
					// Try to construct a resource address from the first two parts
					potentialAddress := parts[0] + "." + parts[1]
					if potentialAddress == address {
						if !contains(references, address) {
							references = append(references, address)
						}
						break
					}

					// Handle data sources (data.type.name)
					if len(parts) >= 3 && parts[0] == "data" {
						potentialAddress = "data." + parts[1] + "." + parts[2]
						if potentialAddress == address {
							if !contains(references, address) {
								references = append(references, address)
							}
							break
						}
					}

					// Handle module resources (module.name.type.name)
					if len(parts) >= 4 && parts[0] == "module" {
						potentialAddress = "module." + parts[1] + "." + parts[2] + "." + parts[3]
						if potentialAddress == address {
							if !contains(references, address) {
								references = append(references, address)
							}
							break
						}
					}
				}
			}
		}
	}

	return references
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TerraformPlan represents the structure of a Terraform plan JSON
type TerraformPlan struct {
	FormatVersion    string        `json:"format_version"`
	TerraformVersion string        `json:"terraform_version"`
	PlannedValues    PlannedValues `json:"planned_values"`
	Configuration    Configuration `json:"configuration"`
}

// PlannedValues contains the planned state
type PlannedValues struct {
	RootModule *Module `json:"root_module"`
}

// Module represents a Terraform module in the plan
type Module struct {
	Resources    []PlanResource `json:"resources"`
	ChildModules []ChildModule  `json:"child_modules"`
}

// ChildModule represents a child module with its address
type ChildModule struct {
	Address   string      `json:"address"`
	Resources []PlanResource `json:"resources"`
	ChildModules []ChildModule `json:"child_modules"`
}

// PlanResource represents a resource in the Terraform plan
type PlanResource struct {
	Address      string                 `json:"address"`
	Mode         string                 `json:"mode"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	ProviderName string                 `json:"provider_name"`
	Values       map[string]interface{} `json:"values"`
}

// Configuration represents the Terraform configuration
type Configuration struct {
	RootModule ConfigModule `json:"root_module"`
}

// ConfigModule represents a module in the configuration
type ConfigModule struct {
	ModuleCalls map[string]ModuleCall `json:"module_calls"`
}

// ModuleCall represents a module call
type ModuleCall struct {
	Source string `json:"source"`
}

// extractResourcesFromModuleWithRedaction recursively extracts resources from a module with redaction
func (a *Analyzer) extractResourcesFromModuleWithRedaction(ctx context.Context, module *Module, parentPath string) []core.Resource {
	var resources []core.Resource

	// Extract resources from this module
	for _, planRes := range module.Resources {
		// Build the full address
		address := planRes.Address
		if parentPath != "" && !strings.HasPrefix(address, parentPath) {
			address = parentPath + "." + address
		}

		// Extract module path from address
		modulePath := ""
		if strings.Contains(address, "module.") {
			parts := strings.Split(address, ".")
			for i, part := range parts {
				if part == "module" && i+1 < len(parts) {
					if modulePath != "" {
						modulePath += "."
					}
					modulePath += "module." + parts[i+1]
					break
				}
			}
		}

		// Redact sensitive data from properties
		redactedProperties, findings := a.redactor.RedactProperties(planRes.Values)
		
		if len(findings) > 0 {
			slog.WarnContext(ctx, "sensitive data redacted from resource properties",
				"resource", address,
				"findings", findings,
			)
		}

		resource := core.Resource{
			ID:           address,
			Type:         planRes.Type,
			Address:      address,
			Properties:   redactedProperties,
			Dependencies: []string{}, // Will be populated in relationship identification
			IsFromPlan:   true,
			ModulePath:   modulePath,
		}

		resources = append(resources, resource)

		slog.DebugContext(ctx, "extracted resource from plan",
			"address", address,
			"type", planRes.Type,
			"module_path", modulePath,
			"redacted", len(findings) > 0,
		)
	}

	// Recursively extract from child modules
	for _, childModule := range module.ChildModules {
		childResources := a.extractResourcesFromChildModuleWithRedaction(ctx, &childModule, childModule.Address)
		resources = append(resources, childResources...)
	}

	return resources
}

// extractResourcesFromChildModuleWithRedaction recursively extracts resources from a child module with redaction
func (a *Analyzer) extractResourcesFromChildModuleWithRedaction(ctx context.Context, childModule *ChildModule, modulePath string) []core.Resource {
	var resources []core.Resource

	// Extract resources from this child module
	for _, planRes := range childModule.Resources {
		address := planRes.Address

		// Redact sensitive data from properties
		redactedProperties, findings := a.redactor.RedactProperties(planRes.Values)
		
		if len(findings) > 0 {
			slog.WarnContext(ctx, "sensitive data redacted from resource properties",
				"resource", address,
				"findings", findings,
			)
		}

		resource := core.Resource{
			ID:           address,
			Type:         planRes.Type,
			Address:      address,
			Properties:   redactedProperties,
			Dependencies: []string{},
			IsFromPlan:   true,
			ModulePath:   modulePath,
		}

		resources = append(resources, resource)

		slog.DebugContext(ctx, "extracted resource from child module",
			"address", address,
			"type", planRes.Type,
			"module_path", modulePath,
			"redacted", len(findings) > 0,
		)
	}

	// Recursively extract from nested child modules
	for _, nestedChild := range childModule.ChildModules {
		nestedResources := a.extractResourcesFromChildModuleWithRedaction(ctx, &nestedChild, nestedChild.Address)
		resources = append(resources, nestedResources...)
	}

	return resources
}

// extractResourcesFromModule recursively extracts resources from a module (kept for backward compatibility)
func extractResourcesFromModule(ctx context.Context, module *Module, parentPath string) []core.Resource {
	var resources []core.Resource

	// Extract resources from this module
	for _, planRes := range module.Resources {
		// Build the full address
		address := planRes.Address
		if parentPath != "" && !strings.HasPrefix(address, parentPath) {
			address = parentPath + "." + address
		}

		// Extract module path from address
		modulePath := ""
		if strings.Contains(address, "module.") {
			parts := strings.Split(address, ".")
			for i, part := range parts {
				if part == "module" && i+1 < len(parts) {
					if modulePath != "" {
						modulePath += "."
					}
					modulePath += "module." + parts[i+1]
					break
				}
			}
		}

		resource := core.Resource{
			ID:           address,
			Type:         planRes.Type,
			Address:      address,
			Properties:   planRes.Values,
			Dependencies: []string{}, // Will be populated in relationship identification
			IsFromPlan:   true,
			ModulePath:   modulePath,
		}

		resources = append(resources, resource)

		slog.DebugContext(ctx, "extracted resource from plan",
			"address", address,
			"type", planRes.Type,
			"module_path", modulePath,
		)
	}

	// Recursively extract from child modules
	for _, childModule := range module.ChildModules {
		childResources := extractResourcesFromChildModule(ctx, &childModule, childModule.Address)
		resources = append(resources, childResources...)
	}

	return resources
}

// extractResourcesFromChildModule recursively extracts resources from a child module
func extractResourcesFromChildModule(ctx context.Context, childModule *ChildModule, modulePath string) []core.Resource {
	var resources []core.Resource

	// Extract resources from this child module
	for _, planRes := range childModule.Resources {
		address := planRes.Address

		resource := core.Resource{
			ID:           address,
			Type:         planRes.Type,
			Address:      address,
			Properties:   planRes.Values,
			Dependencies: []string{},
			IsFromPlan:   true,
			ModulePath:   modulePath,
		}

		resources = append(resources, resource)

		slog.DebugContext(ctx, "extracted resource from child module",
			"address", address,
			"type", planRes.Type,
			"module_path", modulePath,
		)
	}

	// Recursively extract from nested child modules
	for _, nestedChild := range childModule.ChildModules {
		nestedResources := extractResourcesFromChildModule(ctx, &nestedChild, nestedChild.Address)
		resources = append(resources, nestedResources...)
	}

	return resources
}

// HCLResource represents a resource block in HCL
type HCLResource struct {
	Type   string
	Name   string
	Config hcl.Body
	Range  hcl.Range
}

// HCLData represents a data source block in HCL
type HCLData struct {
	Type   string
	Name   string
	Config hcl.Body
	Range  hcl.Range
}

// HCLModule represents a module block in HCL
type HCLModule struct {
	Name   string
	Config hcl.Body
	Range  hcl.Range
}
