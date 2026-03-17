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
	Wants            string
	After            string
	Type             string
	WorkingDirectory string
	ExecStart        string
	User             string
	Group            string
	Environment      map[string]string
	Restart          string
	RestartSec       string
	KillSignal       string
	TimeoutStopSec   string
	LimitNOFILE      string
	StandardOutput   string
	StandardError    string
}

func (u Unit) Render() string {
	var builder strings.Builder
	builder.WriteString("[Unit]\n")
	builder.WriteString("Description=" + u.Description + "\n")
	if u.Wants != "" {
		builder.WriteString("Wants=" + u.Wants + "\n")
	}
	if u.After != "" {
		builder.WriteString("After=" + u.After + "\n")
	} else {
		builder.WriteString("After=network.target\n")
	}
	builder.WriteString("\n[Service]\n")
	if u.Type != "" {
		builder.WriteString("Type=" + u.Type + "\n")
	}
	if u.User != "" {
		builder.WriteString("User=" + u.User + "\n")
	}
	if u.Group != "" {
		builder.WriteString("Group=" + u.Group + "\n")
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
	if u.RestartSec != "" {
		builder.WriteString("RestartSec=" + u.RestartSec + "\n")
	}
	if u.KillSignal != "" {
		builder.WriteString("KillSignal=" + u.KillSignal + "\n")
	}
	if u.TimeoutStopSec != "" {
		builder.WriteString("TimeoutStopSec=" + u.TimeoutStopSec + "\n")
	}
	if u.LimitNOFILE != "" {
		builder.WriteString("LimitNOFILE=" + u.LimitNOFILE + "\n")
	}
	if u.StandardOutput != "" {
		builder.WriteString("StandardOutput=" + u.StandardOutput + "\n")
	}
	if u.StandardError != "" {
		builder.WriteString("StandardError=" + u.StandardError + "\n")
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
