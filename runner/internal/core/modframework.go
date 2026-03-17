package core

import "github.com/openhost/runner/internal/mods"

type ModFramework interface {
	Name() string
	InstallPackage(serverRoot string, pkg mods.PackageIdentifier, archive []byte) error
	ApplyOverlay(serverRoot string, archive []byte) error
}
