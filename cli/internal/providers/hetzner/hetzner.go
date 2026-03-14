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

func (p *Provider) CreateServer(request core.CreateServerRequest) (core.Server, error) {
	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("invalid hetzner settings: %w", err)
	}

	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN environment variable is not set")
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	ctx := context.Background()

	_, anyIPv4, _ := net.ParseCIDR("0.0.0.0/0")

	// Helper to build a port or port-range rule
	buildRule := func(proto hcloud.FirewallRuleProtocol, startPort int, endPort int) hcloud.FirewallRule {
		portStr := fmt.Sprintf("%d", startPort)
		if startPort != endPort {
			portStr = fmt.Sprintf("%d-%d", startPort, endPort)
		}
		return hcloud.FirewallRule{
			Direction: hcloud.FirewallRuleDirectionIn,
			Protocol:  proto,
			Port:      hcloud.Ptr(portStr),
			SourceIPs: []net.IPNet{*anyIPv4},
		}
	}

	// Default rules: SSH always allowed
	rules := []hcloud.FirewallRule{
		buildRule(hcloud.FirewallRuleProtocolTCP, 22, 22),
	}

	// Dynamic Port Generation
	for _, pr := range request.Ports {
		proto := hcloud.FirewallRuleProtocolTCP
		if pr.Protocol == "udp" {
			proto = hcloud.FirewallRuleProtocolUDP
		}

		rules = append(rules, buildRule(proto, pr.From, pr.To))
	}

	if len(request.Ports) == 0 {
		return nil, fmt.Errorf("hetzner provider requires at least one exposed port")
	}

	// Use a unique firewall name based on the game and its primary port
	primaryPort := request.Ports[0].From
	fwName := fmt.Sprintf("fw-%s-%d", request.GameName, primaryPort)
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
		Name:       request.Name,
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
		ServerType: &hcloud.ServerType{Name: settings.Plan},
		Location:   &hcloud.Location{Name: settings.Location},
		UserData:   request.UserData,
		Firewalls: []*hcloud.ServerCreateFirewall{
			{Firewall: *fw},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	_ = client.Action.WaitFor(ctx, result.Action)

	server, _, err := client.Server.GetByID(ctx, result.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve server details: %w", err)
	}

	return &Server{ip: server.PublicNet.IPv4.IP.String()}, nil
}

type Server struct{ ip string }

func (s *Server) IP() string { return s.ip }

func init() {
	core.RegisterProvider("hetzner", func() core.Provider { return &Provider{} })
}
