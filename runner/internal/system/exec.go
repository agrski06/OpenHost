package system

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type Executor interface {
	Run(ctx context.Context, name string, args ...string) error
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (OSExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("run %s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type Manager struct {
	Executor Executor
	Logger   *slog.Logger
}

func NewManager(executor Executor, logger *slog.Logger) *Manager {
	if executor == nil {
		executor = OSExecutor{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{Executor: executor, Logger: logger}
}
