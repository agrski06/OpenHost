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

func (g *Minecraft) Name() string     { return "minecraft" }
func (g *Minecraft) Port() int        { return 25565 }
func (g *Minecraft) Protocol() string { return "tcp" }

func (g *Minecraft) BuildInitCommand() string {
	data := struct {
		JavaPackage string
		DownloadURL string
		MinMem      string
		MaxMem      string
		Port        int
	}{
		JavaPackage: "openjdk-21-jre-headless",
		DownloadURL: "https://piston-data.mojang.com/v1/objects/64bb6d763bed0a9f1d632ec347938594144943ed/server.jar",
		MinMem:      "1G", // TODO: set this dynamically
		MaxMem:      "4G", // Maximum limit
		Port:        g.Port(),
	}

	tmpl, err := template.New("mc_init").Parse(initScript)
	if err != nil {
		return "# Error parsing template: " + err.Error()
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "# Error executing template: " + err.Error()
	}

	return buf.String()
}

func init() {
	core.RegisterGame("minecraft", func() core.Game { return &Minecraft{} })
}
