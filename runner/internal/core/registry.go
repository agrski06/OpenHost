package core

import "fmt"

var gameSetupRegistry = map[string]func() GameSetup{}
var modProviderRegistry = map[string]func() ModProvider{}
var modFrameworkRegistry = map[string]func() ModFramework{}

// RegisterGameSetup registers a factory function for a game setup implementation.
func RegisterGameSetup(name string, f func() GameSetup) {
	gameSetupRegistry[name] = f
}

// GetGameSetup returns a new instance of the named game setup.
func GetGameSetup(name string) (GameSetup, error) {
	f, ok := gameSetupRegistry[name]
	if !ok {
		return nil, fmt.Errorf("game setup not found: %s", name)
	}
	return f(), nil
}

// RegisterModProvider registers a factory function for a mod provider implementation.
func RegisterModProvider(name string, f func() ModProvider) {
	modProviderRegistry[name] = f
}

// GetModProvider returns a new instance of the named mod provider.
func GetModProvider(name string) (ModProvider, error) {
	f, ok := modProviderRegistry[name]
	if !ok {
		return nil, fmt.Errorf("mod provider not found: %s", name)
	}
	return f(), nil
}

// RegisterModFramework registers a factory function for a mod framework implementation.
func RegisterModFramework(name string, f func() ModFramework) {
	modFrameworkRegistry[name] = f
}

// GetModFramework returns a new instance of the named mod framework.
func GetModFramework(name string) (ModFramework, error) {
	f, ok := modFrameworkRegistry[name]
	if !ok {
		return nil, fmt.Errorf("mod framework not found: %s", name)
	}
	return f(), nil
}
