package hetzner

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/openhost/cli/internal/core"
)

type Provider struct{}

type Settings struct {
	Plan     string `mapstructure:"plan"`
	Location string `mapstructure:"location"`
}

func (p *Provider) Name() string {
	return "hetzner"
}

func (p *Provider) RunServer(name string, game core.Game, rawSettings map[string]any) (core.Server, error) {
	var settings Settings
	if err := mapstructure.Decode(rawSettings, &settings); err != nil {
		return nil, fmt.Errorf("invalid hetzner settings: %w", err)
	}

	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN environment variable is not set")
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	ctx := context.Background()

	_, anyIPv4, _ := net.ParseCIDR("0.0.0.0/0")
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
	fw, _, err := client.Firewall.GetByName(ctx, fwName)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing firewall: %w", err)
	}

	if fw == nil {
		res, _, err := client.Firewall.Create(ctx, hcloud.FirewallCreateOpts{
			Name:  fwName,
			Rules: rules,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create firewall: %w", err)
		}
		fw = res.Firewall
	}

	result, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       name,
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
		ServerType: &hcloud.ServerType{Name: settings.Plan},
		Location:   &hcloud.Location{Name: settings.Location},
		UserData:   game.BuildInitCommand(),
		Firewalls: []*hcloud.ServerCreateFirewall{
			{Firewall: *fw},
		},
		// TODO: make this work
		// SSHKeys: []*hcloud.SSHKey{{Name: "default"}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	err = client.Action.WaitFor(ctx, result.Action)
	if err != nil {
		return nil, fmt.Errorf("error waiting for server creation action: %w", err)
	}

	server, _, err := client.Server.GetByID(ctx, result.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve server details: %w", err)
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
