# 04 — SubagentMessenger + send_to_subagent

**Phase:** Communication | **Effort:** 1 day | **Priority:** P1

## Problem

Subagents are fire-and-forget. No way to send mid-task guidance or have subagents coordinate.

## Design

New file `internal/agent/subagent_message.go`:
- `SubagentMessenger` — `map[string]*Agent`, thread-safe register/unregister/send
- `SendToSubagentTool` — model-facing tool, `subagent_id` + `message` params
- Reuses existing `agent.Steer()` + `steerQueue` + `MidTurnSteerPrefix`
- `parallel_tasks` registers subagents on spawn, unregisters on completion

## Files
- `internal/agent/subagent_message.go` (NEW)
- `internal/agent/parallel_tasks.go` — register/unregister
- `internal/agent/task.go` — register background subagents
- `internal/boot/boot.go` — wire messenger

## Acceptance
- [ ] Parent sends message to running subagent by ref or job ID
- [ ] Subagent receives as steer on next tool-call round
- [ ] Thread-safe registration/deregistration
- [ ] Cleanup on subagent completion/cancellation
