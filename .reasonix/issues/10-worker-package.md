# 10 — internal/worker/ Package

**Phase:** Remote | **Effort:** 3 days | **Priority:** P2

## Problem

No worker-side logic for polling work queue, claiming items, executing in containers.

## Design

New package `internal/worker/`:
```
internal/worker/
├── worker.go     — main loop: poll→claim→execute→stream
├── config.go     — worker.toml loading
├── poller.go     — long-poll GET /v1/work/pending
├── executor.go   — spawn Docker container, run subagent
├── streamer.go   — POST /v1/work/{id}/events (SSE relay)
└── health.go     — periodic health + capability report
```

Worker loop: long-poll (30s timeout) → claim → clone repo → Docker container → run → stream events → report completion.

## Files
- `internal/worker/` (NEW package, ~6 files)

## Acceptance
- [ ] Poll→claim→execute→stream cycle works
- [ ] Container isolation: workspace mount, API key env vars, CPU/mem limits
- [ ] Container timeout enforced (default 30 min)
- [ ] Network restricted to configured allowed_hosts
- [ ] Graceful shutdown on SIGTERM (drain in-flight)
- [ ] No orphaned Docker resources
