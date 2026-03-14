package app

import (
	"fmt"

	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/state"
)

func DeleteKnownServer(selector string) (*state.Record, error) {
	store, err := state.DefaultStore()
	if err != nil {
		return nil, err
	}

	record, err := findKnownServerInStore(store, selector)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("no local state record found for %q", selector)
	}

	provider, err := core.GetProvider(record.Provider)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q for server %q: %w", record.Provider, record.Name, err)
	}

	if err := provider.DeleteServer(record.ID); err != nil {
		return nil, fmt.Errorf("delete server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}

	if err := store.RemoveRecord(record.Provider, record.ID); err != nil {
		return nil, fmt.Errorf("remove local state for server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}

	return record, nil
}

func findKnownServerInStore(store *state.Store, selector string) (*state.Record, error) {
	provider, id, ok := splitProviderID(selector)
	if ok {
		return store.GetRecord(provider, id)
	}

	return store.FindByName(selector)
}
