package thunderstore

import "github.com/openhost/runner/internal/mods"

// Format describes the detected format of a Thunderstore profile payload.
type Format int

const (
	// FormatJSON is a standard Thunderstore JSON profile export.
	FormatJSON Format = iota

	// FormatR2Modman is an r2modman base64-encoded zip export.
	FormatR2Modman
)

// Profile represents a resolved Thunderstore profile with its packages.
type Profile struct {
	Format   Format
	Packages []mods.PackageID
	RawData  []byte
}
