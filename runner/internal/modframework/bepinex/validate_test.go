package bepinex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateServerRootDetectsRuntimeArtifacts(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(serverRoot, "BepInEx", "core"), 0o755); err != nil {
		t.Fatalf("mkdir BepInEx core: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(serverRoot, "doorstop_libs"), 0o755); err != nil {
		t.Fatalf("mkdir doorstop libs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverRoot, "BepInEx", "core", "BepInEx.Preloader.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatalf("write preloader: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverRoot, "start_server_bepinex.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}

	status := ValidateServerRoot(serverRoot)
	if status.Launcher == "" {
		t.Fatal("expected launcher to be detected")
	}
	if !status.HasPreloader {
		t.Fatal("expected preloader to be detected")
	}
	if !status.HasDoorstopLib {
		t.Fatal("expected doorstop runtime to be detected")
	}
}
