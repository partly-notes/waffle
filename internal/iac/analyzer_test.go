package iac

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waffle/waffle/internal/core"
)

func TestRetrieveIaCFiles_Success(t *testing.T) {
	// Create a temporary directory with Terraform files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"main.tf": `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		"variables.tf": `variable "region" {
  type = string
  default = "us-east-1"
}`,
		"outputs.tf": `output "bucket_name" {
  value = aws_s3_bucket.example.bucket
}`,
		"terraform.tfvars": `region = "us-west-2"`,
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create analyzer with temp directory
	analyzer := NewAnalyzerWithDir(tmpDir)

	// Retrieve files
	files, err := analyzer.RetrieveIaCFiles(context.Background())

	// Assertions
	require.NoError(t, err)
	assert.Len(t, files, 4)

	// Verify all files were retrieved
	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f.Path] = f.Content
	}

	for filename, expectedContent := range testFiles {
		content, exists := fileMap[filename]
		assert.True(t, exists, "file %s should be retrieved", filename)
		assert.Equal(t, expectedContent, content, "content should match for %s", filename)
	}
}

func TestRetrieveIaCFiles_WithSubdirectories(t *testing.T) {
	// Create a temporary directory with nested structure
	tmpDir := t.TempDir()

	// Create subdirectories
	modulesDir := filepath.Join(tmpDir, "modules", "vpc")
	err := os.MkdirAll(modulesDir, 0755)
	require.NoError(t, err)

	// Create files in root
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("resource \"aws_vpc\" \"main\" {}"), 0644)
	require.NoError(t, err)

	// Create files in subdirectory
	err = os.WriteFile(filepath.Join(modulesDir, "vpc.tf"), []byte("resource \"aws_subnet\" \"private\" {}"), 0644)
	require.NoError(t, err)

	// Create analyzer
	analyzer := NewAnalyzerWithDir(tmpDir)

	// Retrieve files
	files, err := analyzer.RetrieveIaCFiles(context.Background())

	// Assertions
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Verify paths are relative
	for _, f := range files {
		assert.NotContains(t, f.Path, tmpDir, "path should be relative")
	}
}

func TestRetrieveIaCFiles_DirectoryNotExist(t *testing.T) {
	analyzer := NewAnalyzerWithDir("/nonexistent/directory")

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.Error(t, err)
	assert.Nil(t, files)

	var dirErr *core.DirectoryAccessError
	assert.True(t, errors.As(err, &dirErr))
	assert.Contains(t, dirErr.Error(), "does not exist")
}

func TestRetrieveIaCFiles_PathIsFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.tf")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	analyzer := NewAnalyzerWithDir(tmpFile.Name())

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.Error(t, err)
	assert.Nil(t, files)

	var dirErr *core.DirectoryAccessError
	assert.True(t, errors.As(err, &dirErr))
	assert.Contains(t, dirErr.Error(), "not a directory")
}

func TestRetrieveIaCFiles_NoIaCFiles(t *testing.T) {
	// Create a temporary directory with no Terraform files
	tmpDir := t.TempDir()

	// Create non-Terraform files
	err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte("key: value"), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzerWithDir(tmpDir)

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.Error(t, err)
	assert.Nil(t, files)

	var dirErr *core.DirectoryAccessError
	assert.True(t, errors.As(err, &dirErr))
	assert.Contains(t, dirErr.Error(), "no IaC files found")
}

func TestRetrieveIaCFiles_SkipsHiddenDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden directory with Terraform files
	hiddenDir := filepath.Join(tmpDir, ".terraform")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(hiddenDir, "hidden.tf"), []byte("resource \"aws_s3_bucket\" \"hidden\" {}"), 0644)
	require.NoError(t, err)

	// Create visible file
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("resource \"aws_s3_bucket\" \"main\" {}"), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzerWithDir(tmpDir)

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "main.tf", files[0].Path)
}

func TestRetrieveIaCFiles_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules directory
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	err := os.MkdirAll(nodeModulesDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(nodeModulesDir, "module.tf"), []byte("resource \"aws_s3_bucket\" \"module\" {}"), 0644)
	require.NoError(t, err)

	// Create visible file
	err = os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("resource \"aws_s3_bucket\" \"main\" {}"), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzerWithDir(tmpDir)

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "main.tf", files[0].Path)
}

func TestRetrieveIaCFiles_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte("resource \"aws_s3_bucket\" \"main\" {}"), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzerWithDir(tmpDir)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files, err := analyzer.RetrieveIaCFiles(ctx)

	require.Error(t, err)
	assert.Nil(t, files)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetrieveIaCFiles_OnlyTerraformFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various file types
	testFiles := map[string]bool{
		"main.tf":          true,  // Should be included
		"variables.tf":     true,  // Should be included
		"terraform.tfvars": true,  // Should be included
		"README.md":        false, // Should be excluded
		"config.yaml":      false, // Should be excluded
		"script.sh":        false, // Should be excluded
		"data.json":        false, // Should be excluded
	}

	for filename := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte("content"), 0644)
		require.NoError(t, err)
	}

	analyzer := NewAnalyzerWithDir(tmpDir)

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.NoError(t, err)
	assert.Len(t, files, 3, "should only retrieve .tf and .tfvars files")

	// Verify only Terraform files were retrieved
	for _, f := range files {
		shouldInclude, exists := testFiles[f.Path]
		assert.True(t, exists)
		assert.True(t, shouldInclude, "file %s should not be included", f.Path)
	}
}

func TestIsTerraformFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "terraform file",
			path:     "main.tf",
			expected: true,
		},
		{
			name:     "terraform vars file",
			path:     "terraform.tfvars",
			expected: true,
		},
		{
			name:     "uppercase extension",
			path:     "main.TF",
			expected: true,
		},
		{
			name:     "markdown file",
			path:     "README.md",
			expected: false,
		},
		{
			name:     "yaml file",
			path:     "config.yaml",
			expected: false,
		},
		{
			name:     "no extension",
			path:     "Makefile",
			expected: false,
		},
		{
			name:     "terraform in path but not extension",
			path:     "terraform/main.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTerraformFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetrieveIaCFiles_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Create a file with no read permissions
	restrictedFile := filepath.Join(tmpDir, "restricted.tf")
	err := os.WriteFile(restrictedFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Remove read permissions
	err = os.Chmod(restrictedFile, 0000)
	require.NoError(t, err)
	defer os.Chmod(restrictedFile, 0644) // Restore for cleanup

	analyzer := NewAnalyzerWithDir(tmpDir)

	files, err := analyzer.RetrieveIaCFiles(context.Background())

	require.Error(t, err)
	assert.Nil(t, files)

	var dirErr *core.DirectoryAccessError
	assert.True(t, errors.As(err, &dirErr))
	assert.Contains(t, dirErr.Error(), "permission denied")
}

func TestValidateTerraformFiles_Success(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  
  tags = {
    Name = "My bucket"
  }
}`,
		},
		{
			Path: "variables.tf",
			Content: `variable "region" {
  type    = string
  default = "us-east-1"
}

variable "environment" {
  type = string
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.NoError(t, err)
}

func TestValidateTerraformFiles_InvalidSyntax(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "invalid.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  // Missing closing brace
`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.Error(t, err)
	
	var syntaxErr *core.TerraformSyntaxError
	assert.True(t, errors.As(err, &syntaxErr))
	assert.Equal(t, "invalid.tf", syntaxErr.File)
	assert.NotEmpty(t, syntaxErr.Message)
}

func TestValidateTerraformFiles_MissingQuote(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "bad.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.Error(t, err)
	
	var syntaxErr *core.TerraformSyntaxError
	assert.True(t, errors.As(err, &syntaxErr))
	assert.Equal(t, "bad.tf", syntaxErr.File)
}

func TestValidateTerraformFiles_InvalidResourceBlock(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "invalid_resource.tf",
			Content: `resource "aws_s3_bucket" {
  bucket = "my-bucket"
  invalid syntax here @#$
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.Error(t, err)
	
	var syntaxErr *core.TerraformSyntaxError
	assert.True(t, errors.As(err, &syntaxErr))
}

func TestValidateTerraformFiles_EmptyFileList(t *testing.T) {
	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), []core.IaCFile{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files provided")
}

func TestValidateTerraformFiles_NonTerraformFile(t *testing.T) {
	files := []core.IaCFile{
		{
			Path:    "README.md",
			Content: "# This is a readme",
		},
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	// Should succeed, skipping non-Terraform files
	require.NoError(t, err)
}

func TestValidateTerraformFiles_MultipleErrors(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "error1.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  // Missing closing brace
`,
		},
		{
			Path: "error2.tf",
			Content: `resource {
  bucket = "another-bucket"
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.Error(t, err)
	
	var syntaxErr *core.TerraformSyntaxError
	assert.True(t, errors.As(err, &syntaxErr))
	// Should return the first error encountered
	assert.NotEmpty(t, syntaxErr.File)
}

func TestValidateTerraformFiles_ComplexValidSyntax(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "complex.tf",
			Content: `resource "aws_instance" "web" {
  ami           = "ami-abc123"
  instance_type = "t2.micro"

  tags = {
    Name = "web-server"
    Environment = var.environment
  }

  ebs_block_device {
    device_name = "/dev/sda1"
    volume_size = 20
    volume_type = "gp3"
  }

  lifecycle {
    create_before_destroy = true
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }

  owners = ["099720109477"]
}

module "vpc" {
  source = "./modules/vpc"
  
  cidr_block = "10.0.0.0/16"
  azs        = ["us-east-1a", "us-east-1b"]
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.NoError(t, err)
}

func TestValidateTerraformFiles_ContextCancellation(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		},
	}

	analyzer := NewAnalyzer()
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := analyzer.ValidateTerraformFiles(ctx, files)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestValidateTerraformFiles_WithTfvars(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "terraform.tfvars",
			Content: `region = "us-west-2"
environment = "production"
instance_count = 3`,
		},
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		},
	}

	analyzer := NewAnalyzer()
	err := analyzer.ValidateTerraformFiles(context.Background(), files)

	require.NoError(t, err)
}

func TestParseTerraformPlan_Success(t *testing.T) {
	// Create a temporary plan file
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")

	planContent := `{
  "format_version": "1.2",
  "terraform_version": "1.5.0",
  "planned_values": {
    "root_module": {
      "resources": [
        {
          "address": "aws_s3_bucket.example",
          "mode": "managed",
          "type": "aws_s3_bucket",
          "name": "example",
          "provider_name": "registry.terraform.io/hashicorp/aws",
          "values": {
            "bucket": "my-test-bucket",
            "tags": {
              "Environment": "test"
            }
          }
        }
      ],
      "child_modules": []
    }
  },
  "configuration": {
    "root_module": {}
  }
}`

	err := os.WriteFile(planFile, []byte(planContent), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraformPlan(context.Background(), planFile)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "terraform", model.Framework)
	assert.Equal(t, "plan", model.SourceType)
	assert.Len(t, model.Resources, 1)

	// Verify resource details
	resource := model.Resources[0]
	assert.Equal(t, "aws_s3_bucket.example", resource.Address)
	assert.Equal(t, "aws_s3_bucket", resource.Type)
	assert.True(t, resource.IsFromPlan)
	assert.Equal(t, "my-test-bucket", resource.Properties["bucket"])
}

func TestParseTerraformPlan_WithModules(t *testing.T) {
	// Create a temporary plan file with modules
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.json")

	planContent := `{
  "format_version": "1.2",
  "terraform_version": "1.5.0",
  "planned_values": {
    "root_module": {
      "resources": [
        {
          "address": "aws_s3_bucket.root",
          "mode": "managed",
          "type": "aws_s3_bucket",
          "name": "root",
          "values": {
            "bucket": "root-bucket"
          }
        }
      ],
      "child_modules": [
        {
          "address": "module.vpc",
          "resources": [
            {
              "address": "module.vpc.aws_vpc.main",
              "mode": "managed",
              "type": "aws_vpc",
              "name": "main",
              "values": {
                "cidr_block": "10.0.0.0/16"
              }
            }
          ],
          "child_modules": []
        }
      ]
    }
  },
  "configuration": {
    "root_module": {}
  }
}`

	err := os.WriteFile(planFile, []byte(planContent), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraformPlan(context.Background(), planFile)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Len(t, model.Resources, 2)

	// Verify root resource
	var rootResource *core.Resource
	var moduleResource *core.Resource
	for i := range model.Resources {
		if model.Resources[i].Address == "aws_s3_bucket.root" {
			rootResource = &model.Resources[i]
		}
		if model.Resources[i].Address == "module.vpc.aws_vpc.main" {
			moduleResource = &model.Resources[i]
		}
	}

	require.NotNil(t, rootResource)
	assert.Equal(t, "", rootResource.ModulePath)

	require.NotNil(t, moduleResource)
	assert.Equal(t, "module.vpc", moduleResource.ModulePath)
	assert.Equal(t, "aws_vpc", moduleResource.Type)
}

func TestParseTerraformPlan_FileNotExist(t *testing.T) {
	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraformPlan(context.Background(), "/nonexistent/plan.json")

	require.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestParseTerraformPlan_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(planFile, []byte("not valid json {{{"), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraformPlan(context.Background(), planFile)

	require.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "failed to parse IaC file")
}

func TestParseTerraformPlan_EmptyPlan(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "empty.json")

	planContent := `{
  "format_version": "1.2",
  "terraform_version": "1.5.0",
  "planned_values": {
    "root_module": {
      "resources": [],
      "child_modules": []
    }
  },
  "configuration": {
    "root_module": {}
  }
}`

	err := os.WriteFile(planFile, []byte(planContent), 0644)
	require.NoError(t, err)

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraformPlan(context.Background(), planFile)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Len(t, model.Resources, 0)
}

func TestParseTerraform_Success(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  
  tags = {
    Name = "My bucket"
    Environment = "test"
  }
}

resource "aws_s3_bucket_versioning" "example" {
  bucket = aws_s3_bucket.example.id
  
  versioning_configuration {
    status = "Enabled"
  }
}`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "terraform", model.Framework)
	assert.Equal(t, "hcl", model.SourceType)
	assert.Len(t, model.Resources, 2)

	// Verify first resource
	resource := model.Resources[0]
	assert.Equal(t, "aws_s3_bucket.example", resource.Address)
	assert.Equal(t, "aws_s3_bucket", resource.Type)
	assert.False(t, resource.IsFromPlan)
	assert.Equal(t, "main.tf", resource.SourceFile)
	assert.Greater(t, resource.SourceLine, 0)
}

func TestParseTerraform_WithDataSources(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "data.tf",
			Content: `data "aws_ami" "ubuntu" {
  most_recent = true
  
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"]
  }
  
  owners = ["099720109477"]
}

resource "aws_instance" "web" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t2.micro"
}`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Len(t, model.Resources, 2)

	// Find data source
	var dataSource *core.Resource
	for i := range model.Resources {
		if strings.HasPrefix(model.Resources[i].Address, "data.") {
			dataSource = &model.Resources[i]
			break
		}
	}

	require.NotNil(t, dataSource)
	assert.Equal(t, "data.aws_ami.ubuntu", dataSource.Address)
	assert.Equal(t, "aws_ami", dataSource.Type)
}

func TestParseTerraform_WithModules(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `module "vpc" {
  source = "./modules/vpc"
  
  cidr_block = "10.0.0.0/16"
  azs        = ["us-east-1a", "us-east-1b"]
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  subnet_id     = module.vpc.private_subnet_id
}`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.NoError(t, err)
	require.NotNil(t, model)
	// Module calls are not included as resources, only actual resources
	assert.Len(t, model.Resources, 1)
	assert.Equal(t, "aws_instance.web", model.Resources[0].Address)
}

func TestParseTerraform_EmptyFileList(t *testing.T) {
	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), []core.IaCFile{})

	require.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "no files provided")
}

func TestParseTerraform_InvalidHCL(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "invalid.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  // Missing closing brace
`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.Error(t, err)
	assert.Nil(t, model)
	
	var syntaxErr *core.TerraformSyntaxError
	assert.True(t, errors.As(err, &syntaxErr))
}

func TestParseTerraform_MultipleFiles(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		},
		{
			Path: "vpc.tf",
			Content: `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}`,
		},
		{
			Path: "variables.tf",
			Content: `variable "region" {
  type    = string
  default = "us-east-1"
}`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Len(t, model.Resources, 2)
	
	// Verify resources from different files
	addresses := make(map[string]bool)
	for _, res := range model.Resources {
		addresses[res.Address] = true
	}
	
	assert.True(t, addresses["aws_s3_bucket.example"])
	assert.True(t, addresses["aws_vpc.main"])
}

func TestParseTerraform_ComplexNestedBlocks(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "complex.tf",
			Content: `resource "aws_instance" "web" {
  ami           = "ami-abc123"
  instance_type = "t2.micro"

  ebs_block_device {
    device_name = "/dev/sda1"
    volume_size = 20
    volume_type = "gp3"
  }

  lifecycle {
    create_before_destroy = true
  }
}`,
		},
	}

	analyzer := NewAnalyzer()
	model, err := analyzer.ParseTerraform(context.Background(), files)

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Len(t, model.Resources, 1)

	resource := model.Resources[0]
	assert.Equal(t, "aws_instance", resource.Type)
	assert.NotNil(t, resource.Properties)
	
	// Verify nested blocks are captured
	assert.Contains(t, resource.Properties, "ebs_block_device")
	assert.Contains(t, resource.Properties, "lifecycle")
}

func TestParseTerraform_ContextCancellation(t *testing.T) {
	files := []core.IaCFile{
		{
			Path: "main.tf",
			Content: `resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}`,
		},
	}

	analyzer := NewAnalyzer()
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model, err := analyzer.ParseTerraform(ctx, files)

	require.Error(t, err)
	assert.Nil(t, model)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMergeWorkloadModels_BothModels(t *testing.T) {
	planModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:         "aws_s3_bucket.example",
				Type:       "aws_s3_bucket",
				Address:    "aws_s3_bucket.example",
				Properties: map[string]interface{}{"bucket": "my-bucket", "versioning": map[string]interface{}{"enabled": true}},
				IsFromPlan: true,
			},
			{
				ID:         "aws_vpc.main",
				Type:       "aws_vpc",
				Address:    "aws_vpc.main",
				Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16"},
				IsFromPlan: true,
			},
		},
		Framework:  "terraform",
		SourceType: "plan",
		Metadata:   map[string]interface{}{"format_version": "1.2"},
	}

	sourceModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:         "aws_s3_bucket.example",
				Type:       "aws_s3_bucket",
				Address:    "aws_s3_bucket.example",
				Properties: map[string]interface{}{"bucket": "my-bucket"},
				SourceFile: "main.tf",
				SourceLine: 10,
				IsFromPlan: false,
			},
			{
				ID:         "aws_instance.web",
				Type:       "aws_instance",
				Address:    "aws_instance.web",
				Properties: map[string]interface{}{"ami": "ami-12345"},
				SourceFile: "compute.tf",
				SourceLine: 5,
				IsFromPlan: false,
			},
		},
		Framework:  "terraform",
		SourceType: "hcl",
		Metadata:   map[string]interface{}{"file_count": 2},
	}

	analyzer := NewAnalyzer()
	merged, err := analyzer.MergeWorkloadModels(context.Background(), planModel, sourceModel)

	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, "merged", merged.SourceType)
	assert.Len(t, merged.Resources, 3) // 2 from plan + 1 source-only

	// Verify plan resources are prioritized
	var s3Resource *core.Resource
	var vpcResource *core.Resource
	var instanceResource *core.Resource

	for i := range merged.Resources {
		switch merged.Resources[i].Address {
		case "aws_s3_bucket.example":
			s3Resource = &merged.Resources[i]
		case "aws_vpc.main":
			vpcResource = &merged.Resources[i]
		case "aws_instance.web":
			instanceResource = &merged.Resources[i]
		}
	}

	// S3 bucket should have plan data with HCL context
	require.NotNil(t, s3Resource)
	assert.True(t, s3Resource.IsFromPlan)
	assert.Equal(t, "main.tf", s3Resource.SourceFile)
	assert.Equal(t, 10, s3Resource.SourceLine)
	assert.Contains(t, s3Resource.Properties, "versioning") // From plan

	// VPC should be plan-only
	require.NotNil(t, vpcResource)
	assert.True(t, vpcResource.IsFromPlan)

	// Instance should be source-only
	require.NotNil(t, instanceResource)
	assert.False(t, instanceResource.IsFromPlan)
	assert.Equal(t, "compute.tf", instanceResource.SourceFile)
}

func TestMergeWorkloadModels_PlanOnly(t *testing.T) {
	planModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:         "aws_s3_bucket.example",
				Type:       "aws_s3_bucket",
				Address:    "aws_s3_bucket.example",
				Properties: map[string]interface{}{"bucket": "my-bucket"},
				IsFromPlan: true,
			},
		},
		Framework:  "terraform",
		SourceType: "plan",
		Metadata:   map[string]interface{}{"format_version": "1.2"},
	}

	analyzer := NewAnalyzer()
	merged, err := analyzer.MergeWorkloadModels(context.Background(), planModel, nil)

	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, planModel, merged) // Should return plan model as-is
}

func TestMergeWorkloadModels_SourceOnly(t *testing.T) {
	sourceModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:         "aws_s3_bucket.example",
				Type:       "aws_s3_bucket",
				Address:    "aws_s3_bucket.example",
				Properties: map[string]interface{}{"bucket": "my-bucket"},
				SourceFile: "main.tf",
				SourceLine: 10,
				IsFromPlan: false,
			},
		},
		Framework:  "terraform",
		SourceType: "hcl",
		Metadata:   map[string]interface{}{"file_count": 1},
	}

	analyzer := NewAnalyzer()
	merged, err := analyzer.MergeWorkloadModels(context.Background(), nil, sourceModel)

	require.NoError(t, err)
	require.NotNil(t, merged)
	assert.Equal(t, sourceModel, merged) // Should return source model as-is
}

func TestMergeWorkloadModels_BothNil(t *testing.T) {
	analyzer := NewAnalyzer()
	merged, err := analyzer.MergeWorkloadModels(context.Background(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, merged)
	assert.Contains(t, err.Error(), "both plan and source models are nil")
}

func TestMergeWorkloadModels_ContextCancellation(t *testing.T) {
	planModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{Address: "aws_s3_bucket.example", Type: "aws_s3_bucket"},
		},
		Framework:  "terraform",
		SourceType: "plan",
	}

	sourceModel := &core.WorkloadModel{
		Resources: []core.Resource{
			{Address: "aws_s3_bucket.example", Type: "aws_s3_bucket"},
		},
		Framework:  "terraform",
		SourceType: "hcl",
	}

	analyzer := NewAnalyzer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	merged, err := analyzer.MergeWorkloadModels(ctx, planModel, sourceModel)

	require.Error(t, err)
	assert.Nil(t, merged)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExtractResources_Success(t *testing.T) {
	model := &core.WorkloadModel{
		Resources: []core.Resource{
			{
				ID:      "aws_s3_bucket.example",
				Type:    "aws_s3_bucket",
				Address: "aws_s3_bucket.example",
			},
			{
				ID:      "aws_vpc.main",
				Type:    "aws_vpc",
				Address: "aws_vpc.main",
			},
		},
		Framework:  "terraform",
		SourceType: "plan",
	}

	analyzer := NewAnalyzer()
	resources, err := analyzer.ExtractResources(context.Background(), model)

	require.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, model.Resources, resources)
}

func TestExtractResources_NilModel(t *testing.T) {
	analyzer := NewAnalyzer()
	resources, err := analyzer.ExtractResources(context.Background(), nil)

	require.Error(t, err)
	assert.Nil(t, resources)
	assert.Contains(t, err.Error(), "workload model is nil")
}

func TestIdentifyRelationships_SimpleReferences(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example",
			Properties: map[string]interface{}{
				"bucket": "my-bucket",
			},
		},
		{
			ID:      "aws_s3_bucket_versioning.example",
			Type:    "aws_s3_bucket_versioning",
			Address: "aws_s3_bucket_versioning.example",
			Properties: map[string]interface{}{
				"bucket": "aws_s3_bucket.example.id",
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 2)

	// Verify versioning resource depends on bucket
	versioningDeps := graph.Edges["aws_s3_bucket_versioning.example"]
	assert.Contains(t, versioningDeps, "aws_s3_bucket.example")
}

func TestIdentifyRelationships_ModuleReferences(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "module.vpc.aws_vpc.main",
			Type:    "aws_vpc",
			Address: "module.vpc.aws_vpc.main",
			Properties: map[string]interface{}{
				"cidr_block": "10.0.0.0/16",
			},
		},
		{
			ID:      "aws_instance.web",
			Type:    "aws_instance",
			Address: "aws_instance.web",
			Properties: map[string]interface{}{
				"ami":       "ami-12345",
				"subnet_id": "module.vpc.aws_vpc.main.id",
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify instance depends on VPC
	instanceDeps := graph.Edges["aws_instance.web"]
	assert.Contains(t, instanceDeps, "module.vpc.aws_vpc.main")
}

func TestIdentifyRelationships_DataSourceReferences(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "data.aws_ami.ubuntu",
			Type:    "aws_ami",
			Address: "data.aws_ami.ubuntu",
			Properties: map[string]interface{}{
				"most_recent": true,
			},
		},
		{
			ID:      "aws_instance.web",
			Type:    "aws_instance",
			Address: "aws_instance.web",
			Properties: map[string]interface{}{
				"ami":           "data.aws_ami.ubuntu.id",
				"instance_type": "t2.micro",
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify instance depends on data source
	instanceDeps := graph.Edges["aws_instance.web"]
	assert.Contains(t, instanceDeps, "data.aws_ami.ubuntu")
}

func TestIdentifyRelationships_NestedProperties(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_kms_key.example",
			Type:    "aws_kms_key",
			Address: "aws_kms_key.example",
			Properties: map[string]interface{}{
				"description": "KMS key",
			},
		},
		{
			ID:      "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example",
			Properties: map[string]interface{}{
				"bucket": "my-bucket",
				"server_side_encryption_configuration": map[string]interface{}{
					"rule": map[string]interface{}{
						"apply_server_side_encryption_by_default": map[string]interface{}{
							"kms_master_key_id": "aws_kms_key.example.arn",
						},
					},
				},
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify bucket depends on KMS key
	bucketDeps := graph.Edges["aws_s3_bucket.example"]
	assert.Contains(t, bucketDeps, "aws_kms_key.example")
}

func TestIdentifyRelationships_ArrayProperties(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_security_group.web",
			Type:    "aws_security_group",
			Address: "aws_security_group.web",
			Properties: map[string]interface{}{
				"name": "web-sg",
			},
		},
		{
			ID:      "aws_instance.web",
			Type:    "aws_instance",
			Address: "aws_instance.web",
			Properties: map[string]interface{}{
				"ami": "ami-12345",
				"vpc_security_group_ids": []interface{}{
					"aws_security_group.web.id",
				},
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify instance depends on security group
	instanceDeps := graph.Edges["aws_instance.web"]
	assert.Contains(t, instanceDeps, "aws_security_group.web")
}

func TestIdentifyRelationships_ExplicitDependencies(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example",
			Properties: map[string]interface{}{
				"bucket": "my-bucket",
			},
		},
		{
			ID:      "aws_s3_bucket_policy.example",
			Type:    "aws_s3_bucket_policy",
			Address: "aws_s3_bucket_policy.example",
			Properties: map[string]interface{}{
				"bucket": "aws_s3_bucket.example.id",
			},
			Dependencies: []string{"aws_s3_bucket.example"},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify policy depends on bucket
	policyDeps := graph.Edges["aws_s3_bucket_policy.example"]
	assert.Contains(t, policyDeps, "aws_s3_bucket.example")
	assert.Len(t, policyDeps, 1) // Should not duplicate
}

func TestIdentifyRelationships_NoDependencies(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_s3_bucket.example1",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example1",
			Properties: map[string]interface{}{
				"bucket": "bucket1",
			},
		},
		{
			ID:      "aws_s3_bucket.example2",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example2",
			Properties: map[string]interface{}{
				"bucket": "bucket2",
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, graph.Edges, 0) // No dependencies
}

func TestIdentifyRelationships_EmptyResources(t *testing.T) {
	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), []core.Resource{})

	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 0)
	assert.Len(t, graph.Edges, 0)
}

func TestIdentifyRelationships_ContextCancellation(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_s3_bucket.example",
			Type:    "aws_s3_bucket",
			Address: "aws_s3_bucket.example",
		},
	}

	analyzer := NewAnalyzer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	graph, err := analyzer.IdentifyRelationships(ctx, resources)

	require.Error(t, err)
	assert.Nil(t, graph)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestIdentifyRelationships_ComplexDependencyChain(t *testing.T) {
	resources := []core.Resource{
		{
			ID:      "aws_vpc.main",
			Type:    "aws_vpc",
			Address: "aws_vpc.main",
			Properties: map[string]interface{}{
				"cidr_block": "10.0.0.0/16",
			},
		},
		{
			ID:      "aws_subnet.private",
			Type:    "aws_subnet",
			Address: "aws_subnet.private",
			Properties: map[string]interface{}{
				"vpc_id":     "aws_vpc.main.id",
				"cidr_block": "10.0.1.0/24",
			},
		},
		{
			ID:      "aws_security_group.web",
			Type:    "aws_security_group",
			Address: "aws_security_group.web",
			Properties: map[string]interface{}{
				"vpc_id": "aws_vpc.main.id",
			},
		},
		{
			ID:      "aws_instance.web",
			Type:    "aws_instance",
			Address: "aws_instance.web",
			Properties: map[string]interface{}{
				"ami":                    "ami-12345",
				"subnet_id":              "aws_subnet.private.id",
				"vpc_security_group_ids": []interface{}{"aws_security_group.web.id"},
			},
		},
	}

	analyzer := NewAnalyzer()
	graph, err := analyzer.IdentifyRelationships(context.Background(), resources)

	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Len(t, graph.Nodes, 4)

	// Verify dependency chain
	subnetDeps := graph.Edges["aws_subnet.private"]
	assert.Contains(t, subnetDeps, "aws_vpc.main")

	sgDeps := graph.Edges["aws_security_group.web"]
	assert.Contains(t, sgDeps, "aws_vpc.main")

	instanceDeps := graph.Edges["aws_instance.web"]
	assert.Contains(t, instanceDeps, "aws_subnet.private")
	assert.Contains(t, instanceDeps, "aws_security_group.web")
}
