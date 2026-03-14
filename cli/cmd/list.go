package cmd

import (
	"fmt"

	"github.com/openhost/cli/internal/app"
)

func (c *CLI) runList(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("list does not accept any arguments")
	}

	records, err := app.ListKnownServers()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return c.printNoServers()
	}

	for _, record := range records {
		if err := c.printRecordSummary(record); err != nil {
			return err
		}
	}

	return nil
}
