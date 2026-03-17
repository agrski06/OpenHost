package bepinex

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/mods"
)

type Framework struct{}

func New() *Framework { return &Framework{} }

func (f *Framework) Name() string { return "bepinex" }

func (f *Framework) InstallPackage(serverRoot string, pkg mods.PackageIdentifier, archive []byte) error {
	return extractAndInstall(serverRoot, archive, pkg.Name)
}

func (f *Framework) ApplyOverlay(serverRoot string, archive []byte) error {
	return extractAndInstall(serverRoot, archive, "")
}

func extractAndInstall(serverRoot string, archive []byte, bundleName string) error {
	tempDir, err := os.MkdirTemp("", "openhost-bepinex-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if err := mods.ExtractZipToDir(archive, tempDir); err != nil {
		return err
	}

	installRoot, err := resolveInstallRoot(tempDir)
	if err != nil {
		return err
	}
	return installRootContents(serverRoot, installRoot, bundleName)
}

func resolveInstallRoot(base string) (string, error) {
	if hasLauncher(base) {
		return base, nil
	}
	if root, ok, err := findNestedLauncherRoot(base); err != nil {
		return "", err
	} else if ok {
		return root, nil
	}
	if root, ok, err := findNestedContentRoot(base); err != nil {
		return "", err
	} else if ok {
		return root, nil
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return "", err
	}
	var topDirectories []string
	var topNonMetadataEntries []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			topDirectories = append(topDirectories, name)
			continue
		}
		if !isMetadataEntry(name) {
			topNonMetadataEntries = append(topNonMetadataEntries, name)
		}
	}
	if len(topDirectories) == 1 && len(topNonMetadataEntries) == 0 {
		return filepath.Join(base, topDirectories[0]), nil
	}
	return base, nil
}

func hasLauncher(root string) bool {
	for _, name := range []string{"start_server_bepinex.sh", "start_game_bepinex.sh"} {
		if info, err := os.Stat(filepath.Join(root, name)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func findNestedLauncherRoot(base string) (string, bool, error) {
	var candidates []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == base {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		depth := pathDepth(rel)
		if d.IsDir() {
			if depth > 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > 3 {
			return nil
		}
		name := filepath.Base(path)
		if name == "start_server_bepinex.sh" || name == "start_game_bepinex.sh" {
			candidates = append(candidates, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if len(candidates) == 0 {
		return "", false, nil
	}
	sort.Strings(candidates)
	return candidates[0], true, nil
}

func findNestedContentRoot(base string) (string, bool, error) {
	var candidates []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == base {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		depth := pathDepth(rel)
		if d.IsDir() {
			if depth > 3 {
				return filepath.SkipDir
			}
			name := d.Name()
			if name == "BepInEx" || name == "plugins" || name == "patchers" || name == "config" || name == "doorstop_libs" {
				candidates = append(candidates, filepath.Dir(path))
			}
			return nil
		}
		if depth > 3 {
			return nil
		}
		name := filepath.Base(path)
		if name == "start_server_bepinex.sh" || name == "start_game_bepinex.sh" || strings.EqualFold(filepath.Ext(name), ".dll") {
			candidates = append(candidates, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if len(candidates) == 0 {
		return "", false, nil
	}
	sort.Strings(candidates)
	return candidates[0], true, nil
}

func pathDepth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return len(strings.Split(filepath.ToSlash(rel), "/"))
}

func installRootContents(serverRoot string, installRoot string, bundleName string) error {
	if err := os.MkdirAll(filepath.Join(serverRoot, "BepInEx", "plugins"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(serverRoot, "BepInEx", "patchers"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(serverRoot, "BepInEx", "config"), 0o755); err != nil {
		return err
	}

	for _, mapping := range []struct {
		source string
		target string
	}{
		{source: filepath.Join(installRoot, "BepInEx"), target: filepath.Join(serverRoot, "BepInEx")},
		{source: filepath.Join(installRoot, "plugins"), target: filepath.Join(serverRoot, "BepInEx", "plugins")},
		{source: filepath.Join(installRoot, "patchers"), target: filepath.Join(serverRoot, "BepInEx", "patchers")},
		{source: filepath.Join(installRoot, "config"), target: filepath.Join(serverRoot, "BepInEx", "config")},
		{source: filepath.Join(installRoot, "doorstop_libs"), target: filepath.Join(serverRoot, "doorstop_libs")},
	} {
		if info, err := os.Stat(mapping.source); err == nil && info.IsDir() {
			if err := mods.CopyTree(mapping.source, mapping.target); err != nil {
				return err
			}
		}
	}

	for _, name := range []string{"start_server_bepinex.sh", "start_game_bepinex.sh", "doorstop_config.ini", "winhttp.dll"} {
		source := filepath.Join(installRoot, name)
		if info, err := os.Stat(source); err == nil && !info.IsDir() {
			if err := copyFile(source, filepath.Join(serverRoot, name)); err != nil {
				return err
			}
		}
	}

	entries, err := os.ReadDir(installRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		switch name {
		case "BepInEx", "plugins", "patchers", "config", "doorstop_libs", "start_server_bepinex.sh", "start_game_bepinex.sh", "doorstop_config.ini", "winhttp.dll":
			continue
		}
		if isMetadataEntry(name) {
			continue
		}
		source := filepath.Join(installRoot, name)
		target := filepath.Join(serverRoot, name)
		if bundleName != "" {
			target = filepath.Join(serverRoot, "BepInEx", "plugins", bundleName, name)
		}
		if entry.IsDir() {
			if err := mods.CopyTree(source, target); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(source, target); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}

func isMetadataEntry(name string) bool {
	switch {
	case name == "README", strings.HasPrefix(name, "README."), name == "CHANGELOG", strings.HasPrefix(name, "CHANGELOG."), name == "manifest.json", name == "icon.png", name == "LICENSE", strings.HasPrefix(name, "LICENSE."), name == "export.r2x":
		return true
	default:
		return false
	}
}

func init() {
	core.RegisterModFramework("bepinex", func() core.ModFramework { return New() })
}
