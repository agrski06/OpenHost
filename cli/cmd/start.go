package cmd

import (
	"fmt"
	"strings"

	"github.com/openhost/cli/internal/app"
)

const startRecreateFlag = "--recreate"

func (c *CLI) runStart(args []string) error {
	selector := ""
	recreate := false
	for _, arg := range args {
		switch arg {
		case startRecreateFlag:
			recreate = true
		default:
			if strings.HasPrefix(arg, "--") {
				return fmt.Errorf("unknown start flag %q", arg)
			}
			if selector != "" {
				return fmt.Errorf("start requires exactly one selector in server-name or provider:id form")
			}
			selector = arg
		}
	}
	if selector == "" {
		return fmt.Errorf("start requires exactly one selector in server-name or provider:id form")
	}

	result, err := app.StartKnownServer(selector, app.StartOptions{Recreate: recreate})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(
		c.stdout,
		"Started server %s (provider=%s id=%s ip=%s)\n",
		result.OldRecord.Name,
		result.NewServer.Provider,
		result.NewServer.ID,
		result.NewServer.PublicIP,
	)
	return err
}
