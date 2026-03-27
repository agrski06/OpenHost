package valheim

import (
	"time"

	"github.com/openhost/cli/internal/gamestatus"
	"github.com/openhost/cli/internal/gamestatus/a2s"
)

const (
	defaultTimeout = 2 * time.Second
	queryPort      = 2457
)

// Checker implements gamestatus.Checker for Valheim using the shared A2S protocol.
type Checker struct{}

// NewChecker returns a new Valheim status checker.
func NewChecker() *Checker {
	return &Checker{}
}

// GameName returns "valheim".
func (c *Checker) GameName() string {
	return "valheim"
}

// Check queries the Valheim server using the A2S protocol.
func (c *Checker) Check(target gamestatus.Target) (*gamestatus.Status, error) {
	return a2s.Query(target, a2s.Options{
		QueryPort: queryPort,
		Timeout:   defaultTimeout,
	})
}

func init() {
	gamestatus.Register(NewChecker())
}
