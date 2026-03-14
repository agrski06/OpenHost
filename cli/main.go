package main

import (
	"fmt"
	"os"

	godotenv2 "github.com/joho/godotenv"
	"github.com/openhost/cli/cmd"

	// Providers registration
	_ "github.com/openhost/cli/internal/providers/hetzner"
	_ "github.com/openhost/cli/internal/providers/mock"

	// Games registration
	_ "github.com/openhost/cli/internal/games/minecraft"
	_ "github.com/openhost/cli/internal/games/valheim"

	// Game status checker registration
	_ "github.com/openhost/cli/internal/gamestatus/minecraft"
	_ "github.com/openhost/cli/internal/gamestatus/valheim"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv2.Load()

	cli := cmd.New(os.Stdin, os.Stdout, os.Stderr)
	return cli.Execute(os.Args[1:])
}
