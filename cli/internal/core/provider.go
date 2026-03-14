package core

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
}

// Provider defines an interface for cloud providers that can
// prepare and run game servers.
type Provider interface {
	Name() string

	// CreateServer executes the final step to create and start the server
	// on the cloud provider, using the prepared configuration.
	CreateServer(request CreateServerRequest) (*Server, error)

	// GetServerStatus retrieves the infrastructure status for the
	// server identified by the provider-native ID.
	GetServerStatus(id string) (*InfrastructureStatus, error)

	// DeleteServer removes the server identified by the provider-native ID.
	DeleteServer(request DeleteServerRequest) error
}
