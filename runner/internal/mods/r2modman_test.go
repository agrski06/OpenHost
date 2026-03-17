package mods

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeR2ModmanPayloadAndExtractManifest(t *testing.T) {
	t.Parallel()

	archive := newTestArchive(t, map[string]string{
		"profile/export.r2x": "mods:\n  - name: denikson-BepInExPack_Valheim\n    major: 5\n    minor: 4\n    patch: 2333\n    enabled: true\n",
	})
	payload := []byte("#r2modman\n" + base64.StdEncoding.EncodeToString(archive))

	decoded, err := DecodeR2ModmanPayload(payload)
	if err != nil {
		t.Fatalf("DecodeR2ModmanPayload returned error: %v", err)
	}
	if !bytes.Equal(decoded, archive) {
		t.Fatalf("decoded archive did not match original archive")
	}

	member, manifest, err := ExtractManifestFromArchive(decoded)
	if err != nil {
		t.Fatalf("ExtractManifestFromArchive returned error: %v", err)
	}
	if member != "profile/export.r2x" {
		t.Fatalf("unexpected manifest member: %q", member)
	}
	if len(manifest) == 0 {
		t.Fatalf("expected manifest data")
	}

	packages, err := ExtractPackageIdentifiersFromR2X(manifest)
	if err != nil {
		t.Fatalf("ExtractPackageIdentifiersFromR2X returned error: %v", err)
	}
	if len(packages) != 1 || packages[0].String() != "denikson-BepInExPack_Valheim-5.4.2333" {
		t.Fatalf("unexpected packages: %#v", packages)
	}
}

func TestDecodeR2ModmanPayloadRejectsNonExport(t *testing.T) {
	t.Parallel()

	if _, err := DecodeR2ModmanPayload([]byte("{}")); err == nil {
		t.Fatal("expected DecodeR2ModmanPayload to fail for non-r2modman payload")
	}
}

func newTestArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		fileWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := fileWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}
