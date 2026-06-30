# 13 — Docker Sandbox Engine

**Phase:** Remote | **Effort:** 2 days | **Priority:** P2

## Problem

Current sandbox only supports local confinement (Seatbelt/bubblewrap). No container engine.

## Design

New file `internal/sandbox/docker.go`:
1. Pull `reasonix/sandbox:latest` (Debian, includes git, bash, build tools)
2. Create container with workspace mount, API key env vars, resource limits
3. Network restricted via Docker network policy
4. Timeout via `docker run --timeout`
5. Run `reasonix run --prompt "..."` inside
6. Capture stdout/stderr
7. Cleanup on completion

## Files
- `internal/sandbox/docker.go` (NEW)
- `internal/sandbox/sandbox.go` — add Docker engine to Spec

## Acceptance
- [ ] Container starts with correct resource limits
- [ ] Workspace mount read-write
- [ ] API keys injected as env vars, not in container filesystem
- [ ] Network restricted to allowed_hosts
- [ ] Timeout kills + cleans up
- [ ] Works with Docker 24+ and Podman 5+
