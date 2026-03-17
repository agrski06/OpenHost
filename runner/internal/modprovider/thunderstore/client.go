package thunderstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/mods"
	"github.com/openhost/runnerconfig"
)

const defaultBaseURL = "https://thunderstore.io"

type Provider struct {
	client  *http.Client
	baseURL string
}

func New(client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Provider{client: client, baseURL: defaultBaseURL}
}

func (p *Provider) Name() string { return "thunderstore" }

func (p *Provider) Resolve(ctx context.Context, source runnerconfig.ModSource) (*core.ResolvedModpack, error) {
	code := strings.TrimSpace(source.Code)
	if code == "" {
		return nil, fmt.Errorf("thunderstore source code is required")
	}

	profile, err := p.ResolveProfile(ctx, code, source.Settings)
	if err != nil {
		return nil, err
	}

	return &core.ResolvedModpack{
		Source:         source,
		Packages:       profile.Packages,
		OverlayArchive: profile.OverlayArchive,
		Metadata: map[string]any{
			"endpoint": profile.Endpoint,
			"format":   string(profile.Format),
		},
	}, nil
}

func (p *Provider) ResolveProfile(ctx context.Context, code string, settings map[string]any) (*Profile, error) {
	baseURL := p.baseURL
	if raw := strings.TrimSpace(asString(settings["base_url"])); raw != "" {
		baseURL = strings.TrimRight(raw, "/")
	}

	var lastErr error
	for _, endpoint := range buildProfileEndpoints(baseURL, code) {
		payload, err := p.get(ctx, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		profile, err := parseProfilePayload(payload, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		if len(profile.Packages) == 0 {
			lastErr = fmt.Errorf("profile %q resolved from %s but did not contain any installable packages", code, endpoint)
			continue
		}
		return profile, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("failed to resolve profile %q", code)
	}
	return nil, fmt.Errorf("resolve Thunderstore profile %q: %w", code, lastErr)
}

func (p *Provider) DownloadPackage(ctx context.Context, pkg mods.PackageIdentifier) ([]byte, error) {
	url := fmt.Sprintf("%s/package/download/%s/%s/%s/", p.baseURL, pkg.Namespace, pkg.Name, pkg.Version)
	return p.get(ctx, url)
}

func (p *Provider) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GET %s returned %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

func buildProfileEndpoints(baseURL string, code string) []string {
	baseURL = strings.TrimRight(baseURL, "/")
	return []string{
		fmt.Sprintf("%s/api/experimental/legacyprofile/get/valheim/%s/", baseURL, code),
		fmt.Sprintf("%s/api/experimental/legacyprofile/get/valheim/%s", baseURL, code),
		fmt.Sprintf("%s/c/valheim/api/experimental/legacyprofile/get/valheim/%s/", baseURL, code),
		fmt.Sprintf("%s/c/valheim/api/experimental/legacyprofile/get/valheim/%s", baseURL, code),
		fmt.Sprintf("%s/api/experimental/profile/get/%s/", baseURL, code),
		fmt.Sprintf("%s/api/experimental/profile/get/%s", baseURL, code),
		fmt.Sprintf("%s/c/valheim/api/experimental/profile/get/%s/", baseURL, code),
		fmt.Sprintf("%s/c/valheim/api/experimental/profile/get/%s", baseURL, code),
		fmt.Sprintf("%s/api/experimental/legacyprofile/get/%s/", baseURL, code),
		fmt.Sprintf("%s/api/experimental/legacyprofile/get/%s", baseURL, code),
	}
}

func detectProfileFormat(payload []byte) (ProfileFormat, error) {
	trimmed := strings.TrimSpace(string(payload))
	if strings.HasPrefix(trimmed, "#r2modman") {
		return FormatR2Modman, nil
	}
	if json.Valid([]byte(trimmed)) {
		return FormatJSON, nil
	}
	return "", fmt.Errorf("unsupported Thunderstore payload format")
}

func extractPackagesFromJSON(payload []byte) ([]mods.PackageIdentifier, error) {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var packages []mods.PackageIdentifier
	var visit func(value any)
	visit = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			for _, inner := range typed {
				visit(inner)
			}
		case []any:
			for _, inner := range typed {
				visit(inner)
			}
		case string:
			pkg, err := mods.ParseIdentifier(typed)
			if err != nil {
				return
			}
			if _, ok := seen[pkg.String()]; ok {
				return
			}
			seen[pkg.String()] = struct{}{}
			packages = append(packages, pkg)
		}
	}
	visit(decoded)
	return packages, nil
}

func parseProfilePayload(payload []byte, endpoint string) (*Profile, error) {
	format, err := detectProfileFormat(payload)
	if err != nil {
		return nil, err
	}

	profile := &Profile{Format: format, Endpoint: endpoint}
	switch format {
	case FormatJSON:
		profile.Packages, err = extractPackagesFromJSON(payload)
		if err != nil {
			return nil, err
		}
	case FormatR2Modman:
		archive, err := mods.DecodeR2ModmanPayload(payload)
		if err != nil {
			return nil, err
		}
		_, manifest, err := mods.ExtractManifestFromArchive(archive)
		if err != nil {
			return nil, err
		}
		profile.Packages, err = mods.ExtractPackageIdentifiersFromR2X(manifest)
		if err != nil {
			return nil, err
		}
		profile.OverlayArchive = archive
	}
	return profile, nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func init() {
	core.RegisterModProvider("thunderstore", func() core.ModProvider { return New(nil) })
}
