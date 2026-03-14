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
	cli := New(bytes.NewBuffer(nil), &stdout, &stderr)

	require.NoError(t, cli.runDown([]string{"alpha"}))
	assert.Equal(
		t,
		"Deleted server mock:mock-server-1 (alpha) at 203.0.113.10\n",
		stdout.String(),
	)

	record, err := store.GetRecord("mock", "mock-server-1")
	require.NoError(t, err)
	assert.Nil(t, record)
}

func TestParseDownArgs_WithRemoveAssociatedResourcesFlag(t *testing.T) {
	cli := New(bytes.NewBuffer(nil), &bytes.Buffer{}, &bytes.Buffer{})
	selector, removeAssociated, err := cli.parseDownArgs([]string{removeAssociatedResourcesFlag, "alpha"})
	require.NoError(t, err)
	assert.Equal(t, "alpha", selector)
	require.NotNil(t, removeAssociated)
	assert.True(t, *removeAssociated)
}

func TestPromptRemoveAssociatedResources_DefaultNoOnEOF(t *testing.T) {
	var stdout bytes.Buffer
	cli := New(bytes.NewBuffer(nil), &stdout, &bytes.Buffer{})
	remove, err := cli.promptRemoveAssociatedResources("alpha", "hetzner")
	require.NoError(t, err)
	assert.False(t, remove)
	assert.Contains(t, stdout.String(), "Remove associated resources")
}

func TestPromptRemoveAssociatedResources_Yes(t *testing.T) {
	var stdout bytes.Buffer
	cli := New(bytes.NewBufferString("yes\n"), &stdout, &bytes.Buffer{})
	remove, err := cli.promptRemoveAssociatedResources("alpha", "hetzner")
	require.NoError(t, err)
	assert.True(t, remove)
}
