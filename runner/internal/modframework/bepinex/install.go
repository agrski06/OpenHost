package bepinex

import (
	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/mods"
)

type Framework struct{}

func New() *Framework { return &Framework{} }

func (f *Framework) Name() string { return "bepinex" }

func (f *Framework) InstallPackage(serverRoot string, pkg mods.PackageIdentifier, archive []byte) error {
	return mods.InstallArchive(archive, serverRoot)
}

func (f *Framework) ApplyOverlay(serverRoot string, archive []byte) error {
	return mods.ExtractZipToDir(archive, serverRoot)
}

func init() {
	core.RegisterModFramework("bepinex", func() core.ModFramework { return New() })
}
