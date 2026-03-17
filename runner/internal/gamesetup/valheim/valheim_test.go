package valheim

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/openhost/runner/internal/system"
	"github.com/openhost/runnerconfig"
)

type recordedCommand struct {
	name string
	args []string
}

type recordingExecutor struct {
	runs                 []recordedCommand
	combinedOutputs      []recordedCommand
	commandExistsResults map[string]bool
}

func (r *recordingExecutor) Run(_ context.Context, name string, args ...string) error {
	r.runs = append(r.runs, recordedCommand{name: name, args: append([]string(nil), args...)})
	return nil
}

func (r *recordingExecutor) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	r.combinedOutputs = append(r.combinedOutputs, recordedCommand{name: name, args: append([]string(nil), args...)})
	if name == "sh" && len(args) >= 4 && args[len(args)-2] == "sh" {
		if exists := r.commandExistsResults[args[len(args)-1]]; exists {
			return []byte("yes"), nil
		}
	}
	return nil, nil
}

func silentSystemManager(executor system.Executor) *system.Manager {
	return system.NewManager(executor, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

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

func TestProvisionSystemMatchesLegacyBootstrapSequence(t *testing.T) {
	t.Parallel()

	const serverRoot = "/srv/valheim"
	executor := &recordingExecutor{}
	manager := silentSystemManager(executor)

	if err := provisionSystem(context.Background(), manager, serverRoot); err != nil {
		t.Fatalf("provisionSystem returned error: %v", err)
	}

	got := make([]recordedCommand, len(executor.runs))
	copy(got, executor.runs)
	want := []recordedCommand{
		{name: "dpkg", args: []string{"--add-architecture", "i386"}},
		{name: "apt-get", args: []string{"update", "-y"}},
		{name: "apt-get", args: []string{"install", "-y", "software-properties-common"}},
		{name: "add-apt-repository", args: []string{"multiverse", "-y"}},
		{name: "add-apt-repository", args: []string{"universe", "-y"}},
		{name: "sh", args: []string{"-c", "printf '%s\\n' \"$1\" | debconf-set-selections", "sh", "steam steam/question select I AGREE"}},
		{name: "sh", args: []string{"-c", "printf '%s\\n' \"$1\" | debconf-set-selections", "sh", "steam steam/license note ''"}},
		{name: "apt-get", args: []string{"update", "-y"}},
		{name: "apt-get", args: []string{"install", "-y", "steamcmd", "screen", "libpulse0", "libatomic1", "lib32gcc-s1", "curl", "libpulse-dev", "libc6", "jq", "unzip"}},
		{name: "useradd", args: []string{"-m", "-s", "/bin/bash", "valheim"}},
		{name: "chown", args: []string{"-R", "valheim:valheim", "/home/valheim"}},
		{name: "sudo", args: []string{"-u", "valheim", "/usr/games/steamcmd", "+login", "anonymous", "+quit"}},
		{name: "sudo", args: []string{"-u", "valheim", "/usr/games/steamcmd", "+force_install_dir", serverRoot, "+login", "anonymous", "+app_update", appID, "validate", "+quit"}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command sequence\n got: %#v\nwant: %#v", got, want)
	}
	if len(executor.combinedOutputs) != 0 {
		t.Fatalf("expected no CombinedOutput calls during provisionSystem, got: %#v", executor.combinedOutputs)
	}
}

func TestConfigureFirewallSkipsWhenUFWUnavailable(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{commandExistsResults: map[string]bool{"ufw": false}}
	manager := silentSystemManager(executor)

	if err := configureFirewall(context.Background(), manager); err != nil {
		t.Fatalf("configureFirewall returned error: %v", err)
	}
	if len(executor.runs) != 0 {
		t.Fatalf("expected no firewall commands when ufw is unavailable, got: %#v", executor.runs)
	}
	if len(executor.combinedOutputs) != 1 {
		t.Fatalf("expected one command existence check, got: %#v", executor.combinedOutputs)
	}
}

func TestConfigureFirewallAppliesLegacyCommandsWhenUFWAvailable(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{commandExistsResults: map[string]bool{"ufw": true}}
	manager := silentSystemManager(executor)

	if err := configureFirewall(context.Background(), manager); err != nil {
		t.Fatalf("configureFirewall returned error: %v", err)
	}

	want := []recordedCommand{
		{name: "ufw", args: []string{"allow", "2456:2458/udp"}},
		{name: "ufw", args: []string{"reload"}},
	}
	if !reflect.DeepEqual(executor.runs, want) {
		t.Fatalf("unexpected firewall commands\n got: %#v\nwant: %#v", executor.runs, want)
	}
}
