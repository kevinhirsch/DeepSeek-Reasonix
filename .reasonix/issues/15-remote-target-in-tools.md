# 15 — Remote Target in task + workflow Tools

**Phase:** Remote | **Effort:** 1 day | **Priority:** P2

## Problem

No way to specify subagent should run on remote worker instead of locally.

## Design

Add `target` field to `task`, `read_only_task`, `parallel_tasks`, and `workflow` tool schemas:
```json
{"prompt": "Run tests", "target": "build-server", "run_in_background": true}
```

When `target` set to configured remote: spec serialized → work queue → worker claims → executes → streams back. Parent sees same event stream as local. Background panel shows 🌐.

When `target` empty or "local": unchanged behavior.

## Files
- `internal/agent/task.go` — add target, route to remote
- `internal/agent/workflow.go` — add target per stage
- `internal/agent/parallel_tasks.go` — add target per task
- `internal/boot/boot.go` — wire remote client

## Acceptance
- [ ] `target: "build-server"` routes to named remote
- [ ] Remote subagent events stream back identically to local
- [ ] Background panel shows 🌐 for remote
- [ ] Remote offline → clear error, not silent failure
- [ ] `target` omitted or "local" → unchanged
