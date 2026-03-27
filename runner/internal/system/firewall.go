package system

import (
	"fmt"
	"os/exec"
)

// FirewallRule describes a single UFW allow rule.
type FirewallRule struct {
	Protocol string // "tcp", "udp"
	FromPort int
	ToPort   int
}

// ConfigureUFW applies firewall rules via ufw. No-op if ufw is not installed.
func ConfigureUFW(rules []FirewallRule) error {
	if _, err := exec.LookPath("ufw"); err != nil {
		// ufw not installed — skip silently.
		return nil
	}

	for _, rule := range rules {
		portSpec := fmt.Sprintf("%d:%d/%s", rule.FromPort, rule.ToPort, rule.Protocol)
		if rule.FromPort == rule.ToPort {
			portSpec = fmt.Sprintf("%d/%s", rule.FromPort, rule.Protocol)
		}
		if err := run("ufw", "allow", portSpec); err != nil {
			return fmt.Errorf("ufw allow %s: %w", portSpec, err)
		}
	}

	if err := run("ufw", "--force", "enable"); err != nil {
		return fmt.Errorf("ufw enable: %w", err)
	}

	if err := run("ufw", "reload"); err != nil {
		return fmt.Errorf("ufw reload: %w", err)
	}

	return nil
}
