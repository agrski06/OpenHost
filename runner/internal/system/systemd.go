package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Unit struct {
	Description      string
	WorkingDirectory string
	ExecStart        string
	User             string
	Environment      map[string]string
	Restart          string
}

func (u Unit) Render() string {
	var builder strings.Builder
	builder.WriteString("[Unit]\n")
	builder.WriteString("Description=" + u.Description + "\n")
	builder.WriteString("After=network.target\n\n")
	builder.WriteString("[Service]\n")
	if u.User != "" {
		builder.WriteString("User=" + u.User + "\n")
	}
	if u.WorkingDirectory != "" {
		builder.WriteString("WorkingDirectory=" + u.WorkingDirectory + "\n")
	}
	keys := make([]string, 0, len(u.Environment))
	for key := range u.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := u.Environment[key]
		builder.WriteString(fmt.Sprintf("Environment=%s=%s\n", key, value))
	}
	builder.WriteString("ExecStart=" + u.ExecStart + "\n")
	if u.Restart != "" {
		builder.WriteString("Restart=" + u.Restart + "\n")
	}
	builder.WriteString("\n[Install]\nWantedBy=multi-user.target\n")
	return builder.String()
}

func (m *Manager) WriteService(unitName string, unit Unit) (string, error) {
	if unitName == "" {
		return "", fmt.Errorf("unit name is required")
	}
	path := filepath.Join("/etc/systemd/system", unitName)
	m.Logger.Info("writing systemd unit", "path", path)
	if err := os.WriteFile(path, []byte(unit.Render()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) EnableService(ctx context.Context, unitName string) error {
	return m.Executor.Run(ctx, "systemctl", "enable", unitName)
}

func (m *Manager) DaemonReload(ctx context.Context) error {
	return m.Executor.Run(ctx, "systemctl", "daemon-reload")
}

func (m *Manager) StartService(ctx context.Context, unitName string) error {
	return m.Executor.Run(ctx, "systemctl", "start", unitName)
}

func (m *Manager) RestartService(ctx context.Context, unitName string) error {
	return m.Executor.Run(ctx, "systemctl", "restart", unitName)
}
