package minecraft

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runnerconfig"
)

// Minecraft implements GameSetup for the Minecraft Java Edition dedicated server.
type Minecraft struct{}

func (m *Minecraft) Name() string       { return "minecraft" }
func (m *Minecraft) SystemUser() string { return "minecraft" }

// ModFramework returns empty — Minecraft mod frameworks (Fabric/Forge) are not
// yet supported in the runner.
func (m *Minecraft) ModFramework() string { return "" }

func (m *Minecraft) RequiredPackages() []string {
	return []string{"openjdk-21-jre-headless", "wget"}
}

// InstallMethod returns nil — Minecraft uses the config-driven HTTP download
// resolved by the orchestrator from RunnerConfig.Game.Install.
func (m *Minecraft) InstallMethod() core.InstallMethod {
	return nil
}

func (m *Minecraft) ServerPaths() runnerconfig.ServerPaths {
	return runnerconfig.ServerPaths{
		ServerRoot:  "/home/minecraft/server",
		SaveRoot:    "/home/minecraft/saves",
		ModpackRoot: "/home/minecraft/modpack",
	}
}

func (m *Minecraft) BuildLaunchCommand(cfg runnerconfig.GameConfig) core.LaunchConfig {
	minMem := "1G"
	maxMem := "4G"

	if v, ok := cfg.Settings["min_mem"].(string); ok && v != "" {
		minMem = v
	}
	if v, ok := cfg.Settings["max_mem"].(string); ok && v != "" {
		maxMem = v
	}

	paths := m.ServerPaths()

	// Write eula.txt to accept the Minecraft EULA.
	eulaPath := filepath.Join(paths.ServerRoot, "eula.txt")
	_ = os.WriteFile(eulaPath, []byte("eula=true\n"), 0644)

	execStart := fmt.Sprintf(
		"java -Xms%s -Xmx%s -jar server.jar --port 25565 nogui",
		minMem, maxMem,
	)

	return core.LaunchConfig{
		ServiceName:   "openhost-minecraft",
		User:          m.SystemUser(),
		WorkingDir:    paths.ServerRoot,
		ExecStart:     execStart,
		Environment:   map[string]string{},
		RestartPolicy: "on-failure",
	}
}

func init() {
	core.RegisterGameSetup("minecraft", func() core.GameSetup { return &Minecraft{} })
}
