package core

import (
	"context"

	"github.com/openhost/runner/internal/mods"
	"github.com/openhost/runnerconfig"
)

type ResolvedModpack struct {
	Source         runnerconfig.ModSource
	Packages       []mods.PackageIdentifier
	OverlayArchive []byte
	Metadata       map[string]any
}

type ModProvider interface {
	Name() string
	Resolve(ctx context.Context, source runnerconfig.ModSource) (*ResolvedModpack, error)
	DownloadPackage(ctx context.Context, pkg mods.PackageIdentifier) ([]byte, error)
}
