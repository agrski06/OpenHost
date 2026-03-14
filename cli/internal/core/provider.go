package core

type CreateServerRequest struct {
	Name             string
	GameName         string
	Ports            []PortRange
	ProviderSettings map[string]any
	UserData         string
}

// Provider defines an interface for cloud providers that can
// prepare and run game servers.
type Provider interface {
	Name() string

	// CreateServer executes the final step to create and start the server
	// on the cloud provider, using the prepared configuration.
	CreateServer(request CreateServerRequest) (Server, error)
}
