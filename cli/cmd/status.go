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

	record, err := app.FindKnownServer(args[0])
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("no local state record found for %q", args[0])
	}
	return c.printRecord(*record)
}

func (c *CLI) printRecord(record state.Record) error {
	_, err := fmt.Fprintf(
		c.stdout,
		"Provider: %s\nID: %s\nName: %s\nGame: %s\nIP: %s\nConfig: %s\nCreated: %s\n",
		record.Provider,
		record.ID,
		record.Name,
		record.Game,
		record.PublicIP,
		record.ConfigPath,
		record.CreatedAt,
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
