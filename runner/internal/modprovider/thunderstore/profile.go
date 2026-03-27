package thunderstore

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/openhost/runner/internal/mods"
)

var dependencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.]*-[A-Za-z0-9][A-Za-z0-9_.-]*-[0-9A-Za-z][0-9A-Za-z+._-]*$`)

// DetectFormat inspects raw payload data and returns its format.
func DetectFormat(data []byte) Format {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "#r2modman") {
		return FormatR2Modman
	}
	return FormatJSON
}

// ExtractPackages extracts package identifiers from payload data of the given format.
func ExtractPackages(data []byte, format Format) ([]mods.PackageID, error) {
	switch format {
	case FormatR2Modman:
		return mods.DecodeR2ModmanExport(data)
	case FormatJSON:
		return extractJSONPackages(data)
	default:
		return extractJSONPackages(data)
	}
}

// extractJSONPackages walks JSON profile data and extracts package identifiers.
// Ported from the jq pipeline in init.sh that walks nested structures.
func extractJSONPackages(data []byte) ([]mods.PackageID, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var packages []mods.PackageID

	// Collect all strings that match the dependency pattern.
	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case string:
			if dependencyPattern.MatchString(val) && !seen[val] {
				seen[val] = true
				if id, err := mods.ParseIdentifier(val); err == nil {
					packages = append(packages, id)
				}
			}
		case map[string]any:
			// Check known key names for package identifiers.
			for _, key := range []string{"full_name", "package_full_name", "package", "identifier", "dependency_string"} {
				if s, ok := val[key].(string); ok {
					walk(s)
				}
			}
			for _, child := range val {
				walk(child)
			}
		case []any:
			for _, item := range val {
				walk(item)
			}
		}
	}

	walk(raw)
	return packages, nil
}
