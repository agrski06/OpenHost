package config

// ServerConfig is the canonical config shape used by the runtime.
//
// Shared selectors stay flat (`provider.name`, `game.name`) while provider- and
// game-specific configuration lives under `settings` so new integrations can add
// their own properties without bloating the shared schema.
type ServerConfig struct {
	Server   ServerDetail   `mapstructure:"server"`
	Provider ProviderConfig `mapstructure:"provider"`
	Game     GameConfig     `mapstructure:"game"`
}

type ServerDetail struct {
	Name string `mapstructure:"name"`
}

type ProviderConfig struct {
	Name     string         `mapstructure:"name"`
	Settings map[string]any `mapstructure:"settings"`
}

type GameConfig struct {
	Name     string         `mapstructure:"name"`
	Settings map[string]any `mapstructure:"settings"`
}
