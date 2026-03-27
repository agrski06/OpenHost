package app

import (
	"context"
	"fmt"

	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/gamestatus"
	"github.com/openhost/cli/internal/state"
)

type LocalStatus struct {
	State  string
	Detail string
}

type GameStatus struct {
	State  string
	Detail string
}

type ServerStatus struct {
	Record         state.Record
	Local          LocalStatus
	Infrastructure core.InfrastructureStatus
	Game           GameStatus
}

func GetKnownServerStatus(selector string) (*ServerStatus, error) {
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

	report := &ServerStatus{
		Record: *record,
		Local: LocalStatus{
			State:  "tracked",
			Detail: fmt.Sprintf("tracked in local state via %s", record.ConfigPath),
		},
		Infrastructure: core.InfrastructureStatus{
			ID:       record.ID,
			Name:     record.Name,
			PublicIP: record.PublicIP,
			State:    core.InfrastructureStateUnknown,
			Detail:   "infrastructure status not queried yet",
		},
		Game: GameStatus{
			State:  "unknown",
			Detail: "game status not implemented yet",
		},
	}

	provider, err := core.GetProvider(record.Provider)
	if err != nil {
		report.Infrastructure.State = core.InfrastructureStateError
		report.Infrastructure.Detail = fmt.Sprintf("resolve provider %q: %v", record.Provider, err)
		return report, nil
	}

	ctx := context.Background()

	infra, err := provider.GetServerStatus(ctx, record.ID)
	if err != nil {
		report.Infrastructure.State = core.InfrastructureStateError
		report.Infrastructure.Detail = err.Error()
		return report, nil
	}
	if infra != nil {
		report.Infrastructure = *infra
	}

	if report.Infrastructure.ID == "" {
		report.Infrastructure.ID = record.ID
	}
	if report.Infrastructure.Name == "" {
		report.Infrastructure.Name = record.Name
	}
	if report.Infrastructure.PublicIP == "" {
		report.Infrastructure.PublicIP = record.PublicIP
	}

	gameDefinition, err := core.GetGame(record.Game)
	if err != nil {
		report.Game.State = "unknown"
		report.Game.Detail = fmt.Sprintf("resolve game %q: %v", record.Game, err)
		return report, nil
	}

	if report.Infrastructure.State != core.InfrastructureStateRunning {
		report.Game.State = "unknown"
		report.Game.Detail = fmt.Sprintf("game status skipped because infrastructure is %s", report.Infrastructure.State)
		return report, nil
	}
	if report.Infrastructure.PublicIP == "" {
		report.Game.State = "unknown"
		report.Game.Detail = "game status skipped because no public IP is available"
		return report, nil
	}

	checker, err := gamestatus.Get(record.Game)
	if err != nil {
		report.Game.State = "unknown"
		report.Game.Detail = err.Error()
		return report, nil
	}

	gameReport, err := checker.Check(gamestatus.Target{
		GameName: record.Game,
		PublicIP: report.Infrastructure.PublicIP,
		Ports:    gameDefinition.Ports(),
	})
	if err != nil {
		report.Game.State = string(gamestatus.StateQueryFailed)
		report.Game.Detail = err.Error()
		return report, nil
	}
	if gameReport != nil {
		report.Game.State = string(gameReport.State)
		report.Game.Detail = gameReport.Detail
	}

	return report, nil
}
