package valheim

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type renderedRunnerConfig struct {
	Version string `json:"version"`
	Game    struct {
		Name     string             `json:"name"`
		Settings map[string]any     `json:"settings"`
		Mods     *renderedModConfig `json:"mods,omitempty"`
	} `json:"game"`
	Server struct {
		ServerRoot  string `json:"server_root"`
		SaveRoot    string `json:"save_root"`
		ModpackRoot string `json:"modpack_root"`
	} `json:"server"`
}

type renderedModConfig struct {
	Sources []struct {
		Provider string `json:"provider"`
		Code     string `json:"code,omitempty"`
	} `json:"sources"`
}

func extractRunnerConfigJSON(t *testing.T, command string) renderedRunnerConfig {
	t.Helper()

	const startMarker = "cat > \"$CONFIG_PATH\" <<RUNNER_CONFIG_EOF\n"
	start := strings.Index(command, startMarker)
	require.NotEqual(t, -1, start, "runner config heredoc not found")
	start += len(startMarker)

	end := strings.Index(command[start:], "\nRUNNER_CONFIG_EOF")
	require.NotEqual(t, -1, end, "runner config terminator not found")

	jsonBlob := command[start : start+end]
	var cfg renderedRunnerConfig
	require.NoError(t, json.Unmarshal([]byte(jsonBlob), &cfg))
	return cfg
}

func TestBuildInitCommand_VanillaValheim(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":    "DedicatedWorld",
		"password": "secret",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `RUNNER_VERSION="0.1.0"`)
	assert.Contains(t, command, `RUNNER_URL="https://github.com/openhost/OpenHost/releases/download/runner-v0.1.0/openhost-runner-linux-amd64"`)
	assert.Contains(t, command, `RUNNER_BIN="${OPENHOST_RUNNER_BIN:-/usr/local/bin/openhost-runner}"`)
	assert.Contains(t, command, `CONFIG_PATH="${OPENHOST_RUNNER_CONFIG_PATH:-/tmp/openhost-runner-config.json}"`)
	assert.Contains(t, command, `OPENHOST_VALHEIM_LOCAL_DEBUG="${OPENHOST_VALHEIM_LOCAL_DEBUG:-false}"`)
	assert.Contains(t, command, `SERVER_ROOT="${OPENHOST_VALHEIM_SERVER_ROOT:-/home/valheim/server}"`)
	assert.Contains(t, command, `if [ -x "$RUNNER_BIN" ]; then`)
	assert.Contains(t, command, `command -v curl >/dev/null 2>&1 || { apt-get update -y && apt-get install -y curl; }`)
	assert.Contains(t, command, `args=(--config "$CONFIG_PATH")`)
	assert.Contains(t, command, `exec "$RUNNER_BIN" "${args[@]}"`)

	cfg := extractRunnerConfigJSON(t, command)
	assert.Equal(t, "1", cfg.Version)
	assert.Equal(t, "valheim", cfg.Game.Name)
	assert.Equal(t, "DedicatedWorld", cfg.Game.Settings["world"])
	assert.Equal(t, "secret", cfg.Game.Settings["password"])
	assert.Nil(t, cfg.Game.Mods)
	assert.Equal(t, "${SERVER_ROOT}", cfg.Server.ServerRoot)
	assert.Equal(t, "${SAVE_ROOT}", cfg.Server.SaveRoot)
	assert.Equal(t, "${MODPACK_ROOT}", cfg.Server.ModpackRoot)
}

func TestBuildInitCommand_ThunderstoreModpack(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":             "DedicatedWorld",
		"password":          "secret",
		"thunderstore_code": "ABC123_code",
	})
	require.NoError(t, err)

	cfg := extractRunnerConfigJSON(t, command)
	require.NotNil(t, cfg.Game.Mods)
	require.Len(t, cfg.Game.Mods.Sources, 1)
	assert.Equal(t, "thunderstore", cfg.Game.Mods.Sources[0].Provider)
	assert.Equal(t, "ABC123_code", cfg.Game.Mods.Sources[0].Code)
}

func TestBuildInitCommand_UUIDThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":             "DedicatedWorld",
		"password":          "secret",
		"thunderstore_code": "019cf113-4729-c139-63ac-ea85dafcffd6",
	})
	require.NoError(t, err)

	cfg := extractRunnerConfigJSON(t, command)
	require.NotNil(t, cfg.Game.Mods)
	assert.Equal(t, "019cf113-4729-c139-63ac-ea85dafcffd6", cfg.Game.Mods.Sources[0].Code)
}

func TestBuildInitCommand_InvalidThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "bad code with spaces",
	})
	assert.Error(t, err)
	assert.Empty(t, command)
	assert.Contains(t, err.Error(), "thunderstore_code")
}

func TestBuildInitCommand_BlankThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "   ",
	})
	assert.Error(t, err)
	assert.Empty(t, command)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestBuildInitCommand_DefaultWorldWithMods(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "Code123",
	})
	require.NoError(t, err)

	cfg := extractRunnerConfigJSON(t, command)
	assert.Equal(t, "Dedicated", cfg.Game.Settings["world"])
}

func TestBuildInitCommand_RunnerOverrides(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":       "secret",
		"runner_version": "1.2.3-test",
		"runner_url":     "https://example.invalid/openhost-runner-linux-amd64",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `RUNNER_VERSION="1.2.3-test"`)
	assert.Contains(t, command, `RUNNER_URL="https://example.invalid/openhost-runner-linux-amd64"`)
}
