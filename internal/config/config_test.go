package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify Bedrock defaults
	assert.Equal(t, "us-east-1", cfg.Bedrock.Region)
	assert.Equal(t, "us.anthropic.claude-sonnet-4-20250514-v1:0", cfg.Bedrock.ModelID)
	assert.Equal(t, 3, cfg.Bedrock.MaxRetries)
	assert.Equal(t, 60, cfg.Bedrock.Timeout)
	assert.Equal(t, 4096, cfg.Bedrock.MaxTokens)
	assert.Equal(t, 0.7, cfg.Bedrock.Temperature)

	// Verify Storage defaults
	assert.Contains(t, cfg.Storage.SessionDir, ".waffle/sessions")
	assert.Contains(t, cfg.Storage.LogDir, ".waffle/logs")
	assert.Equal(t, 90, cfg.Storage.RetentionDays)

	// Verify IaC defaults
	assert.Equal(t, "terraform", cfg.IaC.Framework)
	assert.Equal(t, 10, cfg.IaC.MaxFileSizeMB)
	assert.Equal(t, 10000, cfg.IaC.MaxFiles)

	// Verify WAFR defaults
	assert.Equal(t, "workload", cfg.WAFR.DefaultScope)
	assert.Equal(t, "wellarchitected", cfg.WAFR.DefaultLens)

	// Verify Logging defaults
	assert.Equal(t, "INFO", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)

	// Verify Security defaults
	assert.True(t, cfg.Security.RedactSensitiveData)
	assert.True(t, cfg.Security.EncryptSessions)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "missing bedrock region",
			modify: func(c *Config) {
				c.Bedrock.Region = ""
			},
			wantErr: true,
			errMsg:  "bedrock.region is required",
		},
		{
			name: "missing bedrock model_id",
			modify: func(c *Config) {
				c.Bedrock.ModelID = ""
			},
			wantErr: true,
			errMsg:  "bedrock.model_id is required",
		},
		{
			name: "negative max_retries",
			modify: func(c *Config) {
				c.Bedrock.MaxRetries = -1
			},
			wantErr: true,
			errMsg:  "bedrock.max_retries must be non-negative",
		},
		{
			name: "invalid timeout",
			modify: func(c *Config) {
				c.Bedrock.Timeout = 0
			},
			wantErr: true,
			errMsg:  "bedrock.timeout must be positive",
		},
		{
			name: "invalid temperature",
			modify: func(c *Config) {
				c.Bedrock.Temperature = 1.5
			},
			wantErr: true,
			errMsg:  "bedrock.temperature must be between 0 and 1",
		},
		{
			name: "missing session_dir",
			modify: func(c *Config) {
				c.Storage.SessionDir = ""
			},
			wantErr: true,
			errMsg:  "storage.session_dir is required",
		},
		{
			name: "invalid retention_days",
			modify: func(c *Config) {
				c.Storage.RetentionDays = -1
			},
			wantErr: true,
			errMsg:  "storage.retention_days must be non-negative",
		},
		{
			name: "missing framework",
			modify: func(c *Config) {
				c.IaC.Framework = ""
			},
			wantErr: true,
			errMsg:  "iac.framework is required",
		},
		{
			name: "invalid log level",
			modify: func(c *Config) {
				c.Logging.Level = "INVALID"
			},
			wantErr: true,
			errMsg:  "logging.level must be one of",
		},
		{
			name: "invalid log format",
			modify: func(c *Config) {
				c.Logging.Format = "xml"
			},
			wantErr: true,
			errMsg:  "logging.format must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	
	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create .waffle directory
	waffleDir := filepath.Join(tmpDir, ".waffle")
	err := os.MkdirAll(waffleDir, 0755)
	require.NoError(t, err)

	// Test loading with no config file (should use defaults)
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", cfg.Bedrock.Region)

	// Create a config file
	configContent := `
bedrock:
  region: us-west-2
  model_id: custom-model
  max_retries: 5
  timeout: 120
  max_tokens: 8192
  temperature: 0.5

storage:
  session_dir: /custom/sessions
  log_dir: /custom/logs
  retention_days: 30

iac:
  framework: terraform
  max_file_size_mb: 20
  max_files: 5000

wafr:
  default_scope: pillar
  default_lens: serverless

logging:
  level: DEBUG
  format: text

security:
  redact_sensitive_data: false
  encrypt_sessions: false

aws:
  profile: test-profile
  region: eu-west-1
`
	configPath := filepath.Join(waffleDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config from file
	cfg, err = Load()
	require.NoError(t, err)

	// Verify loaded values
	assert.Equal(t, "us-west-2", cfg.Bedrock.Region)
	assert.Equal(t, "custom-model", cfg.Bedrock.ModelID)
	assert.Equal(t, 5, cfg.Bedrock.MaxRetries)
	assert.Equal(t, 120, cfg.Bedrock.Timeout)
	assert.Equal(t, 8192, cfg.Bedrock.MaxTokens)
	assert.Equal(t, 0.5, cfg.Bedrock.Temperature)

	assert.Equal(t, "/custom/sessions", cfg.Storage.SessionDir)
	assert.Equal(t, "/custom/logs", cfg.Storage.LogDir)
	assert.Equal(t, 30, cfg.Storage.RetentionDays)

	assert.Equal(t, "terraform", cfg.IaC.Framework)
	assert.Equal(t, 20, cfg.IaC.MaxFileSizeMB)
	assert.Equal(t, 5000, cfg.IaC.MaxFiles)

	assert.Equal(t, "pillar", cfg.WAFR.DefaultScope)
	assert.Equal(t, "serverless", cfg.WAFR.DefaultLens)

	assert.Equal(t, "DEBUG", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)

	assert.False(t, cfg.Security.RedactSensitiveData)
	assert.False(t, cfg.Security.EncryptSessions)

	assert.Equal(t, "test-profile", cfg.AWS.Profile)
	assert.Equal(t, "eu-west-1", cfg.AWS.Region)
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	
	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Set environment variables
	os.Setenv("AWS_PROFILE", "env-profile")
	defer os.Unsetenv("AWS_PROFILE")

	os.Setenv("AWS_REGION", "ap-southeast-1")
	defer os.Unsetenv("AWS_REGION")

	os.Setenv("WAFFLE_LOG_LEVEL", "ERROR")
	defer os.Unsetenv("WAFFLE_LOG_LEVEL")

	// Load config
	cfg, err := Load()
	require.NoError(t, err)

	// Verify environment variables override defaults
	assert.Equal(t, "env-profile", cfg.AWS.Profile)
	assert.Equal(t, "ap-southeast-1", cfg.AWS.Region)
	assert.Equal(t, "ERROR", cfg.Logging.Level)
	// Bedrock region should also be overridden by AWS_REGION
	assert.Equal(t, "ap-southeast-1", cfg.Bedrock.Region)
}

func TestSaveConfig(t *testing.T) {
	// Create a temporary directory for test config
	tmpDir := t.TempDir()
	
	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a config
	cfg := DefaultConfig()
	cfg.Bedrock.Region = "eu-central-1"
	cfg.Bedrock.MaxRetries = 10
	cfg.AWS.Profile = "saved-profile"

	// Save config
	err := Save(cfg)
	require.NoError(t, err)

	// Verify file was created
	configPath := filepath.Join(tmpDir, ".waffle", "config.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load config back
	loadedCfg, err := Load()
	require.NoError(t, err)

	// Verify values match
	assert.Equal(t, "eu-central-1", loadedCfg.Bedrock.Region)
	assert.Equal(t, 10, loadedCfg.Bedrock.MaxRetries)
	assert.Equal(t, "saved-profile", loadedCfg.AWS.Profile)
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "path with tilde",
			input:    "~/.waffle/sessions",
			contains: ".waffle/sessions",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			contains: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			contains: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Contains(t, result, tt.contains)
		})
	}
}
