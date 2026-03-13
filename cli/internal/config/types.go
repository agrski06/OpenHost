package config

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
	Settings map[string]any `mapstructure:"settings"` // The "bucket"
}

type GameConfig struct {
	Name     string         `mapstructure:"name"`
	Settings map[string]any `mapstructure:"settings"` // If game also has settings
}
