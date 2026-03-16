package cmd

import (
	"fmt"
	"strings"

	"github.com/openhost/cli/internal/app"
)

const snapshotDescriptionFlagPrefix = "--snapshot-description="
const stopDeleteFlag = "--delete"
const stopNoSnapshotFlag = "--no-snapshot"

func (c *CLI) runStop(args []string) error {
	selector, snapshotDescription, deleteServer, noSnapshot, err := c.parseStopArgs(args)
	if err != nil {
		return err
	}

	result, err := app.StopKnownServer(selector, app.StopOptions{
		SnapshotDescription: snapshotDescription,
		CreateSnapshot:      !noSnapshot,
		DeleteServer:        deleteServer,
	})
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Stopped server %s:%s (%s) at %s", result.Record.Provider, result.Record.ID, result.Record.Name, result.Record.PublicIP)
	if result.Snapshot != nil {
		message += fmt.Sprintf(" and created snapshot %s (%s)", result.Snapshot.SnapshotID, result.Snapshot.SnapshotDescription)
	}
	if result.Record.Deleted {
		message += " and deleted the original server"
	}
	_, err = fmt.Fprintln(c.stdout, message)
	return err
}

func (c *CLI) parseStopArgs(args []string) (selector string, snapshotDescription string, deleteServer bool, noSnapshot bool, err error) {
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, snapshotDescriptionFlagPrefix):
			snapshotDescription = strings.TrimPrefix(arg, snapshotDescriptionFlagPrefix)
			if snapshotDescription == "" {
				return "", "", false, false, fmt.Errorf("stop flag %q cannot be empty", snapshotDescriptionFlagPrefix)
			}
		case arg == stopDeleteFlag:
			deleteServer = true
		case arg == stopNoSnapshotFlag:
			noSnapshot = true
		case strings.HasPrefix(arg, "--"):
			return "", "", false, false, fmt.Errorf("unknown stop flag %q", arg)
		default:
			if selector != "" {
				return "", "", false, false, fmt.Errorf("stop requires exactly one selector in server-name or provider:id form")
			}
			selector = arg
		}
	}

	if selector == "" {
		return "", "", false, false, fmt.Errorf("stop requires exactly one selector in server-name or provider:id form")
	}
	if deleteServer && noSnapshot {
		return "", "", false, false, fmt.Errorf("stop flags %q and %q cannot be used together", stopDeleteFlag, stopNoSnapshotFlag)
	}

	return selector, snapshotDescription, deleteServer, noSnapshot, nil
}
