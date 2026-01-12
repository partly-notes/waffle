package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigWithOverrides(t *testing.T) {
	tests := []struct {
		name                string
		regionFlag          string
		profileFlag         string
		expectedBedrockRegion string
		expectedAWSRegion   string
		wantErr             bool
	}{
		{
			name:                "no overrides uses defaults",
			regionFlag:          "",
			profileFlag:         "",
			expectedBedrockRegion: "us-east-1", // default Bedrock region
			expectedAWSRegion:   "",            // AWS region is empty by default
			wantErr:             false,
		},
		{
			name:                "region flag overrides config",
			regionFlag:          "eu-west-1",
			profileFlag:         "",
			expectedBedrockRegion: "eu-west-1",
			expectedAWSRegion:   "eu-west-1",
			wantErr:             false,
		},
		{
			name:                "profile flag is applied",
			regionFlag:          "",
			profileFlag:         "test-profile",
			expectedBedrockRegion: "us-east-1", // default Bedrock region
			expectedAWSRegion:   "",            // AWS region is empty by default
			wantErr:             false,
		},
		{
			name:                "both flags are applied",
			regionFlag:          "ap-southeast-1",
			profileFlag:         "prod-profile",
			expectedBedrockRegion: "ap-southeast-1",
			expectedAWSRegion:   "ap-southeast-1",
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command with flags
			cmd := &cobra.Command{}
			cmd.Flags().String("region", "", "AWS region")
			cmd.Flags().String("profile", "", "AWS profile")

			// Set flag values
			if tt.regionFlag != "" {
				cmd.Flags().Set("region", tt.regionFlag)
			}
			if tt.profileFlag != "" {
				cmd.Flags().Set("profile", tt.profileFlag)
			}

			// Load config with overrides
			cfg, err := loadConfigWithOverrides(cmd)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAWSRegion, cfg.AWS.Region)
			assert.Equal(t, tt.expectedBedrockRegion, cfg.Bedrock.Region)

			if tt.profileFlag != "" {
				assert.Equal(t, tt.profileFlag, cfg.AWS.Profile)
			}
		})
	}
}

func TestRegionFlagOverridesBothBedrockAndAWS(t *testing.T) {
	// Create a test command with region flag
	cmd := &cobra.Command{}
	cmd.Flags().String("region", "", "AWS region")
	cmd.Flags().String("profile", "", "AWS profile")

	// Set region flag
	cmd.Flags().Set("region", "us-west-2")

	// Load config
	cfg, err := loadConfigWithOverrides(cmd)
	require.NoError(t, err)

	// Verify both AWS and Bedrock regions are set
	assert.Equal(t, "us-west-2", cfg.AWS.Region, "AWS region should be overridden")
	assert.Equal(t, "us-west-2", cfg.Bedrock.Region, "Bedrock region should be overridden")
}

func TestPersistentFlagsAvailableOnAllCommands(t *testing.T) {
	// Test that region and profile flags are available on all commands
	commands := []*cobra.Command{
		reviewCmd,
		statusCmd,
		resultsCmd,
		initCmd,
	}

	for _, cmd := range commands {
		t.Run(cmd.Name(), func(t *testing.T) {
			// Check that persistent flags are inherited
			regionFlag := cmd.Flag("region")
			profileFlag := cmd.Flag("profile")

			// Flags should be available (either local or inherited from parent)
			assert.NotNil(t, regionFlag, "region flag should be available on %s command", cmd.Name())
			assert.NotNil(t, profileFlag, "profile flag should be available on %s command", cmd.Name())
		})
	}
}
