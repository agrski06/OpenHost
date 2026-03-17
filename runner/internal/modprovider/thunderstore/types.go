package thunderstore

import "github.com/openhost/runner/internal/mods"

type ProfileFormat string

const (
	FormatJSON     ProfileFormat = "json"
	FormatR2Modman ProfileFormat = "r2modman"
)

type Profile struct {
	Format         ProfileFormat
	Endpoint       string
	Packages       []mods.PackageIdentifier
	OverlayArchive []byte
}
