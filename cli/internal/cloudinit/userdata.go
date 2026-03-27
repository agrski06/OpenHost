// Package cloudinit generates cloud-init user-data scripts that bootstrap
// the OpenHost runner on a freshly provisioned VPS.
package cloudinit

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/openhost/runnerconfig"
)

//go:embed bootstrap.sh.tmpl
var bootstrapTemplate string

// BuildUserData serializes the RunnerConfig to JSON and embeds it into a
// cloud-init bootstrap script that downloads the runner binary and executes it.
func BuildUserData(cfg *runnerconfig.RunnerConfig, runnerVersion string) (string, error) {
	configJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal runner config: %w", err)
	}

	tmpl, err := template.New("bootstrap").Parse(bootstrapTemplate)
	if err != nil {
		return "", fmt.Errorf("parse bootstrap template: %w", err)
	}

	data := struct {
		ConfigJSON    string
		RunnerVersion string
	}{
		ConfigJSON:    string(configJSON),
		RunnerVersion: runnerVersion,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute bootstrap template: %w", err)
	}

	return buf.String(), nil
}
