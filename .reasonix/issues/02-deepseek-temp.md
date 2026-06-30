# 02 — DeepSeek Subagent Temperature Override

**Phase:** Intelligence | **Effort:** 1 hour | **Priority:** P0

## Problem

reasonix defaults to `temperature: 0.0`. DeepSeek recommends `0.6` (0.5-0.7). At 0.0, DeepSeek loops infinitely — #1 failure mode.

## Design

In `internal/boot/boot.go`, when constructing a subagent: if provider is DeepSeek AND configured temp is 0.0, override to 0.6. Applies only to subagents, not parent. Emit info notice on first override.

## Files
- `internal/boot/boot.go` — add `deepSeekSubagentTemperature()` helper

## Acceptance
- [ ] DeepSeek subagents at temp 0.6 when configured is 0.0
- [ ] Non-DeepSeek subagents unaffected
- [ ] Info notice emitted
- [ ] Parent turn not affected
