package core

import "fmt"

var (
	gameSetups    = map[string]func() GameSetup{}
	modProviders  = map[string]func() ModProvider{}
	modFrameworks = map[string]func() ModFramework{}
)

func RegisterGameSetup(name string, factory func() GameSetup) {
	gameSetups[name] = factory
}

func GetGameSetup(name string) (GameSetup, error) {
	factory, ok := gameSetups[name]
	if !ok {
		return nil, fmt.Errorf("game setup %q is not registered", name)
	}
	return factory(), nil
}

func RegisterModProvider(name string, factory func() ModProvider) {
	modProviders[name] = factory
}

func GetModProvider(name string) (ModProvider, error) {
	factory, ok := modProviders[name]
	if !ok {
		return nil, fmt.Errorf("mod provider %q is not registered", name)
	}
	return factory(), nil
}

func RegisterModFramework(name string, factory func() ModFramework) {
	modFrameworks[name] = factory
}

func GetModFramework(name string) (ModFramework, error) {
	factory, ok := modFrameworks[name]
	if !ok {
		return nil, fmt.Errorf("mod framework %q is not registered", name)
	}
	return factory(), nil
}
