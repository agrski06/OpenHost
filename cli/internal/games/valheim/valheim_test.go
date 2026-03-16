package valheim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInitCommand_VanillaValheim(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":    "DedicatedWorld",
		"password": "secret",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `OPENHOST_VALHEIM_HAS_MODS="false"`)
	assert.Contains(t, command, `OPENHOST_VALHEIM_THUNDERSTORE_CODE=""`)
	assert.Contains(t, command, `OPENHOST_VALHEIM_LOCAL_DEBUG="${OPENHOST_VALHEIM_LOCAL_DEBUG:-false}"`)
	assert.Contains(t, command, `SERVER_ROOT="${OPENHOST_VALHEIM_SERVER_ROOT:-/home/valheim/server}"`)
	assert.Contains(t, command, `./valheim_server.x86_64 \`)
	assert.NotContains(t, command, `./start_server_bepinex.sh \`)
}

func TestBuildInitCommand_ThunderstoreModpack(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":             "DedicatedWorld",
		"password":          "secret",
		"thunderstore_code": "ABC123_code",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `OPENHOST_VALHEIM_HAS_MODS="true"`)
	assert.Contains(t, command, `OPENHOST_VALHEIM_THUNDERSTORE_CODE="ABC123_code"`)
	assert.Contains(t, command, `https://thunderstore.io/api/experimental/profile/get/${code}/`)
	assert.Contains(t, command, `https://thunderstore.io/api/experimental/legacyprofile/get/valheim/${code}/`)
	assert.Contains(t, command, `resolve_thunderstore_profile "$OPENHOST_VALHEIM_THUNDERSTORE_CODE" "$PROFILE_PAYLOAD"`)
	assert.Contains(t, command, `local debug mode enabled; privileged setup, ownership changes, firewall updates, and automatic server start will be skipped`)
	assert.Contains(t, command, `local debug mode skipping system package setup, user creation, and Steam installation`)
	assert.Contains(t, command, `returned an unsupported payload`)
	assert.Contains(t, command, `resolved Thunderstore payload format '`)
	assert.Contains(t, command, `detect_profile_payload_format`)
	assert.Contains(t, command, `log_package_summary`)
	assert.Contains(t, command, `install_r2modman_export`)
	assert.Contains(t, command, `decoded r2modman payload`)
	assert.Contains(t, command, `applied exported r2modman overlay files`)
	assert.Contains(t, command, `resolve_package_install_root`)
	assert.Contains(t, command, `start_server_bepinex.sh`)
	assert.Contains(t, command, `doorstop_libs`)
	assert.Contains(t, command, `README|README.*|CHANGELOG|CHANGELOG.*|manifest.json|icon.png|LICENSE|LICENSE.*`)
	assert.Contains(t, command, `plugins`)
	assert.Contains(t, command, `-iname '*.dll'`)
	assert.Contains(t, command, `using detected nested package root`)
	assert.Contains(t, command, `install_package_root_contents`)
	assert.Contains(t, command, `BepInEx/plugins/${package_name}`)
	assert.Contains(t, command, `merging BepInEx tree`)
	assert.Contains(t, command, `merging plugin directory`)
	assert.Contains(t, command, `copying package bundle entry`)
	assert.Contains(t, command, `ensure_server_root_launcher`)
	assert.Contains(t, command, `promoted launcher`)
	assert.Contains(t, command, `log_bepinex_runtime_status`)
	assert.Contains(t, command, `BepInEx preloader found`)
	assert.Contains(t, command, `doorstop runtime found`)
	assert.Contains(t, command, `launcher 'start_server_bepinex.sh' is executable`)
	assert.Contains(t, command, `launcher 'start_game_bepinex.sh'`)
	assert.Contains(t, command, `log_bepinex_plugin_status`)
	assert.Contains(t, command, `detected ${#plugin_dlls[@]} plugin DLL(s) under`)
	assert.Contains(t, command, `plugin DLLs:`)
	assert.Contains(t, command, `local debug mode skipping firewall changes`)
	assert.Contains(t, command, `local debug mode skipping automatic server start`)
	assert.Contains(t, command, `extract_r2modman_export_packages`)
	assert.Contains(t, command, `found r2modman manifest`)
	assert.Contains(t, command, `extracted r2modman manifest`)
	assert.Contains(t, command, `extract_package_identifiers_from_r2x`)
	assert.Contains(t, command, `printf "%s-%s.%s.%s\n", name, major, minor, patch`)
	assert.Contains(t, command, `https://thunderstore.io/package/download/${namespace}/${package_name}/${version}/`)
	assert.Contains(t, command, `produced ${#packages[@]} installable package(s)`)
	assert.Contains(t, command, `package list:`)
	assert.Contains(t, command, `installing Thunderstore package ${namespace}/${package_name}@${version}`)
	assert.Contains(t, command, `export.r2x`)
	assert.Contains(t, command, `"format": "r2modman"`)
	assert.Contains(t, command, `"source": "legacyprofile_export"`)
	assert.Contains(t, command, `PACKAGE_LIST="${MODPACK_ROOT}/packages.txt"`)
	assert.Contains(t, command, `export DOORSTOP_ENABLED=1`)
	assert.Contains(t, command, `export DOORSTOP_TARGET_ASSEMBLY=./BepInEx/core/BepInEx.Preloader.dll`)
	assert.Contains(t, command, `export LD_PRELOAD="libdoorstop_x64.so:$LD_PRELOAD"`)
	assert.Contains(t, command, `launching Valheim server with injected BepInEx environment`)
	assert.Contains(t, command, `./valheim_server.x86_64 \`)
}

func TestBuildInitCommand_UUIDThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"world":             "DedicatedWorld",
		"password":          "secret",
		"thunderstore_code": "019cf113-4729-c139-63ac-ea85dafcffd6",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `OPENHOST_VALHEIM_THUNDERSTORE_CODE="019cf113-4729-c139-63ac-ea85dafcffd6"`)
	assert.Contains(t, command, `api/experimental/legacyprofile/get/valheim/${code}`)
}

func TestBuildInitCommand_InvalidThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "bad code with spaces",
	})
	assert.Error(t, err)
	assert.Empty(t, command)
	assert.Contains(t, err.Error(), "thunderstore_code")
}

func TestBuildInitCommand_BlankThunderstoreCode(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "   ",
	})
	assert.Error(t, err)
	assert.Empty(t, command)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestBuildInitCommand_DefaultWorldWithMods(t *testing.T) {
	game := &Valheim{}

	command, err := game.BuildInitCommand(map[string]any{
		"password":          "secret",
		"thunderstore_code": "Code123",
	})
	require.NoError(t, err)

	assert.Contains(t, command, `-world "Dedicated"`)
}
