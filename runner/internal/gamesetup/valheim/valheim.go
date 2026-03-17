package valheim

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/modframework/bepinex"
	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

const (
	appID           = "896660"
	serviceName     = "openhost-valheim.service"
	startupScript   = "start_valheim_custom.sh"
	defaultUser     = "valheim"
	defaultHomeRoot = "/home/valheim"
	defaultName     = "OpenHost-Valheim"
	defaultWorld    = "Dedicated"
	defaultUDPFrom  = 2456
	defaultUDPTo    = 2458
)

type GameSetup struct{}

type Settings struct {
	World    string `json:"world"`
	Password string `json:"password"`
}

func New() *GameSetup { return &GameSetup{} }

func (g *GameSetup) Name() string { return "valheim" }

func (g *GameSetup) Setup(ctx context.Context, cfg runnerconfig.RunnerConfig, env *core.SetupEnvironment) error {
	if env == nil {
		env = &core.SetupEnvironment{}
	}
	if env.System == nil {
		return fmt.Errorf("system manager is required")
	}

	settings, err := decodeSettings(cfg.Game.Settings)
	if err != nil {
		return fmt.Errorf("decode valheim settings: %w", err)
	}
	if settings.World == "" {
		settings.World = defaultWorld
	}
	if strings.TrimSpace(cfg.Server.ServerRoot) == "" || strings.TrimSpace(cfg.Server.SaveRoot) == "" || strings.TrimSpace(cfg.Server.ModpackRoot) == "" {
		return fmt.Errorf("server_root, save_root, and modpack_root are required")
	}

	logger := env.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if err := ensureDirectories(cfg.Server); err != nil {
		return err
	}

	if cfg.Debug.LocalMode {
		logger.Info("local mode enabled; skipping apt, steamcmd, firewall, user creation, and systemd operations")
	} else {
		if err := provisionSystem(ctx, env.System, cfg.Server.ServerRoot); err != nil {
			return err
		}
	}

	if cfg.Game.Mods != nil && len(cfg.Game.Mods.Sources) > 0 {
		if err := installMods(ctx, cfg, env); err != nil {
			return err
		}
	}

	startupPath, err := writeStartupScript(cfg.Server, settings)
	if err != nil {
		return err
	}
	logger.Info("wrote startup script", "path", startupPath)

	if cfg.Debug.LocalMode {
		logger.Info("local mode skipping systemd service creation", "script", startupPath)
		return nil
	}

	if err := env.System.ChownR(ctx, cfg.Server.ServerRoot, defaultUser); err != nil {
		return fmt.Errorf("chown server root: %w", err)
	}
	if err := env.System.ChownR(ctx, cfg.Server.SaveRoot, defaultUser); err != nil {
		return fmt.Errorf("chown save root: %w", err)
	}
	if err := configureFirewall(ctx, env.System); err != nil {
		return err
	}
	if err := writeSystemdService(ctx, env.System, cfg.Server.ServerRoot, startupPath); err != nil {
		return err
	}
	if cfg.Debug.SkipServerStart {
		logger.Info("skip_server_start enabled; leaving Valheim service disabled at runtime")
		return nil
	}
	if err := env.System.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("start %s: %w", serviceName, err)
	}
	return nil
}

func decodeSettings(raw map[string]any) (Settings, error) {
	payload, err := json.Marshal(raw)
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	if err := json.Unmarshal(payload, &settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func ensureDirectories(paths runnerconfig.ServerPaths) error {
	for _, dir := range []string{paths.ServerRoot, paths.SaveRoot, paths.ModpackRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}
	return nil
}

func provisionSystem(ctx context.Context, manager *system.Manager, serverRoot string) error {
	if manager == nil {
		return fmt.Errorf("system manager is required")
	}
	if err := manager.AddArchitecture(ctx, "i386"); err != nil {
		return fmt.Errorf("add dpkg architecture: %w", err)
	}
	if err := manager.AptUpdate(ctx); err != nil {
		return fmt.Errorf("apt update: %w", err)
	}
	if err := manager.InstallAptPackages(ctx, "software-properties-common"); err != nil {
		return fmt.Errorf("install software-properties-common: %w", err)
	}
	if err := manager.AddAptRepository(ctx, "multiverse"); err != nil {
		return fmt.Errorf("add multiverse repository: %w", err)
	}
	if err := manager.AddAptRepository(ctx, "universe"); err != nil {
		return fmt.Errorf("add universe repository: %w", err)
	}
	if err := manager.PreseedDebconfSelection(ctx, "steam steam/question select I AGREE"); err != nil {
		return fmt.Errorf("preseed steam question: %w", err)
	}
	if err := manager.PreseedDebconfSelection(ctx, "steam steam/license note ''"); err != nil {
		return fmt.Errorf("preseed steam license: %w", err)
	}
	if err := manager.AptUpdate(ctx); err != nil {
		return fmt.Errorf("apt update (post-repo): %w", err)
	}
	if err := manager.InstallAptPackages(ctx,
		"steamcmd",
		"screen",
		"libpulse0",
		"libatomic1",
		"lib32gcc-s1",
		"curl",
		"libpulse-dev",
		"libc6",
		"jq",
		"unzip",
	); err != nil {
		return fmt.Errorf("install valheim packages: %w", err)
	}
	if err := manager.CreateUser(ctx, defaultUser); err != nil {
		return fmt.Errorf("create user %q: %w", defaultUser, err)
	}
	if err := manager.ChownR(ctx, defaultHomeRoot, defaultUser); err != nil {
		return fmt.Errorf("chown %s: %w", defaultHomeRoot, err)
	}
	if err := manager.SteamCMDAnonymousLogin(ctx, defaultUser); err != nil {
		return fmt.Errorf("warm steamcmd: %w", err)
	}
	if err := manager.SteamAppUpdateAsUser(ctx, defaultUser, appID, serverRoot); err != nil {
		return fmt.Errorf("install/update valheim server: %w", err)
	}
	return nil
}

func configureFirewall(ctx context.Context, manager *system.Manager) error {
	if manager == nil {
		return fmt.Errorf("system manager is required")
	}

	hasUFW, err := manager.CommandExists(ctx, "ufw")
	if err != nil {
		return fmt.Errorf("check ufw availability: %w", err)
	}
	if !hasUFW {
		return nil
	}
	if err := manager.AllowUDPRange(ctx, defaultUDPFrom, defaultUDPTo); err != nil {
		return fmt.Errorf("configure firewall: %w", err)
	}
	if err := manager.ReloadFirewall(ctx); err != nil {
		return fmt.Errorf("reload firewall: %w", err)
	}
	return nil
}

func installMods(ctx context.Context, cfg runnerconfig.RunnerConfig, env *core.SetupEnvironment) error {
	logger := slog.Default()
	if env != nil && env.Logger != nil {
		logger = env.Logger
	}

	framework, err := core.GetModFramework("bepinex")
	if err != nil {
		return err
	}

	for _, source := range cfg.Game.Mods.Sources {
		provider, err := core.GetModProvider(source.Provider)
		if err != nil {
			return err
		}
		resolved, err := provider.Resolve(ctx, source)
		if err != nil {
			return fmt.Errorf("resolve %s mod source: %w", source.Provider, err)
		}
		if len(resolved.OverlayArchive) > 0 {
			if err := framework.ApplyOverlay(cfg.Server.ServerRoot, resolved.OverlayArchive); err != nil {
				return fmt.Errorf("apply overlay from %s: %w", source.Provider, err)
			}
		}
		for _, pkg := range resolved.Packages {
			logger.Info("installing mod package", "provider", source.Provider, "package", pkg.String())
			archive, err := provider.DownloadPackage(ctx, pkg)
			if err != nil {
				return fmt.Errorf("download package %s: %w", pkg.String(), err)
			}
			if err := framework.InstallPackage(cfg.Server.ServerRoot, pkg, archive); err != nil {
				return fmt.Errorf("install package %s: %w", pkg.String(), err)
			}
		}
	}
	return nil
}

func writeStartupScript(paths runnerconfig.ServerPaths, settings Settings) (string, error) {
	status := bepinex.ValidateServerRoot(paths.ServerRoot)
	launcherName := filepath.Base(status.Launcher)
	if launcherName == "." || launcherName == string(filepath.Separator) || launcherName == "" {
		launcherName = "valheim_server.x86_64"
	}

	scriptPath := filepath.Join(paths.ServerRoot, startupScript)
	content := fmt.Sprintf("#!/bin/bash\nset -euo pipefail\ncd %q\nexec ./%s -name %q -port %d -world %q -password %q -savedir %q -public 1\n",
		paths.ServerRoot,
		launcherName,
		defaultName,
		defaultUDPFrom,
		settings.World,
		settings.Password,
		paths.SaveRoot,
	)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		return "", fmt.Errorf("write startup script %q: %w", scriptPath, err)
	}
	return scriptPath, nil
}

func writeSystemdService(ctx context.Context, manager *system.Manager, serverRoot string, startupPath string) error {
	if _, err := manager.WriteService(serviceName, system.Unit{
		Description:      "OpenHost Valheim dedicated server",
		WorkingDirectory: serverRoot,
		ExecStart:        startupPath,
		User:             defaultUser,
		Restart:          "on-failure",
	}); err != nil {
		return fmt.Errorf("write %s: %w", serviceName, err)
	}
	if err := manager.DaemonReload(ctx); err != nil {
		return fmt.Errorf("systemd daemon-reload: %w", err)
	}
	if err := manager.EnableService(ctx, serviceName); err != nil {
		return fmt.Errorf("enable %s: %w", serviceName, err)
	}
	return nil
}

func init() {
	core.RegisterGameSetup("valheim", func() core.GameSetup { return New() })
}
