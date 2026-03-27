package core

import "context"

type CreateServerRequest struct {
	Name             string
	GameName         string
	Ports            []PortRange
	ProviderSettings map[string]any
	UserData         string
}

type DeleteServerRequest struct {
	ID                        string
	GameName                  string
	Ports                     []PortRange
	AssociatedResources       []ResourceRef
	RemoveAssociatedResources bool

	// Optional: snapshot image IDs associated with this server that should be
	// removed when RemoveAssociatedResources is true.
	SnapshotIDs []string
}

// StopServerAndSnapshotRequest describes a provider-specific workflow where
// a server should be stopped gracefully and then snapshotted.
//
// Providers may interpret "graceful" differently; for example, it can mean
// powering off the VM (after an in-guest service stop) before creating an image.
type StopServerAndSnapshotRequest struct {
	ID string

	// Optional metadata for provider-side tagging/naming.
	Name     string
	GameName string

	// Optional: the server's public IP, useful for providers that perform an
	// in-guest graceful stop (e.g., via SSH/systemd) before powering off.
	PublicIP string

	// Optional snapshot description/name.
	SnapshotDescription string
}

type StopServerRequest struct {
	ID string
}

type StartServerRequest struct {
	ID string
}

type SnapshotResult struct {
	// Provider-specific identifier of the created snapshot image.
	SnapshotID string

	// Human readable description/name.
	SnapshotDescription string
}

// StartServerFromSnapshotRequest describes creating (or restoring) a server from
// a provider snapshot image.
//
// For providers like Hetzner Cloud, this is typically implemented as creating a
// new server using the snapshot Image as the boot disk.
type StartServerFromSnapshotRequest struct {
	// SnapshotID is the provider-specific snapshot image identifier.
	SnapshotID string

	// Name is the new server name.
	Name string

	// GameName influences firewall naming and port exposure.
	GameName string

	// Ports that must be exposed for the game to work.
	Ports []PortRange

	// ProviderSettings carries cloud provider specific settings (e.g.
	// Hetzner plan/location).
	ProviderSettings map[string]any
}

// Provider defines an interface for cloud providers that can
// prepare and run game servers.
type Provider interface {
	Name() string

	// CreateServer executes the final step to create and start the server
	// on the cloud provider, using the prepared configuration.
	CreateServer(ctx context.Context, request CreateServerRequest) (*Server, error)

	// GetServerStatus retrieves the infrastructure status for the
	// server identified by the provider-native ID.
	GetServerStatus(ctx context.Context, id string) (*InfrastructureStatus, error)

	// StopServer stops (powers off) a server.
	StopServer(ctx context.Context, request StopServerRequest) error

	// StartServer starts (powers on) a server.
	StartServer(ctx context.Context, request StartServerRequest) error

	// DeleteServer removes the server identified by the provider-native ID.
	DeleteServer(ctx context.Context, request DeleteServerRequest) error

	// StopServerAndSnapshot gracefully stops a server (including any in-guest
	// services where supported) and then creates a provider snapshot image.
	StopServerAndSnapshot(ctx context.Context, request StopServerAndSnapshotRequest) (*SnapshotResult, error)

	// StartServerFromSnapshot creates a new server from a snapshot image.
	StartServerFromSnapshot(ctx context.Context, request StartServerFromSnapshotRequest) (*Server, error)
}
