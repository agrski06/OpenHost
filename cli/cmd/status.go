package cmd

import (
	"fmt"

	"github.com/openhost/cli/internal/app"
	"github.com/openhost/cli/internal/state"
)

func (c *CLI) runStatus(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("status accepts at most one selector")
	}
	if len(args) == 0 {
		return c.runList(nil)
	}

	report, err := app.GetKnownServerStatus(args[0])
	if err != nil {
		return err
	}
	return c.printStatus(*report)
}

func (c *CLI) printStatus(report app.ServerStatus) error {
	_, err := fmt.Fprintf(
		c.stdout,
		"Name: %s\nProvider: %s\nProvider ID: %s\n\nLocal:\n  State: %s\n  Detail: %s\n  Config: %s\n  Created: %s\n  Deleted: %t\n  Last Snapshot ID: %s\n  Last Snapshot Description: %s\n  Last Snapshot Created: %s\n\nInfrastructure:\n  State: %s\n  Detail: %s\n  Name: %s\n  Public IP: %s\n\nGame:\n  State: %s\n  Detail: %s\n",
		report.Record.Name,
		report.Record.Provider,
		report.Record.ID,
		report.Local.State,
		report.Local.Detail,
		report.Record.ConfigPath,
		report.Record.CreatedAt,
		report.Record.Deleted,
		report.Record.LastSnapshotID,
		report.Record.LastSnapshotDescription,
		report.Record.LastSnapshotCreatedAt,
		report.Infrastructure.State,
		report.Infrastructure.Detail,
		report.Infrastructure.Name,
		report.Infrastructure.PublicIP,
		report.Game.State,
		report.Game.Detail,
	)
	return err
}

func (c *CLI) printRecordSummary(record state.Record) error {
	_, err := fmt.Fprintf(c.stdout, "%s:%s  name=%s  game=%s  ip=%s\n", record.Provider, record.ID, record.Name, record.Game, record.PublicIP)
	return err
}

func (c *CLI) printNoServers() error {
	_, err := fmt.Fprintln(c.stdout, "No servers found in local state.")
	return err
}
