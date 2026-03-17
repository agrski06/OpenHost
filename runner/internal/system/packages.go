package system

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func (m *Manager) AddArchitecture(ctx context.Context, architecture string) error {
	if strings.TrimSpace(architecture) == "" {
		return nil
	}

	m.Logger.Info("adding dpkg architecture", "architecture", architecture)
	return m.Executor.Run(ctx, "dpkg", "--add-architecture", architecture)
}

func (m *Manager) AddAptRepository(ctx context.Context, repository string) error {
	if strings.TrimSpace(repository) == "" {
		return nil
	}

	m.Logger.Info("adding apt repository", "repository", repository)
	return m.Executor.Run(ctx, "add-apt-repository", repository, "-y")
}

func (m *Manager) PreseedDebconfSelection(ctx context.Context, selection string) error {
	if strings.TrimSpace(selection) == "" {
		return nil
	}

	m.Logger.Info("preseeding debconf selection")
	return m.Executor.Run(ctx, "sh", "-c", "printf '%s\\n' \"$1\" | debconf-set-selections", "sh", selection)
}

func (m *Manager) AptUpdate(ctx context.Context) error {
	m.Logger.Info("updating apt package indexes")
	return m.Executor.Run(ctx, "apt-get", "update", "-y")
}

func (m *Manager) InstallAptPackages(ctx context.Context, packages ...string) error {
	filtered := make([]string, 0, len(packages))
	for _, pkg := range packages {
		if strings.TrimSpace(pkg) == "" {
			continue
		}
		filtered = append(filtered, pkg)
	}
	if len(filtered) == 0 {
		return nil
	}

	m.Logger.Info("installing apt packages", "packages", filtered)
	args := append([]string{"install", "-y"}, filtered...)
	return m.Executor.Run(ctx, "apt-get", args...)
}

func (m *Manager) EnsureAptPackages(ctx context.Context, packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	unique := map[string]struct{}{}
	for _, pkg := range packages {
		if pkg == "" {
			continue
		}
		unique[pkg] = struct{}{}
	}

	ordered := make([]string, 0, len(unique))
	for pkg := range unique {
		ordered = append(ordered, pkg)
	}
	sort.Strings(ordered)

	m.Logger.Info("ensuring apt packages", "packages", ordered)
	if err := m.AptUpdate(ctx); err != nil {
		return fmt.Errorf("apt update: %w", err)
	}
	if err := m.InstallAptPackages(ctx, ordered...); err != nil {
		return fmt.Errorf("install apt packages: %w", err)
	}
	return nil
}
