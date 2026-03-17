package core

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

type SetupEnvironment struct {
	Logger     *slog.Logger
	HTTPClient *http.Client
	System     *system.Manager
}

type GameSetup interface {
	Name() string
	Setup(ctx context.Context, cfg runnerconfig.RunnerConfig, env *SetupEnvironment) error
}
