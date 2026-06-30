# 20 — Beyond-Structural Compensations

**Phase:** Quality (ships after Phase 3) | **Effort:** 14 days total | **Priority:** P1

## Problem

Structural compensations 1-10 close the quality gap. But DeepSeek's 50× cost advantage enables an entirely different class of capability — things that are ONLY economically viable when tokens cost 1/50th of Anthropic's rates. These go beyond "closing the gap" into genuinely novel territory.

## Design

### Comp 11: Ambient Code Guardian — 2 days
Always-on background subagent (Flash) that watches every file save. Proactively flags bugs, security issues, and pattern violations before the user discovers them in production. Silent on safe changes.

### Comp 12: Idle-Time Pre-Computation — 2 days
When reasonix is idle for 2 minutes, spawns explorer subagents to pre-compute understanding of the most-referenced unexplored parts of the codebase. Answers stored in local knowledge base. User gets instant answers to questions that haven't been asked yet.

### Comp 13: Dependency Watchdog — 1.5 days
Weekly cron: checks every dependency for updates. For each update: clones into git worktree, applies update, runs tests, reports "safe" or "breaks N tests." Fully automated dependency management.

### Comp 14: Self-Play Code Generation — 2 days
For feature implementation requests: spawns 3 executors with different approach hints (simplicity, clarity, performance). Judge subagent picks best and grafts ideas from runners-up.

### Comp 15: Ensemble Code Review — 1 day
On pre-commit: spawns 3 reviewers (security, correctness, performance). Synthesizer merges findings. Catches what single reviewers miss.

### Comp 16: Cross-Session Knowledge Base — 3 days
Everything reasonix learns about the codebase is embedded and stored. Repeat questions answered instantly. Semantic search across all prior analyses. Grows more valuable over time.

### Comp 17: Bug Pattern Recognition — 1.5 days
Extracts patterns from every bug fix. Flags new code that matches known bug patterns BEFORE commit. Pattern library grows organically.

### Comp 18: Impact Analysis Pre-Computation — 1.5 days
On function signature change: automatically analyzes every call site. Reports what breaks, where, and how to fix it — before the user asks.

## Files
- `internal/guardian/ambient.go` (NEW) — file watcher + Guardian subagent
- `internal/idle/precompute.go` (NEW) — idle detection + explorer spawning
- `internal/watchdog/dependency.go` (NEW) — dependency update checker
- `internal/agent/selfplay.go` (NEW) — ensemble code generation
- `internal/agent/ensemble_review.go` (NEW) — ensemble code review
- `internal/knowledge/store.go` (NEW) — embeddings + semantic search
- `internal/knowledge/query.go` (NEW) — knowledge retrieval
- `internal/pattern/recognize.go` (NEW) — bug pattern extraction + matching
- `internal/impact/analyze.go` (NEW) — call site analysis
- `internal/config/config.go` — add `[compensations]` section

## Acceptance (Per Compensation)
- [ ] Works with mock provider for deterministic testing
- [ ] Config toggle per compensation (user can disable any)
- [ ] Cost tracking shows per-compensation daily/monthly spend
- [ ] Degrades gracefully when disabled
- [ ] No perceptible latency impact on user interactions
