package minecraft

import (
	"fmt"

	"github.com/openhost/cli/internal/core"
)

type Minecraft struct{}

func (g *Minecraft) Name() string {
	return "minecraft"
}

func (g *Minecraft) Port() int {
	return 25565
}

func (g *Minecraft) Protocol() string {
	return "tcp"
}

// BuildInitCommand builds the actual startup command
func (g *Minecraft) BuildInitCommand() string {
	// Variables we might want to customize later
	memMin := "1G"
	memMax := "8G"
	// Latest Vanilla 1.21 download URL (Check Minecraft's site for the newest)
	serverJarURL := "https://piston-data.mojang.com/v1/objects/450698d18625b03f09071b56da456d2f347895f3/server.jar"

	const scriptTemplate = `#!/bin/bash
# 1. System Updates & Dependencies
apt-get update -y
apt-get install -y openjdk-21-jre-headless wget screen

# 2. Create a dedicated user for security (don't run as root!)
useradd -m minecraft
mkdir -p /home/minecraft/server
cd /home/minecraft/server

# 3. Download the server JAR
wget -O server.jar %s

# 4. Accept the EULA automatically
echo "eula=true" > eula.txt

# 5. Set permissions
chown -R minecraft:minecraft /home/minecraft/server

# 6. Start the server in a detached screen session as the minecraft user
# This allows the server to keep running and lets you 'attach' to the console later
sudo -u minecraft screen -dmS mc-server java -Xms%s -Xmx%s -jar server.jar --port %d nogui
`

	return fmt.Sprintf(scriptTemplate, serverJarURL, memMin, memMax, g.Port())
}

func init() {
	core.RegisterGame("minecraft", func() core.Game { return &Minecraft{} })
}
