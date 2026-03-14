package app

import (
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/core"
	_ "github.com/openhost/cli/internal/providers/mock"
	"github.com/openhost/cli/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteKnownServerWithOptions_WarnsForSharedAssociatedResources(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("OPENHOST_STATE_DIR", stateDir)

	sharedFirewall := core.ResourceRef{Type: "firewall", ID: "fw-1", Name: "shared-fw"}
	store := state.NewStore(filepath.Join(stateDir, "instances.json"))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider: "mock",
		ID:       "mock-alpha",
		Name:     "alpha",
		Game:     "minecraft",
		AssociatedResources: []core.ResourceRef{
			sharedFirewall,
		},
		CreatedAt: "2026-03-14T00:00:00Z",
	}))
	require.NoError(t, store.SaveRecord(state.Record{
		Provider: "mock",
		ID:       "mock-beta",
		Name:     "beta",
		Game:     "minecraft",
		AssociatedResources: []core.ResourceRef{
			sharedFirewall,
		},
		CreatedAt: "2026-03-14T00:01:00Z",
	}))

	result, err := DeleteKnownServerWithOptions("alpha", true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Record)
	assert.Equal(t, "alpha", result.Record.Name)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "was not removed because it is also referenced")
	assert.Contains(t, result.Warnings[0], "mock:mock-beta (beta)")

	alpha, err := store.GetRecord("mock", "mock-alpha")
	require.NoError(t, err)
	assert.Nil(t, alpha)

	beta, err := store.GetRecord("mock", "mock-beta")
	require.NoError(t, err)
	require.NotNil(t, beta)
	require.Len(t, beta.AssociatedResources, 1)
	assert.Equal(t, sharedFirewall, beta.AssociatedResources[0])
}
