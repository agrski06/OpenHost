package cmd

import (
	"fmt"
	"io"
	"strings"
)

const DefaultConfigPath = "openhost_config.yaml"

type CLI struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func New(stdin io.Reader, stdout io.Writer, stderr io.Writer) *CLI {
	return &CLI{stdin: stdin, stdout: stdout, stderr: stderr}
}

func (c *CLI) Execute(args []string) error {
	if len(args) == 0 {
		return c.runUp(nil)
	}

	switch args[0] {
	case "up":
		return c.runUp(args[1:])
	case "list":
		return c.runList(args[1:])
	case "status":
		return c.runStatus(args[1:])
	case "down":
		return c.runDown(args[1:])
	case "help", "-h", "--help":
		return c.printUsage()
	default:
		if strings.HasPrefix(args[0], "-") {
			if err := c.printUsage(); err != nil {
				return err
			}
			return fmt.Errorf("unknown command %q", args[0])
		}

		// Compatibility path: treat a bare config path as `up <config>`.
		return c.runUp(args)
	}
}

func (c *CLI) printUsage() error {
	usage := "Usage:\n" +
		"  openhost up [config-file]\n" +
		"  openhost list\n" +
		"  openhost status [server-name|provider:id]\n" +
		"  openhost down [--remove-associated-resources] <server-name|provider:id>\n" +
		"  openhost help\n\n" +
		"Notes:\n" +
		"  - If no arguments are provided, the CLI runs `up` with the default config.\n" +
		"  - For compatibility, a bare config path is treated like `up <config-file>`.\n" +
		"  - `down` prompts before removing associated resources unless the flag is provided."

	_, err := fmt.Fprintln(c.stdout, usage)
	return err
}
