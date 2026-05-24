# Wails3 selfupdate service snapshot

This package is a local snapshot of the official Wails3 `pkg/services/selfupdate`
service from the upstream `v3/feat/self-update` branch.

It is used because the latest published Go module tag available to this project,
`github.com/wailsapp/wails/v3 v3.0.0-alpha.95`, does not yet include the updater
runtime service described by the Wails3 documentation.

Local integration changes:

- Download progress is forwarded through `Config.OnProgress`.
- GitHub release asset `digest` values with `sha256:` are mapped into
  `UpdateResult.Checksum`.
