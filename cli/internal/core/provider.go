package core

// Provider defines an interface for cloud providers that can
// prepare and run game servers.
type Provider interface {
	Name() string

	// RunServer executes the final step to create and start the server
	// on the cloud provider, using the prepared configuration.
	RunServer(name string, game Game, rawSettings map[string]any, gameSettings map[string]any) (Server, error)
}
