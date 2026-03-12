package config

type ServerConfig struct {
	Name     string         `mapstructure:"name"`
	Provider ProviderConfig `mapstructure:"provider"`
	Game     GameConfig     `mapstructure:"game"`
}

type ProviderConfig struct {
	Name string `mapstructure:"name"`
	Plan string `mapstructure:"plan"`
}

type GameConfig struct {
	Type string `mapstructure:"type"`
}
