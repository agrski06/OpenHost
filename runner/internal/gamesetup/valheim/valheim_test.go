package valheim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openhost/runnerconfig"
)

func TestWriteStartupScriptPrefersDetectedLauncher(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	saveRoot := filepath.Join(t.TempDir(), "saves")
	if err := os.MkdirAll(filepath.Join(serverRoot, "BepInEx", "core"), 0o755); err != nil {
		t.Fatalf("create BepInEx dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverRoot, "start_server_bepinex.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}

	scriptPath, err := writeStartupScript(runnerconfig.ServerPaths{
		ServerRoot:  serverRoot,
		SaveRoot:    saveRoot,
		ModpackRoot: filepath.Join(t.TempDir(), "modpack"),
	}, Settings{World: "DedicatedWorld", Password: "secret"})
	if err != nil {
		t.Fatalf("writeStartupScript returned error: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read startup script: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "./start_server_bepinex.sh") {
		t.Fatalf("expected script to use BepInEx launcher, got: %s", text)
	}
	if !strings.Contains(text, `-world "DedicatedWorld"`) && !strings.Contains(text, "-world \"DedicatedWorld\"") {
		t.Fatalf("expected world flag in script, got: %s", text)
	}
}

func TestWriteStartupScriptFallsBackToVanillaLauncher(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	saveRoot := filepath.Join(t.TempDir(), "saves")
	scriptPath, err := writeStartupScript(runnerconfig.ServerPaths{
		ServerRoot:  serverRoot,
		SaveRoot:    saveRoot,
		ModpackRoot: filepath.Join(t.TempDir(), "modpack"),
	}, Settings{World: "Dedicated", Password: "secret"})
	if err != nil {
		t.Fatalf("writeStartupScript returned error: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read startup script: %v", err)
	}
	if !strings.Contains(string(content), "./valheim_server.x86_64") {
		t.Fatalf("expected fallback vanilla launcher, got: %s", string(content))
	}
}
