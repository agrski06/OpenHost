package system

import "fmt"

// EnsureSteamCMD installs SteamCMD via apt if not already present.
func EnsureSteamCMD() error {
	// Add i386 architecture for SteamCMD dependencies.
	if err := run("dpkg", "--add-architecture", "i386"); err != nil {
		return fmt.Errorf("dpkg add i386: %w", err)
	}

	if err := run("apt-get", "update", "-y"); err != nil {
		return fmt.Errorf("apt-get update for steamcmd: %w", err)
	}

	// Pre-accept the Steam license to avoid interactive prompts.
	if err := run("bash", "-c",
		`echo "steamcmd steam/question select I AGREE" | debconf-set-selections && `+
			`echo "steamcmd steam/license note ''" | debconf-set-selections`); err != nil {
		return fmt.Errorf("pre-accept steam license: %w", err)
	}

	if err := run("apt-get", "install", "-y", "steamcmd"); err != nil {
		return fmt.Errorf("apt-get install steamcmd: %w", err)
	}

	return nil
}

// InstallApp uses SteamCMD to install or update a Steam app into serverRoot.
func InstallApp(user, appID, serverRoot string, anonymous bool) error {
	loginArgs := "anonymous"
	if !anonymous {
		return fmt.Errorf("non-anonymous SteamCMD login not yet supported")
	}

	args := []string{
		"-u", user, "--",
		"/usr/games/steamcmd",
		"+force_install_dir", serverRoot,
		"+login", loginArgs,
		"+app_update", appID, "validate",
		"+quit",
	}

	if err := run("sudo", args...); err != nil {
		return fmt.Errorf("steamcmd install app %s: %w", appID, err)
	}

	return nil
}
