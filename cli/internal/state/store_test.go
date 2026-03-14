package state

import (
	"path/filepath"
	"testing"

	"github.com/openhost/cli/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreLoadMissingFileReturnsEmptySnapshot(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "instances.json"))

	snapshot, err := store.Load()
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	assert.Equal(t, currentVersion, snapshot.Version)
	assert.Empty(t, snapshot.Servers)
}

func TestStoreSaveRecordCreatesAndLoadsSnapshot(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "instances.json"))

	err := store.SaveRecord(Record{
		Provider:   "mock",
		ID:         "mock-server-1",
		Name:       "server-1",
		PublicIP:   "203.0.113.10",
		Game:       "minecraft",
		ConfigPath: "example/mock_minecraft_config.yaml",
		AssociatedResources: []core.ResourceRef{
			{Type: "firewall", ID: "fw-1", Name: "fw-test"},
		},
		CreatedAt: "2026-03-14T00:00:00Z",
	})
	require.NoError(t, err)

	snapshot, err := store.Load()
	require.NoError(t, err)
	require.Len(t, snapshot.Servers, 1)
	assert.Equal(t, "mock", snapshot.Servers[0].Provider)
	assert.Equal(t, "mock-server-1", snapshot.Servers[0].ID)
	assert.Equal(t, "server-1", snapshot.Servers[0].Name)
	require.Len(t, snapshot.Servers[0].AssociatedResources, 1)
	assert.Equal(t, "fw-1", snapshot.Servers[0].AssociatedResources[0].ID)
}

func TestStoreSaveRecordUpsertsByProviderAndID(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "instances.json"))

	first := Record{
		Provider:   "mock",
		ID:         "same-id",
		Name:       "first-name",
		PublicIP:   "203.0.113.10",
		Game:       "minecraft",
		ConfigPath: "first.yaml",
		CreatedAt:  "2026-03-14T00:00:00Z",
	}
	second := Record{
		Provider:   "mock",
		ID:         "same-id",
		Name:       "second-name",
		PublicIP:   "203.0.113.11",
		Game:       "valheim",
		ConfigPath: "second.yaml",
		CreatedAt:  "2026-03-14T01:00:00Z",
	}

	require.NoError(t, store.SaveRecord(first))
	require.NoError(t, store.SaveRecord(second))

	snapshot, err := store.Load()
	require.NoError(t, err)
	require.Len(t, snapshot.Servers, 1)
	assert.Equal(t, second, snapshot.Servers[0])
}

func TestStoreLookupHelpers(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "instances.json"))
	record := Record{
		Provider:   "mock",
		ID:         "mock-server-1",
		Name:       "server-1",
		PublicIP:   "203.0.113.10",
		Game:       "minecraft",
		ConfigPath: "example/mock_minecraft_config.yaml",
		CreatedAt:  "2026-03-14T00:00:00Z",
	}

	require.NoError(t, store.SaveRecord(record))

	records, err := store.ListRecords()
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, record, records[0])

	byID, err := store.GetRecord("mock", "mock-server-1")
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, record, *byID)

	byName, err := store.FindByName("server-1")
	require.NoError(t, err)
	require.NotNil(t, byName)
	assert.Equal(t, record, *byName)
}

func TestStoreRemoveRecord(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "instances.json"))
	record := Record{
		Provider:  "mock",
		ID:        "mock-server-1",
		Name:      "server-1",
		CreatedAt: "2026-03-14T00:00:00Z",
	}

	require.NoError(t, store.SaveRecord(record))
	require.NoError(t, store.RemoveRecord("mock", "mock-server-1"))

	records, err := store.ListRecords()
	require.NoError(t, err)
	assert.Empty(t, records)
}
