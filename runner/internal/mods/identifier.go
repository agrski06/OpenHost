package mods

import (
	"fmt"
	"regexp"
	"strings"
)

type PackageIdentifier struct {
	Namespace string
	Name      string
	Version   string
}

var identifierPattern = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.]*)-([A-Za-z0-9][A-Za-z0-9_.-]*)-([0-9A-Za-z][0-9A-Za-z+._-]*)$`)

func ParseIdentifier(value string) (PackageIdentifier, error) {
	value = strings.TrimSpace(value)
	matches := identifierPattern.FindStringSubmatch(value)
	if matches == nil {
		return PackageIdentifier{}, fmt.Errorf("invalid package identifier %q", value)
	}

	return PackageIdentifier{
		Namespace: matches[1],
		Name:      matches[2],
		Version:   matches[3],
	}, nil
}

func (p PackageIdentifier) String() string {
	return p.Namespace + "-" + p.Name + "-" + p.Version
}
