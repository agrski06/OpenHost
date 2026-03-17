#!/bin/bash
set -euo pipefail

RUNNER_VERSION="{{ .RunnerVersion }}"
RUNNER_URL="{{ .RunnerURL }}"
RUNNER_BIN="${OPENHOST_RUNNER_BIN:-/usr/local/bin/openhost-runner}"
CONFIG_PATH="${OPENHOST_RUNNER_CONFIG_PATH:-/tmp/openhost-runner-config.json}"
OPENHOST_VALHEIM_LOCAL_DEBUG="${OPENHOST_VALHEIM_LOCAL_DEBUG:-false}"
OPENHOST_VALHEIM_SKIP_SERVER_START="${OPENHOST_VALHEIM_SKIP_SERVER_START:-$OPENHOST_VALHEIM_LOCAL_DEBUG}"
SERVER_ROOT="${OPENHOST_VALHEIM_SERVER_ROOT:-/home/valheim/server}"
SAVE_ROOT="${OPENHOST_VALHEIM_SAVE_ROOT:-/home/valheim/saves}"
MODPACK_ROOT="${OPENHOST_VALHEIM_MODPACK_ROOT:-/home/valheim/modpack}"

mkdir -p "$(dirname "$CONFIG_PATH")"
cat > "$CONFIG_PATH" <<RUNNER_CONFIG_EOF
{{ .RunnerConfigJSON }}
RUNNER_CONFIG_EOF

if [ -x "$RUNNER_BIN" ]; then
    echo "OpenHost: using existing runner binary at ${RUNNER_BIN}" >&2
else
    command -v curl >/dev/null 2>&1 || { apt-get update -y && apt-get install -y curl; }

    echo "OpenHost: downloading runner v${RUNNER_VERSION} from ${RUNNER_URL}" >&2
    for i in 1 2 3 4 5; do
        if curl -fsSL "$RUNNER_URL" -o "$RUNNER_BIN"; then
            chmod +x "$RUNNER_BIN"
            break
        fi
        echo "OpenHost: runner download attempt $i failed, retrying in ${i}0s..." >&2
        sleep "${i}0"
    done
fi

    if [ ! -x "$RUNNER_BIN" ]; then
      echo "OpenHost: runner binary is not executable at ${RUNNER_BIN}" >&2
      exit 1
    fi

args=(--config "$CONFIG_PATH")
if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" = "true" ]; then
    args+=(--local)
fi
if [ "$OPENHOST_VALHEIM_SKIP_SERVER_START" = "true" ]; then
    args+=(--skip-server-start)
fi

exec "$RUNNER_BIN" "${args[@]}"


