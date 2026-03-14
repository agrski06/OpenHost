package mock

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/openhost/cli/internal/core"
)

const (
	defaultIP          = "203.0.113.10"
	defaultTokenEnvVar = "OPENHOST_MOCK_TOKEN"
)

type Provider struct{}

type Settings struct {
	IP           string `mapstructure:"ip"`
	Fail         bool   `mapstructure:"fail"`
	FailMessage  string `mapstructure:"fail_message"`
	RequireToken bool   `mapstructure:"require_token"`
	TokenEnvVar  string `mapstructure:"token_env_var"`
}

func (p *Provider) Name() string {
	return "mock"
}

func (p *Provider) GetServerStatus(id string) (*core.InfrastructureStatus, error) {
	log.Printf("mock provider: starting GetServerStatus id=%q", id)
	if id == "" {
		log.Printf("mock provider: rejected status because server id is empty")
		return nil, fmt.Errorf("mock: server id cannot be empty")
	}
	if !strings.HasPrefix(id, "mock-") {
		log.Printf("mock provider: server id %q not found", id)
		return &core.InfrastructureStatus{
			ID:     id,
			State:  core.InfrastructureStateNotFound,
			Detail: "mock server id not found",
		}, nil
	}

	name := strings.TrimPrefix(id, "mock-")
	status := &core.InfrastructureStatus{
		ID:       id,
		Name:     name,
		PublicIP: defaultIP,
		State:    core.InfrastructureStateRunning,
		Detail:   "mock provider always reports synthetic servers as running",
	}
	log.Printf("mock provider: returning status id=%q state=%q", id, status.State)
	return status, nil
}

func (p *Provider) DeleteServer(id string) error {
	log.Printf("mock provider: starting DeleteServer id=%q", id)
	if id == "" {
		log.Printf("mock provider: rejected delete because server id is empty")
		return fmt.Errorf("mock: server id cannot be empty")
	}

	log.Printf("mock provider: deleted fake server id=%q", id)
	return nil
}

func (p *Provider) CreateServer(request core.CreateServerRequest) (*core.Server, error) {
	log.Printf(
		"mock provider: starting CreateServer name=%q game=%q providerSettingKeys=%v ports=%v userDataBytes=%d",
		request.Name,
		request.GameName,
		sortedKeys(request.ProviderSettings),
		request.Ports,
		len(request.UserData),
	)

	if request.Name == "" {
		log.Printf("mock provider: rejected request because server name is empty")
		return nil, fmt.Errorf("mock: server name cannot be empty")
	}
	if request.GameName == "" {
		log.Printf("mock provider: rejected request because game name is empty")
		return nil, fmt.Errorf("mock: game name cannot be empty")
	}

	log.Printf("mock provider: resolved game=%q ports=%v", request.GameName, request.Ports)

	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		log.Printf("mock provider: failed to decode settings: %v", err)
		return nil, fmt.Errorf("mock: invalid settings: %w", err)
	}
	log.Printf(
		"mock provider: decoded settings ip=%q fail=%t failMessageSet=%t requireToken=%t tokenEnvVar=%q",
		settings.IP,
		settings.Fail,
		settings.FailMessage != "",
		settings.RequireToken,
		settings.TokenEnvVar,
	)

	if settings.TokenEnvVar == "" {
		settings.TokenEnvVar = defaultTokenEnvVar
	}

	if settings.RequireToken {
		token := os.Getenv(settings.TokenEnvVar)
		if token == "" {
			log.Printf("mock provider: required token env var %q is not set", settings.TokenEnvVar)
			return nil, fmt.Errorf("mock: %s environment variable is not set", settings.TokenEnvVar)
		}

		log.Printf("mock provider: resolved token from env var %q (length=%d)", settings.TokenEnvVar, len(token))
	}

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

	if request.UserData == "" {
		log.Printf("mock provider: rejected request because user-data is empty")
		return nil, fmt.Errorf("mock: user-data cannot be empty")
	}
	log.Printf("mock provider: received bootstrap script bytes=%d", len(request.UserData))

	log.Printf("mock provider: returning fake server ip=%q", settings.IP)

	return &core.Server{
		ID:       fmt.Sprintf("mock-%s", request.Name),
		Provider: p.Name(),
		Name:     request.Name,
		PublicIP: settings.IP,
	}, nil
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
