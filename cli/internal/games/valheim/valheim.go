package valheim

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
)

//go:embed init.sh
var initScript string

type Valheim struct{}

type Settings struct {
	World    string `mapstructure:"world"`
	Password string `mapstructure:"password"`
}

func (g *Valheim) Name() string { return "valheim" }
func (g *Valheim) Ports() []core.PortRange {
	return []core.PortRange{
		{Protocol: "udp", From: 2456, To: 2458},
	}
}
func (g *Valheim) Protocol() string { return "udp" }

func (g *Valheim) BuildInitCommand(rawSettings map[string]any) (string, error) {
	s := Settings{World: "Dedicated"}
	if err := mapstructure.Decode(rawSettings, &s); err != nil {
		return "", err
	}

	// Pull range from the interface definition
	portRange := g.Ports()[0]

	data := struct {
		AppID      string
		ServerName string
		WorldName  string
		Password   string
		Port       int
		PortEnd    int
	}{
		AppID:      "896660",
		ServerName: "OpenHost-Valheim",
		WorldName:  s.World,
		Password:   s.Password,
		Port:       portRange.From,
		PortEnd:    portRange.To,
	}

	tmpl, err := template.New("valheim_init").Parse(initScript)
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
	core.RegisterGame("valheim", func() core.Game { return &Valheim{} })
}
