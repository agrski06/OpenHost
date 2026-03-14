package valheim

import (
	"testing"

	"github.com/openhost/cli/internal/core"
	"github.com/openhost/cli/internal/gamestatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrimaryPort_UsesValheimQueryPort(t *testing.T) {
	port, err := primaryPort(gamestatus.Target{
		Ports: []core.PortRange{{Protocol: "udp", From: 2456, To: 2458}},
	})
	require.NoError(t, err)
	assert.Equal(t, queryPort, port)
}

func TestPrimaryPort_FallsBackToSecondRangeStart(t *testing.T) {
	port, err := primaryPort(gamestatus.Target{
		Ports: []core.PortRange{{Protocol: "udp", From: 3000, To: 3002}},
	})
	require.NoError(t, err)
	assert.Equal(t, 3001, port)
}

func TestParseInfoResponse(t *testing.T) {
	payload := []byte{
		0x11,
		'O', 'p', 'e', 'n', 'H', 'o', 's', 't', 0x00,
		'M', 'i', 's', 't', 'l', 'a', 'n', 'd', 's', 0x00,
		'v', 'a', 'l', 'h', 'e', 'i', 'm', 0x00,
		'V', 'a', 'l', 'h', 'e', 'i', 'm', 0x00,
		0x3A, 0xE3,
		0x03,
		0x0A,
		0x00,
		'd',
		'l',
		0x00,
		0x01,
		'1', '.', '2', '3', '.', '4', 0x00,
	}

	info, err := parseInfoResponse(payload)
	require.NoError(t, err)
	assert.Equal(t, "OpenHost", info.Name)
	assert.Equal(t, "Mistlands", info.Map)
	assert.Equal(t, 3, info.Players)
	assert.Equal(t, 10, info.MaxPlayers)
	assert.Equal(t, "1.23.4", info.Version)
}

func TestParseInfoResponse_ShortPayload(t *testing.T) {
	_, err := parseInfoResponse([]byte{0x11})
	require.Error(t, err)
}
