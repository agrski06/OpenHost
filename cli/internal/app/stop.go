package app

import (
	"fmt"
	"time"

	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/state"
)

type StopResult struct {
	Record   *state.Record
	Snapshot *core.SnapshotResult
}

type StopOptions struct {
	CreateSnapshot      bool
	DeleteServer        bool
	SnapshotDescription string
}

func StopKnownServer(selector string, opts StopOptions) (*StopResult, error) {
	if !opts.CreateSnapshot && opts.DeleteServer {
		return nil, fmt.Errorf("delete requires snapshot")
	}

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

	result := &StopResult{Record: record}

	if opts.CreateSnapshot {
		snapshot, err := provider.StopServerAndSnapshot(core.StopServerAndSnapshotRequest{
			ID:                  record.ID,
			Name:                record.Name,
			GameName:            record.Game,
			PublicIP:            record.PublicIP,
			SnapshotDescription: opts.SnapshotDescription,
		})
		if err != nil {
			return nil, fmt.Errorf("stop+snapshot server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
		}
		result.Snapshot = snapshot
		if snapshot != nil {
			record.LastSnapshotID = snapshot.SnapshotID
			record.LastSnapshotDescription = snapshot.SnapshotDescription
			record.LastSnapshotCreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
	} else {
		if err := provider.StopServer(core.StopServerRequest{ID: record.ID}); err != nil {
			return nil, fmt.Errorf("stop server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
		}
	}

	if opts.DeleteServer {
		if err := provider.DeleteServer(core.DeleteServerRequest{ID: record.ID}); err != nil {
			return nil, fmt.Errorf("delete server after snapshot %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
		}
		record.Deleted = true
	} else {
		record.Deleted = false
	}

	if err := store.SaveRecord(*record); err != nil {
		return nil, fmt.Errorf("persist state after stop for server %q (%s:%s): %w", record.Name, record.Provider, record.ID, err)
	}

	return result, nil
}
