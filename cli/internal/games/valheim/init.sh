#!/bin/bash
set -euo pipefail

OPENHOST_VALHEIM_HAS_MODS="{{ .HasMods }}"
OPENHOST_VALHEIM_THUNDERSTORE_CODE="{{ .ThunderstoreCode }}"
OPENHOST_VALHEIM_LOCAL_DEBUG="${OPENHOST_VALHEIM_LOCAL_DEBUG:-false}"
SERVER_ROOT="${OPENHOST_VALHEIM_SERVER_ROOT:-/home/valheim/server}"
SAVE_ROOT="${OPENHOST_VALHEIM_SAVE_ROOT:-/home/valheim/saves}"
MODPACK_ROOT="${OPENHOST_VALHEIM_MODPACK_ROOT:-/home/valheim/modpack}"

if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" = "true" ]; then
    echo "OpenHost: local debug mode enabled; privileged setup, ownership changes, firewall updates, and automatic server start will be skipped" >&2
    echo "OpenHost: local debug roots server='${SERVER_ROOT}' saves='${SAVE_ROOT}' modpack='${MODPACK_ROOT}'" >&2
fi

thunderstore_dependency_pattern='^[A-Za-z0-9][A-Za-z0-9_.]*-[A-Za-z0-9][A-Za-z0-9_.-]*-[0-9A-Za-z][0-9A-Za-z+._-]*$'

print_payload_preview() {
    local payload_path="$1"

    if [ ! -f "$payload_path" ]; then
        echo "OpenHost: debug payload '${payload_path}' was not found" >&2
        return
    fi

    echo "OpenHost: payload preview from ${payload_path}" >&2
    sed -n '1,40p' "$payload_path" >&2 || true
}

detect_profile_payload_format() {
    local payload_path="$1"
    local first_line

    if [ ! -f "$payload_path" ]; then
        echo "missing"
        return 1
    fi

    first_line="$(sed -n '1p' "$payload_path")"
    if [ "$first_line" = "#r2modman" ]; then
        echo "r2modman"
        return 0
    fi

    if jq empty "$payload_path" >/dev/null 2>&1; then
        echo "json"
        return 0
    fi

    echo "unknown"
    return 1
}

log_package_summary() {
    local source_label="$1"
    shift
    local packages=("$@")

    echo "OpenHost: ${source_label} produced ${#packages[@]} installable package(s)" >&2
    if [ "${#packages[@]}" -gt 0 ]; then
        printf 'OpenHost: package list: %s\n' "${packages[*]}" >&2
    fi
}

resolve_thunderstore_profile() {
    local code="$1"
    local destination="$2"
    local payload_format
    local url
    local -a urls=(
        "https://thunderstore.io/api/experimental/legacyprofile/get/valheim/${code}/"
        "https://thunderstore.io/api/experimental/legacyprofile/get/valheim/${code}"
        "https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/valheim/${code}/"
        "https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/valheim/${code}"
        "https://thunderstore.io/api/experimental/profile/get/${code}/"
        "https://thunderstore.io/api/experimental/profile/get/${code}"
        "https://thunderstore.io/c/valheim/api/experimental/profile/get/${code}/"
        "https://thunderstore.io/c/valheim/api/experimental/profile/get/${code}"
        "https://thunderstore.io/api/experimental/legacyprofile/get/${code}/"
        "https://thunderstore.io/api/experimental/legacyprofile/get/${code}"
        "https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/${code}/"
        "https://thunderstore.io/c/valheim/api/experimental/legacyprofile/get/${code}"
    )

    for url in "${urls[@]}"; do
        if curl -fsSL "$url" -o "$destination"; then
            if payload_format="$(detect_profile_payload_format "$destination")"; then
                echo "OpenHost: Thunderstore endpoint '${url}' returned ${payload_format} payload" >&2
                return 0
            fi

            echo "OpenHost: Thunderstore endpoint '${url}' returned an unsupported payload" >&2
            print_payload_preview "$destination"
        fi
    done

    echo "OpenHost: failed to resolve Thunderstore profile code '${code}' using known endpoints" >&2
    printf 'OpenHost: attempted endpoints:\n' >&2
    printf '  %s\n' "${urls[@]}" >&2
    return 1
}

extract_profile_packages() {
    local profile_json="$1"

    if ! jq empty "$profile_json" >/dev/null 2>&1; then
        echo "OpenHost: Thunderstore profile payload at '${profile_json}' is not valid JSON" >&2
        print_payload_preview "$profile_json"
        return 1
    fi

    jq -r --arg dependency_pattern "$thunderstore_dependency_pattern" '
        def package_identifier:
            select(type == "string")
            | select(test($dependency_pattern));

        (
            .mods[]?,
            .packages[]?,
            .profile.mods[]?,
            .profile.packages[]?,
            .data.mods[]?,
            .data.packages[]?,
            .. | objects | (.full_name?, .package_full_name?, .package?, .identifier?, .dependency_string?),
            .. | strings
        )
        | package_identifier
    ' "$profile_json" | awk 'NF && !seen[$0]++'
}

extract_package_identifiers_from_r2x() {
    local source_file="$1"

    awk '
        function trim(value) {
            sub(/^[[:space:]]+/, "", value)
            sub(/[[:space:]]+$/, "", value)
            gsub(/^"|"$/, "", value)
            return value
        }

        function emit() {
            if (name != "" && enabled == "true" && major != "" && minor != "" && patch != "") {
                printf "%s-%s.%s.%s\n", name, major, minor, patch
            }
        }

        /^[[:space:]]*-[[:space:]]+name:[[:space:]]*/ {
            emit()
            line = $0
            sub(/^[[:space:]]*-[[:space:]]+name:[[:space:]]*/, "", line)
            name = trim(line)
            major = ""
            minor = ""
            patch = ""
            enabled = ""
            next
        }

        /^[[:space:]]*major:[[:space:]]*/ {
            line = $0
            sub(/^[[:space:]]*major:[[:space:]]*/, "", line)
            major = trim(line)
            next
        }

        /^[[:space:]]*minor:[[:space:]]*/ {
            line = $0
            sub(/^[[:space:]]*minor:[[:space:]]*/, "", line)
            minor = trim(line)
            next
        }

        /^[[:space:]]*patch:[[:space:]]*/ {
            line = $0
            sub(/^[[:space:]]*patch:[[:space:]]*/, "", line)
            patch = trim(line)
            next
        }

        /^[[:space:]]*enabled:[[:space:]]*/ {
            line = $0
            sub(/^[[:space:]]*enabled:[[:space:]]*/, "", line)
            enabled = trim(line)
            next
        }

        END {
            emit()
        }
    ' "$source_file" | awk 'NF && !seen[$0]++'
}

extract_package_identifiers_from_file() {
    local source_file="$1"

    if jq empty "$source_file" >/dev/null 2>&1; then
        jq -r --arg dependency_pattern "$thunderstore_dependency_pattern" '
            def package_identifier:
                select(type == "string")
                | select(test($dependency_pattern));

            (
                .. | strings,
                .. | objects | (.full_name?, .package_full_name?, .package?, .identifier?, .dependency_string?)
            )
            | package_identifier
        ' "$source_file" | awk 'NF && !seen[$0]++'
        return
    fi

    if grep -Eq '^[[:space:]]*profileName:[[:space:]]|^[[:space:]]*mods:[[:space:]]' "$source_file"; then
        extract_package_identifiers_from_r2x "$source_file"
        return
    fi

    grep -Eo "$thunderstore_dependency_pattern" "$source_file" | awk 'NF && !seen[$0]++' || true
}

find_r2modman_manifest_member() {
    local archive_path="$1"

    unzip -Z1 "$archive_path" | awk '/(^|\/)export\.r2x$/ { print; exit }'
}

extract_r2modman_export_packages() {
    local archive_path="$1"
    local manifest_path="$2"
    local manifest_member

    manifest_member="$(find_r2modman_manifest_member "$archive_path")"
    if [ -z "$manifest_member" ]; then
        echo "OpenHost: r2modman archive '${archive_path}' did not contain export.r2x" >&2
        return 1
    fi

    echo "OpenHost: found r2modman manifest '${manifest_member}' in '${archive_path}'" >&2
    unzip -p "$archive_path" "$manifest_member" > "$manifest_path"
    echo "OpenHost: extracted r2modman manifest to '${manifest_path}'" >&2
    extract_package_identifiers_from_file "$manifest_path"
}

install_r2modman_export() {
    local payload_path="$1"
    local archive_path="$2"

    if [ "$(detect_profile_payload_format "$payload_path")" != "r2modman" ]; then
        echo "OpenHost: payload '${payload_path}' is not an r2modman export" >&2
        return 1
    fi

    tail -n +2 "$payload_path" | tr -d '\r\n' | base64 --decode > "$archive_path"
    echo "OpenHost: decoded r2modman payload '${payload_path}' to '${archive_path}'" >&2
    unzip -oq "$archive_path" -d "$SERVER_ROOT"
    echo "OpenHost: applied exported r2modman overlay files into '${SERVER_ROOT}'" >&2
}

resolve_package_install_root() {
    local unpack_dir="$1"
    local nested_launcher_root
    local nested_content_root
    local entry
    local -a top_directories=()
    local -a top_non_metadata_entries=()

    is_metadata_entry() {
        case "$1" in
            README|README.*|CHANGELOG|CHANGELOG.*|manifest.json|icon.png|LICENSE|LICENSE.*)
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }

    if [ -x "$unpack_dir/start_server_bepinex.sh" ] || [ -x "$unpack_dir/start_game_bepinex.sh" ]; then
        echo "$unpack_dir"
        return 0
    fi

    nested_launcher_root="$(find "$unpack_dir" -mindepth 2 -maxdepth 3 \( -name 'start_server_bepinex.sh' -o -name 'start_game_bepinex.sh' \) -printf '%h\n' | sort -u | head -n 1)"
    if [ -n "$nested_launcher_root" ]; then
        echo "$nested_launcher_root"
        return 0
    fi

    nested_content_root="$(find "$unpack_dir" -mindepth 2 -maxdepth 3 \( -type f \( -name 'start_server_bepinex.sh' -o -name 'start_game_bepinex.sh' -o -iname '*.dll' \) -o -type d \( -name 'BepInEx' -o -name 'plugins' -o -name 'patchers' -o -name 'config' -o -name 'doorstop_libs' \) \) -printf '%h\n' | sort -u | head -n 1)"
    if [ -n "$nested_content_root" ]; then
        echo "$nested_content_root"
        return 0
    fi

    while IFS= read -r entry; do
        [ -z "$entry" ] && continue
        if [ -d "$unpack_dir/$entry" ]; then
            top_directories+=("$entry")
        elif ! is_metadata_entry "$entry"; then
            top_non_metadata_entries+=("$entry")
        fi
    done < <(find "$unpack_dir" -mindepth 1 -maxdepth 1 -printf '%P\n' | sort)

    if [ "${#top_directories[@]}" -eq 1 ] && [ "${#top_non_metadata_entries[@]}" -eq 0 ]; then
        echo "$unpack_dir/${top_directories[0]}"
        return 0
    fi

    echo "$unpack_dir"
}

install_package_root_contents() {
    local install_root="$1"
    local package_name="$2"
    local entry
    local bundle_root="${SERVER_ROOT}/BepInEx/plugins/${package_name}"

    mkdir -p "${SERVER_ROOT}/BepInEx/plugins" "${SERVER_ROOT}/BepInEx/patchers" "${SERVER_ROOT}/BepInEx/config"

    if [ -d "${install_root}/BepInEx" ]; then
        echo "OpenHost: merging BepInEx tree from '${install_root}/BepInEx'" >&2
        mkdir -p "${SERVER_ROOT}/BepInEx"
        cp -a "${install_root}/BepInEx"/. "${SERVER_ROOT}/BepInEx"/
    fi

    if [ -d "${install_root}/plugins" ]; then
        echo "OpenHost: merging plugin directory from '${install_root}/plugins'" >&2
        cp -a "${install_root}/plugins"/. "${SERVER_ROOT}/BepInEx/plugins"/
    fi

    if [ -d "${install_root}/patchers" ]; then
        echo "OpenHost: merging patcher directory from '${install_root}/patchers'" >&2
        cp -a "${install_root}/patchers"/. "${SERVER_ROOT}/BepInEx/patchers"/
    fi

    if [ -d "${install_root}/config" ]; then
        echo "OpenHost: merging config directory from '${install_root}/config'" >&2
        cp -a "${install_root}/config"/. "${SERVER_ROOT}/BepInEx/config"/
    fi

    if [ -d "${install_root}/doorstop_libs" ]; then
        echo "OpenHost: merging doorstop runtime from '${install_root}/doorstop_libs'" >&2
        mkdir -p "${SERVER_ROOT}/doorstop_libs"
        cp -a "${install_root}/doorstop_libs"/. "${SERVER_ROOT}/doorstop_libs"/
    fi

    for entry in start_server_bepinex.sh start_game_bepinex.sh doorstop_config.ini winhttp.dll; do
        if [ -f "${install_root}/${entry}" ]; then
            echo "OpenHost: copying runtime file '${entry}' to server root" >&2
            cp -f "${install_root}/${entry}" "${SERVER_ROOT}/${entry}"
        fi
    done

    while IFS= read -r entry; do
        [ -z "$entry" ] && continue
        case "$entry" in
            BepInEx|plugins|patchers|config|doorstop_libs|start_server_bepinex.sh|start_game_bepinex.sh|doorstop_config.ini|winhttp.dll|README|README.*|CHANGELOG|CHANGELOG.*|manifest.json|icon.png|LICENSE|LICENSE.*)
                continue
                ;;
        esac

        mkdir -p "$bundle_root"
        echo "OpenHost: copying package bundle entry '${entry}' into '${bundle_root}'" >&2
        cp -a "${install_root}/${entry}" "$bundle_root/"
    done < <(find "$install_root" -mindepth 1 -maxdepth 1 -printf '%P\n' | sort)
}

download_package_zip() {
    local namespace="$1"
    local package_name="$2"
    local version="$3"
    local destination="$4"
    local url

    for url in \
        "https://thunderstore.io/package/download/${namespace}/${package_name}/${version}/" \
        "https://thunderstore.io/c/valheim/package/download/${namespace}/${package_name}/${version}/"; do
        if curl -fsSL "$url" -o "$destination"; then
            return 0
        fi
    done

    echo "OpenHost: failed to download Thunderstore package ${namespace}-${package_name}-${version}" >&2
    return 1
}

install_package_from_identifier() {
    local identifier="$1"
    local version="${identifier##*-}"
    local without_version="${identifier%-*}"
    local namespace="${without_version%%-*}"
    local package_name="${without_version#*-}"
    local archive_path="${MODPACK_ROOT}/${identifier}.zip"
    local unpack_dir="${MODPACK_ROOT}/unpack/${identifier}"
    local install_root

    if [ -z "$version" ] || [ "$without_version" = "$identifier" ] || [ -z "$namespace" ] || [ -z "$package_name" ] || [ "$package_name" = "$without_version" ]; then
        echo "OpenHost: unsupported Thunderstore package identifier '${identifier}'" >&2
        return 1
    fi

    echo "OpenHost: installing Thunderstore package ${namespace}/${package_name}@${version}" >&2
    download_package_zip "$namespace" "$package_name" "$version" "$archive_path"
    mkdir -p "$unpack_dir"
    unzip -oq "$archive_path" -d "$unpack_dir"
    install_root="$(resolve_package_install_root "$unpack_dir")"
    if [ "$install_root" != "$unpack_dir" ]; then
        echo "OpenHost: using detected nested package root '${install_root}' for ${identifier}" >&2
    else
        echo "OpenHost: using package root '${install_root}' for ${identifier}" >&2
    fi
    install_package_root_contents "$install_root" "$package_name"
}

ensure_server_root_launcher() {
    local launcher_name="$1"
    local source_path

    if [ -f "${SERVER_ROOT}/${launcher_name}" ]; then
        chmod +x "${SERVER_ROOT}/${launcher_name}" || true
        echo "OpenHost: launcher '${launcher_name}' is present at server root" >&2
        return 0
    fi

    source_path="$(find "$SERVER_ROOT" -mindepth 2 -maxdepth 4 -type f -name "$launcher_name" | sort | head -n 1)"
    if [ -n "$source_path" ]; then
        cp -f "$source_path" "${SERVER_ROOT}/${launcher_name}"
        chmod +x "${SERVER_ROOT}/${launcher_name}" || true
        echo "OpenHost: promoted launcher '${launcher_name}' from '${source_path}' to server root" >&2
        return 0
    fi

    echo "OpenHost: launcher '${launcher_name}' was not found under '${SERVER_ROOT}'" >&2
    return 1
}

log_server_launcher_status() {
    local launcher_name="$1"

    if [ -x "${SERVER_ROOT}/${launcher_name}" ]; then
        echo "OpenHost: launcher '${launcher_name}' is executable at '${SERVER_ROOT}/${launcher_name}'" >&2
    elif [ -f "${SERVER_ROOT}/${launcher_name}" ]; then
        echo "OpenHost: launcher '${launcher_name}' exists at '${SERVER_ROOT}/${launcher_name}' but is not executable" >&2
    else
        echo "OpenHost: launcher '${launcher_name}' is absent from '${SERVER_ROOT}'" >&2
    fi
}

log_bepinex_plugin_status() {
    local plugin_root="${SERVER_ROOT}/BepInEx/plugins"
    local -a plugin_dlls=()

    if [ ! -d "$plugin_root" ]; then
        echo "OpenHost: BepInEx plugin directory '${plugin_root}' is absent" >&2
        return 0
    fi

    mapfile -t plugin_dlls < <(find "$plugin_root" -type f -iname '*.dll' | sort)
    echo "OpenHost: detected ${#plugin_dlls[@]} plugin DLL(s) under '${plugin_root}'" >&2
    if [ "${#plugin_dlls[@]}" -gt 0 ]; then
        printf 'OpenHost: plugin DLLs: %s\n' "${plugin_dlls[*]}" >&2
    fi
}

log_bepinex_runtime_status() {
    if [ -f "${SERVER_ROOT}/BepInEx/core/BepInEx.Preloader.dll" ]; then
        echo "OpenHost: BepInEx preloader found at '${SERVER_ROOT}/BepInEx/core/BepInEx.Preloader.dll'" >&2
    else
        echo "OpenHost: BepInEx preloader missing from '${SERVER_ROOT}/BepInEx/core/BepInEx.Preloader.dll'" >&2
    fi

    if [ -d "${SERVER_ROOT}/doorstop_libs" ]; then
        echo "OpenHost: doorstop runtime found at '${SERVER_ROOT}/doorstop_libs'" >&2
    else
        echo "OpenHost: doorstop runtime missing from '${SERVER_ROOT}/doorstop_libs'" >&2
    fi
}

mkdir -p "$SERVER_ROOT" "$SAVE_ROOT" "$MODPACK_ROOT/unpack"

if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" != "true" ]; then
    # 1. System Requirements & 32-bit Architecture
    dpkg --add-architecture i386
    add-apt-repository multiverse -y
    add-apt-repository universe -y

    # 2. Pre-seed Steam License
    echo steam steam/question select I AGREE | debconf-set-selections
    echo steam steam/license note '' | debconf-set-selections

    # 3. Update and Install Dependencies
    apt-get update -y
    apt-get install -y steamcmd screen libpulse0 libatomic1 lib32gcc-s1 curl libpulse-dev libc6 jq unzip

    # 4. Create User and dedicated Save folder
    useradd -m -s /bin/bash valheim
    chown -R valheim:valheim /home/valheim

    # 5. Fix SteamCMD "Missing Configuration" & Install Valheim
    # We run it twice: once to update SteamCMD itself, then to download the game
    sudo -u valheim /usr/games/steamcmd +login anonymous +quit
    sudo -u valheim /usr/games/steamcmd \
        +force_install_dir "$SERVER_ROOT" \
        +login anonymous \
        +app_update {{ .AppID }} validate \
        +quit
else
    echo "OpenHost: local debug mode skipping system package setup, user creation, and Steam installation" >&2
fi

# 6. Optional Thunderstore/r2modman modpack import
if [ "$OPENHOST_VALHEIM_HAS_MODS" = "true" ]; then
    PROFILE_PAYLOAD="${MODPACK_ROOT}/profile.data"
    PACKAGE_LIST="${MODPACK_ROOT}/packages.txt"
    resolve_thunderstore_profile "$OPENHOST_VALHEIM_THUNDERSTORE_CODE" "$PROFILE_PAYLOAD"

    PAYLOAD_FORMAT="$(detect_profile_payload_format "$PROFILE_PAYLOAD")"
    echo "OpenHost: resolved Thunderstore payload format '${PAYLOAD_FORMAT}' for code '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}'" >&2
    if [ "$PAYLOAD_FORMAT" = "r2modman" ]; then
        PROFILE_ARCHIVE="${MODPACK_ROOT}/profile.r2modman.zip"
        PROFILE_EXPORT_MANIFEST="${MODPACK_ROOT}/export.r2x"
        install_r2modman_export "$PROFILE_PAYLOAD" "$PROFILE_ARCHIVE"

        if ! extract_r2modman_export_packages "$PROFILE_ARCHIVE" "$PROFILE_EXPORT_MANIFEST" > "$PACKAGE_LIST"; then
            echo "OpenHost: failed to read package metadata from r2modman export for profile '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}'" >&2
            exit 1
        fi

        mapfile -t THUNDERSTORE_PACKAGES < "$PACKAGE_LIST"
        if [ "${#THUNDERSTORE_PACKAGES[@]}" -eq 0 ]; then
            echo "OpenHost: r2modman export for profile '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}' did not contain any installable packages" >&2
            exit 1
        fi
        log_package_summary "r2modman export '${PROFILE_EXPORT_MANIFEST}'" "${THUNDERSTORE_PACKAGES[@]}"

        printf '{\n  "thunderstore_code": "%s",\n  "format": "r2modman",\n  "source": "legacyprofile_export",\n  "packages": [\n' "$OPENHOST_VALHEIM_THUNDERSTORE_CODE" > "${SERVER_ROOT}/openhost-modpack.json"
        for i in "${!THUNDERSTORE_PACKAGES[@]}"; do
            package_identifier="${THUNDERSTORE_PACKAGES[$i]}"
            install_package_from_identifier "$package_identifier"

            if [ "$i" -gt 0 ]; then
                printf ',\n' >> "${SERVER_ROOT}/openhost-modpack.json"
            fi
            printf '    "%s"' "$package_identifier" >> "${SERVER_ROOT}/openhost-modpack.json"
        done
        printf '\n  ]\n}\n' >> "${SERVER_ROOT}/openhost-modpack.json"
    elif [ "$PAYLOAD_FORMAT" = "json" ]; then
        if ! extract_profile_packages "$PROFILE_PAYLOAD" > "$PACKAGE_LIST"; then
            echo "OpenHost: failed to extract installable packages from Thunderstore profile '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}'" >&2
            exit 1
        fi

        mapfile -t THUNDERSTORE_PACKAGES < "$PACKAGE_LIST"
        if [ "${#THUNDERSTORE_PACKAGES[@]}" -eq 0 ]; then
            echo "OpenHost: Thunderstore profile '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}' did not contain any installable packages" >&2
            print_payload_preview "$PROFILE_PAYLOAD"
            exit 1
        fi
        log_package_summary "json profile '${PROFILE_PAYLOAD}'" "${THUNDERSTORE_PACKAGES[@]}"

        printf '{\n  "thunderstore_code": "%s",\n  "format": "json",\n  "packages": [\n' "$OPENHOST_VALHEIM_THUNDERSTORE_CODE" > "${SERVER_ROOT}/openhost-modpack.json"
        for i in "${!THUNDERSTORE_PACKAGES[@]}"; do
            package_identifier="${THUNDERSTORE_PACKAGES[$i]}"
            install_package_from_identifier "$package_identifier"

            if [ "$i" -gt 0 ]; then
                printf ',\n' >> "${SERVER_ROOT}/openhost-modpack.json"
            fi
            printf '    "%s"' "$package_identifier" >> "${SERVER_ROOT}/openhost-modpack.json"
        done
        printf '\n  ]\n}\n' >> "${SERVER_ROOT}/openhost-modpack.json"
    else
        echo "OpenHost: unsupported Thunderstore payload format for code '${OPENHOST_VALHEIM_THUNDERSTORE_CODE}'" >&2
        print_payload_preview "$PROFILE_PAYLOAD"
        exit 1
    fi

    ensure_server_root_launcher "start_server_bepinex.sh" || true
    ensure_server_root_launcher "start_game_bepinex.sh" || true
    log_server_launcher_status "start_server_bepinex.sh"
    log_server_launcher_status "start_game_bepinex.sh"
    log_bepinex_runtime_status
    log_bepinex_plugin_status

    if [ ! -f "${SERVER_ROOT}/BepInEx/core/BepInEx.Preloader.dll" ] || [ ! -d "${SERVER_ROOT}/doorstop_libs" ]; then
        echo "OpenHost: Thunderstore profile installed but the BepInEx runtime is incomplete. Ensure the shared profile includes denikson-BepInExPack_Valheim." >&2
        exit 1
    fi
fi

# 7. Create the Dynamic Startup Script (Matching official logic)
cat << 'EOF' > "$SERVER_ROOT/start_valheim_custom.sh"
#!/bin/bash
SAVE_ROOT="/home/valheim/saves"
export SteamAppId=892970

echo "Starting server PRESS CTRL-C to exit"

{{ if .HasMods }}
if [ ! -f "./BepInEx/core/BepInEx.Preloader.dll" ]; then
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

{{ end }}

export LD_LIBRARY_PATH="./linux64:$LD_LIBRARY_PATH"

exec ./valheim_server.x86_64 \
    -batchmode \
    -nographics \
    -name "{{ .ServerName }}" \
    -port {{ .Port }} \
    -world "{{ .WorldName }}" \
    -password "{{ .Password }}" \
    -savedir "${SAVE_ROOT}" \
    -public 1 \
EOF

# 8. Finalize Permissions
chmod +x "$SERVER_ROOT/start_valheim_custom.sh"
if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" != "true" ]; then
    chown -R valheim:valheim /home/valheim
fi

# 9. Firewall - Open Range for Query/Join ports
if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" = "true" ]; then
    echo "OpenHost: local debug mode skipping firewall changes" >&2
elif command -v ufw > /dev/null; then
    ufw allow {{ .Port }}:{{ .PortEnd }}/udp
    ufw reload
fi

# 10. Start in Screen
if [ "$OPENHOST_VALHEIM_LOCAL_DEBUG" = "true" ]; then
    echo "OpenHost: local debug mode skipping automatic server start" >&2
else
    LOG_DIR="${SERVER_ROOT}/logs"
    SCREEN_LOG_FILE="${LOG_DIR}/screen-valheim-server.log"
    SERVER_OUT_LOG_FILE="${LOG_DIR}/valheim.out.log"
    SERVER_ERR_LOG_FILE="${LOG_DIR}/valheim.err.log"

    mkdir -p "$LOG_DIR"
    chown -R valheim:valheim "$LOG_DIR" || true

    echo "OpenHost: Valheim logs: screen='${SCREEN_LOG_FILE}' stdout='${SERVER_OUT_LOG_FILE}' stderr='${SERVER_ERR_LOG_FILE}'" >&2

    sudo -u valheim screen \
        -L \
        -Logfile "${SCREEN_LOG_FILE}" \
        -dmS valheim-server \
        bash -lc "cd '${SERVER_ROOT}' && exec ./start_valheim_custom.sh >>'${SERVER_OUT_LOG_FILE}' 2>>'${SERVER_ERR_LOG_FILE}'"
fi
