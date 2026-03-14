package cmd

import (
	"fmt"

	"github.com/openhost/cli/internal/app"
)

func (c *CLI) runDown(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("down requires exactly one selector in server-name or provider:id form")
	}

	record, err := app.FindKnownServer(args[0])
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("no local state record found for %q", args[0])
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"Down is not implemented yet. Known local record: provider=%s id=%s name=%s ip=%s\n",
		record.Provider,
		record.ID,
		record.Name,
		record.PublicIP,
	)
	return err
}
