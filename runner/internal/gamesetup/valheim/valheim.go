package valheim

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openhost/runner/internal/core"
	"github.com/openhost/runner/internal/install"
	"github.com/openhost/runnerconfig"
)

// Valheim implements GameSetup for the Valheim dedicated server.
type Valheim struct{}

func (v *Valheim) Name() string       { return "valheim" }
func (v *Valheim) SystemUser() string { return "valheim" }

func (v *Valheim) ModFramework() string { return "bepinex" }

func (v *Valheim) RequiredPackages() []string {
	return []string{"libpulse0", "libatomic1", "lib32gcc-s1", "libpulse-dev", "libc6"}
}

func (v *Valheim) InstallMethod() core.InstallMethod {
	return &install.SteamCMDInstall{AppID: "896660", Anonymous: true}
}

func (v *Valheim) ServerPaths() runnerconfig.ServerPaths {
	return runnerconfig.ServerPaths{
		ServerRoot:  "/home/valheim/server",
		SaveRoot:    "/home/valheim/saves",
		ModpackRoot: "/home/valheim/modpack",
	}
}

func (v *Valheim) BuildLaunchCommand(cfg runnerconfig.GameConfig) core.LaunchConfig {
	world := "Dedicated"
	password := ""
	if w, ok := cfg.Settings["world"].(string); ok && w != "" {
		world = w
	}
	if p, ok := cfg.Settings["password"].(string); ok {
		password = p
	}

	paths := v.ServerPaths()
	hasMods := cfg.Mods != nil && len(cfg.Mods.Sources) > 0

	// Write the startup script to ServerRoot.
	scriptPath := filepath.Join(paths.ServerRoot, "start_valheim_custom.sh")
	script := buildStartScript(paths, world, password, hasMods)
	_ = os.WriteFile(scriptPath, []byte(script), 0755)

	env := map[string]string{
		"SteamAppId": "892970",
	}

	return core.LaunchConfig{
		ServiceName:   "openhost-valheim",
		User:          v.SystemUser(),
		WorkingDir:    paths.ServerRoot,
		ExecStart:     fmt.Sprintf("/bin/bash -lc '%s'", scriptPath),
		Environment:   env,
		RestartPolicy: "always",
	}
}

func buildStartScript(paths runnerconfig.ServerPaths, world, password string, hasMods bool) string {
	script := `#!/bin/bash
export SteamAppId=892970

echo "Starting server PRESS CTRL-C to exit"

`
	if hasMods {
		script += `if [ ! -f "./BepInEx/core/BepInEx.Preloader.dll" ]; then
    echo "OpenHost: expected BepInEx preloader at ./BepInEx/core/BepInEx.Preloader.dll" >&2
    exit 1
fi

if [ ! -d "./doorstop_libs" ]; then
    echo "OpenHost: expected doorstop runtime directory at ./doorstop_libs" >&2
    exit 1
fi

export DOORSTOP_ENABLED=1
export DOORSTOP_TARGET_ASSEMBLY=./BepInEx/core/BepInEx.Preloader.dll
export LD_LIBRARY_PATH="./doorstop_libs:$LD_LIBRARY_PATH"
export LD_PRELOAD="libdoorstop_x64.so:$LD_PRELOAD"

echo "OpenHost: launching Valheim server with injected BepInEx environment" >&2

`
	}

	script += fmt.Sprintf(`export LD_LIBRARY_PATH="./linux64:$LD_LIBRARY_PATH"

exec ./valheim_server.x86_64 \
    -batchmode \
    -nographics \
    -name "OpenHost-Valheim" \
    -port 2456 \
    -world "%s" \
    -password "%s" \
    -savedir "%s" \
    -public 1
`, world, password, paths.SaveRoot)

	return script
}

func init() {
	core.RegisterGameSetup("valheim", func() core.GameSetup { return &Valheim{} })
}
