package valheim

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
	"github.com/openhost/runnerconfig"
)

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

func (g *Valheim) BuildRunnerConfig(rawSettings map[string]any) (*runnerconfig.RunnerConfig, error) {
	s := Settings{World: "Dedicated"}
	if err := mapstructure.Decode(rawSettings, &s); err != nil {
		return nil, err
	}
	if rawCode, ok := rawSettings["thunderstore_code"]; ok && strings.TrimSpace(fmt.Sprint(rawCode)) == "" {
		return nil, fmt.Errorf("game.settings.thunderstore_code cannot be empty when provided")
	}

	s.ThunderstoreCode = strings.TrimSpace(s.ThunderstoreCode)
	if err := validateThunderstoreCode(s.ThunderstoreCode); err != nil {
		return nil, err
	}

	settings := map[string]any{
		"world":    s.World,
		"password": s.Password,
	}

	cfg := &runnerconfig.RunnerConfig{
		Version: "1",
		Game: runnerconfig.GameConfig{
			Name:     "valheim",
			Settings: settings,
			Install: runnerconfig.InstallConfig{
				Method:     "steamcmd",
				SteamAppID: "896660",
				Anonymous:  true,
			},
		},
		Server: runnerconfig.ServerPaths{
			ServerRoot:  "/home/valheim/server",
			SaveRoot:    "/home/valheim/saves",
			ModpackRoot: "/home/valheim/modpack",
		},
	}

	if s.ThunderstoreCode != "" {
		cfg.Game.Mods = &runnerconfig.ModConfig{
			Sources: []runnerconfig.ModSource{
				{
					Provider: "thunderstore",
					Code:     s.ThunderstoreCode,
				},
			},
		}
	}

	return cfg, nil
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
