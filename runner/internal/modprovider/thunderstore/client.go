package thunderstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Option configures the Thunderstore client.
type Option func(*Client)

// Client fetches Thunderstore profile data.
type Client struct {
	baseURLs   []string
	httpClient *http.Client
}

// DefaultBaseURLs is the multi-endpoint fallback list ported from the bash script.
var DefaultBaseURLs = []string{
	"https://thunderstore.io/api/experimental/legacyprofile/get/valheim/%s/",
	"https://thunderstore.io/api/experimental/legacyprofile/get/valheim/%s",
	"https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/valheim/%s/",
	"https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/valheim/%s",
	"https://thunderstore.io/api/experimental/profile/get/%s/",
	"https://thunderstore.io/api/experimental/profile/get/%s",
	"https://thunderstore.io/c/valheim/api/experimental/profile/get/%s/",
	"https://thunderstore.io/c/valheim/api/experimental/profile/get/%s",
	"https://thunderstore.io/api/experimental/legacyprofile/get/%s/",
	"https://thunderstore.io/api/experimental/legacyprofile/get/%s",
	"https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/%s/",
	"https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/%s",
}

// NewClient creates a Thunderstore client with optional configuration.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURLs:   DefaultBaseURLs,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithBaseURLs overrides the endpoint list.
func WithBaseURLs(urls []string) Option {
	return func(c *Client) {
		c.baseURLs = urls
	}
}

// ResolveProfile fetches a Thunderstore profile by code, trying all endpoints.
func (c *Client) ResolveProfile(ctx context.Context, code string) (*Profile, error) {
	var lastErr error

	for _, urlTemplate := range c.baseURLs {
		url := fmt.Sprintf(urlTemplate, code)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			continue
		}

		format := DetectFormat(body)
		packages, err := ExtractPackages(body, format)
		if err != nil {
			lastErr = fmt.Errorf("extract packages from %s: %w", url, err)
			continue
		}

		return &Profile{
			Format:   format,
			Packages: packages,
			RawData:  body,
		}, nil
	}

	return nil, fmt.Errorf("failed to resolve Thunderstore profile %q: %w", code, lastErr)
}
