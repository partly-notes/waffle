package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the complete Waffle configuration
type Config struct {
	Bedrock  BedrockConfig  `mapstructure:"bedrock"`
	Storage  StorageConfig  `mapstructure:"storage"`
	IaC      IaCConfig      `mapstructure:"iac"`
	WAFR     WAFRConfig     `mapstructure:"wafr"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
	AWS      AWSConfig      `mapstructure:"aws"`
}

// BedrockConfig contains Bedrock-specific configuration
type BedrockConfig struct {
	Region      string  `mapstructure:"region"`
	ModelID     string  `mapstructure:"model_id"`
	MaxRetries  int     `mapstructure:"max_retries"`
	Timeout     int     `mapstructure:"timeout"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

// StorageConfig contains storage-related configuration
type StorageConfig struct {
	SessionDir    string `mapstructure:"session_dir"`
	LogDir        string `mapstructure:"log_dir"`
	RetentionDays int    `mapstructure:"retention_days"`
}

// IaCConfig contains IaC analysis configuration
type IaCConfig struct {
	Framework      string `mapstructure:"framework"`
	MaxFileSizeMB  int    `mapstructure:"max_file_size_mb"`
	MaxFiles       int    `mapstructure:"max_files"`
}

// WAFRConfig contains WAFR-specific configuration
type WAFRConfig struct {
	DefaultScope string `mapstructure:"default_scope"`
	DefaultLens  string `mapstructure:"default_lens"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	RedactSensitiveData bool `mapstructure:"redact_sensitive_data"`
	EncryptSessions     bool `mapstructure:"encrypt_sessions"`
}

// AWSConfig contains AWS-specific configuration
type AWSConfig struct {
	Profile string `mapstructure:"profile"`
	Region  string `mapstructure:"region"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	waffleDir := filepath.Join(homeDir, ".waffle")

	return &Config{
		Bedrock: BedrockConfig{
			Region:      "us-east-1",
			ModelID:     "us.anthropic.claude-sonnet-4-20250514-v1:0",
			MaxRetries:  3,
			Timeout:     60,
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		Storage: StorageConfig{
			SessionDir:    filepath.Join(waffleDir, "sessions"),
			LogDir:        filepath.Join(waffleDir, "logs"),
			RetentionDays: 90,
		},
		IaC: IaCConfig{
			Framework:     "terraform",
			MaxFileSizeMB: 10,
			MaxFiles:      10000,
		},
		WAFR: WAFRConfig{
			DefaultScope: "workload",
			DefaultLens:  "wellarchitected",
		},
		Logging: LoggingConfig{
			Level:  "INFO",
			Format: "json",
		},
		Security: SecurityConfig{
			RedactSensitiveData: true,
			EncryptSessions:     true,
		},
		AWS: AWSConfig{
			Profile: "",
			Region:  "",
		},
	}
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Set up viper
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add config paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	waffleDir := filepath.Join(homeDir, ".waffle")
	v.AddConfigPath(waffleDir)
	v.AddConfigPath(".")

	// Set environment variable prefix
	v.SetEnvPrefix("WAFFLE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with environment variables if set
	if awsProfile := os.Getenv("AWS_PROFILE"); awsProfile != "" {
		cfg.AWS.Profile = awsProfile
	}
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		cfg.AWS.Region = awsRegion
		// Also set Bedrock region if AWS_REGION is set
		if cfg.Bedrock.Region == "us-east-1" {
			cfg.Bedrock.Region = awsRegion
		}
	}
	if logLevel := os.Getenv("WAFFLE_LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	// Expand home directory in paths
	cfg.Storage.SessionDir = expandPath(cfg.Storage.SessionDir)
	cfg.Storage.LogDir = expandPath(cfg.Storage.LogDir)

	return cfg, nil
}

// Save saves the configuration to the config file
func Save(cfg *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	waffleDir := filepath.Join(homeDir, ".waffle")
	if err := os.MkdirAll(waffleDir, 0755); err != nil {
		return fmt.Errorf("failed to create waffle directory: %w", err)
	}

	configPath := filepath.Join(waffleDir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set all config values with proper structure
	v.Set("bedrock.region", cfg.Bedrock.Region)
	v.Set("bedrock.model_id", cfg.Bedrock.ModelID)
	v.Set("bedrock.max_retries", cfg.Bedrock.MaxRetries)
	v.Set("bedrock.timeout", cfg.Bedrock.Timeout)
	v.Set("bedrock.max_tokens", cfg.Bedrock.MaxTokens)
	v.Set("bedrock.temperature", cfg.Bedrock.Temperature)

	v.Set("storage.session_dir", cfg.Storage.SessionDir)
	v.Set("storage.log_dir", cfg.Storage.LogDir)
	v.Set("storage.retention_days", cfg.Storage.RetentionDays)

	v.Set("iac.framework", cfg.IaC.Framework)
	v.Set("iac.max_file_size_mb", cfg.IaC.MaxFileSizeMB)
	v.Set("iac.max_files", cfg.IaC.MaxFiles)

	v.Set("wafr.default_scope", cfg.WAFR.DefaultScope)
	v.Set("wafr.default_lens", cfg.WAFR.DefaultLens)

	v.Set("logging.level", cfg.Logging.Level)
	v.Set("logging.format", cfg.Logging.Format)

	v.Set("security.redact_sensitive_data", cfg.Security.RedactSensitiveData)
	v.Set("security.encrypt_sessions", cfg.Security.EncryptSessions)

	v.Set("aws.profile", cfg.AWS.Profile)
	v.Set("aws.region", cfg.AWS.Region)

	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Bedrock config
	if c.Bedrock.Region == "" {
		return fmt.Errorf("bedrock.region is required")
	}
	if c.Bedrock.ModelID == "" {
		return fmt.Errorf("bedrock.model_id is required")
	}
	if c.Bedrock.MaxRetries < 0 {
		return fmt.Errorf("bedrock.max_retries must be non-negative")
	}
	if c.Bedrock.Timeout <= 0 {
		return fmt.Errorf("bedrock.timeout must be positive")
	}
	if c.Bedrock.MaxTokens <= 0 {
		return fmt.Errorf("bedrock.max_tokens must be positive")
	}
	if c.Bedrock.Temperature < 0 || c.Bedrock.Temperature > 1 {
		return fmt.Errorf("bedrock.temperature must be between 0 and 1")
	}

	// Validate Storage config
	if c.Storage.SessionDir == "" {
		return fmt.Errorf("storage.session_dir is required")
	}
	if c.Storage.LogDir == "" {
		return fmt.Errorf("storage.log_dir is required")
	}
	if c.Storage.RetentionDays < 0 {
		return fmt.Errorf("storage.retention_days must be non-negative")
	}

	// Validate IaC config
	if c.IaC.Framework == "" {
		return fmt.Errorf("iac.framework is required")
	}
	if c.IaC.MaxFileSizeMB <= 0 {
		return fmt.Errorf("iac.max_file_size_mb must be positive")
	}
	if c.IaC.MaxFiles <= 0 {
		return fmt.Errorf("iac.max_files must be positive")
	}

	// Validate WAFR config
	if c.WAFR.DefaultScope == "" {
		return fmt.Errorf("wafr.default_scope is required")
	}
	if c.WAFR.DefaultLens == "" {
		return fmt.Errorf("wafr.default_lens is required")
	}

	// Validate Logging config
	validLevels := map[string]bool{
		"DEBUG": true, "INFO": true, "WARNING": true, "WARN": true, "ERROR": true,
	}
	if !validLevels[strings.ToUpper(c.Logging.Level)] {
		return fmt.Errorf("logging.level must be one of: DEBUG, INFO, WARNING, ERROR")
	}

	validFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validFormats[strings.ToLower(c.Logging.Format)] {
		return fmt.Errorf("logging.format must be one of: json, text")
	}

	return nil
}

// expandPath expands ~ to home directory in paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
