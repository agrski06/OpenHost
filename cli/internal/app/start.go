package app

import (
	"context"
	"fmt"
	"time"

	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/state"
)

type StartResult struct {
	OldRecord *state.Record
	NewServer *core.Server
}

type StartOptions struct {
	Recreate bool
}

// StartKnownServer creates a new server from the last snapshot
// stored in local state (created by `openhost stop`) and updates local state to
// track the new provider-native server ID.
func StartKnownServer(selector string, opts StartOptions) (*StartResult, error) {
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

	ctx := context.Background()

	shouldRecreate := opts.Recreate || record.Deleted
	if !shouldRecreate {
		status, err := provider.GetServerStatus(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		if status != nil && status.State == core.InfrastructureStateNotFound {
			shouldRecreate = true
		}
	}

	if !shouldRecreate {
		// Power on the existing server.
		if err := provider.StartServer(ctx, core.StartServerRequest{ID: record.ID}); err != nil {
			return nil, fmt.Errorf("start server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
		}
		record.Deleted = false
		if err := store.SaveRecord(*record); err != nil {
			return nil, fmt.Errorf("persist state after start for server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
		}
		return &StartResult{OldRecord: record, NewServer: &core.Server{ID: record.ID, Provider: record.Provider, Name: record.Name, PublicIP: record.PublicIP, AssociatedResources: record.AssociatedResources}}, nil
	}

	if record.LastSnapshotID == "" {
		return nil, fmt.Errorf("no snapshot found in local state for %q; run `openhost stop` first", selector)
	}
	if record.ConfigPath == "" {
		return nil, fmt.Errorf("no config_path stored for %q; cannot determine provider settings / ports", selector)
	}

	parsedConfig, err := config.ParseYAML(record.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", record.ConfigPath, err)
	}

	game, err := core.GetGame(record.Game)
	if err != nil {
		return nil, fmt.Errorf("resolve game %q for server %q: %w", record.Game, record.Name, err)
	}

	newServer, err := provider.StartServerFromSnapshot(ctx, core.StartServerFromSnapshotRequest{
		SnapshotID:       record.LastSnapshotID,
		Name:             record.Name,
		GameName:         game.Name(),
		Ports:            game.Ports(),
		ProviderSettings: parsedConfig.Provider.Settings,
	})
	if err != nil {
		return nil, fmt.Errorf("start server %q from snapshot %q (%s:%s): %w", record.Name, record.LastSnapshotID, record.Provider, record.ID, err)
	}

	// Replace tracked server ID with the new server.
	old := *record
	record.ID = newServer.ID
	record.PublicIP = newServer.PublicIP
	record.AssociatedResources = newServer.AssociatedResources
	record.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	record.Deleted = false
	if err := store.SaveRecord(*record); err != nil {
		return nil, fmt.Errorf("persist state for started server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}
	// Remove old record if provider ID changed.
	if old.ID != record.ID {
		_ = store.RemoveRecord(old.Provider, old.ID)
	}

	return &StartResult{OldRecord: &old, NewServer: newServer}, nil
}
