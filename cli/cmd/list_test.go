package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunList_NoServers(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(&stdout, &stderr)

	require.NoError(t, cli.runList(nil))
	assert.Equal(t, "No servers found in local state.\n", stdout.String())
}

func TestRunList_PrintsSummaries(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	store := state.NewStore(filepath.Join(stateDir, "instances.json"))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider:  "mock",
		ID:        "mock-server-1",
		Name:      "alpha",
		Game:      "minecraft",
		PublicIP:  "203.0.113.10",
		CreatedAt: "2026-03-14T00:00:00Z",
	}))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider:  "hetzner",
		ID:        "12345",
		Name:      "beta",
		Game:      "valheim",
		PublicIP:  "198.51.100.20",
		CreatedAt: "2026-03-14T01:00:00Z",
	}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(&stdout, &stderr)

	require.NoError(t, cli.runList(nil))
	output := stdout.String()
	assert.Contains(t, output, "mock:mock-server-1  name=alpha  game=minecraft  ip=203.0.113.10")
	assert.Contains(t, output, "hetzner:12345  name=beta  game=valheim  ip=198.51.100.20")
}

func TestExecute_ListRoute(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(&stdout, &stderr)

	require.NoError(t, cli.Execute([]string{"list"}))
	assert.Equal(t, "No servers found in local state.\n", stdout.String())
}
