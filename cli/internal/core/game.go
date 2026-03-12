package core

type Game interface {
	Name() string
	Port() int
	Protocol() string // tcp, udp, both

	// BuildInitCommand returns the actual command(s) for this game,
	// based on the per-server GameConfig
	BuildInitCommand() string
}
