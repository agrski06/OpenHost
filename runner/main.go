// Package main is the entry point for the OpenHost runner binary.
//
// The runner is responsible for bootstrapping and managing game server processes
// on provisioned VPS instances. It receives configuration from the CLI via a
// RunnerConfig JSON blob passed at launch time.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/openhost/runner/internal/setup"
	"github.com/openhost/runnerconfig"

	// Game setup registrations.
	_ "github.com/openhost/runner/internal/gamesetup/minecraft"
	_ "github.com/openhost/runner/internal/gamesetup/valheim"

	// Mod provider registrations.
	_ "github.com/openhost/runner/internal/modprovider/thunderstore"

	// Mod framework registrations.
	_ "github.com/openhost/runner/internal/modframework/bepinex"
)

func main() {
	configPath := flag.String("config", "", "Path to runner config JSON")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "openhost-runner: -config flag is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openhost-runner: read config: %v\n", err)
		os.Exit(1)
	}

	var cfg runnerconfig.RunnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "openhost-runner: parse config: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[runner] starting setup for game %q (version=%s)", cfg.Game.Name, cfg.Version)

	if err := setup.Run(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "openhost-runner: setup failed: %v\n", err)
		os.Exit(1)
	}

	log.Println("[runner] setup completed successfully")
}
