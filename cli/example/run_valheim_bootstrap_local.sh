#!/bin/bash
set -euo pipefail

BOOTSTRAP_PATH="${1:-./.openhost/debug/valheim-bootstrap.sh}"
DEBUG_ROOT="${2:-./.openhost/valheim-local}"

BOOTSTRAP_PATH="$(readlink -f "$BOOTSTRAP_PATH")"
DEBUG_ROOT="$(readlink -f "$DEBUG_ROOT")"

mkdir -p "$DEBUG_ROOT/server" "$DEBUG_ROOT/saves" "$DEBUG_ROOT/modpack"

export OPENHOST_VALHEIM_LOCAL_DEBUG=true
export OPENHOST_VALHEIM_SERVER_ROOT="$DEBUG_ROOT/server"
export OPENHOST_VALHEIM_SAVE_ROOT="$DEBUG_ROOT/saves"
export OPENHOST_VALHEIM_MODPACK_ROOT="$DEBUG_ROOT/modpack"

printf 'OpenHost local Valheim bootstrap debug\n' >&2
printf '  bootstrap: %s\n' "$BOOTSTRAP_PATH" >&2
printf '  server:    %s\n' "$OPENHOST_VALHEIM_SERVER_ROOT" >&2
printf '  saves:     %s\n' "$OPENHOST_VALHEIM_SAVE_ROOT" >&2
printf '  modpack:   %s\n' "$OPENHOST_VALHEIM_MODPACK_ROOT" >&2

exec bash "$BOOTSTRAP_PATH"

