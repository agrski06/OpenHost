#!/bin/bash
# 1. System Setup
apt-get update -y
apt-get install -y {{ .JavaPackage }} wget screen

# 2. User & Directory Setup
useradd -m minecraft
mkdir -p /home/minecraft/server
cd /home/minecraft/server

# 3. Game Files & EULA
wget -O server.jar {{ .DownloadURL }}
echo "eula=true" > eula.txt

# 4. Permissions
chown -R minecraft:minecraft /home/minecraft/server

# 5. Launch
# -Xms is the minimum/starting memory, -Xmx is the maximum limit
sudo -u minecraft screen -dmS mc-server java -Xms{{ .MinMem }} -Xmx{{ .MaxMem }} -jar server.jar --port {{ .Port }} nogui

echo "OpenHost: Minecraft setup complete on port {{ .Port }} with {{ .MaxMem }} RAM"