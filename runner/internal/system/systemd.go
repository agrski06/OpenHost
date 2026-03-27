package system

import (
	"fmt"
	"os"
	"text/template"

	"github.com/openhost/runner/internal/core"
)

const unitTemplate = `[Unit]
Description=OpenHost {{ .ServiceName }}
After=network.target

[Service]
Type=simple
User={{ .User }}
WorkingDirectory={{ .WorkingDir }}
ExecStart={{ .ExecStart }}
Restart={{ .RestartPolicy }}
{{ range $key, $value := .Environment }}Environment="{{ $key }}={{ $value }}"
{{ end }}
[Install]
WantedBy=multi-user.target
`

// CreateAndStartService writes a systemd unit file and starts the service.
func CreateAndStartService(cfg core.LaunchConfig) error {
	if cfg.RestartPolicy == "" {
		cfg.RestartPolicy = "on-failure"
	}

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.ServiceName)

	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return fmt.Errorf("parse unit template: %w", err)
	}

	f, err := os.Create(unitPath)
	if err != nil {
		return fmt.Errorf("create unit file %s: %w", unitPath, err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("render unit file: %w", err)
	}

	if err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}

	if err := run("systemctl", "enable", cfg.ServiceName); err != nil {
		return fmt.Errorf("systemctl enable %s: %w", cfg.ServiceName, err)
	}

	if err := run("systemctl", "restart", cfg.ServiceName); err != nil {
		return fmt.Errorf("systemctl restart %s: %w", cfg.ServiceName, err)
	}

	return nil
}
