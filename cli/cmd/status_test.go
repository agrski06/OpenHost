package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStatus_PrintsCombinedStatus(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	store := state.NewStore(filepath.Join(stateDir, "instances.json"))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider:   "mock",
		ID:         "mock-alpha",
		Name:       "alpha",
		PublicIP:   "203.0.113.10",
		Game:       "minecraft",
		ConfigPath: "example/mock_minecraft_config.yaml",
		CreatedAt:  "2026-03-14T00:00:00Z",
	}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(bytes.NewBuffer(nil), &stdout, &stderr)

	require.NoError(t, cli.runStatus([]string{"alpha"}))
	output := stdout.String()
	assert.Contains(t, output, "Name: alpha")
	assert.Contains(t, output, "Provider: mock")
	assert.Contains(t, output, "Provider ID: mock-alpha")
	assert.Contains(t, output, "Local:")
	assert.Contains(t, output, "State: tracked")
	assert.Contains(t, output, "Infrastructure:")
	assert.Contains(t, output, "State: running")
	assert.Contains(t, output, "Game:")
	assert.Contains(t, output, "Detail: minecraft status check not implemented yet")
}
