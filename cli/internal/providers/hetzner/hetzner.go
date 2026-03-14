package hetzner

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

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

func (p *Provider) GetServerStatus(id string) (*core.InfrastructureStatus, error) {
	if id == "" {
		return nil, fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid hetzner server id %q: %w", id, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve hetzner server %q: %w", id, err)
	}
	if server == nil {
		return &core.InfrastructureStatus{
			ID:     id,
			State:  core.InfrastructureStateNotFound,
			Detail: "server not found in Hetzner",
		}, nil
	}

	publicIP := ""
	if server.PublicNet.IPv4.IP != nil {
		publicIP = server.PublicNet.IPv4.IP.String()
	}

	return &core.InfrastructureStatus{
		ID:       id,
		Name:     server.Name,
		PublicIP: publicIP,
		State:    mapServerState(server.Status),
		Detail:   fmt.Sprintf("hetzner server status=%s", server.Status),
	}, nil
}

func (p *Provider) DeleteServer(id string) error {
	if id == "" {
		return fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid hetzner server id %q: %w", id, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	ctx := context.Background()
	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to retrieve hetzner server %q before deletion: %w", id, err)
	}
	if server == nil {
		return nil
	}

	if _, _, err := client.Server.DeleteWithResult(ctx, server); err != nil {
		return fmt.Errorf("failed to delete hetzner server %q: %w", id, err)
	}

	return nil
}

func (p *Provider) CreateServer(request core.CreateServerRequest) (*core.Server, error) {
	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("invalid hetzner settings: %w", err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return nil, err
	}
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

	publicIP := ""
	if server.PublicNet.IPv4.IP != nil {
		publicIP = server.PublicNet.IPv4.IP.String()
	}

	return &core.Server{
		ID:       fmt.Sprintf("%d", server.ID),
		Provider: p.Name(),
		Name:     server.Name,
		PublicIP: publicIP,
	}, nil
}

func init() {
	core.RegisterProvider("hetzner", func() core.Provider { return &Provider{} })
}

func mapServerState(status hcloud.ServerStatus) core.InfrastructureState {
	switch status {
	case hcloud.ServerStatusStarting:
		return core.InfrastructureStateCreating
	case hcloud.ServerStatusRunning:
		return core.InfrastructureStateRunning
	case hcloud.ServerStatusStopping:
		return core.InfrastructureStateDeleting
	case hcloud.ServerStatusOff:
		return core.InfrastructureStateStopped
	case hcloud.ServerStatusDeleting:
		return core.InfrastructureStateDeleting
	case hcloud.ServerStatusMigrating:
		return core.InfrastructureStateCreating
	case hcloud.ServerStatusRebuilding:
		return core.InfrastructureStateCreating
	case hcloud.ServerStatusUnknown:
		return core.InfrastructureStateUnknown
	default:
		return core.InfrastructureStateUnknown
	}
}

func newClientFromEnv() (*hcloud.Client, error) {
	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN environment variable is not set")
	}

	return hcloud.NewClient(hcloud.WithToken(token)), nil
}
