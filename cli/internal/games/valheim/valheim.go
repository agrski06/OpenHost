package valheim

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
)

//go:embed init.sh
var initScript string

type Valheim struct{}

type Settings struct {
	World            string `mapstructure:"world"`
	Password         string `mapstructure:"password"`
	ThunderstoreCode string `mapstructure:"thunderstore_code"`
}

var thunderstoreCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

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
	if rawCode, ok := rawSettings["thunderstore_code"]; ok && strings.TrimSpace(fmt.Sprint(rawCode)) == "" {
		return "", fmt.Errorf("game.settings.thunderstore_code cannot be empty when provided")
	}

	s.ThunderstoreCode = strings.TrimSpace(s.ThunderstoreCode)
	if err := validateThunderstoreCode(s.ThunderstoreCode); err != nil {
		return "", err
	}

	// Pull range from the interface definition
	portRange := g.Ports()[0]

	data := struct {
		AppID            string
		ServerName       string
		WorldName        string
		Password         string
		Port             int
		PortEnd          int
		HasMods          bool
		ThunderstoreCode string
	}{
		AppID:            "896660",
		ServerName:       "OpenHost-Valheim",
		WorldName:        s.World,
		Password:         s.Password,
		Port:             portRange.From,
		PortEnd:          portRange.To,
		HasMods:          s.ThunderstoreCode != "",
		ThunderstoreCode: s.ThunderstoreCode,
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

func validateThunderstoreCode(code string) error {
	if code == "" {
		return nil
	}
	if !thunderstoreCodePattern.MatchString(code) {
		return fmt.Errorf("game.settings.thunderstore_code must be a bare Thunderstore/r2modman export code")
	}
	return nil
}
