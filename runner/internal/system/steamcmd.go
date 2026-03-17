package system

import "context"

func (m *Manager) EnsureSteamCMD(ctx context.Context) error {
	m.Logger.Info("installing steamcmd dependencies")
	if err := m.EnsureAptPackages(ctx, "steamcmd", "lib32gcc-s1", "ca-certificates"); err != nil {
		return err
	}
	return nil
}

func (m *Manager) SteamAppUpdate(ctx context.Context, appID string, installDir string) error {
	m.Logger.Info("running steamcmd app_update", "app_id", appID, "install_dir", installDir)
	return m.Executor.Run(ctx, "steamcmd", "+force_install_dir", installDir, "+login", "anonymous", "+app_update", appID, "validate", "+quit")
}
