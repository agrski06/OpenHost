// Package bepinex implements the BepInEx mod framework for the runner.
// It extracts and merges downloaded mod packages into the game server directory,
// handling the nested BepInEx directory structure.
package bepinex

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/mods"
)

// Framework implements core.ModFramework for BepInEx.
type Framework struct{}

func (f *Framework) Name() string { return "bepinex" }

// Install extracts each downloaded mod zip and merges the contents into the
// server root following BepInEx conventions.
func (f *Framework) Install(downloaded []core.DownloadedMod, serverRoot string) error {
	// Ensure base directories exist.
	for _, dir := range []string{
		filepath.Join(serverRoot, "BepInEx", "plugins"),
		filepath.Join(serverRoot, "BepInEx", "patchers"),
		filepath.Join(serverRoot, "BepInEx", "config"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	for _, mod := range downloaded {
		unpackDir := mod.LocalPath + ".unpack"
		if err := os.MkdirAll(unpackDir, 0755); err != nil {
			return fmt.Errorf("create unpack dir: %w", err)
		}

		if err := mods.ExtractZip(mod.LocalPath, unpackDir); err != nil {
			return fmt.Errorf("extract %s: %w", mod.Identifier, err)
		}

		installRoot := resolveInstallRoot(unpackDir)
		log.Printf("[bepinex] installing %s from %s", mod.Identifier, installRoot)

		// Extract package name from identifier (Namespace-Name-Version → Name).
		packageName := extractPackageName(mod.Identifier)
		if err := installPackageContents(installRoot, serverRoot, packageName); err != nil {
			return fmt.Errorf("install %s: %w", mod.Identifier, err)
		}
	}

	return nil
}

// resolveInstallRoot finds the "real" root of the mod package contents,
// handling nested BepInEx directory structures. Ported from the bash
// resolve_package_install_root function.
func resolveInstallRoot(unpackDir string) string {
	// Check for BepInEx launcher scripts at top level.
	for _, launcher := range []string{"start_server_bepinex.sh", "start_game_bepinex.sh"} {
		if fileExists(filepath.Join(unpackDir, launcher)) {
			return unpackDir
		}
	}

	// Search nested directories for launcher scripts.
	launcherRoot := findNestedPath(unpackDir, func(path string, info fs.FileInfo) bool {
		name := info.Name()
		return name == "start_server_bepinex.sh" || name == "start_game_bepinex.sh"
	})
	if launcherRoot != "" {
		return launcherRoot
	}

	// Search for BepInEx content markers.
	contentRoot := findNestedPath(unpackDir, func(path string, info fs.FileInfo) bool {
		name := info.Name()
		if info.IsDir() {
			return name == "BepInEx" || name == "plugins" || name == "patchers" || name == "config" || name == "doorstop_libs"
		}
		return strings.EqualFold(filepath.Ext(name), ".dll") ||
			name == "start_server_bepinex.sh" || name == "start_game_bepinex.sh"
	})
	if contentRoot != "" {
		return contentRoot
	}

	// If there's a single subdirectory with no non-metadata files, use it.
	entries, err := os.ReadDir(unpackDir)
	if err == nil {
		var dirs []string
		hasNonMetadata := false
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			} else if !isMetadataEntry(e.Name()) {
				hasNonMetadata = true
			}
		}
		if len(dirs) == 1 && !hasNonMetadata {
			return filepath.Join(unpackDir, dirs[0])
		}
	}

	return unpackDir
}

// installPackageContents merges the install root into the server root.
func installPackageContents(installRoot, serverRoot, packageName string) error {
	// Merge known BepInEx directories.
	mergeMap := map[string]string{
		"BepInEx":       filepath.Join(serverRoot, "BepInEx"),
		"plugins":       filepath.Join(serverRoot, "BepInEx", "plugins"),
		"patchers":      filepath.Join(serverRoot, "BepInEx", "patchers"),
		"config":        filepath.Join(serverRoot, "BepInEx", "config"),
		"doorstop_libs": filepath.Join(serverRoot, "doorstop_libs"),
	}

	for srcName, destDir := range mergeMap {
		srcDir := filepath.Join(installRoot, srcName)
		if dirExists(srcDir) {
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}
			if err := copyDirContents(srcDir, destDir); err != nil {
				return fmt.Errorf("merge %s: %w", srcName, err)
			}
		}
	}

	// Copy launcher/runtime files to server root.
	runtimeFiles := []string{
		"start_server_bepinex.sh",
		"start_game_bepinex.sh",
		"doorstop_config.ini",
		"winhttp.dll",
	}
	for _, name := range runtimeFiles {
		src := filepath.Join(installRoot, name)
		if fileExists(src) {
			if err := copyFile(src, filepath.Join(serverRoot, name)); err != nil {
				return fmt.Errorf("copy %s: %w", name, err)
			}
		}
	}

	// Bundle remaining files into BepInEx/plugins/<package_name>/.
	bundleRoot := filepath.Join(serverRoot, "BepInEx", "plugins", packageName)
	skipSet := map[string]bool{
		"BepInEx": true, "plugins": true, "patchers": true, "config": true,
		"doorstop_libs": true, "start_server_bepinex.sh": true,
		"start_game_bepinex.sh": true, "doorstop_config.ini": true,
		"winhttp.dll": true,
	}

	entries, err := os.ReadDir(installRoot)
	if err != nil {
		return err
	}

	for _, e := range entries {
		name := e.Name()
		if skipSet[name] || isMetadataEntry(name) {
			continue
		}

		if err := os.MkdirAll(bundleRoot, 0755); err != nil {
			return err
		}

		src := filepath.Join(installRoot, name)
		dst := filepath.Join(bundleRoot, name)
		if e.IsDir() {
			if err := copyDirContents(src, dst); err != nil {
				return fmt.Errorf("bundle %s: %w", name, err)
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("bundle %s: %w", name, err)
			}
		}
	}

	return nil
}

func extractPackageName(identifier string) string {
	parts := strings.SplitN(identifier, "-", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return identifier
}

func isMetadataEntry(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case lower == "manifest.json", lower == "icon.png":
		return true
	case strings.HasPrefix(lower, "readme"), strings.HasPrefix(lower, "changelog"),
		strings.HasPrefix(lower, "license"):
		return true
	}
	return false
}

func findNestedPath(root string, match func(string, fs.FileInfo) bool) string {
	var result string
	_ = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil || path == root {
			return nil
		}
		// Limit depth to 3 levels.
		rel, _ := filepath.Rel(root, path)
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > 3 {
			return filepath.SkipDir
		}
		if match(path, info) {
			result = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	return result
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyDirContents(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

func init() {
	core.RegisterModFramework("bepinex", func() core.ModFramework { return &Framework{} })
}
