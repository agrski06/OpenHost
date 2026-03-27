package core

// ModFramework handles installation of mods into a game server directory.
type ModFramework interface {
	// Name returns the framework name (e.g. "bepinex", "fabric", "forge").
	Name() string

	// Install extracts and merges downloaded mods into the server root.
	Install(mods []DownloadedMod, serverRoot string) error

	// Validate checks that the mod framework was installed correctly.
	Validate(serverRoot string) error
}
