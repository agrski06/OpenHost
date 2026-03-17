package thunderstore

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"
)

func TestParseProfilePayloadJSON(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"mods":["denikson-BepInExPack_Valheim-5.4.2333","denikson-BepInExPack_Valheim-5.4.2333","MyAuthor-AnotherMod-1.0.0"]}`)
	profile, err := parseProfilePayload(payload, "https://example.invalid/profile")
	if err != nil {
		t.Fatalf("parseProfilePayload returned error: %v", err)
	}
	if profile.Format != FormatJSON {
		t.Fatalf("unexpected format: %q", profile.Format)
	}
	if len(profile.Packages) != 2 {
		t.Fatalf("expected 2 unique packages, got %d", len(profile.Packages))
	}
}

func TestParseProfilePayloadR2Modman(t *testing.T) {
	t.Parallel()

	archive := thunderstoreArchive(t, map[string]string{
		"export.r2x": "mods:\n  - name: denikson-BepInExPack_Valheim\n    major: 5\n    minor: 4\n    patch: 2333\n    enabled: true\n",
	})
	payload := []byte("#r2modman\n" + base64.StdEncoding.EncodeToString(archive))

	profile, err := parseProfilePayload(payload, "https://example.invalid/profile")
	if err != nil {
		t.Fatalf("parseProfilePayload returned error: %v", err)
	}
	if profile.Format != FormatR2Modman {
		t.Fatalf("unexpected format: %q", profile.Format)
	}
	if len(profile.OverlayArchive) == 0 {
		t.Fatal("expected overlay archive data for r2modman payload")
	}
	if len(profile.Packages) != 1 || profile.Packages[0].String() != "denikson-BepInExPack_Valheim-5.4.2333" {
		t.Fatalf("unexpected packages: %#v", profile.Packages)
	}
}

func thunderstoreArchive(t *testing.T, files map[string]string) []byte {
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
