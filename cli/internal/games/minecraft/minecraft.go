package minecraft

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
)

//go:embed init.sh
var initScript string

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

func (g *Minecraft) BuildInitCommand(rawSettings map[string]any) (string, error) {
	var s Settings
	if err := mapstructure.Decode(rawSettings, &s); err != nil {
		return "", fmt.Errorf("decode minecraft settings: %w", err)
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
		return "", fmt.Errorf("unsupported minecraft version %q; supported: %v", s.Version, supported)
	}

	if s.MinMem == "" {
		s.MinMem = "1G"
	}
	if s.MaxMem == "" {
		s.MaxMem = "4G"
	}

	primaryPort := g.Ports()[0].From

	data := struct {
		JavaPackage string
		DownloadURL string
		MinMem      string
		MaxMem      string
		Port        int
	}{
		JavaPackage: "openjdk-21-jre-headless",
		DownloadURL: downloadURL,
		MinMem:      s.MinMem,
		MaxMem:      s.MaxMem,
		Port:        primaryPort,
	}

	tmpl, err := template.New("mc_init").Parse(initScript)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func init() {
	core.RegisterGame("minecraft", func() core.Game { return &Minecraft{} })
}
