package app

import (
	"fmt"

	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/core"
)

// DeployFromConfig parses a config file, resolves the registered provider and
// game implementations, and provisions the server through the current runtime
// flow.
func DeployFromConfig(configPath string) (*core.Server, error) {
	parsedConfig, err := config.ParseYAML(configPath)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	provider, err := core.GetProvider(parsedConfig.Provider.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %q: %w", parsedConfig.Provider.Name, err)
	}

	game, err := core.GetGame(parsedConfig.Game.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve game %q: %w", parsedConfig.Game.Name, err)
	}

	userData, err := game.BuildInitCommand(parsedConfig.Game.Settings)
	if err != nil {
		return nil, fmt.Errorf("build init command for game %q: %w", parsedConfig.Game.Name, err)
	}

	server, err := provider.CreateServer(core.CreateServerRequest{
		Name:             parsedConfig.Server.Name,
		GameName:         game.Name(),
		Ports:            game.Ports(),
		ProviderSettings: parsedConfig.Provider.Settings,
		UserData:         userData,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"provision server %q with provider %q for game %q: %w",
			parsedConfig.Server.Name,
			parsedConfig.Provider.Name,
			parsedConfig.Game.Name,
			err,
		)
	}

	return server, nil
}
