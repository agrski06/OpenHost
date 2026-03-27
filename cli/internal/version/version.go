// Package version holds build-time version information injected via ldflags.
//
// Build with:
//
//	go build -ldflags "-X github.com/openhost/cli/internal/version.RunnerVersion=0.2.0"
package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RunnerVersion is the version of the runner binary the CLI will download
// during cloud deployments. It is set at build time via -ldflags. When set
// to "dev" (default for local builds), ResolveRunnerVersion queries the
// latest GitHub release tag at runtime.
var RunnerVersion = "dev"

const githubRepo = "agrski06/openhost"

// ResolveRunnerVersion returns RunnerVersion if it was set at build time,
// otherwise queries GitHub for the latest runner-v* release tag.
func ResolveRunnerVersion() (string, error) {
	if RunnerVersion != "dev" && RunnerVersion != "" {
		return RunnerVersion, nil
	}

	ver, err := fetchLatestRunnerVersion()
	if err != nil {
		return "", fmt.Errorf("runner version is 'dev' and failed to resolve latest from GitHub: %w", err)
	}
	return ver, nil
}

func fetchLatestRunnerVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", githubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}

	for _, r := range releases {
		if strings.HasPrefix(r.TagName, "runner-v") {
			return strings.TrimPrefix(r.TagName, "runner-v"), nil
		}
	}

	return "", fmt.Errorf("no runner-v* release found in %s", githubRepo)
}
