// Package core defines the runner-side interfaces for game setup, mod
// providers, and mod frameworks.
package core

import (
	"context"

	"github.com/openhost/runnerconfig"
)

// GameSetup describes everything the runner needs to know about a game in
// order to install and launch it on a VPS.
type GameSetup interface {
	// Name returns the game identifier (e.g. "valheim", "minecraft").
	Name() string

	// SystemUser returns the OS user that runs the game server process.
	SystemUser() string

	// RequiredPackages returns apt packages that must be installed.
	RequiredPackages() []string

	// InstallMethod returns the installer for the game server binary.
	// May return nil if the install is fully config-driven (resolved by the
	// orchestrator from RunnerConfig.Game.Install).
	InstallMethod() InstallMethod

	// ModFramework returns the name of the mod framework this game uses
	// (e.g. "bepinex", "fabric", "forge"). Empty string means no mod framework.
	ModFramework() string

	// BuildLaunchCommand builds the service/process configuration for the game.
	BuildLaunchCommand(cfg runnerconfig.GameConfig, paths runnerconfig.ServerPaths) LaunchConfig

	// ServerPaths returns the filesystem layout for this game.
	ServerPaths() runnerconfig.ServerPaths
}

// InstallMethod abstracts game server installation (SteamCMD, HTTP download, etc.).
type InstallMethod interface {
	Install(ctx context.Context, user string, serverRoot string) error
}

// LaunchConfig describes how to run the game server as a systemd service.
type LaunchConfig struct {
	// ServiceName is the systemd unit name (e.g. "openhost-valheim").
	ServiceName string

	// User is the OS user to run the service as.
	User string

	// WorkingDir is the working directory for the service.
	WorkingDir string

	// ExecStart is the command line to execute.
	ExecStart string

	// Environment holds KEY=VALUE pairs for the service environment.
	Environment map[string]string

	// RestartPolicy is the systemd restart policy ("always", "on-failure", "no").
	RestartPolicy string
}
