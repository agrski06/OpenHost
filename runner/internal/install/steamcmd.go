// Package install provides InstallMethod implementations for installing game
// server binaries via SteamCMD or HTTP download.
package install

import (
	"context"
	"fmt"

	"github.com/openhost/runner/internal/system"
)

// SteamCMDInstall installs a game server via SteamCMD.
type SteamCMDInstall struct {
	AppID      string
	Anonymous  bool
	BetaBranch string
}

// Install ensures SteamCMD is available and downloads the app.
func (s *SteamCMDInstall) Install(_ context.Context, user, serverRoot string) error {
	if err := system.EnsureSteamCMD(); err != nil {
		return fmt.Errorf("ensure steamcmd: %w", err)
	}

	if err := system.InstallApp(user, s.AppID, serverRoot, s.Anonymous); err != nil {
		return fmt.Errorf("steamcmd install app %s: %w", s.AppID, err)
	}

	return nil
}
