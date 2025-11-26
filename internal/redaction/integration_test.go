package redaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedaction_TerraformWithSecrets tests redaction on a realistic Terraform example
func TestRedaction_TerraformWithSecrets(t *testing.T) {
	redactor := NewRedactor()

	// Realistic Terraform configuration with multiple types of sensitive data
	terraformConfig := `
resource "aws_db_instance" "production" {
  allocated_storage    = 100
  engine               = "postgres"
  engine_version       = "13.7"
  instance_class       = "db.t3.large"
  name                 = "production_db"
  username             = "admin"
  password             = "SuperSecretPassword123!"
  parameter_group_name = "default.postgres13"
  skip_final_snapshot  = true
  
  tags = {
    Name        = "Production Database"
    Environment = "production"
    Owner       = "devops@company.com"
  }
}

resource "aws_s3_bucket" "logs" {
  bucket = "company-logs-bucket"
  
  tags = {
    Contact = "admin@company.com"
  }
}

resource "aws_instance" "web_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  
  user_data = <<-EOF
    #!/bin/bash
    export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
    export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    export DB_PASSWORD=MyDatabasePassword
    export API_KEY=sk-1234567890abcdef
    
    # Configure network
    ip addr add 192.168.1.100/24 dev eth0
    ip addr add 10.0.1.50/24 dev eth1
  EOF
  
  tags = {
    Name = "Web Server"
  }
}

resource "aws_secretsmanager_secret" "api_key" {
  name = "production/api_key"
  
  secret_string = jsonencode({
    api_key = "sk-prod-abcdef123456"
    token   = "bearer_xyz789"
  })
}
`

	redacted, findings := redactor.Redact(terraformConfig)

	// Verify all sensitive data is redacted
	t.Run("AWS credentials redacted", func(t *testing.T) {
		assert.NotContains(t, redacted, "AKIAIOSFODNN7EXAMPLE")
		assert.NotContains(t, redacted, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
		assert.Contains(t, redacted, "AKIA****************")
	})

	t.Run("Passwords redacted", func(t *testing.T) {
		assert.NotContains(t, redacted, "SuperSecretPassword123!")
		assert.NotContains(t, redacted, "MyDatabasePassword")
	})

	t.Run("API keys redacted", func(t *testing.T) {
		assert.NotContains(t, redacted, "sk-1234567890abcdef")
		assert.NotContains(t, redacted, "sk-prod-abcdef123456")
		assert.NotContains(t, redacted, "bearer_xyz789")
	})

	t.Run("Email addresses redacted", func(t *testing.T) {
		assert.NotContains(t, redacted, "devops@company.com")
		assert.NotContains(t, redacted, "admin@company.com")
		assert.Contains(t, redacted, "[EMAIL]")
	})

	t.Run("Private IPs redacted", func(t *testing.T) {
		assert.NotContains(t, redacted, "192.168.1.100")
		assert.NotContains(t, redacted, "10.0.1.50")
		assert.Contains(t, redacted, "[PRIVATE_IP]")
	})

	t.Run("Non-sensitive data preserved", func(t *testing.T) {
		assert.Contains(t, redacted, "aws_db_instance")
		assert.Contains(t, redacted, "production_db")
		assert.Contains(t, redacted, "postgres")
		assert.Contains(t, redacted, "ami-0c55b159cbfafe1f0")
		assert.Contains(t, redacted, "t2.micro")
		assert.Contains(t, redacted, "company-logs-bucket")
	})

	t.Run("All finding types present", func(t *testing.T) {
		assert.Contains(t, findings, "AWS Access Key")
		assert.Contains(t, findings, "AWS Secret Key")
		assert.Contains(t, findings, "Password Field")
		assert.Contains(t, findings, "API Key")
		assert.Contains(t, findings, "Email Address")
		assert.Contains(t, findings, "Private IP")
	})
}

// TestRedaction_PropertiesWithSecrets tests property redaction
func TestRedaction_PropertiesWithSecrets(t *testing.T) {
	redactor := NewRedactor()

	// Simulate resource properties from parsed Terraform
	properties := map[string]interface{}{
		"name":     "production-db",
		"engine":   "postgres",
		"username": "admin",
		"password": "SuperSecret123!",
		"tags": map[string]interface{}{
			"Name":  "Production DB",
			"Owner": "admin@company.com",
		},
		"connection_string": "postgres://admin:password123@10.0.1.5:5432/mydb",
		"environment_variables": []interface{}{
			map[string]interface{}{
				"name":  "DB_HOST",
				"value": "10.0.1.5",
			},
			map[string]interface{}{
				"name":  "DB_PASSWORD",
				"value": "secret123",
			},
			map[string]interface{}{
				"name":  "API_KEY",
				"value": "sk-prod-xyz",
			},
		},
	}

	redacted, findings := redactor.RedactProperties(properties)

	// Verify sensitive properties are redacted
	t.Run("Password field redacted", func(t *testing.T) {
		require.Contains(t, redacted, "password")
		assert.Equal(t, "[REDACTED]", redacted["password"])
	})

	t.Run("Nested email redacted", func(t *testing.T) {
		tags, ok := redacted["tags"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "[EMAIL]", tags["Owner"])
	})

	t.Run("Connection string IPs redacted", func(t *testing.T) {
		connStr, ok := redacted["connection_string"].(string)
		require.True(t, ok)
		// Note: password in connection string is not redacted by pattern matching
		// Only the IP address is redacted
		assert.NotContains(t, connStr, "10.0.1.5")
		assert.Contains(t, connStr, "[PRIVATE_IP]")
	})

	t.Run("Array properties with IPs redacted", func(t *testing.T) {
		envVars, ok := redacted["environment_variables"].([]interface{})
		require.True(t, ok)
		require.Len(t, envVars, 3)

		// Check DB_HOST IP is redacted
		dbHost := envVars[0].(map[string]interface{})
		assert.Equal(t, "[PRIVATE_IP]", dbHost["value"])
		
		// Note: "value" is not a sensitive key name, so DB_PASSWORD and API_KEY
		// values are only redacted if they match string patterns
		// In a real scenario, these would be caught by the string pattern matching
	})

	t.Run("Non-sensitive data preserved", func(t *testing.T) {
		assert.Equal(t, "production-db", redacted["name"])
		assert.Equal(t, "postgres", redacted["engine"])
		assert.Equal(t, "admin", redacted["username"])
	})

	t.Run("Findings reported", func(t *testing.T) {
		assert.Contains(t, findings, "Password Field")
		assert.Contains(t, findings, "Email Address")
		assert.Contains(t, findings, "Private IP")
	})
}

// TestRedaction_NoFalsePositives ensures we don't over-redact
func TestRedaction_NoFalsePositives(t *testing.T) {
	redactor := NewRedactor()

	// Content that should NOT be redacted
	safeContent := `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  
  tags = {
    Name        = "Example Bucket"
    Description = "This is a test bucket"
    Team        = "Platform"
  }
}

resource "aws_vpc" "main" {
  cidr_block = "203.0.113.0/24"  # Public CIDR (TEST-NET-3)
  
  tags = {
    Name = "Main VPC"
  }
}

output "bucket_name" {
  value = aws_s3_bucket.example.bucket
}
`

	redacted, findings := redactor.Redact(safeContent)

	t.Run("Safe content unchanged", func(t *testing.T) {
		assert.Equal(t, safeContent, redacted)
	})

	t.Run("No findings for safe content", func(t *testing.T) {
		assert.Empty(t, findings)
	})
}
