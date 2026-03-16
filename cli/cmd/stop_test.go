package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStopArgs_RequiresSelector(t *testing.T) {
	cli := New(bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{})
	_, _, _, _, err := cli.parseStopArgs(nil)
	assert.Error(t, err)
}

func TestParseStopArgs_ParsesSnapshotDescription(t *testing.T) {
	cli := New(bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{})
	selector, desc, del, noSnap, err := cli.parseStopArgs([]string{"--snapshot-description=my-snap", "alpha"})
	require.NoError(t, err)
	assert.Equal(t, "alpha", selector)
	assert.Equal(t, "my-snap", desc)
	assert.False(t, del)
	assert.False(t, noSnap)
}

func TestRunStop_ResolvesServerName(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	store := state.NewStore(filepath.Join(stateDir, "instances.json"))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider:  "mock",
		ID:        "mock-alpha",
		Name:      "alpha",
		PublicIP:  "203.0.113.10",
		Game:      "minecraft",
		CreatedAt: "2026-03-14T00:00:00Z",
	}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(bytes.NewBuffer(nil), &stdout, &stderr)

	err := cli.runStop([]string{"alpha"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop+snapshot")
}
