# OpenHost Runner

`openhost-runner` is the target-host setup binary used by the CLI bootstrap scripts.

Current scope in this repository:

- loads `runnerconfig` JSON from `--config`
- validates schema version `1`
- dispatches to registered game setup implementations
- includes first-pass Valheim setup plus Thunderstore/BepInEx support

The CLI renders a minimal cloud-init/bootstrap shell wrapper that downloads this binary, writes the JSON config, and invokes:

```bash
./openhost-runner --config /tmp/openhost-runner-config.json
```

