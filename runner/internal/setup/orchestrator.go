// Package setup implements the runner's orchestration pipeline: resolve game,
// install packages, create user, install game server, handle mods, launch.
package setup

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/install"
	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

// Run executes the full runner pipeline for the given configuration.
func Run(cfg *runnerconfig.RunnerConfig) error {
	ctx := context.Background()

	// 1. Resolve GameSetup from registry.
	game, err := core.GetGameSetup(cfg.Game.Name)
	if err != nil {
		return fmt.Errorf("resolve game setup: %w", err)
	}

	// Use paths from config if provided, otherwise fall back to game defaults.
	paths := cfg.Server
	if paths.ServerRoot == "" || paths.SaveRoot == "" || paths.ModpackRoot == "" {
		defaults := game.ServerPaths()
		if paths.ServerRoot == "" {
			paths.ServerRoot = defaults.ServerRoot
		}
		if paths.SaveRoot == "" {
			paths.SaveRoot = defaults.SaveRoot
		}
		if paths.ModpackRoot == "" {
			paths.ModpackRoot = defaults.ModpackRoot
		}
	}
	user := game.SystemUser()

	// 2. System bootstrap (skip in local/debug mode).
	if !cfg.Debug.LocalMode {
		log.Println("[setup] installing system packages...")
		if err := system.InstallPackages(game.RequiredPackages()); err != nil {
			return fmt.Errorf("install packages: %w", err)
		}

		log.Println("[setup] creating system user:", user)
		if err := system.CreateUser(user); err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		// Configure firewall from RunnerConfig or game defaults.
		log.Println("[setup] configuring firewall...")
		// Firewall rules could be derived from config; for now use game defaults.
	}

	// 3. Create directories.
	for _, dir := range []string{paths.ServerRoot, paths.SaveRoot, paths.ModpackRoot} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// 4. Install game server binary.
	installer := resolveInstallMethod(cfg.Game.Install, game)
	if installer != nil {
		log.Println("[setup] installing game server...")
		if err := installer.Install(ctx, user, paths.ServerRoot); err != nil {
			return fmt.Errorf("install game server: %w", err)
		}
	}

	// 5. Mod pipeline (wired in Phase 4).
	if cfg.Game.Mods != nil {
		log.Println("[setup] processing mods...")
		if err := runModPipeline(ctx, cfg.Game.Mods, game, paths); err != nil {
			return fmt.Errorf("mod pipeline: %w", err)
		}
	}

	// 6. Build launch configuration.
	launchCfg := game.BuildLaunchCommand(cfg.Game, paths)

	// 7. Set file ownership.
	if !cfg.Debug.LocalMode {
		log.Println("[setup] setting file ownership...")
		for _, dir := range []string{paths.ServerRoot, paths.SaveRoot, paths.ModpackRoot} {
			if err := system.SetOwnership(user, dir); err != nil {
				return fmt.Errorf("set ownership on %s: %w", dir, err)
			}
		}
	}

	// 8. Start the game server.
	if !cfg.Debug.SkipServerStart {
		if cfg.Debug.LocalMode {
			log.Println("[setup] starting game server process directly:", launchCfg.ExecStart)
			if err := startProcessDirectly(launchCfg); err != nil {
				return fmt.Errorf("start process: %w", err)
			}
		} else {
			log.Println("[setup] starting game server service:", launchCfg.ServiceName)
			if err := system.CreateAndStartService(launchCfg); err != nil {
				return fmt.Errorf("start service: %w", err)
			}
		}
	}

	log.Println("[setup] done!")
	return nil
}

// resolveInstallMethod builds the appropriate InstallMethod from the config.
// Falls back to the game's own InstallMethod() if nothing is specified in config.
func resolveInstallMethod(cfg runnerconfig.InstallConfig, game core.GameSetup) core.InstallMethod {
	switch cfg.Method {
	case "steamcmd":
		return &install.SteamCMDInstall{
			AppID:      cfg.SteamAppID,
			Anonymous:  cfg.Anonymous,
			BetaBranch: cfg.BetaBranch,
		}
	case "http":
		return &install.HTTPDownloadInstall{
			URL:          cfg.DownloadURL,
			DestFilename: cfg.DestFilename,
			ExtractZip:   cfg.ExtractZip,
			ExtractTar:   cfg.ExtractTar,
		}
	default:
		return game.InstallMethod()
	}
}

// runModPipeline resolves, downloads, and installs mods for each source.
func runModPipeline(ctx context.Context, modCfg *runnerconfig.ModConfig, game core.GameSetup, paths runnerconfig.ServerPaths) error {
	for _, source := range modCfg.Sources {
		provider, err := core.GetModProvider(source.Provider)
		if err != nil {
			return fmt.Errorf("resolve mod provider %q: %w", source.Provider, err)
		}

		packages, err := provider.Resolve(ctx, source)
		if err != nil {
			return fmt.Errorf("resolve mods from %q: %w", source.Provider, err)
		}

		downloaded, err := downloadAll(packages, paths.ModpackRoot)
		if err != nil {
			return fmt.Errorf("download mods: %w", err)
		}

		frameworkName := game.ModFramework()
		if frameworkName != "" {
			framework, err := core.GetModFramework(frameworkName)
			if err != nil {
				return fmt.Errorf("resolve mod framework %q: %w", frameworkName, err)
			}

			if err := framework.Install(downloaded, paths.ServerRoot); err != nil {
				return fmt.Errorf("install mods via %q: %w", frameworkName, err)
			}

			if err := framework.Validate(paths.ServerRoot); err != nil {
				return fmt.Errorf("validate mod framework %q: %w", frameworkName, err)
			}
		}
	}

	return nil
}

// downloadAll downloads all packages into destDir and returns the results.
func downloadAll(packages []core.Package, destDir string) ([]core.DownloadedMod, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create mod dir %s: %w", destDir, err)
	}

	var downloaded []core.DownloadedMod
	for _, pkg := range packages {
		destPath := fmt.Sprintf("%s/%s.zip", destDir, pkg.Identifier)
		if err := system.DownloadWithRetry(pkg.URL, destPath, 3); err != nil {
			return nil, fmt.Errorf("download %s: %w", pkg.Identifier, err)
		}
		downloaded = append(downloaded, core.DownloadedMod{
			Identifier: pkg.Identifier,
			LocalPath:  destPath,
		})
	}

	return downloaded, nil
}

// startProcessDirectly launches the game server as a local process (for local/debug mode).
func startProcessDirectly(cfg core.LaunchConfig) error {
	parts := strings.Fields(cfg.ExecStart)
	if len(parts) == 0 {
		return fmt.Errorf("empty ExecStart command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = cfg.WorkingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for k, v := range cfg.Environment {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	log.Printf("[setup] launching: %s (workdir=%s)", cfg.ExecStart, cfg.WorkingDir)
	return cmd.Start()
}
