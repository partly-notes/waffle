package redaction

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedact_AWSAccessKey(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "AWS access key in plain text",
			input:    "aws_access_key_id = AKIAIOSFODNN7EXAMPLE",
			expected: "aws_access_key_id = AKIA****************",
			findings: []string{"AWS Access Key"},
		},
		{
			name:     "AWS access key in JSON",
			input:    `{"access_key": "AKIAI44QH8DHBEXAMPLE"}`,
			expected: `{"access_key": "AKIA****************"}`,
			findings: []string{"AWS Access Key"},
		},
		{
			name:     "Multiple AWS access keys",
			input:    "key1=AKIAIOSFODNN7EXAMPLE key2=AKIAI44QH8DHBEXAMPLE",
			expected: "key1=AKIA**************** key2=AKIA****************",
			findings: []string{"AWS Access Key"},
		},
		{
			name:     "No AWS access key",
			input:    "some normal text without keys",
			expected: "some normal text without keys",
			findings: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_AWSSecretKey(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "AWS secret key",
			input:    "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expected: "aws_secret_access_key: \"[REDACTED]\"",
			findings: []string{"AWS Secret Key", "Secret Field"},
		},
		{
			name:     "40-character base64 string",
			input:    "secret: abcdefghijklmnopqrstuvwxyz01234567890AB",
			expected: "secret: \"[REDACTED]\"",
			findings: []string{"Secret Field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			// Check that all expected findings are present (order may vary)
			for _, expectedFinding := range tt.findings {
				assert.Contains(t, findings, expectedFinding)
			}
		})
	}
}

func TestRedact_PasswordField(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "password with colon",
			input:    "password: mysecretpass123",
			expected: `password: "[REDACTED]"`,
			findings: []string{"Password Field"},
		},
		{
			name:     "password with equals",
			input:    "password=mysecretpass123",
			expected: `password: "[REDACTED]"`,
			findings: []string{"Password Field"},
		},
		{
			name:     "passwd field",
			input:    "passwd: admin123",
			expected: `passwd: "[REDACTED]"`,
			findings: []string{"Password Field"},
		},
		{
			name:     "pwd field",
			input:    "pwd=test123",
			expected: `pwd: "[REDACTED]"`,
			findings: []string{"Password Field"},
		},
		{
			name:     "case insensitive",
			input:    "PASSWORD: MySecret",
			expected: `PASSWORD: "[REDACTED]"`,
			findings: []string{"Password Field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_APIKey(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "api_key field",
			input:    "api_key: sk-1234567890abcdef",
			expected: `api_key: "[REDACTED]"`,
			findings: []string{"API Key"},
		},
		{
			name:     "apikey field",
			input:    "apikey=myapikey123",
			expected: `apikey: "[REDACTED]"`,
			findings: []string{"API Key"},
		},
		{
			name:     "api-key field",
			input:    "api-key: xyz789",
			expected: `api-key: "[REDACTED]"`,
			findings: []string{"API Key"},
		},
		{
			name:     "token field",
			input:    "token: bearer_token_123",
			expected: `token: "[REDACTED]"`,
			findings: []string{"API Key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_SecretField(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "secret field",
			input:    "secret: my_secret_value",
			expected: `secret: "[REDACTED]"`,
			findings: []string{"Secret Field"},
		},
		{
			name:     "secret_key field",
			input:    "secret_key=abc123",
			expected: `secret_key: "[REDACTED]"`,
			findings: []string{"Secret Field"},
		},
		{
			name:     "secret_access_key field",
			input:    "secret_access_key: xyz789",
			expected: `secret_access_key: "[REDACTED]"`,
			findings: []string{"Secret Field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_EmailAddress(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "simple email",
			input:    "user@example.com",
			expected: "[EMAIL]",
			findings: []string{"Email Address"},
		},
		{
			name:     "email with dots",
			input:    "john.doe@company.co.uk",
			expected: "[EMAIL]",
			findings: []string{"Email Address"},
		},
		{
			name:     "email in text",
			input:    "Contact us at support@example.com for help",
			expected: "Contact us at [EMAIL] for help",
			findings: []string{"Email Address"},
		},
		{
			name:     "multiple emails",
			input:    "admin@test.com and user@test.com",
			expected: "[EMAIL] and [EMAIL]",
			findings: []string{"Email Address"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_PrivateIP(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    string
		expected string
		findings []string
	}{
		{
			name:     "10.x.x.x range",
			input:    "server at 10.0.1.5",
			expected: "server at [PRIVATE_IP]",
			findings: []string{"Private IP"},
		},
		{
			name:     "172.16-31.x.x range",
			input:    "database: 172.16.0.10",
			expected: "database: [PRIVATE_IP]",
			findings: []string{"Private IP"},
		},
		{
			name:     "192.168.x.x range",
			input:    "router: 192.168.1.1",
			expected: "router: [PRIVATE_IP]",
			findings: []string{"Private IP"},
		},
		{
			name:     "multiple private IPs",
			input:    "10.0.0.1 and 192.168.1.1",
			expected: "[PRIVATE_IP] and [PRIVATE_IP]",
			findings: []string{"Private IP"},
		},
		{
			name:     "public IP not redacted",
			input:    "8.8.8.8",
			expected: "8.8.8.8",
			findings: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.Redact(tt.input)
			assert.Equal(t, tt.expected, redacted)
			assert.Equal(t, tt.findings, findings)
		})
	}
}

func TestRedact_MultiplePatterns(t *testing.T) {
	redactor := NewRedactor()

	input := `
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  
  tags = {
    Name  = "example-instance"
    Email = "admin@company.com"
  }
  
  user_data = <<-EOF
    #!/bin/bash
    export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
    export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    export DB_PASSWORD=mysecretpassword
    export API_KEY=sk-1234567890abcdef
    
    # Configure network
    ip addr add 192.168.1.100/24 dev eth0
  EOF
}
`

	redacted, findings := redactor.Redact(input)

	// Verify all sensitive data is redacted
	assert.NotContains(t, redacted, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, redacted, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	assert.NotContains(t, redacted, "mysecretpassword")
	assert.NotContains(t, redacted, "sk-1234567890abcdef")
	assert.NotContains(t, redacted, "admin@company.com")
	assert.NotContains(t, redacted, "192.168.1.100")

	// Verify findings include all matched patterns
	assert.Contains(t, findings, "AWS Access Key")
	assert.Contains(t, findings, "AWS Secret Key")
	assert.Contains(t, findings, "Password Field")
	assert.Contains(t, findings, "API Key")
	assert.Contains(t, findings, "Email Address")
	assert.Contains(t, findings, "Private IP")
}

func TestRedactProperties(t *testing.T) {
	redactor := NewRedactor()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
		findings []string
	}{
		{
			name: "simple properties",
			input: map[string]interface{}{
				"name":     "my-bucket",
				"password": "secret123",
				"email":    "user@example.com",
			},
			expected: map[string]interface{}{
				"name":     "my-bucket",
				"password": "[REDACTED]",
				"email":    "[EMAIL]",
			},
			findings: []string{"Password Field", "Email Address"},
		},
		{
			name: "nested properties",
			input: map[string]interface{}{
				"name": "my-resource",
				"config": map[string]interface{}{
					"api_key": "sk-123456",
					"host":    "10.0.1.5",
				},
			},
			expected: map[string]interface{}{
				"name": "my-resource",
				"config": map[string]interface{}{
					"api_key": "[REDACTED]",
					"host":    "[PRIVATE_IP]",
				},
			},
			findings: []string{"API Key", "Private IP"},
		},
		{
			name: "array properties",
			input: map[string]interface{}{
				"emails": []interface{}{
					"admin@test.com",
					"user@test.com",
				},
				"servers": []interface{}{
					map[string]interface{}{
						"ip":       "192.168.1.1",
						"password": "admin123",
					},
				},
			},
			expected: map[string]interface{}{
				"emails": []interface{}{
					"[EMAIL]",
					"[EMAIL]",
				},
				"servers": []interface{}{
					map[string]interface{}{
						"ip":       "[PRIVATE_IP]",
						"password": "[REDACTED]",
					},
				},
			},
			findings: []string{"Email Address", "Private IP", "Password Field"},
		},
		{
			name:     "nil properties",
			input:    nil,
			expected: nil,
			findings: nil,
		},
		{
			name:     "empty properties",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
			findings: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted, findings := redactor.RedactProperties(tt.input)
			assert.Equal(t, tt.expected, redacted)
			
			// Check that all expected findings are present
			for _, expectedFinding := range tt.findings {
				assert.Contains(t, findings, expectedFinding)
			}
		})
	}
}

func TestNewRedactorWithRules(t *testing.T) {
	customRules := []RedactionRule{
		{
			Name:        "Custom Pattern",
			Pattern:     regexp.MustCompile(`CUSTOM-\d{4}`),
			Replacement: "CUSTOM-****",
		},
	}

	redactor := NewRedactorWithRules(customRules)

	input := "My code is CUSTOM-1234"
	expected := "My code is CUSTOM-****"

	redacted, findings := redactor.Redact(input)

	assert.Equal(t, expected, redacted)
	assert.Equal(t, []string{"Custom Pattern"}, findings)
}

func TestDefaultRedactionRules(t *testing.T) {
	rules := DefaultRedactionRules()

	// Verify we have all expected rules
	require.NotEmpty(t, rules)

	ruleNames := make(map[string]bool)
	for _, rule := range rules {
		ruleNames[rule.Name] = true
	}

	expectedRules := []string{
		"AWS Access Key",
		"AWS Secret Key",
		"Password Field",
		"API Key",
		"Secret Field",
		"Email Address",
		"Private IP",
	}

	for _, expected := range expectedRules {
		assert.True(t, ruleNames[expected], "Expected rule %s not found", expected)
	}
}

func TestRedact_NoSensitiveData(t *testing.T) {
	redactor := NewRedactor()

	input := `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  
  tags = {
    Name        = "My bucket"
    Environment = "Dev"
  }
}
`

	redacted, findings := redactor.Redact(input)

	// Content should remain unchanged
	assert.Equal(t, input, redacted)
	// No findings
	assert.Empty(t, findings)
}

func TestRedact_TerraformExample(t *testing.T) {
	redactor := NewRedactor()

	input := `
resource "aws_db_instance" "default" {
  allocated_storage    = 10
  engine               = "mysql"
  engine_version       = "5.7"
  instance_class       = "db.t3.micro"
  name                 = "mydb"
  username             = "admin"
  password             = "SuperSecret123!"
  parameter_group_name = "default.mysql5.7"
  skip_final_snapshot  = true
  
  tags = {
    Owner = "admin@company.com"
  }
}
`

	redacted, findings := redactor.Redact(input)

	// Verify sensitive data is redacted
	assert.NotContains(t, redacted, "SuperSecret123!")
	assert.NotContains(t, redacted, "admin@company.com")

	// Verify findings
	assert.Contains(t, findings, "Password Field")
	assert.Contains(t, findings, "Email Address")

	// Verify non-sensitive data remains
	assert.Contains(t, redacted, "aws_db_instance")
	assert.Contains(t, redacted, "mysql")
	assert.Contains(t, redacted, "admin")
}
