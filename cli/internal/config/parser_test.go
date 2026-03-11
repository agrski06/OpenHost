package config_test

import (
	"os"
	"testing"

	"github.com/openhost/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseYAML_ValidConfig(t *testing.T) {
	// Create a temporary valid YAML file
	content := `
name: "test-server"
provider:
  name: "hetzner"
  region: "fsn1"
  plan: "cx11"
game:
  type: "minecraft"
  image: "itzg/minecraft-server"
  persistence: "10GB"
automation:
  auto-stop-timeout: "1h"
  trigger: "no-players"
`
	filePath := "test_valid.yaml"
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	cfg, err := config.ParseYAML(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "test-server", cfg.Name)
	assert.Equal(t, "hetzner", cfg.Provider.Name)
	assert.Equal(t, "fsn1", cfg.Provider.Region)
	assert.Equal(t, "cx11", cfg.Provider.Plan)
	assert.Equal(t, "minecraft", cfg.Game.Type)
	assert.Equal(t, "itzg/minecraft-server", cfg.Game.Image)
	assert.Equal(t, "10GB", cfg.Game.Persistence)
	assert.Equal(t, "1h", cfg.Automation.AutoStopTimeout)
	assert.Equal(t, "no-players", cfg.Automation.Trigger)
}

func TestParseYAML_Defaults(t *testing.T) {
	// Create a temporary YAML file with missing optional fields
	content := `
name: "test-server-defaults"
provider:
  name: "hetzner"
  region: "nbg1"
  plan: "cx21"
game:
  type: "valheim"
  image: "lloesche/valheim-server"
`
	filePath := "test_defaults.yaml"
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	cfg, err := config.ParseYAML(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "none", cfg.Game.Persistence)
	assert.Equal(t, "disabled", cfg.Automation.AutoStopTimeout)
}

func TestParseYAML_InvalidConfig(t *testing.T) {
	// Test cases for invalid configurations
	tests := []struct {
		name        string
		content     string
		expectedErr string
	}{
		{
			name: "missing server name",
			content: `
provider:
  name: "hetzner"
  region: "fsn1"
  plan: "cx11"
game:
  type: "minecraft"
  image: "itzg/minecraft-server"
`,
			expectedErr: "config validation failed: server name cannot be empty",
		},
		{
			name: "missing provider name",
			content: `
name: "test-server"
provider:
  region: "fsn1"
  plan: "cx11"
game:
  type: "minecraft"
  image: "itzg/minecraft-server"
`,
			expectedErr: "config validation failed: provider name cannot be empty",
		},
		{
			name: "invalid auto-stop-timeout",
			content: `
name: "test-server"
provider:
  name: "hetzner"
  region: "fsn1"
  plan: "cx11"
game:
  type: "minecraft"
  image: "itzg/minecraft-server"
automation:
  auto-stop-timeout: "invalid-duration"
`,
			expectedErr: "config validation failed: invalid auto-stop-timeout format: time: invalid duration \"invalid-duration\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := "test_invalid_" + tt.name + ".yaml"
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			assert.NoError(t, err)
			defer os.Remove(filePath)

			cfg, err := config.ParseYAML(filePath)
			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestParseYAML_NonExistentFile(t *testing.T) {
	cfg, err := config.ParseYAML("non_existent_file.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read config file")
}
