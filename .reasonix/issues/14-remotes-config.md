# 14 — Remotes Config + Settings Panel

**Phase:** Remote | **Effort:** 1.5 days | **Priority:** P2

## Problem

No config schema for remote workers. No settings UI.

## Design

Add `[[remotes]]` to reasonix.toml:
```toml
[[remotes]]
name = "build-server"
url = "https://build-server:9090"
auth_token = "${REASONIX_REMOTE_TOKEN}"
max_concurrent = 4
prefer_for = ["test", "build"]
```

New file `internal/config/remote.go` with `RemoteEntry` + validation. Desktop settings UI for management. CLI `/remotes` command.

## Files
- `internal/config/config.go` — add Remotes field
- `internal/config/remote.go` (NEW)
- `desktop/settings_app.go` — remotes management
- `internal/cli/remote.go` — `/remotes` slash command

## Acceptance
- [ ] Remotes declared in reasonix.toml
- [ ] Auth token from env var (never in config file)
- [ ] Remote status: online/offline, load, capabilities
- [ ] `/remotes` command lists all remotes
- [ ] Validation rejects invalid URLs, duplicate names
