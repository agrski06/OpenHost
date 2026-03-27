package hetzner

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/openhost/cli/internal/core"
)

type Provider struct{}

type Settings struct {
	Plan     string `mapstructure:"plan"`
	Location string `mapstructure:"location"`
}

const (
	deleteWaitTimeout  = 2 * time.Minute
	deletePollInterval = 2 * time.Second
)

func (p *Provider) Name() string {
	return "hetzner"
}

func (p *Provider) GetServerStatus(ctx context.Context, id string) (*core.InfrastructureStatus, error) {
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

func (p *Provider) StopServer(ctx context.Context, request core.StopServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid hetzner server id %q: %w", request.ID, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to retrieve hetzner server %q: %w", request.ID, err)
	}
	if server == nil {
		return nil
	}
	if server.Status == hcloud.ServerStatusOff {
		return nil
	}

	action, _, err := client.Server.Shutdown(ctx, server)
	if err != nil {
		return fmt.Errorf("shutdown hetzner server %q: %w", request.ID, err)
	}
	if action != nil {
		if err := client.Action.WaitFor(ctx, action); err != nil {
			return fmt.Errorf("wait for shutdown of hetzner server %q: %w", request.ID, err)
		}
	}
	return nil
}

func (p *Provider) StartServer(ctx context.Context, request core.StartServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid hetzner server id %q: %w", request.ID, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to retrieve hetzner server %q: %w", request.ID, err)
	}
	if server == nil {
		return fmt.Errorf("hetzner server %q not found", request.ID)
	}
	if server.Status == hcloud.ServerStatusRunning {
		return nil
	}

	action, _, err := client.Server.Poweron(ctx, server)
	if err != nil {
		return fmt.Errorf("power on hetzner server %q: %w", request.ID, err)
	}
	if action != nil {
		if err := client.Action.WaitFor(ctx, action); err != nil {
			return fmt.Errorf("wait for power on of hetzner server %q: %w", request.ID, err)
		}
	}
	return nil
}

func (p *Provider) DeleteServer(ctx context.Context, request core.DeleteServerRequest) error {
	if request.ID == "" {
		return fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid hetzner server id %q: %w", request.ID, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return err
	}

	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to retrieve hetzner server %q before deletion: %w", request.ID, err)
	}
	if server == nil {
		return nil
	}

	deleteResult, _, err := client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to delete hetzner server %q: %w", request.ID, err)
	}
	if deleteResult != nil && deleteResult.Action != nil {
		if err := client.Action.WaitFor(ctx, deleteResult.Action); err != nil {
			return fmt.Errorf("wait for deletion of hetzner server %q: %w", request.ID, err)
		}
	}
	if err := waitForServerDeletion(ctx, client, serverID); err != nil {
		return fmt.Errorf("confirm deletion of hetzner server %q: %w", request.ID, err)
	}

	if err := p.deleteAssociatedResources(ctx, client, request); err != nil {
		return err
	}

	if request.RemoveAssociatedResources && len(request.SnapshotIDs) > 0 {
		for _, snapshotID := range request.SnapshotIDs {
			imageID, err := strconv.ParseInt(snapshotID, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse snapshot image id %q for server %q: %w", snapshotID, request.ID, err)
			}
			image, _, err := client.Image.GetByID(ctx, imageID)
			if err != nil {
				return fmt.Errorf("failed to look up snapshot image %q for server %q: %w", snapshotID, request.ID, err)
			}
			if image == nil {
				continue
			}
			if _, err := client.Image.Delete(ctx, image); err != nil {
				return fmt.Errorf("failed to delete snapshot image %q for server %q: %w", snapshotID, request.ID, err)
			}
		}
	}

	return nil
}

func (p *Provider) CreateServer(ctx context.Context, request core.CreateServerRequest) (*core.Server, error) {
	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("invalid hetzner settings: %w", err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return nil, err
	}

	if len(request.Ports) == 0 {
		return nil, fmt.Errorf("hetzner provider requires at least one exposed port")
	}

	rules := buildFirewallRules(request.Ports)
	fwName := firewallName(request.GameName, request.Ports)
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
		AssociatedResources: []core.ResourceRef{
			{
				Type: "firewall",
				ID:   fmt.Sprintf("%d", fw.ID),
				Name: fw.Name,
			},
		},
	}, nil
}

func (p *Provider) StopServerAndSnapshot(ctx context.Context, request core.StopServerAndSnapshotRequest) (*core.SnapshotResult, error) {
	if request.ID == "" {
		return nil, fmt.Errorf("hetzner server id cannot be empty")
	}

	serverID, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid hetzner server id %q: %w", request.ID, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return nil, err
	}

	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve hetzner server %q: %w", request.ID, err)
	}
	if server == nil {
		return nil, fmt.Errorf("hetzner server %q not found", request.ID)
	}

	// 1) Attempt a graceful shutdown for filesystem consistency.
	// (If the server is already off, this will typically fail; in that case we proceed.)
	if server.Status != hcloud.ServerStatusOff {
		action, _, err := client.Server.Shutdown(ctx, server)
		if err == nil && action != nil {
			if err := client.Action.WaitFor(ctx, action); err != nil {
				return nil, fmt.Errorf("wait for shutdown of hetzner server %q: %w", request.ID, err)
			}
		}
	}

	// Wait until the server is actually powered off.
	for {
		refreshed, _, err := client.Server.GetByID(ctx, serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh hetzner server %q status: %w", request.ID, err)
		}
		if refreshed == nil {
			return nil, fmt.Errorf("hetzner server %q not found while waiting for shutdown", request.ID)
		}
		if refreshed.Status == hcloud.ServerStatusOff {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// 2) Create the snapshot image.
	description := request.SnapshotDescription
	if description == "" {
		// Keep it human-friendly but deterministic.
		base := server.Name
		if request.GameName != "" {
			base = fmt.Sprintf("%s-%s", server.Name, request.GameName)
		}
		description = fmt.Sprintf("openhost snapshot: %s %s", base, time.Now().UTC().Format(time.RFC3339))
	}

	res, _, err := client.Server.CreateImage(ctx, server, &hcloud.ServerCreateImageOpts{
		Type:        hcloud.ImageTypeSnapshot,
		Description: hcloud.Ptr(description),
		Labels: map[string]string{
			"managed-by": "openhost",
			"server-id":  fmt.Sprintf("%d", server.ID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create snapshot for hetzner server %q: %w", request.ID, err)
	}
	if res.Action != nil {
		if err := client.Action.WaitFor(ctx, res.Action); err != nil {
			return nil, fmt.Errorf("wait for snapshot action for hetzner server %q: %w", request.ID, err)
		}
	}
	if res.Image == nil {
		return nil, fmt.Errorf("hetzner snapshot action completed but no image was returned")
	}

	return &core.SnapshotResult{
		SnapshotID:          fmt.Sprintf("%d", res.Image.ID),
		SnapshotDescription: res.Image.Description,
	}, nil
}

func (p *Provider) StartServerFromSnapshot(ctx context.Context, request core.StartServerFromSnapshotRequest) (*core.Server, error) {
	if request.SnapshotID == "" {
		return nil, fmt.Errorf("hetzner snapshot id cannot be empty")
	}
	if request.Name == "" {
		return nil, fmt.Errorf("server name cannot be empty")
	}
	if request.GameName == "" {
		return nil, fmt.Errorf("game name cannot be empty")
	}
	if len(request.Ports) == 0 {
		return nil, fmt.Errorf("hetzner provider requires at least one exposed port")
	}

	var settings Settings
	if err := mapstructure.Decode(request.ProviderSettings, &settings); err != nil {
		return nil, fmt.Errorf("invalid hetzner settings: %w", err)
	}

	snapshotID, err := strconv.ParseInt(request.SnapshotID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid hetzner snapshot id %q: %w", request.SnapshotID, err)
	}

	client, err := newClientFromEnv()
	if err != nil {
		return nil, err
	}

	rules := buildFirewallRules(request.Ports)
	fwName := firewallName(request.GameName, request.Ports)
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
		Image:      &hcloud.Image{ID: snapshotID},
		ServerType: &hcloud.ServerType{Name: settings.Plan},
		Location:   &hcloud.Location{Name: settings.Location},
		Firewalls: []*hcloud.ServerCreateFirewall{
			{Firewall: *fw},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create server from snapshot: %w", err)
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
		AssociatedResources: []core.ResourceRef{
			{Type: "firewall", ID: fmt.Sprintf("%d", fw.ID), Name: fw.Name},
		},
	}, nil
}

func init() {
	core.RegisterProvider("hetzner", func() core.Provider { return &Provider{} })
}

// buildFirewallRules returns the standard set of Hetzner firewall rules for a
// game server: SSH (TCP 22) is always allowed, plus one rule per game port range.
func buildFirewallRules(ports []core.PortRange) []hcloud.FirewallRule {
	_, anyIPv4, _ := net.ParseCIDR("0.0.0.0/0")

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

	rules := []hcloud.FirewallRule{
		buildRule(hcloud.FirewallRuleProtocolTCP, 22, 22),
	}
	for _, pr := range ports {
		proto := hcloud.FirewallRuleProtocolTCP
		if pr.Protocol == "udp" {
			proto = hcloud.FirewallRuleProtocolUDP
		}
		rules = append(rules, buildRule(proto, pr.From, pr.To))
	}
	return rules
}

// firewallName returns a deterministic name for a game-server firewall based on
// the game name and its primary (first) port.
func firewallName(gameName string, ports []core.PortRange) string {
	primaryPort := ports[0].From
	return fmt.Sprintf("fw-%s-%d", gameName, primaryPort)
}

func (p *Provider) deleteAssociatedResources(ctx context.Context, client *hcloud.Client, request core.DeleteServerRequest) error {
	if !request.RemoveAssociatedResources {
		return nil
	}
	for _, resource := range request.AssociatedResources {
		switch resource.Type {
		case "firewall":
			firewallID, err := strconv.ParseInt(resource.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse associated Hetzner firewall ID %q for server %q: %w", resource.ID, request.ID, err)
			}
			firewall, _, err := client.Firewall.GetByID(ctx, firewallID)
			if err != nil {
				return fmt.Errorf("failed to look up associated Hetzner firewall %q for server %q: %w", resource.ID, request.ID, err)
			}
			if firewall == nil {
				continue
			}
			if _, err := client.Firewall.Delete(ctx, firewall); err != nil {
				return fmt.Errorf("failed to delete associated Hetzner firewall %q for server %q: %w", resource.ID, request.ID, err)
			}
		}
	}

	return nil
}

func waitForServerDeletion(ctx context.Context, client *hcloud.Client, serverID int64) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, deleteWaitTimeout)
	defer cancel()

	ticker := time.NewTicker(deletePollInterval)
	defer ticker.Stop()

	for {
		server, _, err := client.Server.GetByID(deadlineCtx, serverID)
		if err != nil {
			return err
		}
		if server == nil {
			return nil
		}

		select {
		case <-deadlineCtx.Done():
			return fmt.Errorf("timed out waiting for server %d to be deleted", serverID)
		case <-ticker.C:
		}
	}
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
