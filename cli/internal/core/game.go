package core

import "github.com/openhost/runnerconfig"

type PortRange struct {
	Protocol string // "tcp" or "udp"
	From     int
	To       int
}

type Game interface {
	Name() string
	Ports() []PortRange
	BuildRunnerConfig(rawSettings map[string]any) (*runnerconfig.RunnerConfig, error)
}
