# 06 — Enriched jobs.View with Live State

**Phase:** Monitoring | **Effort:** 0.5 day | **Priority:** P1

## Problem

`jobs.View` has 5 fields. No reasoning stream, tool count, model/effort info, or result preview.

## Design

Add to `Job.View`:
- `Model`, `Effort` — subagent identity
- `ToolCalls` — running count
- `LastTool` — most recent tool name
- `LastReasoning` — rolling 200-char window of reasoning
- `Usage` — accumulated token counts
- `DependsOn`, `Blocks` — dependency graph
- `ResultPreview` — first line of partial output

Thread-safe: subagent goroutine writes, TUI reads.

## Files
- `internal/jobs/jobs.go` — enrich View, add setters
- `internal/agent/task.go` — pipe tool state into job
- `internal/agent/parallel_tasks.go` — pipe subagent events

## Acceptance
- [ ] All new fields populated during subagent run
- [ ] Thread-safe reads from TUI goroutine
- [ ] `LastReasoning` shows live reasoning tail
