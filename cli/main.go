package main

import (
	"fmt"
	"os"

	godotenv2 "github.com/joho/godotenv"
	"github.com/openhost/cli/internal/app"

	// Providers registration
	_ "github.com/openhost/cli/internal/providers/hetzner"
	_ "github.com/openhost/cli/internal/providers/mock"

	// Games registration
	_ "github.com/openhost/cli/internal/games/minecraft"
	_ "github.com/openhost/cli/internal/games/valheim"
)

const defaultConfigPath = "example/mock_minecraft_config.yaml"

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv2.Load()

	configPath := defaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	server, err := app.DeployFromConfig(configPath)
	if err != nil {
		return err
	}

	_, err = fmt.Println("Server running at:", server.IP())
	return err
}
