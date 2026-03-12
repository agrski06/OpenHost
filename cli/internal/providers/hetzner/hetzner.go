package hetzner

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/openhost/cli/internal/core"
)

type Provider struct{}

func (p *Provider) Name() string {
	return "hetzner"
}

// RunServer creates the VPS on Hetzner and runs init command
func (p *Provider) RunServer(name string, game core.Game) (core.Server, error) {
	token := os.Getenv("HCLOUD_TOKEN")

	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN environment variable is not set")
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	ctx := context.Background()

	_, anyIPv4, _ := net.ParseCIDR("0.0.0.0/0")

	// Helper function to build a rule
	buildRule := func(p hcloud.FirewallRuleProtocol, port int) hcloud.FirewallRule {
		return hcloud.FirewallRule{
			Direction: hcloud.FirewallRuleDirectionIn,
			Protocol:  p,
			Port:      hcloud.Ptr(fmt.Sprintf("%d", port)),
			SourceIPs: []net.IPNet{*anyIPv4},
		}
	}

	rules := []hcloud.FirewallRule{
		buildRule(hcloud.FirewallRuleProtocolTCP, 22), // Always allow SSH
	}

	if game.Protocol() == "tcp" || game.Protocol() == "both" {
		rules = append(rules, buildRule(hcloud.FirewallRuleProtocolTCP, game.Port()))
	}
	if game.Protocol() == "udp" || game.Protocol() == "both" {
		rules = append(rules, buildRule(hcloud.FirewallRuleProtocolUDP, game.Port()))
	}

	fwName := fmt.Sprintf("fw-%s-%d", game.Name(), game.Port())
	fw, _, _ := client.Firewall.Create(ctx, hcloud.FirewallCreateOpts{
		Name:  fwName,
		Rules: rules,
	})

	// Create the Server
	result, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       name,
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
		ServerType: &hcloud.ServerType{Name: "cx23"},
		UserData:   game.BuildInitCommand(),
		Firewalls: []*hcloud.ServerCreateFirewall{
			{Firewall: *fw.Firewall},
		},
		SSHKeys: []*hcloud.SSHKey{{Name: "default"}},
	})

	if err != nil {
		return nil, err
	}

	// Note: The IP might not be fully assigned until the 'Create Server' action is finished
	err = client.Action.WaitFor(ctx, result.Action)
	if err != nil {
		return nil, err
	}

	server, _, err := client.Server.GetByID(ctx, result.Server.ID)
	if err != nil {
		return nil, err
	}

	return &Server{
		ip: server.PublicNet.IPv4.IP.String(),
	}, nil
}

type Server struct {
	ip string
}

func (s *Server) IP() string { return s.ip }

func init() {
	core.RegisterProvider("hetzner", func() core.Provider {
		return &Provider{}
	})
}
