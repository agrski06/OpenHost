package mods

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// DecodeR2ModmanExport decodes an r2modman export payload (starts with "#r2modman"
// header, followed by base64-encoded zip data) and extracts the package list
// from the export.r2x manifest inside the zip.
func DecodeR2ModmanExport(data []byte) ([]PackageID, error) {
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "#r2modman" {
		return nil, fmt.Errorf("not an r2modman export: missing #r2modman header")
	}

	// The rest after the header is base64-encoded zip data (may have whitespace).
	encoded := strings.ReplaceAll(lines[1], "\r", "")
	encoded = strings.ReplaceAll(encoded, "\n", "")
	encoded = strings.TrimSpace(encoded)

	zipData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode r2modman payload: %w", err)
	}

	return extractR2XPackages(zipData)
}

// extractR2XPackages opens a zip archive in memory and finds the export.r2x
// manifest, then parses package identifiers from it.
func extractR2XPackages(zipData []byte) ([]PackageID, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open r2modman zip: %w", err)
	}

	var manifestFile *zip.File
	for _, f := range reader.File {
		name := f.Name
		// Match "export.r2x" at any nesting level.
		if strings.HasSuffix(name, "export.r2x") || name == "export.r2x" {
			manifestFile = f
			break
		}
	}

	if manifestFile == nil {
		return nil, fmt.Errorf("r2modman archive does not contain export.r2x")
	}

	rc, err := manifestFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open export.r2x: %w", err)
	}
	defer func() { _ = rc.Close() }()

	manifestData, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read export.r2x: %w", err)
	}

	return parseR2XManifest(manifestData)
}

// parseR2XManifest parses the YAML-like r2modman export.r2x format.
// It extracts enabled packages with their name and version (major.minor.patch).
func parseR2XManifest(data []byte) ([]PackageID, error) {
	type entry struct {
		name    string
		major   string
		minor   string
		patch   string
		enabled string
	}

	var packages []PackageID
	var current entry
	seen := map[string]bool{}

	nameRe := regexp.MustCompile(`^\s*-\s+name:\s*(.+)$`)
	majorRe := regexp.MustCompile(`^\s*major:\s*(.+)$`)
	minorRe := regexp.MustCompile(`^\s*minor:\s*(.+)$`)
	patchRe := regexp.MustCompile(`^\s*patch:\s*(.+)$`)
	enabledRe := regexp.MustCompile(`^\s*enabled:\s*(.+)$`)

	emit := func() {
		if current.name != "" && current.enabled == "true" &&
			current.major != "" && current.minor != "" && current.patch != "" {
			version := fmt.Sprintf("%s.%s.%s", current.major, current.minor, current.patch)
			id := fmt.Sprintf("%s-%s", current.name, version)
			if !seen[id] {
				seen[id] = true
				parsed, err := ParseIdentifier(id)
				if err == nil {
					packages = append(packages, parsed)
				}
			}
		}
		current = entry{}
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if m := nameRe.FindStringSubmatch(line); m != nil {
			emit()
			current.name = trimQuotes(strings.TrimSpace(m[1]))
		} else if m := majorRe.FindStringSubmatch(line); m != nil {
			current.major = trimQuotes(strings.TrimSpace(m[1]))
		} else if m := minorRe.FindStringSubmatch(line); m != nil {
			current.minor = trimQuotes(strings.TrimSpace(m[1]))
		} else if m := patchRe.FindStringSubmatch(line); m != nil {
			current.patch = trimQuotes(strings.TrimSpace(m[1]))
		} else if m := enabledRe.FindStringSubmatch(line); m != nil {
			current.enabled = trimQuotes(strings.TrimSpace(m[1]))
		}
	}
	emit() // flush last entry

	return packages, nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
