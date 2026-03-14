package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/openhost/cli/internal/app"
)

const removeAssociatedResourcesFlag = "--remove-associated-resources"

func (c *CLI) runDown(args []string) error {
	selector, removeAssociatedResources, err := c.parseDownArgs(args)
	if err != nil {
		return err
	}

	record, err := app.FindKnownServer(selector)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("no local state record found for %q", selector)
	}

	if removeAssociatedResources == nil && len(record.AssociatedResources) > 0 && supportsAssociatedResources(record.Provider) {
		choice, err := c.promptRemoveAssociatedResources(record.Name, record.Provider)
		if err != nil {
			return err
		}
		removeAssociatedResources = &choice
	}

	removeAssociated := false
	if removeAssociatedResources != nil {
		removeAssociated = *removeAssociatedResources
	}

	deleteResult, err := app.DeleteKnownServerWithOptions(selector, removeAssociated)
	if err != nil {
		return err
	}
	record = deleteResult.Record

	_, err = fmt.Fprintf(
		c.stdout,
		"Deleted server %s:%s (%s) at %s\n",
		record.Provider,
		record.ID,
		record.Name,
		record.PublicIP,
	)
	if err != nil {
		return err
	}
	for _, warning := range deleteResult.Warnings {
		if _, err := fmt.Fprintf(c.stdout, "Warning: %s\n", warning); err != nil {
			return err
		}
	}
	return nil
}

func (c *CLI) parseDownArgs(args []string) (selector string, removeAssociatedResources *bool, err error) {
	for _, arg := range args {
		switch arg {
		case removeAssociatedResourcesFlag:
			value := true
			removeAssociatedResources = &value
		default:
			if strings.HasPrefix(arg, "--") {
				return "", nil, fmt.Errorf("unknown down flag %q", arg)
			}
			if selector != "" {
				return "", nil, fmt.Errorf("down requires exactly one selector in server-name or provider:id form")
			}
			selector = arg
		}
	}

	if selector == "" {
		return "", nil, fmt.Errorf("down requires exactly one selector in server-name or provider:id form")
	}

	return selector, removeAssociatedResources, nil
}

func (c *CLI) promptRemoveAssociatedResources(serverName string, provider string) (bool, error) {
	if _, err := fmt.Fprintf(c.stdout, "Remove associated resources for %s via provider %s? [y/N]: ", serverName, provider); err != nil {
		return false, err
	}
	if c.stdin == nil {
		return false, nil
	}

	line, err := bufio.NewReader(c.stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return false, nil
	}

	response := strings.TrimSpace(strings.ToLower(line))
	return response == "y" || response == "yes", nil
}

func supportsAssociatedResources(provider string) bool {
	switch provider {
	case "hetzner":
		return true
	default:
		return false
	}
}
