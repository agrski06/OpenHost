package config_test

import (
	"os"
	"testing"

	"github.com/openhost/cli/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseYAML_ValidConfig(t *testing.T) {
	content := `
server:
  name: "test-server"
provider:
  name: "hetzner"
  settings:
    location: "fsn1"
    plan: "cx11"
game:
  name: "minecraft"
  settings:
    difficulty: "hard"
`
	filePath := "test_valid.yaml"
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	cfg, err := config.ParseYAML(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "test-server", cfg.Server.Name)
	assert.Equal(t, "hetzner", cfg.Provider.Name)
	assert.Equal(t, "fsn1", cfg.Provider.Settings["location"])
	assert.Equal(t, "cx11", cfg.Provider.Settings["plan"])
	assert.Equal(t, "minecraft", cfg.Game.Name)
	assert.Equal(t, "hard", cfg.Game.Settings["difficulty"])
}

func TestParseYAML_InitializesMissingSettingsMaps(t *testing.T) {
	content := `
server:
  name: "test-server-defaults"
provider:
  name: "hetzner"
game:
  name: "valheim"
`
	filePath := "test_defaults.yaml"
	err := os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	cfg, err := config.ParseYAML(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.NotNil(t, cfg.Provider.Settings)
	assert.NotNil(t, cfg.Game.Settings)
}

func TestParseYAML_InvalidConfig(t *testing.T) {
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
  settings:
    location: "fsn1"
    plan: "cx11"
game:
  name: "minecraft"
`,
			expectedErr: "config validation failed: server name cannot be empty",
		},
		{
			name: "top-level name is rejected",
			content: `
name: "test-server"
provider:
  name: "hetzner"
  settings:
    location: "fsn1"
    plan: "cx11"
game:
  name: "minecraft"
`,
			expectedErr: "invalid config shape: top-level name is not supported; use server.name",
		},
		{
			name: "flat provider plan is rejected",
			content: `
server:
  name: "test-server"
provider:
  name: "hetzner"
  plan: "cx11"
game:
  name: "minecraft"
`,
			expectedErr: "invalid config shape: provider.plan is not supported; use provider.settings.plan",
		},
		{
			name: "game type alias is rejected",
			content: `
server:
  name: "test-server"
provider:
  name: "hetzner"
game:
  type: "minecraft"
`,
			expectedErr: "invalid config shape: game.type is not supported; use game.name",
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
