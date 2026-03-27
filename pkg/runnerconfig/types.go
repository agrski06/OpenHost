// Package runnerconfig defines the JSON contract between the CLI (which
// serializes the config) and the runner binary (which reads it on the target
// VPS). Both cli/go.mod and runner/go.mod reference this package via replace
// directives pointing at ../pkg/runnerconfig.
package runnerconfig

// RunnerConfig is the top-level configuration blob serialized by the CLI and
// consumed by the runner binary on the target VPS.
type RunnerConfig struct {
	// Version is the schema version for forward compatibility.
	Version string `json:"version"`

	// Game describes the game to install and its settings.
	Game GameConfig `json:"game"`

	// Server describes filesystem paths on the target VPS.
	Server ServerPaths `json:"server"`

	// Debug contains optional flags for local development/testing.
	Debug DebugConfig `json:"debug,omitempty"`
}

// GameConfig holds the game name, game-specific settings, and optional mod
// configuration.
type GameConfig struct {
	// Name is the game identifier, e.g. "valheim", "minecraft".
	Name string `json:"name"`

	// Settings holds game-specific key/value pairs (world name, password, memory, etc.).
	Settings map[string]any `json:"settings"`

	// Mods is optional mod configuration. Nil means no mods.
	Mods *ModConfig `json:"mods,omitempty"`

	// Install describes how to obtain the game server binary.
	Install InstallConfig `json:"install"`
}

// InstallConfig describes the installation method for the game server binary.
type InstallConfig struct {
	// Method is "steamcmd" or "http".
	Method string `json:"method"`

	// SteamCMD-specific fields.
	SteamAppID string `json:"steam_app_id,omitempty"`
	Anonymous  bool   `json:"anonymous,omitempty"`
	BetaBranch string `json:"beta_branch,omitempty"`

	// HTTP-download-specific fields.
	DownloadURL  string `json:"download_url,omitempty"`
	DestFilename string `json:"dest_filename,omitempty"`
	ExtractZip   bool   `json:"extract_zip,omitempty"`
	ExtractTar   bool   `json:"extract_tar,omitempty"`
}

// ModConfig is provider-agnostic. Each game declares which mod sources it uses.
// The runner resolves the right ModProvider implementation from the source name.
type ModConfig struct {
	// Sources lists one or more mod sources to install from.
	Sources []ModSource `json:"sources"`
}

// ModSource describes a single mod provider + lookup key.
// This is the extensibility point for adding new mod providers (Thunderstore,
// CurseForge, Modrinth, Steam Workshop, etc.).
type ModSource struct {
	// Provider is the mod provider name, e.g. "thunderstore", "curseforge",
	// "modrinth", "workshop".
	Provider string `json:"provider"`

	// Code is a profile/collection code (e.g. a Thunderstore export code).
	Code string `json:"code,omitempty"`

	// Settings holds provider-specific extra configuration.
	Settings map[string]any `json:"settings,omitempty"`
}

// ServerPaths describes filesystem layout on the target VPS.
type ServerPaths struct {
	// ServerRoot is where the game server binary/files are installed.
	ServerRoot string `json:"server_root"`

	// SaveRoot is where game save/world data lives.
	SaveRoot string `json:"save_root"`

	// ModpackRoot is where extracted mod files are staged.
	ModpackRoot string `json:"modpack_root"`
}

// DebugConfig contains optional flags for local development and testing.
type DebugConfig struct {
	// LocalMode skips apt, steamcmd, user creation — useful for testing on a
	// developer machine.
	LocalMode bool `json:"local_mode"`

	// SkipServerStart prevents the game server process from launching.
	SkipServerStart bool `json:"skip_server_start"`
}
