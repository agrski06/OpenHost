package runnerconfig

type RunnerConfig struct {
	Version string      `json:"version"`
	Game    GameConfig  `json:"game"`
	Server  ServerPaths `json:"server"`
	Debug   DebugConfig `json:"debug,omitempty"`
}

type GameConfig struct {
	Name     string         `json:"name"`
	Settings map[string]any `json:"settings"`
	Mods     *ModConfig     `json:"mods,omitempty"`
}

type ModConfig struct {
	Sources []ModSource `json:"sources"`
}

type ModSource struct {
	Provider string         `json:"provider"`
	Code     string         `json:"code,omitempty"`
	Settings map[string]any `json:"settings,omitempty"`
}

type ServerPaths struct {
	ServerRoot  string `json:"server_root"`
	SaveRoot    string `json:"save_root"`
	ModpackRoot string `json:"modpack_root"`
}

type DebugConfig struct {
	LocalMode       bool `json:"local_mode,omitempty"`
	SkipServerStart bool `json:"skip_server_start,omitempty"`
}
