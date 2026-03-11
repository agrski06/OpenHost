package config

type ServerConfig struct {
	Name       string           `mapstructure:"name"`
	Provider   ProviderConfig   `mapstructure:"provider"`
	Game       GameConfig       `mapstructure:"game"`
	Automation AutomationConfig `mapstructure:"automation"`
}

type ProviderConfig struct {
	Name   string `mapstructure:"name"`
	Region string `mapstructure:"region"`
	Plan   string `mapstructure:"plan"`
}

type GameConfig struct {
	Type        string `mapstructure:"type"`
	Image       string `mapstructure:"image"`
	Persistence string `mapstructure:"persistence"`
}

type AutomationConfig struct {
	AutoStopTimeout string `mapstructure:"auto-stop-timeout"`
	Trigger         string `mapstructure:"trigger"`
}
