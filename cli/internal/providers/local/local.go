// Package local implements a Provider that runs the game server on the current
// machine rather than provisioning a cloud VPS. It writes the RunnerConfig to a
// local file and invokes the runner binary (or the setup pipeline inline) with
// debug.local_mode enabled, skipping privileged operations like apt, user
// creation, and firewall changes.
package local

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
	"github.com/openhost/runnerconfig"
)

// Provider runs game servers on the local machine.
type Provider struct{}

// Settings holds local-provider-specific configuration from provider.settings.
type Settings struct {
	// WorkDir is the root working directory for the local server. Defaults to
	// ~/.openhost/local/<server-name>.
	WorkDir string `mapstructure:"work_dir"`

	// RunnerBin is an optional path to a pre-built openhost-runner binary. When
	// empty, the provider invokes the runner setup pipeline inline (by writing
	// config and running the binary found on PATH as "openhost-runner").
	RunnerBin string `mapstructure:"runner_bin"`

	// SkipServerStart if true, runs the setup pipeline but does not actually
	// start the game server process. Useful for testing the install flow.
	SkipServerStart bool `mapstructure:"skip_server_start"`
}

const pidFileName = "runner.pid"

func (p *Provider) Name() string { return "local" }

// CreateServer writes the runner config to the work directory and executes the
// runner binary locally. The RunnerConfig is patched to enable local_mode and
// override server paths to point at the work directory.
func (p *Provider) CreateServer(_ context.Context, request core.CreateServerRequest) (*core.Server, error) {
	if request.Name == "" {
		return nil, fmt.Errorf("local: server name cannot be empty")
	}
	if request.GameName == "" {
		return nil, fmt.Errorf("local: game name cannot be empty")
	}
	if request.UserData == "" {
		return nil, fmt.Errorf("local: user-data (runner config) cannot be empty")
	}

	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("local: decode settings: %w", err)
	}

	workDir, err := resolveWorkDir(settings.WorkDir, request.Name)
	if err != nil {
		return nil, fmt.Errorf("local: resolve work dir: %w", err)
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("local: create work dir %s: %w", workDir, err)
	}

	// Parse the RunnerConfig from user-data. The deploy flow serializes the
	// config into a cloud-init script; for local mode we need the raw JSON.
	// We re-derive it from the provider settings instead. The caller already
	// built the user-data through cloudinit.BuildUserData which embeds the
	// JSON. For local, we intercept the config from provider settings.
	//
	// However, the simpler approach: re-read the RunnerConfig that was
	// embedded in user-data. The user-data is a bash script with the JSON
	// between heredoc markers. Let's extract it.
	runnerCfg, err := extractRunnerConfigFromUserData(request.UserData)
	if err != nil {
		return nil, fmt.Errorf("local: extract runner config from user-data: %w", err)
	}

	// Patch for local mode.
	runnerCfg.Debug.LocalMode = true
	runnerCfg.Debug.SkipServerStart = settings.SkipServerStart

	// Override server paths to be under the work directory.
	runnerCfg.Server = runnerconfig.ServerPaths{
		ServerRoot:  filepath.Join(workDir, "server"),
		SaveRoot:    filepath.Join(workDir, "saves"),
		ModpackRoot: filepath.Join(workDir, "modpack"),
	}

	// Write the patched runner config to disk.
	configPath := filepath.Join(workDir, "runner-config.json")
	configJSON, err := json.MarshalIndent(runnerCfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("local: marshal runner config: %w", err)
	}
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("local: write runner config: %w", err)
	}

	// Resolve and invoke the runner binary.
	runnerBin, err := resolveRunnerBin(settings.RunnerBin)
	if err != nil {
		return nil, fmt.Errorf("local: resolve runner binary: %w", err)
	}

	logFile, logPath, err := openLogFile(workDir)
	if err != nil {
		return nil, fmt.Errorf("local: open log file: %w", err)
	}

	cmd := exec.Command(runnerBin, "-config", configPath)
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("local: start runner %q: %w", runnerBin, err)
	}

	// Persist the PID so we can track/stop it later.
	pidPath := filepath.Join(workDir, pidFileName)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		slog.Warn("local provider: failed to write PID file", "path", pidPath, "err", err)
	}

	// Wait for the runner to complete in a goroutine so we don't block the CLI.
	go func() {
		defer func() { _ = logFile.Close() }()
		if err := cmd.Wait(); err != nil {
			slog.Error("local provider: runner exited with error, check logs", "log", logPath, "err", err)
		}
	}()

	// Give the runner a moment to start before returning.
	time.Sleep(500 * time.Millisecond)

	localIP := detectLocalIP()

	return &core.Server{
		ID:       fmt.Sprintf("local-%s", request.Name),
		Provider: p.Name(),
		Name:     request.Name,
		PublicIP: localIP,
		AssociatedResources: []core.ResourceRef{
			{Type: "log", ID: logPath, Name: "runner.log"},
			{Type: "workdir", ID: workDir, Name: "work directory"},
		},
	}, nil
}

func (p *Provider) GetServerStatus(_ context.Context, id string) (*core.InfrastructureStatus, error) {
	if id == "" {
		return nil, fmt.Errorf("local: server id cannot be empty")
	}
	if !strings.HasPrefix(id, "local-") {
		return &core.InfrastructureStatus{
			ID:     id,
			State:  core.InfrastructureStateNotFound,
			Detail: "not a local server id",
		}, nil
	}

	name := strings.TrimPrefix(id, "local-")
	workDir, err := resolveWorkDir("", name)
	if err != nil {
		return &core.InfrastructureStatus{
			ID:     id,
			Name:   name,
			State:  core.InfrastructureStateError,
			Detail: fmt.Sprintf("resolve work dir: %v", err),
		}, nil
	}

	// Primary check: try connecting to the game port.
	if port, proto, ok := readGamePort(workDir); ok {
		if isPortListening(port, proto) {
			detail := fmt.Sprintf("local server listening on port %d/%s", port, proto)
			if pid, alive := readPIDFile(filepath.Join(workDir, "game.pid")); alive {
				detail = fmt.Sprintf("local server running (pid %d) on port %d/%s", pid, port, proto)
			}
			return &core.InfrastructureStatus{
				ID:       id,
				Name:     name,
				PublicIP: detectLocalIP(),
				State:    core.InfrastructureStateRunning,
				Detail:   detail,
			}, nil
		}
	}

	// Fallback: check PID files.
	for _, pidFile := range []string{"game.pid", pidFileName} {
		if pid, alive := readPIDFile(filepath.Join(workDir, pidFile)); alive {
			return &core.InfrastructureStatus{
				ID:       id,
				Name:     name,
				PublicIP: detectLocalIP(),
				State:    core.InfrastructureStateRunning,
				Detail:   fmt.Sprintf("local server running (pid %d via %s)", pid, pidFile),
			}, nil
		}
	}

	return &core.InfrastructureStatus{
		ID:       id,
		Name:     name,
		PublicIP: detectLocalIP(),
		State:    core.InfrastructureStateStopped,
		Detail:   "local server process not running",
	}, nil
}

func (p *Provider) StopServer(_ context.Context, request core.StopServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("local: server id cannot be empty")
	}

	name := strings.TrimPrefix(request.ID, "local-")
	workDir, err := resolveWorkDir("", name)
	if err != nil {
		return fmt.Errorf("local: resolve work dir: %w", err)
	}

	killed := false
	// Stop processes tracked by PID files.
	for _, pidFile := range []string{"game.pid", pidFileName} {
		pidPath := filepath.Join(workDir, pidFile)
		pid, running := readPIDFile(pidPath)
		if !running {
			continue
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		slog.Info("local provider: sending SIGTERM", "pid", pid, "file", pidFile)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			slog.Warn("local provider: SIGTERM failed, trying SIGKILL", "pid", pid, "err", err)
			_ = process.Kill()
		}
		_ = os.Remove(pidPath)
		killed = true
	}

	// Also try to stop by port if PID files didn't help.
	if !killed {
		if port, proto, ok := readGamePort(workDir); ok && isPortListening(port, proto) {
			if pid := findProcessByPort(port); pid > 0 {
				slog.Info("local provider: killing process on game port", "pid", pid, "port", port)
				if process, err := os.FindProcess(pid); err == nil {
					_ = process.Signal(syscall.SIGTERM)
				}
			}
		}
	}

	// Also stop the systemd service if it exists.
	serviceName := fmt.Sprintf("openhost-%s", name)
	if isSystemdAvailable() {
		_ = execCommand("systemctl", "stop", serviceName)
	}

	slog.Info("local provider: stopped server", "id", request.ID)
	return nil
}

func (p *Provider) StartServer(_ context.Context, request core.StartServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("local: server id cannot be empty")
	}

	name := strings.TrimPrefix(request.ID, "local-")

	// Try to restart via systemd first (the runner creates a service).
	serviceName := fmt.Sprintf("openhost-%s", name)
	if isSystemdAvailable() {
		if err := execCommand("systemctl", "restart", serviceName); err == nil {
			slog.Info("local provider: restarted via systemd", "service", serviceName)
			return nil
		}
	}

	// Fallback: re-run the runner with the stored config.
	workDir, err := resolveWorkDir("", name)
	if err != nil {
		return fmt.Errorf("local: resolve work dir: %w", err)
	}

	configPath := filepath.Join(workDir, "runner-config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("local: runner config not found at %s; use 'openhost up' to deploy first", configPath)
	}

	runnerBin, err := resolveRunnerBin("")
	if err != nil {
		return fmt.Errorf("local: resolve runner binary: %w", err)
	}

	logFile, logPath, err := openLogFile(workDir)
	if err != nil {
		return fmt.Errorf("local: open log file: %w", err)
	}

	cmd := exec.Command(runnerBin, "-config", configPath)
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("local: start runner: %w", err)
	}

	pidPath := filepath.Join(workDir, pidFileName)
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	go func() {
		defer func() { _ = logFile.Close() }()
		_ = cmd.Wait()
	}()

	slog.Info("local provider: started server", "id", request.ID, "log", logPath)
	return nil
}

func (p *Provider) DeleteServer(_ context.Context, request core.DeleteServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("local: server id cannot be empty")
	}

	name := strings.TrimPrefix(request.ID, "local-")
	workDir, err := resolveWorkDir("", name)
	if err != nil {
		return fmt.Errorf("local: resolve work dir: %w", err)
	}

	// Stop any running process first.
	_ = p.StopServer(context.Background(), core.StopServerRequest{ID: request.ID})

	// Disable systemd service if it exists.
	serviceName := fmt.Sprintf("openhost-%s", name)
	if isSystemdAvailable() {
		_ = execCommand("systemctl", "disable", serviceName)
		_ = os.Remove(fmt.Sprintf("/etc/systemd/system/%s.service", serviceName))
		_ = execCommand("systemctl", "daemon-reload")
	}

	if request.RemoveAssociatedResources {
		slog.Info("local provider: removing work directory", "path", workDir)
		if err := os.RemoveAll(workDir); err != nil {
			return fmt.Errorf("local: remove work dir %s: %w", workDir, err)
		}
	}

	slog.Info("local provider: deleted server", "id", request.ID)
	return nil
}

func (p *Provider) StopServerAndSnapshot(_ context.Context, request core.StopServerAndSnapshotRequest) (*core.SnapshotResult, error) {
	// Stop the server process.
	_ = p.StopServer(context.Background(), core.StopServerRequest{ID: request.ID})

	// For local mode, a "snapshot" is just a note that the work dir exists.
	// The server data is already on the local filesystem.
	description := request.SnapshotDescription
	if description == "" {
		description = fmt.Sprintf("local snapshot of %s (data preserved in work dir)", request.ID)
	}

	return &core.SnapshotResult{
		SnapshotID:          fmt.Sprintf("local-snap-%s", request.ID),
		SnapshotDescription: description,
	}, nil
}

func (p *Provider) StartServerFromSnapshot(_ context.Context, request core.StartServerFromSnapshotRequest) (*core.Server, error) {
	// For local mode, "start from snapshot" just means start the server again.
	// The data is already on disk.
	if request.Name == "" {
		return nil, fmt.Errorf("local: server name cannot be empty")
	}

	if err := p.StartServer(context.Background(), core.StartServerRequest{
		ID: fmt.Sprintf("local-%s", request.Name),
	}); err != nil {
		return nil, err
	}

	return &core.Server{
		ID:       fmt.Sprintf("local-%s", request.Name),
		Provider: "local",
		Name:     request.Name,
		PublicIP: detectLocalIP(),
	}, nil
}

// --- helpers ---

// resolveWorkDir returns the work directory for a local server.
func resolveWorkDir(explicit, serverName string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	return filepath.Join(home, ".openhost", "local", serverName), nil
}

// openLogFile creates (or appends to) the runner log file in workDir.
// Returns the file handle, the absolute path, and any error.
func openLogFile(workDir string) (*os.File, string, error) {
	logPath := filepath.Join(workDir, "runner.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("open %s: %w", logPath, err)
	}
	return f, logPath, nil
}

// resolveRunnerBin returns the absolute path to the runner binary.
func resolveRunnerBin(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}

	// Check if the runner is on PATH.
	if path, err := exec.LookPath("openhost-runner"); err == nil {
		return filepath.Abs(path)
	}

	// Fallback: look for it relative to the CLI binary.
	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "openhost-runner")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}

	return "", fmt.Errorf("openhost-runner binary not found on PATH or next to CLI binary; set provider.settings.runner_bin")
}

// extractRunnerConfigFromUserData extracts the JSON config from the cloud-init
// heredoc in the user-data script.
func extractRunnerConfigFromUserData(userData string) (*runnerconfig.RunnerConfig, error) {
	// The bootstrap.sh.tmpl embeds the config between:
	//   << 'OPENHOST_CONFIG_EOF'
	//   { ... json ... }
	//   OPENHOST_CONFIG_EOF
	//
	// We match the opening heredoc line (which has quotes around the marker)
	// and the bare closing marker on its own line.
	const startMarker = "'OPENHOST_CONFIG_EOF'\n"
	const endMarker = "\nOPENHOST_CONFIG_EOF"

	startIdx := strings.Index(userData, startMarker)
	if startIdx == -1 {
		return nil, fmt.Errorf("start marker %q not found in user-data", startMarker)
	}
	startIdx += len(startMarker)

	endIdx := strings.Index(userData[startIdx:], endMarker)
	if endIdx == -1 {
		return nil, fmt.Errorf("end marker %q not found in user-data", endMarker)
	}

	jsonData := userData[startIdx : startIdx+endIdx]

	var cfg runnerconfig.RunnerConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal runner config JSON: %w", err)
	}

	return &cfg, nil
}

// readPIDFile reads a PID from a file and checks if the process is alive.
func readPIDFile(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return pid, false
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return pid, false
	}
	return pid, true
}

// readGamePort reads the runner config from workDir and returns the primary
// game port, protocol, and true if found.
func readGamePort(workDir string) (int, string, bool) {
	configPath := filepath.Join(workDir, "runner-config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, "", false
	}
	var cfg runnerconfig.RunnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, "", false
	}

	// Map game names to their primary listen port and protocol.
	type portInfo struct {
		port  int
		proto string
	}
	knownPorts := map[string]portInfo{
		"minecraft": {25565, "tcp"},
		"valheim":   {2456, "udp"},
	}
	if info, ok := knownPorts[cfg.Game.Name]; ok {
		return info.port, info.proto, true
	}
	return 0, "", false
}

// isPortListening checks if a port is in use on localhost.
func isPortListening(port int, proto string) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if proto == "udp" {
		// For UDP we can't reliably dial, check /proc/net/udp instead.
		return isUDPPortInUse(port)
	}
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// isUDPPortInUse checks /proc/net/udp for a listening UDP port (Linux only).
func isUDPPortInUse(port int) bool {
	data, err := os.ReadFile("/proc/net/udp")
	if err != nil {
		return false
	}
	// /proc/net/udp format: local_address is hex IP:PORT
	hexPort := fmt.Sprintf(":%04X", port)
	return strings.Contains(string(data), hexPort)
}

// findProcessByPort attempts to find a PID listening on the given TCP port
// by scanning /proc (Linux only). Returns 0 if not found.
func findProcessByPort(port int) int {
	// Best-effort: use lsof or ss if available
	out, err := exec.Command("ss", "-tlnp", fmt.Sprintf("sport = :%d", port)).CombinedOutput()
	if err != nil {
		return 0
	}
	// Parse "pid=12345" from ss output
	for _, line := range strings.Split(string(out), "\n") {
		if idx := strings.Index(line, "pid="); idx >= 0 {
			pidStr := line[idx+4:]
			if commaIdx := strings.IndexAny(pidStr, ",)"); commaIdx >= 0 {
				pidStr = pidStr[:commaIdx]
			}
			if pid, err := strconv.Atoi(pidStr); err == nil {
				return pid
			}
		}
	}
	return 0
}

// detectLocalIP returns the machine's local IP address.
func detectLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}

	return "127.0.0.1"
}

// isSystemdAvailable checks if systemctl is on PATH.
func isSystemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// execCommand runs a command and returns an error if it fails.
func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, string(output))
	}
	return nil
}

func init() {
	core.RegisterProvider("local", func() core.Provider { return &Provider{} })
}
