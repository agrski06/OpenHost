package setup

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

type Orchestrator struct {
	logger     *slog.Logger
	httpClient *http.Client
	system     *system.Manager
}

func NewOrchestrator(logger *slog.Logger, httpClient *http.Client, systemManager *system.Manager) *Orchestrator {
	if logger == nil {
		logger = slog.Default()
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if systemManager == nil {
		systemManager = system.NewManager(nil, logger)
	}
	return &Orchestrator{logger: logger, httpClient: httpClient, system: systemManager}
}

func (o *Orchestrator) Run(ctx context.Context, cfg runnerconfig.RunnerConfig) error {
	if cfg.Version != "1" {
		return fmt.Errorf("unsupported config version %q; this runner supports version 1", cfg.Version)
	}
	if cfg.Game.Name == "" {
		return fmt.Errorf("game.name is required")
	}

	gameSetup, err := core.GetGameSetup(cfg.Game.Name)
	if err != nil {
		return err
	}

	o.logger.Info("starting game setup", "game", cfg.Game.Name, "local_mode", cfg.Debug.LocalMode, "skip_server_start", cfg.Debug.SkipServerStart)
	return gameSetup.Setup(ctx, cfg, &core.SetupEnvironment{
		Logger:     o.logger,
		HTTPClient: o.httpClient,
		System:     o.system,
	})
}
