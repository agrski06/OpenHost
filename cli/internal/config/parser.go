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

	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateConfig performs basic validation on the ServerConfig.
func validateConfig(config *ServerConfig) error {
	if config.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	if config.Provider.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if config.Provider.Plan == "" {
		return fmt.Errorf("provider plan cannot be empty")
	}
	if config.Game.Type == "" {
		return fmt.Errorf("game type cannot be empty")
	}

	return nil
}
