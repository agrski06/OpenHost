package core

import "fmt"

var providerRegistry = map[string]func() Provider{}
var gameRegistry = map[string]func() Game{}

func RegisterProvider(name string, f func() Provider) {
	providerRegistry[name] = f
}

func RegisterGame(name string, f func() Game) {
	gameRegistry[name] = f
}

func GetProvider(name string) (Provider, error) {
	f, ok := providerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return f(), nil
}

func GetGame(name string) (Game, error) {
	f, ok := gameRegistry[name]
	if !ok {
		return nil, fmt.Errorf("game not found: %s", name)
	}
	return f(), nil
}
