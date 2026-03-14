package mock

import (
	"testing"

	"github.com/openhost/cli/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderCreateServer_DefaultSuccessWithoutToken(t *testing.T) {
	provider := &Provider{}

	server, err := provider.CreateServer(core.CreateServerRequest{
		Name:     "mock-server",
		GameName: "minecraft",
		Ports: []core.PortRange{
			{Protocol: "tcp", From: 25565, To: 25565},
		},
		ProviderSettings: map[string]any{
			"ip": "203.0.113.77",
		},
		UserData: "#!/bin/bash\necho ok\n",
	})

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "mock-mock-server", server.ID)
	assert.Equal(t, "mock", server.Provider)
	assert.Equal(t, "mock-server", server.Name)
	assert.Equal(t, "203.0.113.77", server.PublicIP)
}

func TestProviderCreateServer_RequireTokenMissing(t *testing.T) {
	t.Setenv(defaultTokenEnvVar, "")

	provider := &Provider{}
	server, err := provider.CreateServer(core.CreateServerRequest{
		Name:     "mock-server",
		GameName: "minecraft",
		Ports: []core.PortRange{
			{Protocol: "tcp", From: 25565, To: 25565},
		},
		ProviderSettings: map[string]any{
			"require_token": true,
		},
		UserData: "#!/bin/bash\necho ok\n",
	})

	require.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), defaultTokenEnvVar)
}

func TestProviderCreateServer_RequireTokenDefaultEnvVar(t *testing.T) {
	t.Setenv(defaultTokenEnvVar, "mock-token")

	provider := &Provider{}
	server, err := provider.CreateServer(core.CreateServerRequest{
		Name:     "mock-server",
		GameName: "minecraft",
		Ports: []core.PortRange{
			{Protocol: "tcp", From: 25565, To: 25565},
		},
		ProviderSettings: map[string]any{
			"require_token": true,
		},
		UserData: "#!/bin/bash\necho ok\n",
	})

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, defaultIP, server.PublicIP)
}

func TestProviderCreateServer_RequireTokenCustomEnvVar(t *testing.T) {
	const envVar = "CUSTOM_MOCK_TOKEN"
	t.Setenv(envVar, "custom-token")

	provider := &Provider{}
	server, err := provider.CreateServer(core.CreateServerRequest{
		Name:     "mock-server",
		GameName: "minecraft",
		Ports: []core.PortRange{
			{Protocol: "tcp", From: 25565, To: 25565},
		},
		ProviderSettings: map[string]any{
			"require_token": true,
			"token_env_var": envVar,
		},
		UserData: "#!/bin/bash\necho ok\n",
	})

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.Equal(t, "mock", server.Provider)
}

func TestProviderDeleteServer_Succeeds(t *testing.T) {
	provider := &Provider{}
	require.NoError(t, provider.DeleteServer("mock-server"))
}

func TestProviderDeleteServer_RejectsEmptyID(t *testing.T) {
	provider := &Provider{}
	err := provider.DeleteServer("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server id cannot be empty")
}
