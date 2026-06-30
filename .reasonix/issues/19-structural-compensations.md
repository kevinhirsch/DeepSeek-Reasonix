# 19 — Structural Compensation Package

**Phase:** Quality | **Effort:** 8 days total | **Priority:** P1 (ships after Phase 1-3 foundation)

## Problem

Anthropic has model-internal quality control (adaptive thinking, server-side structured output, task budgets, stop_details). DeepSeek doesn't. We compensate with agent-loop-level mechanisms that leverage DeepSeek's 50× cost advantage to do things Anthropic can't economically match.

## Compensation 1: Speculative Execution — 1 day
### Design
Run the same subagent task N times (default 3), merge results by consensus. A finding found by 1/3 subagents is "uncertain." Found by 2+/3 is "confirmed."
### Files: `internal/agent/speculative.go` (NEW), `internal/tool/builtin/explore_speculative.go` (NEW)

## Compensation 2: Multi-Model Cross-Validation — 1 day
### Design
Run the same verification on Flash AND Pro. If verdicts match → high confidence. If they differ → surface dissent.
### Files: `internal/agent/cross_validate.go` (NEW), `internal/workflow/stage.go` (modify)

## Compensation 3: Post-Processing Passes — 1 day
### Design
After every subagent output: correctness pass (flag unsupported claims), conciseness pass (strip 20% tokens), completeness pass (check against original task).
### Files: `internal/agent/postprocess.go` (NEW)

## Compensation 4: Confidence Calibration — 1 day
### Design
Per-(role, model, language) accuracy tracking. Subagents routed to their strongest profile. Weighted voting by historical accuracy.
### Files: `internal/agent/confidence.go` (NEW), `internal/agent/confidence_test.go` (NEW)

## Compensation 5: Prompt Regression Bisection — 0.5 day
### Design
When drift detected (Phase E), auto-bisect to find the prompt change responsible. Revert only the breaking change, keep improvements.
### Files: `internal/prompt/bisect.go` (NEW)

## Compensation 6: Consensus Weighting — 0.5 day
### Design
Verifier votes weighted by that verifier's historical accuracy on the same claim type. Handles edge cases where weighted consensus differs from simple majority.
### Files: `internal/workflow/voting.go` (modify)

## Compensation 7: Self-Healing Prompt Library — 1 day
### Design
On model version bump: evaluate all prompt variants, auto-select best. No manual rewriting needed.
### Files: `internal/prompt/library.go` (NEW)

## Compensation 8: Context Janitor — 1 day
### Design
Every 5 turns: cheap Flash subagent semantically compacts context — keeps what matters, drops dead ends.
### Files: `internal/agent/janitor.go` (NEW)

## Compensation 9: Pre-Mortem Analysis — 0.5 day
### Design
Before workflow execution: cheap subagent predicts failure modes, injects preventive guidance into finder prompts.
### Files: `internal/workflow/premortem.go` (NEW), `internal/workflow/workflow.go` (modify)

## Compensation 10: Semantic Diff — 0.5 day
### Design
When speculative execution produces N outputs: semantic diff identifies agreements, disagreements, and contradictions.
### Files: `internal/agent/semantic_diff.go` (NEW)

## Acceptance (Per Compensation)
- [ ] Works with mock provider for deterministic testing
- [ ] ≤5% overhead on subagent latency (cost of extra LLM calls is the design, not the overhead)
- [ ] Degrades gracefully when turned off (compensations are additive, not load-bearing)
- [ ] Config toggle per compensation (user can disable any they don't want)
- [ ] Cost tracking shows per-compensation token spend
