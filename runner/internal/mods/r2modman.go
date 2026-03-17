package mods

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

func DecodeR2ModmanPayload(payload []byte) ([]byte, error) {
	lines := bytes.SplitN(payload, []byte("\n"), 2)
	if len(lines) != 2 || strings.TrimSpace(string(lines[0])) != "#r2modman" {
		return nil, fmt.Errorf("payload is not an r2modman export")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(lines[1])))
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func ExtractManifestFromArchive(archive []byte) (string, []byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return "", nil, err
	}

	for _, file := range reader.File {
		name := filepathLike(file.Name)
		if strings.HasSuffix(name, "/export.r2x") || name == "export.r2x" {
			rc, err := file.Open()
			if err != nil {
				return "", nil, err
			}
			data, err := io.ReadAll(rc)
			closeErr := rc.Close()
			if err != nil {
				return "", nil, err
			}
			if closeErr != nil {
				return "", nil, closeErr
			}
			return file.Name, data, nil
		}
	}

	return "", nil, fmt.Errorf("export.r2x not found in r2modman archive")
}

func ExtractPackageIdentifiersFromR2X(manifest []byte) ([]PackageIdentifier, error) {
	scanner := bufio.NewScanner(bytes.NewReader(manifest))
	type entry struct {
		name    string
		major   string
		minor   string
		patch   string
		enabled bool
	}
	current := entry{}
	seen := map[string]struct{}{}
	var identifiers []PackageIdentifier

	emit := func() error {
		if current.name == "" || !current.enabled || current.major == "" || current.minor == "" || current.patch == "" {
			return nil
		}
		pkg, err := ParseIdentifier(fmt.Sprintf("%s-%s.%s.%s", current.name, current.major, current.minor, current.patch))
		if err != nil {
			return err
		}
		if _, ok := seen[pkg.String()]; ok {
			return nil
		}
		seen[pkg.String()] = struct{}{}
		identifiers = append(identifiers, pkg)
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "- name:"):
			if err := emit(); err != nil {
				return nil, err
			}
			current = entry{name: trimR2Value(strings.TrimPrefix(line, "- name:"))}
		case strings.HasPrefix(line, "name:"):
			if err := emit(); err != nil {
				return nil, err
			}
			current = entry{name: trimR2Value(strings.TrimPrefix(line, "name:"))}
		case strings.HasPrefix(line, "major:"):
			current.major = trimR2Value(strings.TrimPrefix(line, "major:"))
		case strings.HasPrefix(line, "minor:"):
			current.minor = trimR2Value(strings.TrimPrefix(line, "minor:"))
		case strings.HasPrefix(line, "patch:"):
			current.patch = trimR2Value(strings.TrimPrefix(line, "patch:"))
		case strings.HasPrefix(line, "enabled:"):
			current.enabled = strings.EqualFold(trimR2Value(strings.TrimPrefix(line, "enabled:")), "true")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := emit(); err != nil {
		return nil, err
	}
	return identifiers, nil
}

func filepathLike(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

func trimR2Value(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return value
}
