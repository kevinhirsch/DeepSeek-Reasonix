# 12 — Work Queue Endpoints in reasonix serve

**Phase:** Remote | **Effort:** 1.5 days | **Priority:** P2

## Problem

`reasonix serve` has no work queue endpoints for remote workers to poll.

## Design

Add to `internal/serve/`:
```
GET  /v1/work/pending       — long-poll for pending work
POST /v1/work/{id}/claim    — atomic claim (first wins)
POST /v1/work/{id}/events   — SSE relay from worker
POST /v1/work/{id}/complete — mark complete with result
GET  /v1/work/stats         — queue depth, active workers
```

Auth: same token/password as existing serve endpoints. Work items created when subagent has `target: "remote-name"`.

## Files
- `internal/serve/workqueue.go` (NEW)
- `internal/serve/serve.go` — register routes

## Acceptance
- [ ] Workers long-poll for pending work
- [ ] Atomic claim (no double-assignment)
- [ ] Event relay preserves ordering
- [ ] Stats show accurate queue depth and active workers
- [ ] Unclaimed work expires after configurable timeout
