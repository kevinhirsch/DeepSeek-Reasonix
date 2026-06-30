# 22 — Stress Test Gaps Found During Architecture Audit

**Phase:** Resilience | **Effort:** 5 days total | **Priority:** P1

## What We Found

A systematic stress test of the entire architecture against compositional, timing, state, provider, resource, and user-workflow failure modes found 24 new gaps across 8 categories.

## Gaps Requiring Implementation

### Gap A: Global Subagent Concurrency Limiter (Cat 1B)
Add a global subagent spawn limiter that respects provider rate limits across ALL subagent sources (workflows, compensations, ambient, idle). With priority: user-initiated > workflow > compensations > ambient.
- **Files:** `internal/agent/limiter.go` (NEW), `internal/agent/task.go` (modify)
- **Effort:** 1 day

### Gap B: Compensation Trigger Chain Detection (Cat 1A)
Track when compensation A's output triggers compensation B. Cap chain depth at 5. Log chains for cost analysis.
- **Files:** `internal/agent/compensation_chain.go` (NEW)
- **Effort:** 0.5 day

### Gap C: Idle Detection Hysteresis (Cat 2B)
Debounce idle detection to prevent spawn/cancel churn when user briefly pauses reading. 2-second grace period after any activity before idle timer starts.
- **Files:** `internal/idle/detector.go` (NEW or modify)
- **Effort:** 0.5 day

### Gap D: Instant Prompt Rollback on Drift (Cat 2C)
On drift detection: roll back to last known good prompt version IMMEDIATELY (file swap, <1s), then run recalibration in background. User subagents never use degraded prompts during the recalibration window.
- **Files:** `internal/prompt/library.go` (modify)
- **Effort:** 0.5 day

### Gap E: Knowledge Base Entry Verification (Cat 3A)
Before storing KB entries: cross-validate key claims with a second explorer (Comp 2). Tag entries with producing model version. Flag stale entries on access. Re-verify entries from significantly different model versions.
- **Files:** `internal/knowledge/verify.go` (NEW), `internal/knowledge/store.go` (modify)
- **Effort:** 1 day

### Gap F: Confidence Profile Decay on Model Bump (Cat 3B)
On model version change: halve profile weight until 10 new samples accumulate. Flag pre-update profiles in UI.
- **Files:** `internal/agent/confidence.go` (modify)
- **Effort:** 0.5 day

### Gap G: Provider Price Change Response (Cat 4A)
Track per-compensation cost against user-configured thresholds. On provider price change: recalculate, flag exceeding compensations, offer mass-pause. On extreme spike (10×+): auto-pause all compensations with prominent notice.
- **Files:** `internal/agent/compensation_cost.go` (NEW), `internal/provider/pricing_watch.go` (NEW)
- **Effort:** 1 day

### Gap H: Transcript Retention Policy (Cat 5A)
Auto-rotate subagent transcripts: keep 30 days or 1,000 latest, whichever is larger. Preserve transcripts referenced by active workflows or recent sessions.
- **Files:** `internal/agent/retention.go` (NEW)
- **Effort:** 0.5 day

### Gap I: KB Growth Management (Cat 5B)
Size limit with intelligent eviction (least valuable by recency, access freq, confidence, uniqueness). Semantic dedup of identical analyses.
- **Files:** `internal/knowledge/eviction.go` (NEW)
- **Effort:** 0.5 day

### Gap J: Granular Undo Per Subagent (Cat 6A)
Per-subagent revert button in transcript panel. Handles overlapping changes with conflict warning.
- **Files:** `internal/checkpoint/checkpoint.go` (modify: per-subagent snapshots), `desktop/` (UI)
- **Effort:** 1 day

### Gap K: One-Click Fix from Finding (Cat 6B)
"Fix this" button on each finding spawns isolated executor in worktree with finding context pre-loaded.
- **Files:** `internal/agent/task.go` (modify: accept finding context), `desktop/` (UI)
- **Effort:** 0.5 day

### Gap L: Session Fork (Cat 6C)
Full session fork into new tab. Fork point marked in both transcripts.
- **Files:** `desktop/sessions.go` (modify), `desktop/app.go` (modify)
- **Effort:** 1 day

### Gap M: Change Audit Trail (Cat 6D)
Attribution of every file change to the subagent that made it. Searchable timeline.
- **Files:** `internal/agent/audit.go` (NEW), `desktop/` (UI)
- **Effort:** 1 day

### Gap N: Held-Out Statistical Power Warning (Cat 7A)
Advisory notice when held-out set has low power (N=1). Does not block shipping.
- **Files:** `internal/prompt/eval.go` (modify)
- **Effort:** 0.5 day

### Gap O: Degradation Fatigue Handling (Cat 7B)
After 5 degradation cycles in 4 weeks: escalate to user decision. Offer: keep recalibrating, pin model version, or accept degraded.
- **Files:** `internal/prompt/library.go` (modify)
- **Effort:** 0.5 day

### Gap P: Tool Output Cross-Reference (Cat 8A)
Post-processing pass that verifies subagent claims against actual recorded tool output, not just whether the file was read.
- **Files:** `internal/agent/postprocess.go` (modify)
- **Effort:** 0.5 day

### Gap Q: Language Switch Detection (Cat 8B)
Detect when reasoning switches languages. Non-blocking notice.
- **Files:** `internal/agent/reasoning_language.go` (modify)
- **Effort:** 0.5 day

## Acceptance
- [ ] Each gap has a corresponding Gherkin scenario in `compositional-edge-cases.feature` or `deep-stress-tests.feature`
- [ ] Global subagent limiter prevents rate limit stampedes
- [ ] Compensation trigger chains are detected, logged, and capped
- [ ] Idle detection has hysteresis — no spawn churn
- [ ] Degraded prompts roll back instantly, recalibration runs in background
- [ ] KB entries are verified before storage and tagged with model version
- [ ] Confidence profiles decay on model bump
- [ ] Provider price changes trigger compensation review
- [ ] KB and transcript storage have size limits with intelligent eviction
- [ ] Granular undo, one-click fix, session fork, and audit trail are functional
