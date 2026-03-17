package system

import (
	"context"
	"fmt"
)

func (m *Manager) AllowUDPRange(ctx context.Context, from int, to int) error {
	m.Logger.Info("opening udp firewall range", "from", from, "to", to)
	return m.Executor.Run(ctx, "ufw", "allow", fmt.Sprintf("%d:%d/udp", from, to))
}

func (m *Manager) AllowTCPPort(ctx context.Context, port int) error {
	m.Logger.Info("opening tcp firewall port", "port", port)
	return m.Executor.Run(ctx, "ufw", "allow", fmt.Sprintf("%d/tcp", port))
}
