package bepinex

import (
	"os"
	"path/filepath"
)

type RuntimeStatus struct {
	Launcher       string
	HasPreloader   bool
	HasDoorstopLib bool
}

func ValidateServerRoot(serverRoot string) RuntimeStatus {
	status := RuntimeStatus{}
	for _, launcher := range []string{"start_server_bepinex.sh", "start_game_bepinex.sh", "valheim_server.x86_64"} {
		candidate := filepath.Join(serverRoot, launcher)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			status.Launcher = candidate
			break
		}
	}
	if _, err := os.Stat(filepath.Join(serverRoot, "BepInEx", "core", "BepInEx.Preloader.dll")); err == nil {
		status.HasPreloader = true
	}
	if _, err := os.Stat(filepath.Join(serverRoot, "doorstop_libs")); err == nil {
		status.HasDoorstopLib = true
	}
	return status
}
