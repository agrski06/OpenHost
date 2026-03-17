package system

import (
	"context"
	"sort"
)

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
	if err := m.Executor.Run(ctx, "apt-get", "update", "-y"); err != nil {
		return err
	}
	args := append([]string{"install", "-y"}, ordered...)
	return m.Executor.Run(ctx, "apt-get", args...)
}
