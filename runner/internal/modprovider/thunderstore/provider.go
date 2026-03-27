package thunderstore

import (
	"context"
	"fmt"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runnerconfig"
)

// Provider implements core.ModProvider for Thunderstore.
type Provider struct {
	client *Client
}

func (p *Provider) Name() string { return "thunderstore" }

// Resolve fetches a Thunderstore profile by the source code and returns
// downloadable packages.
func (p *Provider) Resolve(ctx context.Context, source runnerconfig.ModSource) ([]core.Package, error) {
	if source.Code == "" {
		return nil, fmt.Errorf("thunderstore source requires a profile code")
	}

	profile, err := p.client.ResolveProfile(ctx, source.Code)
	if err != nil {
		return nil, fmt.Errorf("resolve thunderstore profile %q: %w", source.Code, err)
	}

	var packages []core.Package
	for _, pkg := range profile.Packages {
		downloadURL := fmt.Sprintf(
			"https://thunderstore.io/package/download/%s/%s/%s/",
			pkg.Namespace, pkg.Name, pkg.Version,
		)
		packages = append(packages, core.Package{
			Identifier: pkg.String(),
			URL:        downloadURL,
		})
	}

	return packages, nil
}

func init() {
	core.RegisterModProvider("thunderstore", func() core.ModProvider {
		return &Provider{client: NewClient()}
	})
}
