package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveStateDir_UsesOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-openhost-state")
	t.Setenv(stateDirEnvVar, override)

	path, err := resolveStateDir()
	require.NoError(t, err)
	assert.Equal(t, override, path)
}

func TestResolveDefaultStatePath_UsesOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-openhost-state")
	t.Setenv(stateDirEnvVar, override)

	path, err := resolveDefaultStatePath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(override, "instances.json"), path)
}

func TestStateDirForGOOS_LinuxUsesXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state-home"))

	path, err := stateDirForGOOS("linux")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(os.Getenv("XDG_STATE_HOME"), "openhost"), path)
}

func TestStateDirForGOOS_WindowsUsesLocalAppData(t *testing.T) {
	localAppData := filepath.Join(t.TempDir(), "LocalAppData")
	t.Setenv("LOCALAPPDATA", localAppData)

	path, err := stateDirForGOOS("windows")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(localAppData, "OpenHost"), path)
}

func TestStateDirForGOOS_DarwinUsesApplicationSupport(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	path, err := stateDirForGOOS("darwin")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(homeDir, "Library", "Application Support", "OpenHost"), path)
}
