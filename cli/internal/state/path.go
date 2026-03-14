package state

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const stateDirEnvVar = "OPENHOST_STATE_DIR"

func resolveDefaultStatePath() (string, error) {
	baseDir, err := resolveStateDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "instances.json"), nil
}

func resolveStateDir() (string, error) {
	if override := os.Getenv(stateDirEnvVar); override != "" {
		return override, nil
	}

	return stateDirForGOOS(runtime.GOOS)
}

func stateDirForGOOS(goos string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}

	switch goos {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "OpenHost"), nil
		}
		return filepath.Join(homeDir, "AppData", "Local", "OpenHost"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "OpenHost"), nil
	case "linux":
		if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
			return filepath.Join(xdgStateHome, "openhost"), nil
		}
		return filepath.Join(homeDir, ".local", "state", "openhost"), nil
	default:
		return filepath.Join(homeDir, ".openhost"), nil
	}
}
