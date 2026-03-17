package bepinex

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/openhost/runner/internal/mods"
)

func TestInstallPackageNormalizesNestedBepInExPackLayout(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	archive := bepinexArchive(t, map[string]string{
		"BepInExPack_Valheim/README.md":                              "readme",
		"BepInExPack_Valheim/BepInEx/core/BepInEx.Preloader.dll":     "dll",
		"BepInExPack_Valheim/doorstop_libs/libdoorstop_x64.so":       "so",
		"BepInExPack_Valheim/start_server_bepinex.sh":                "#!/bin/bash\n",
		"BepInExPack_Valheim/doorstop_config.ini":                    "config",
		"BepInExPack_Valheim/BepInEx/plugins/RuntimeFixer/fixer.dll": "plugin",
	})
	pkg := mods.PackageIdentifier{Namespace: "denikson", Name: "BepInExPack_Valheim", Version: "5.4.2333"}

	if err := New().InstallPackage(serverRoot, pkg, archive); err != nil {
		t.Fatalf("InstallPackage returned error: %v", err)
	}

	for _, target := range []string{
		filepath.Join(serverRoot, "BepInEx", "core", "BepInEx.Preloader.dll"),
		filepath.Join(serverRoot, "doorstop_libs", "libdoorstop_x64.so"),
		filepath.Join(serverRoot, "start_server_bepinex.sh"),
		filepath.Join(serverRoot, "doorstop_config.ini"),
		filepath.Join(serverRoot, "BepInEx", "plugins", "RuntimeFixer", "fixer.dll"),
	} {
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("expected extracted runtime file at %q: %v", target, err)
		}
	}
}

func TestInstallPackageMapsTopLevelPluginDirectoriesIntoBepInExTree(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	archive := bepinexArchive(t, map[string]string{
		"MyAuthor-MapPin/plugins/MapPin.dll":         "plugin",
		"MyAuthor-MapPin/config/map-pin.cfg":         "cfg",
		"MyAuthor-MapPin/extra/readme-for-users.txt": "notes",
	})
	pkg := mods.PackageIdentifier{Namespace: "MyAuthor", Name: "MapPin", Version: "1.0.0"}

	if err := New().InstallPackage(serverRoot, pkg, archive); err != nil {
		t.Fatalf("InstallPackage returned error: %v", err)
	}

	for _, target := range []string{
		filepath.Join(serverRoot, "BepInEx", "plugins", "MapPin.dll"),
		filepath.Join(serverRoot, "BepInEx", "config", "map-pin.cfg"),
		filepath.Join(serverRoot, "BepInEx", "plugins", "MapPin", "extra", "readme-for-users.txt"),
	} {
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("expected mapped package content at %q: %v", target, err)
		}
	}
}

func TestApplyOverlayNormalizesProfileWrappedContent(t *testing.T) {
	t.Parallel()

	serverRoot := t.TempDir()
	archive := bepinexArchive(t, map[string]string{
		"profile/BepInEx/config/server-sync.cfg":   "cfg",
		"profile/plugins/OverlayPlugin.dll":        "plugin",
		"profile/doorstop_libs/libdoorstop_x64.so": "so",
		"profile/export.r2x":                       "manifest",
	})

	if err := New().ApplyOverlay(serverRoot, archive); err != nil {
		t.Fatalf("ApplyOverlay returned error: %v", err)
	}

	for _, target := range []string{
		filepath.Join(serverRoot, "BepInEx", "config", "server-sync.cfg"),
		filepath.Join(serverRoot, "BepInEx", "plugins", "OverlayPlugin.dll"),
		filepath.Join(serverRoot, "doorstop_libs", "libdoorstop_x64.so"),
	} {
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("expected normalized overlay content at %q: %v", target, err)
		}
	}
	if _, err := os.Stat(filepath.Join(serverRoot, "export.r2x")); !os.IsNotExist(err) {
		t.Fatalf("expected overlay metadata to be skipped, got err=%v", err)
	}
}

func bepinexArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}
