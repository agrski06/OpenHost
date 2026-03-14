package minecraft

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/openhost/cli/internal/core"
)

//go:embed init.sh
var initScript string

type Minecraft struct{}

func (g *Minecraft) Name() string { return "minecraft" }

func (g *Minecraft) Ports() []core.PortRange {
	return []core.PortRange{
		{Protocol: "tcp", From: 25565, To: 25565},
	}
}

func (g *Minecraft) BuildInitCommand(rawSettings map[string]any) (string, error) {
	primaryPort := g.Ports()[0].From

	data := struct {
		JavaPackage string
		DownloadURL string
		MinMem      string
		MaxMem      string
		Port        int
	}{
		JavaPackage: "openjdk-21-jre-headless",
		// Minecraft 1.21.x Server JAR
		DownloadURL: "https://piston-data.mojang.com/v1/objects/64bb6d763bed0a9f1d632ec347938594144943ed/server.jar",
		MinMem:      "1G", // TODO: set this dynamically based on provider plan
		MaxMem:      "4G",
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
