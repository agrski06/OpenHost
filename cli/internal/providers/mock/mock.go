package mock

import (
	"fmt"
	"log"
	"sort"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
)

const defaultIP = "203.0.113.10"

type Provider struct{}

type Settings struct {
	IP          string `mapstructure:"ip"`
	Fail        bool   `mapstructure:"fail"`
	FailMessage string `mapstructure:"fail_message"`
}

func (p *Provider) Name() string {
	return "mock"
}

func (p *Provider) RunServer(name string, game core.Game, rawSettings map[string]any, gameSettings map[string]any) (core.Server, error) {
	log.Printf("mock provider: starting RunServer name=%q rawSettingKeys=%v gameSettingKeys=%v", name, sortedKeys(rawSettings), sortedKeys(gameSettings))

	if name == "" {
		log.Printf("mock provider: rejected request because server name is empty")
		return nil, fmt.Errorf("mock: server name cannot be empty")
	}
	if game == nil {
		log.Printf("mock provider: rejected request because game is nil")
		return nil, fmt.Errorf("mock: game cannot be nil")
	}

	ports := game.Ports()
	log.Printf("mock provider: resolved game=%q ports=%v", game.Name(), ports)

	var settings Settings
	if err := mapstructure.Decode(rawSettings, &settings); err != nil {
		log.Printf("mock provider: failed to decode settings: %v", err)
		return nil, fmt.Errorf("mock: invalid settings: %w", err)
	}
	log.Printf("mock provider: decoded settings ip=%q fail=%t failMessageSet=%t", settings.IP, settings.Fail, settings.FailMessage != "")

	if settings.Fail {
		if settings.FailMessage == "" {
			settings.FailMessage = "mock provider forced failure"
		}
		log.Printf("mock provider: forcing failure message=%q", settings.FailMessage)
		return nil, fmt.Errorf("mock: %s", settings.FailMessage)
	}

	if settings.IP == "" {
		settings.IP = defaultIP
		log.Printf("mock provider: using default IP %q", settings.IP)
	}

	// Exercise the current game bootstrap path so config/game regressions are still
	// visible during lightweight local runs.
	initCommand, err := game.BuildInitCommand(gameSettings)
	if err != nil {
		log.Printf("mock provider: game bootstrap failed: %v", err)
		return nil, fmt.Errorf("mock: build init command for game %q: %w", game.Name(), err)
	}
	log.Printf("mock provider: generated bootstrap script bytes=%d", len(initCommand))

	log.Printf("mock provider: returning fake server ip=%q", settings.IP)

	return &Server{ip: settings.IP}, nil
}

type Server struct {
	ip string
}

func (s *Server) IP() string {
	return s.ip
}

func init() {
	core.RegisterProvider("mock", func() core.Provider { return &Provider{} })
}

func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}

	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
