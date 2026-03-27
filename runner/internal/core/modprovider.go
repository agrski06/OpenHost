package core

import (
	"context"

	"github.com/openhost/runnerconfig"
)

// Package describes a resolved mod package ready for download.
type Package struct {
	// Identifier is the package identifier (e.g. "Namespace-Name-1.2.3").
	Identifier string

	// URL is the download URL for the package archive.
	URL string
}

// DownloadedMod represents a mod package that has been downloaded to disk.
type DownloadedMod struct {
	// Identifier is the package identifier.
	Identifier string

	// LocalPath is the path to the downloaded zip/jar on disk.
	LocalPath string
}

// ModProvider resolves mod sources into downloadable packages.
type ModProvider interface {
	// Name returns the provider name (e.g. "thunderstore", "curseforge").
	Name() string

	// Resolve takes a mod source and returns the list of packages to download.
	Resolve(ctx context.Context, source runnerconfig.ModSource) ([]Package, error)
}
