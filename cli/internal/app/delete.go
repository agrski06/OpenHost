package app

import (
	"fmt"

	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/state"
)

type DeleteResult struct {
	Record   *state.Record
	Warnings []string
}

func DeleteKnownServerWithOptions(selector string, removeAssociatedResources bool) (*DeleteResult, error) {
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
	result := &DeleteResult{Record: record}

	provider, err := core.GetProvider(record.Provider)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q for server %q: %w", record.Provider, record.Name, err)
	}

	deleteRequest := core.DeleteServerRequest{
		ID:                        record.ID,
		GameName:                  record.Game,
		AssociatedResources:       record.AssociatedResources,
		RemoveAssociatedResources: removeAssociatedResources,
		SnapshotIDs:               nil,
	}
	if removeAssociatedResources && record.LastSnapshotID != "" {
		deleteRequest.SnapshotIDs = []string{record.LastSnapshotID}
	}
	if removeAssociatedResources {
		deleteRequest.AssociatedResources, result.Warnings, err = filterSharedAssociatedResources(store, *record)
		if err != nil {
			return nil, err
		}
	}
	if gameDefinition, err := core.GetGame(record.Game); err == nil {
		deleteRequest.GameName = gameDefinition.Name()
		deleteRequest.Ports = gameDefinition.Ports()
	}

	if err := provider.DeleteServer(deleteRequest); err != nil {
		return nil, fmt.Errorf("delete server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}

	if err := store.RemoveRecord(record.Provider, record.ID); err != nil {
		return nil, fmt.Errorf("remove local state for server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}

	return result, nil
}

func filterSharedAssociatedResources(store *state.Store, record state.Record) ([]core.ResourceRef, []string, error) {
	records, err := store.ListRecords()
	if err != nil {
		return nil, nil, err
	}

	allowed := make([]core.ResourceRef, 0, len(record.AssociatedResources))
	warnings := []string{}
	for _, resource := range record.AssociatedResources {
		sharedWith := findOtherResourceReferences(records, record, resource)
		if len(sharedWith) == 0 {
			allowed = append(allowed, resource)
			continue
		}

		warning := fmt.Sprintf(
			"associated resource %s:%s (%s) was not removed because it is also referenced by %s",
			resource.Type,
			resource.ID,
			resource.Name,
			sharedWith[0],
		)
		if len(sharedWith) > 1 {
			warning = fmt.Sprintf("%s and %d other tracked server(s)", warning, len(sharedWith)-1)
		}
		warnings = append(warnings, warning)
	}

	return allowed, warnings, nil
}

func findOtherResourceReferences(records []state.Record, current state.Record, resource core.ResourceRef) []string {
	references := []string{}
	for _, other := range records {
		if other.Provider != current.Provider {
			continue
		}
		if other.Provider == current.Provider && other.ID == current.ID {
			continue
		}
		for _, otherResource := range other.AssociatedResources {
			if otherResource.Type == resource.Type && otherResource.ID == resource.ID {
				references = append(references, fmt.Sprintf("%s:%s (%s)", other.Provider, other.ID, other.Name))
				break
			}
		}
	}
	return references
}

func findKnownServerInStore(store *state.Store, selector string) (*state.Record, error) {
	provider, id, ok := splitProviderID(selector)
	if ok {
		return store.GetRecord(provider, id)
	}

	return store.FindByName(selector)
}
