package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ParseYAML parses a YAML file into a ServerConfig struct.
func ParseYAML(filePath string) (*ServerConfig, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := validateCanonicalConfigShape(v.AllSettings()); err != nil {
		return nil, fmt.Errorf("invalid config shape: %w", err)
	}

	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := normalizeConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to normalize config: %w", err)
	}

	if err := validateMandatoryConfigFields(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func normalizeConfig(config *ServerConfig) error {
	if config.Provider.Settings == nil {
		config.Provider.Settings = map[string]any{}
	}
	if config.Game.Settings == nil {
		config.Game.Settings = map[string]any{}
	}

	return nil
}

// validateMandatoryConfigFields performs basic validation on the ServerConfig.
func validateMandatoryConfigFields(config *ServerConfig) error {
	if config.Server.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	if config.Provider.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if config.Game.Name == "" {
		return fmt.Errorf("game type cannot be empty")
	}

	return nil
}

func validateCanonicalConfigShape(raw map[string]any) error {
	if _, ok := raw["name"]; ok {
		return fmt.Errorf("top-level name is not supported; use server.name")
	}

	providerSection, _ := nestedMap(raw, "provider")
	for key := range providerSection {
		switch key {
		case "region", "location", "plan":
			return fmt.Errorf("provider.%s is not supported; use provider.settings.%s", key, key)
		}
	}

	gameSection, _ := nestedMap(raw, "game")
	if _, ok := gameSection["type"]; ok {
		return fmt.Errorf("game.type is not supported; use game.name")
	}

	return nil
}

func nestedMap(raw map[string]any, key string) (map[string]any, bool) {
	value, ok := raw[key]
	if !ok || value == nil {
		return nil, false
	}

	section, ok := value.(map[string]any)
	return section, ok
}
