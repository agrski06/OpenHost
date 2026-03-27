package mock

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	UserDataPath string `mapstructure:"user_data_output_path"`
}

func (p *Provider) Name() string {
	return "mock"
}

func (p *Provider) GetServerStatus(id string) (*core.InfrastructureStatus, error) {
	slog.Debug("mock provider: GetServerStatus", "id", id)
	if id == "" {
		return nil, fmt.Errorf("mock: server id cannot be empty")
	}
	if !strings.HasPrefix(id, "mock-") {
		slog.Debug("mock provider: server id not found", "id", id)
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
	slog.Debug("mock provider: returning status", "id", id, "state", status.State)
	return status, nil
}

func (p *Provider) StopServer(request core.StopServerRequest) error {
	slog.Debug("mock provider: StopServer", "id", request.ID)
	if request.ID == "" {
		return fmt.Errorf("mock: server id cannot be empty")
	}
	slog.Info("mock provider: stopped server", "id", request.ID)
	return nil
}

func (p *Provider) StartServer(request core.StartServerRequest) error {
	slog.Debug("mock provider: StartServer", "id", request.ID)
	if request.ID == "" {
		return fmt.Errorf("mock: server id cannot be empty")
	}
	slog.Info("mock provider: started server", "id", request.ID)
	return nil
}

func (p *Provider) DeleteServer(request core.DeleteServerRequest) error {
	slog.Debug("mock provider: DeleteServer", "id", request.ID, "removeAssociatedResources", request.RemoveAssociatedResources)
	if request.ID == "" {
		return fmt.Errorf("mock: server id cannot be empty")
	}
	slog.Info("mock provider: deleted server", "id", request.ID)
	return nil
}

func (p *Provider) StopServerAndSnapshot(request core.StopServerAndSnapshotRequest) (*core.SnapshotResult, error) {
	slog.Debug("mock provider: StopServerAndSnapshot", "id", request.ID)
	if request.ID == "" {
		return nil, fmt.Errorf("mock: server id cannot be empty")
	}

	snapshotID := fmt.Sprintf("snap-%s", request.ID)
	description := request.SnapshotDescription
	if description == "" {
		description = fmt.Sprintf("mock snapshot of %s", request.ID)
	}

	slog.Info("mock provider: stopped server and created snapshot", "id", request.ID, "snapshotID", snapshotID)
	return &core.SnapshotResult{
		SnapshotID:          snapshotID,
		SnapshotDescription: description,
	}, nil
}

func (p *Provider) StartServerFromSnapshot(request core.StartServerFromSnapshotRequest) (*core.Server, error) {
	slog.Debug("mock provider: StartServerFromSnapshot", "snapshotID", request.SnapshotID, "name", request.Name)
	if request.SnapshotID == "" {
		return nil, fmt.Errorf("mock: snapshot id cannot be empty")
	}
	if request.Name == "" {
		return nil, fmt.Errorf("mock: server name cannot be empty")
	}

	server := &core.Server{
		ID:       fmt.Sprintf("mock-%s", request.Name),
		Provider: p.Name(),
		Name:     request.Name,
		PublicIP: defaultIP,
	}

	slog.Info("mock provider: started server from snapshot", "snapshotID", request.SnapshotID, "serverID", server.ID)
	return server, nil
}

func (p *Provider) CreateServer(request core.CreateServerRequest) (*core.Server, error) {
	slog.Debug("mock provider: CreateServer",
		"name", request.Name,
		"game", request.GameName,
		"providerSettingKeys", sortedKeys(request.ProviderSettings),
		"ports", request.Ports,
		"userDataBytes", len(request.UserData),
	)

	if request.Name == "" {
		return nil, fmt.Errorf("mock: server name cannot be empty")
	}
	if request.GameName == "" {
		return nil, fmt.Errorf("mock: game name cannot be empty")
	}

	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("mock: invalid settings: %w", err)
	}
	slog.Debug("mock provider: decoded settings",
		"ip", settings.IP,
		"fail", settings.Fail,
		"requireToken", settings.RequireToken,
		"tokenEnvVar", settings.TokenEnvVar,
		"userDataPath", settings.UserDataPath,
	)

	if settings.TokenEnvVar == "" {
		settings.TokenEnvVar = defaultTokenEnvVar
	}

	if settings.RequireToken {
		token := os.Getenv(settings.TokenEnvVar)
		if token == "" {
			return nil, fmt.Errorf("mock: %s environment variable is not set", settings.TokenEnvVar)
		}
		slog.Debug("mock provider: resolved token", "envVar", settings.TokenEnvVar, "length", len(token))
	}

	if settings.Fail {
		if settings.FailMessage == "" {
			settings.FailMessage = "mock provider forced failure"
		}
		slog.Debug("mock provider: forcing failure", "message", settings.FailMessage)
		return nil, fmt.Errorf("mock: %s", settings.FailMessage)
	}

	if settings.IP == "" {
		settings.IP = defaultIP
		slog.Debug("mock provider: using default IP", "ip", settings.IP)
	}

	if request.UserData == "" {
		return nil, fmt.Errorf("mock: user-data cannot be empty")
	}

	if settings.UserDataPath != "" {
		if err := os.MkdirAll(filepath.Dir(settings.UserDataPath), 0o755); err != nil {
			return nil, fmt.Errorf("mock: create bootstrap output directory: %w", err)
		}

		if err := os.WriteFile(settings.UserDataPath, []byte(request.UserData), 0o600); err != nil {
			return nil, fmt.Errorf("mock: write bootstrap script: %w", err)
		}

		slog.Debug("mock provider: wrote bootstrap script", "path", settings.UserDataPath)
	}

	slog.Info("mock provider: created server", "name", request.Name, "ip", settings.IP)

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
