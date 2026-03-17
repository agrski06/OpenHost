package valheim

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
	"github.com/openhost/runnerconfig"
)

//go:embed bootstrap.sh
var bootstrapScript string

const defaultRunnerDownloadURL = "https://github.com/agrski06/OpenGameHost/releases/download/runner-v%s/openhost-runner-linux-amd64"

// RunnerVersion is intended to be overridden at build time via -ldflags.
var RunnerVersion = "0.1.0"

type Valheim struct{}

type Settings struct {
	World            string `mapstructure:"world"`
	Password         string `mapstructure:"password"`
	ThunderstoreCode string `mapstructure:"thunderstore_code"`
	RunnerVersion    string `mapstructure:"runner_version"`
	RunnerURL        string `mapstructure:"runner_url"`
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

	data := struct {
		RunnerVersion    string
		RunnerURL        string
		RunnerConfigJSON string
	}{
		RunnerVersion: resolveRunnerVersion(s.RunnerVersion),
		RunnerURL:     resolveRunnerURL(s.RunnerVersion, s.RunnerURL),
	}

	runnerConfig := runnerconfig.RunnerConfig{
		Version: "1",
		Game: runnerconfig.GameConfig{
			Name: "valheim",
			Settings: map[string]any{
				"world":    s.World,
				"password": s.Password,
			},
		},
		Server: runnerconfig.ServerPaths{
			ServerRoot:  "${SERVER_ROOT}",
			SaveRoot:    "${SAVE_ROOT}",
			ModpackRoot: "${MODPACK_ROOT}",
		},
	}
	if s.ThunderstoreCode != "" {
		runnerConfig.Game.Mods = &runnerconfig.ModConfig{
			Sources: []runnerconfig.ModSource{{
				Provider: "thunderstore",
				Code:     s.ThunderstoreCode,
			}},
		}
	}

	configJSON, err := json.MarshalIndent(runnerConfig, "", "  ")
	if err != nil {
		return "", err
	}
	data.RunnerConfigJSON = string(configJSON)

	tmpl, err := template.New("valheim_bootstrap").Parse(bootstrapScript)
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

func resolveRunnerVersion(setting string) string {
	if trimmed := strings.TrimSpace(setting); trimmed != "" {
		return trimmed
	}
	return RunnerVersion
}

func resolveRunnerURL(settingVersion string, settingURL string) string {
	if trimmed := strings.TrimSpace(settingURL); trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf(defaultRunnerDownloadURL, resolveRunnerVersion(settingVersion))
}
