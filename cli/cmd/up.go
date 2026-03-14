package cmd

import (
	"fmt"

	"github.com/openhost/cli/internal/app"
)

func (c *CLI) runUp(args []string) error {
	configPath := DefaultConfigPath
	if len(args) > 1 {
		return fmt.Errorf("up accepts at most one config path")
	}
	if len(args) == 1 {
		configPath = args[0]
	}

	server, err := app.DeployFromConfig(configPath)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.stdout, "Created server %s:%s (%s) at %s\n", server.Provider, server.ID, server.Name, server.IP()); err != nil {
		return err
	}

	return nil
}
