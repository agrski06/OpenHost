package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/openhost/runner/internal/gamesetup/valheim"
	_ "github.com/openhost/runner/internal/modframework/bepinex"
	_ "github.com/openhost/runner/internal/modprovider/thunderstore"
	"github.com/openhost/runner/internal/setup"
	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	var (
		configPath      string
		localMode       bool
		skipServerStart bool
		showVersion     bool
	)

	flag.StringVar(&configPath, "config", "", "path to runner config JSON")
	flag.BoolVar(&localMode, "local", false, "enable local debug mode")
	flag.BoolVar(&skipServerStart, "skip-server-start", false, "skip automatic server start")
	flag.BoolVar(&showVersion, "version", false, "print runner version")
	flag.Parse()

	if showVersion {
		_, _ = fmt.Fprintln(os.Stdout, version)
		return 0
	}
	if configPath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "openhost-runner: --config is required")
		return 2
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Error("load config", "path", configPath, "error", err)
		return 1
	}
	if localMode {
		cfg.Debug.LocalMode = true
	}
	if skipServerStart {
		cfg.Debug.SkipServerStart = true
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	orchestrator := setup.NewOrchestrator(logger, httpClient, system.NewManager(nil, logger))
	if err := orchestrator.Run(context.Background(), cfg); err != nil {
		logger.Error("runner failed", "error", err)
		return 1
	}

	logger.Info("runner completed", "game", cfg.Game.Name, "version", version)
	return 0
}

func loadConfig(path string) (runnerconfig.RunnerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runnerconfig.RunnerConfig{}, err
	}

	var cfg runnerconfig.RunnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return runnerconfig.RunnerConfig{}, err
	}
	return cfg, nil
}
