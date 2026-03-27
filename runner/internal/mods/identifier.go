// Package mods provides shared mod utility functions: identifier parsing,
// package downloading, and zip extraction.
package mods

import (
	"fmt"
	"strings"
)

// PackageID represents a parsed Thunderstore-style package identifier.
type PackageID struct {
	Namespace string
	Name      string
	Version   string
}

// String returns the canonical "Namespace-Name-Version" string.
func (p PackageID) String() string {
	return fmt.Sprintf("%s-%s-%s", p.Namespace, p.Name, p.Version)
}

// ParseIdentifier parses a "Namespace-PackageName-1.2.3" string into a PackageID.
func ParseIdentifier(s string) (PackageID, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, "-", 3)
	if len(parts) < 3 {
		return PackageID{}, fmt.Errorf("invalid package identifier %q: expected Namespace-Name-Version", s)
	}

	return PackageID{
		Namespace: parts[0],
		Name:      parts[1],
		Version:   parts[2],
	}, nil
}
