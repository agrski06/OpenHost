// Package main is the entry point for the OpenHost runner binary.
//
// The runner is responsible for bootstrapping and managing game server processes
// on provisioned VPS instances. It receives configuration from the CLI via a
// RunnerConfig JSON blob passed at launch time.
package main

import (
	"fmt"
	"os"
)

func main() {
	_, _ = fmt.Fprintln(os.Stderr, "openhost-runner: not yet implemented")
	os.Exit(1)
}
