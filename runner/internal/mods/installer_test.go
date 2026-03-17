package mods

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipToDirNormalizesWindowsPaths(t *testing.T) {
	t.Parallel()

	archive := archiveBytes(t, map[string]string{
		"BepInEx\\plugins\\MyMod\\plugin.dll": "dll",
	})
	destination := t.TempDir()

	if err := ExtractZipToDir(archive, destination); err != nil {
		t.Fatalf("ExtractZipToDir returned error: %v", err)
	}

	target := filepath.Join(destination, "BepInEx", "plugins", "MyMod", "plugin.dll")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected normalized extracted file at %q: %v", target, err)
	}
}

func TestSanitizeZipEntryPathRejectsTraversal(t *testing.T) {
	t.Parallel()

	if _, err := SanitizeZipEntryPath("../evil.dll"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func archiveBytes(t *testing.T, files map[string]string) []byte {
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
