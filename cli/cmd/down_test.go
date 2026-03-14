package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDown_ResolvesServerName(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	store := state.NewStore(filepath.Join(stateDir, "instances.json"))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider:  "mock",
		ID:        "mock-server-1",
		Name:      "alpha",
		PublicIP:  "203.0.113.10",
		Game:      "minecraft",
		CreatedAt: "2026-03-14T00:00:00Z",
	}))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(&stdout, &stderr)

	require.NoError(t, cli.runDown([]string{"alpha"}))
	assert.Equal(
		t,
		"Down is not implemented yet. Known local record: provider=mock id=mock-server-1 name=alpha ip=203.0.113.10\n",
		stdout.String(),
	)
}
