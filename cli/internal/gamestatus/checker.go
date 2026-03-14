package gamestatus

import (
	"fmt"

	"github.com/openhost/cli/internal/core"
)

type State string

const (
	StateUnknown     State = "unknown"
	StateReady       State = "ready"
	StateUnreachable State = "unreachable"
	StateQueryFailed State = "query_failed"
	StateStarting    State = "starting"
)

type Target struct {
	GameName string
	PublicIP string
	Ports    []core.PortRange
}

type Status struct {
	State       State
	Detail      string
	Reachable   bool
	PlayerCount *int
}

type Checker interface {
	GameName() string
	Check(target Target) (*Status, error)
}

var registry = map[string]Checker{}

func Register(checker Checker) {
	registry[checker.GameName()] = checker
}

func Get(gameName string) (Checker, error) {
	checker, ok := registry[gameName]
	if !ok {
		return nil, fmt.Errorf("game status checker not found: %s", gameName)
	}
	return checker, nil
}
