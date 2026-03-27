package bepinex

import (
	"fmt"
	"os"
	"path/filepath"
)

// Validate checks that the BepInEx mod framework was installed correctly
// by verifying critical files and directories exist.
func (f *Framework) Validate(serverRoot string) error {
	preloaderPath := filepath.Join(serverRoot, "BepInEx", "core", "BepInEx.Preloader.dll")
	if _, err := os.Stat(preloaderPath); os.IsNotExist(err) {
		return fmt.Errorf("BepInEx validation failed: preloader not found at %s", preloaderPath)
	}

	doorstopDir := filepath.Join(serverRoot, "doorstop_libs")
	if _, err := os.Stat(doorstopDir); os.IsNotExist(err) {
		return fmt.Errorf("BepInEx validation failed: doorstop_libs directory not found at %s", doorstopDir)
	}

	return nil
}
