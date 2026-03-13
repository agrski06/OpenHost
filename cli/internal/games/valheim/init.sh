#!/bin/bash

# 1. System Requirements & 32-bit Architecture
dpkg --add-architecture i386
add-apt-repository multiverse -y
add-apt-repository universe -y

# 2. Pre-seed Steam License
echo steam steam/question select I AGREE | debconf-set-selections
echo steam steam/license note '' | debconf-set-selections

# 3. Update and Install Dependencies
apt-get update -y
apt-get install -y steamcmd screen libpulse0 libatomic1 lib32gcc-s1 curl libpulse-dev libc6

# 4. Create User and dedicated Save folder
useradd -m -s /bin/bash valheim
mkdir -p /home/valheim/server /home/valheim/saves
chown -R valheim:valheim /home/valheim

# 5. Fix SteamCMD "Missing Configuration" & Install Valheim
# We run it twice: once to update SteamCMD itself, then to download the game
sudo -u valheim /usr/games/steamcmd +login anonymous +quit
sudo -u valheim /usr/games/steamcmd \
    +force_install_dir /home/valheim/server \
    +login anonymous \
    +app_update {{ .AppID }} validate \
    +quit

# 6. Create the Dynamic Startup Script (Matching official logic)
cat << 'EOF' > /home/valheim/server/start_valheim_custom.sh
#!/bin/bash
export templdpath=$LD_LIBRARY_PATH
export LD_LIBRARY_PATH=./linux64:$LD_LIBRARY_PATH
export SteamAppId=892970

echo "Starting server PRESS CTRL-C to exit"

./valheim_server.x86_64 \
    -name "{{ .ServerName }}" \
    -port {{ .Port }} \
    -world "{{ .WorldName }}" \
    -password "{{ .Password }}" \
    -savedir "/home/valheim/saves" \
    -public 1 \

export LD_LIBRARY_PATH=$templdpath
EOF

# 7. Finalize Permissions
chmod +x /home/valheim/server/start_valheim_custom.sh
chown -R valheim:valheim /home/valheim

# 8. Firewall - Open Range for Query/Join ports
if command -v ufw > /dev/null; then
    ufw allow {{ .Port }}:{{ .PortEnd }}/udp
    ufw reload
fi

# 9. Start in Screen
sudo -u valheim screen -dmS valheim-server bash -c "cd /home/valheim/server && ./start_valheim_custom.sh"