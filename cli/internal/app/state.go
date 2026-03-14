package app

import (
	"fmt"
	"strings"

	"github.com/openhost/cli/internal/state"
)

func ListKnownServers() ([]state.Record, error) {
	store, err := state.DefaultStore()
	if err != nil {
		return nil, err
	}

	return store.ListRecords()
}

func FindKnownServer(selector string) (*state.Record, error) {
	store, err := state.DefaultStore()
	if err != nil {
		return nil, err
	}

	provider, id, ok := splitProviderID(selector)
	if ok {
		return store.GetRecord(provider, id)
	}

	return store.FindByName(selector)
}

func ParseProviderID(selector string) (provider string, id string, err error) {
	provider, id, ok := splitProviderID(selector)
	if !ok {
		return "", "", fmt.Errorf("invalid server selector %q; expected provider:id", selector)
	}
	return provider, id, nil
}

func splitProviderID(selector string) (provider string, id string, ok bool) {
	parts := strings.SplitN(selector, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}
