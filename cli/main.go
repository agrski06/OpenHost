package main

import (
	"fmt"
	"log"
	"os"

	godotenv2 "github.com/joho/godotenv"
	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/core"

	// Providers registration
	_ "github.com/openhost/cli/internal/providers/hetzner"

	// Games registration
	_ "github.com/openhost/cli/internal/games/minecraft"
	_ "github.com/openhost/cli/internal/games/valheim"
)

func main() {
	err := godotenv2.Load()
	if err != nil {
		log.Fatal("Could not load .env")
	}

	parsedConfig, _ := config.ParseYAML("example/hetzner_valheim_config.yaml")

	provider, _ := core.GetProvider(parsedConfig.Provider.Name)
	game, _ := core.GetGame(parsedConfig.Game.Name)

	// TODO: I guess I will need to pass whole ProviderConfig anyway
	server, err := provider.RunServer(
		parsedConfig.Server.Name,
		game,
		parsedConfig.Provider.Settings,
		parsedConfig.Game.Settings,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Server running at:", server.IP())
}
