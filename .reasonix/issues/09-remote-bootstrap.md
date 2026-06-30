# 09 — reasonix remote bootstrap Command

**Phase:** Remote | **Effort:** 1 day | **Priority:** P2

## Problem

Setting up a remote build machine requires manual Docker, worker, systemd steps.

## Design

New file `internal/cli/remote.go` with `bootstrap` subcommand. Generates self-contained shell script (~200 lines):
1. Detects OS
2. Installs Docker if missing
3. Probes CPU/RAM/disk
4. Auto-calculates safe limits (80% of system)
5. Downloads reasonix-worker binary
6. Writes worker.toml with auto-detected limits
7. Prompts for API keys interactively
8. Installs systemd service with security hardening
9. Starts worker

## Files
- `internal/cli/remote.go` (NEW) — bootstrap + script generator
- `internal/cli/cli.go` — register `remote` subcommand

## Acceptance
- [ ] `reasonix remote bootstrap --name my-worker` generates runnable script
- [ ] Works on Ubuntu 22.04/24.04, Debian 12, Fedora 40+
- [ ] Auto-detected limits leave 20% for system
- [ ] systemd service has NoNewPrivileges, ProtectSystem, etc.
- [ ] Idempotent
