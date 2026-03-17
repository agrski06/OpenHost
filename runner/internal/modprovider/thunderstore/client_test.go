package thunderstore

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveProfileFallsBackAcrossEndpoints(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/experimental/legacyprofile/get/valheim/ABC123/", "/api/experimental/legacyprofile/get/valheim/ABC123":
			http.NotFound(w, r)
		case "/c/valheim/api/experimental/legacyprofile/get/valheim/ABC123/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mods":["denikson-BepInExPack_Valheim-5.4.2333"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := New(&http.Client{Timeout: 2 * time.Second})
	profile, err := provider.ResolveProfile(context.Background(), "ABC123", map[string]any{"base_url": server.URL})
	if err != nil {
		t.Fatalf("ResolveProfile returned error: %v", err)
	}
	if profile.Endpoint != server.URL+"/c/valheim/api/experimental/legacyprofile/get/valheim/ABC123/" {
		t.Fatalf("unexpected endpoint: %q", profile.Endpoint)
	}
	if len(profile.Packages) != 1 || profile.Packages[0].String() != "denikson-BepInExPack_Valheim-5.4.2333" {
		t.Fatalf("unexpected packages: %#v", profile.Packages)
	}
}
