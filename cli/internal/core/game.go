package core

type PortRange struct {
	Protocol string // "tcp" or "udp"
	From     int
	To       int
}

type Game interface {
	Name() string
	Ports() []PortRange
	BuildInitCommand(rawSettings map[string]any) (string, error)
}
