// Package system provides OS-level primitives for the runner: apt packages,
// user management, firewall, systemd, and SteamCMD.
package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// InstallPackages runs apt-get update and installs the given packages.
func InstallPackages(packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	if err := run("apt-get", "update", "-y"); err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}

	args := append([]string{"install", "-y"}, packages...)
	if err := run("apt-get", args...); err != nil {
		return fmt.Errorf("apt-get install %s: %w", strings.Join(packages, " "), err)
	}

	return nil
}

// run is a small helper that executes a command and returns a wrapped error
// containing stderr on failure.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, string(output))
	}
	return nil
}
