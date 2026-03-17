package mods

import "testing"

func TestParseIdentifier(t *testing.T) {
	t.Parallel()

	pkg, err := ParseIdentifier("denikson-BepInExPack_Valheim-5.4.2333")
	if err != nil {
		t.Fatalf("ParseIdentifier returned error: %v", err)
	}

	if pkg.Namespace != "denikson" || pkg.Name != "BepInExPack_Valheim" || pkg.Version != "5.4.2333" {
		t.Fatalf("unexpected package identifier: %#v", pkg)
	}
	if pkg.String() != "denikson-BepInExPack_Valheim-5.4.2333" {
		t.Fatalf("unexpected String() value: %q", pkg.String())
	}
}

func TestParseIdentifierRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "bad code with spaces", "namespace-package", "a/b/c"} {
		if _, err := ParseIdentifier(value); err == nil {
			t.Fatalf("expected ParseIdentifier(%q) to fail", value)
		}
	}
}
