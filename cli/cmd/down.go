package cmd

import (
	"fmt"

	"github.com/openhost/cli/internal/app"
)

func (c *CLI) runDown(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("down requires exactly one selector in server-name or provider:id form")
	}

	record, err := app.DeleteKnownServer(args[0])
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"Deleted server %s:%s (%s) at %s\n",
		record.Provider,
		record.ID,
		record.Name,
		record.PublicIP,
	)
	return err
}
