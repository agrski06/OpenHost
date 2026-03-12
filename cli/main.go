package main

import (
	"fmt"
	"os"

	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/core"

	// Providers registration
	_ "github.com/openhost/cli/internal/providers/hetzner"

	// Games registration
	_ "github.com/openhost/cli/internal/games/minecraft"
)

func main() {

	parsedConfig, _ := config.ParseYAML("tests/test_config.yaml")

	provider, _ := core.GetProvider(parsedConfig.Provider.Name)
	game, _ := core.GetGame(parsedConfig.Game.Type)

	// TODO: I guess I will need to pass whole ProviderConfig anyway
	server, err := provider.RunServer(parsedConfig.Name, game)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Server running at:", server.IP())
}
