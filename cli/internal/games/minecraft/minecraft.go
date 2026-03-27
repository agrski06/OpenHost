package minecraft

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
	"github.com/openhost/runnerconfig"
)

type Minecraft struct{}

type Settings struct {
	// Version selects the Minecraft server JAR. Supported values:
	//   "1.21.4" (default), "1.21.3", "1.20.6"
	// Future: resolve dynamically via the Mojang version manifest API.
	Version string `mapstructure:"version"`

	// MinMem is the JVM -Xms value (e.g. "1G", "512M"). Defaults to "1G".
	MinMem string `mapstructure:"min_mem"`

	// MaxMem is the JVM -Xmx value (e.g. "4G", "8G"). Defaults to "4G".
	// Increase for larger worlds or more players. Should not exceed the
	// available RAM on the provider plan minus ~512M for OS overhead.
	MaxMem string `mapstructure:"max_mem"`
}

// knownVersions maps Minecraft version strings to their Mojang download URLs.
// These are the server JAR URLs from https://piston-data.mojang.com.
var knownVersions = map[string]string{
	"1.21.4": "https://piston-data.mojang.com/v1/objects/4707d00eb834b446575d89a61a11b5d548d8c001/server.jar",
	"1.21.3": "https://piston-data.mojang.com/v1/objects/45810d238246d90e811d896f87b14695b7fb6839/server.jar",
	"1.21.2": "https://piston-data.mojang.com/v1/objects/64bb6d763bed0a9f1d632ec347938594144943ed/server.jar",
	"1.20.6": "https://piston-data.mojang.com/v1/objects/145ff0858209bcfc164571aac2d39d577a5a18f3/server.jar",
}

const defaultVersion = "1.21.4"

func (g *Minecraft) Name() string { return "minecraft" }

func (g *Minecraft) Ports() []core.PortRange {
	return []core.PortRange{
		{Protocol: "tcp", From: 25565, To: 25565},
	}
}

func (g *Minecraft) BuildRunnerConfig(rawSettings map[string]any) (*runnerconfig.RunnerConfig, error) {
	var s Settings
	if err := mapstructure.Decode(rawSettings, &s); err != nil {
		return nil, fmt.Errorf("decode minecraft settings: %w", err)
	}

	if s.Version == "" {
		s.Version = defaultVersion
	}

	downloadURL, ok := knownVersions[s.Version]
	if !ok {
		supported := make([]string, 0, len(knownVersions))
		for v := range knownVersions {
			supported = append(supported, v)
		}
		return nil, fmt.Errorf("unsupported minecraft version %q; supported: %v", s.Version, supported)
	}

	if s.MinMem == "" {
		s.MinMem = "1G"
	}
	if s.MaxMem == "" {
		s.MaxMem = "4G"
	}

	settings := map[string]any{
		"min_mem": s.MinMem,
		"max_mem": s.MaxMem,
		"version": s.Version,
	}

	cfg := &runnerconfig.RunnerConfig{
		Version: "1",
		Game: runnerconfig.GameConfig{
			Name:     "minecraft",
			Settings: settings,
			Install: runnerconfig.InstallConfig{
				Method:       "http",
				DownloadURL:  downloadURL,
				DestFilename: "server.jar",
			},
		},
		Server: runnerconfig.ServerPaths{
			ServerRoot:  "/home/minecraft/server",
			SaveRoot:    "/home/minecraft/saves",
			ModpackRoot: "/home/minecraft/modpack",
		},
	}

	return cfg, nil
}

func init() {
	core.RegisterGame("minecraft", func() core.Game { return &Minecraft{} })
}
