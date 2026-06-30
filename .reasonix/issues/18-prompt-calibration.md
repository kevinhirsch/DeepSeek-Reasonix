# 18 — Prompt Calibration Framework

**Phase:** Foundation | **Effort:** 2 days | **Priority:** P0 (must ship before any subagent code)

## Problem

We can't copy Claude Code's prompts — they're Anthropic-proprietary and Claude-tuned. We need DeepSeek-adapted prompts that produce equivalent behavioral quality, but there's no framework to evaluate and calibrate them. Without this, every subagent role ships with untested prompts that might loop, over-narrate, or miss findings.

## Design

### Prompt Structure
Every DeepSeek subagent prompt follows a 7-part structure (role identity, behavioral rules, tool guidance, output format, stop conditions, DeepSeek-specific notes, counter-examples). Full structure in `PROMPT_CALIBRATION.md`.

### Evaluation Harness
New package `internal/prompt/eval.go` — takes a prompt, a role, and a scenario file, runs the prompt against DeepSeek, scores pass/fail against expectations.

### Standard Scenarios
17 eval scenarios across 5 roles in `.reasonix/prompts/eval/`. Each scenario has: task input, expected tool usage patterns, expected output patterns, forbidden patterns.

### Calibration Threshold
≥90% pass rate across all scenarios before any prompt ships. Estimated cost: $0.12 total for full calibration.

## Files
- `internal/prompt/eval.go` (NEW) — evaluation harness
- `internal/prompt/eval_test.go` (NEW) — tests for the harness itself
- `.reasonix/prompts/v1/explorer.md` (NEW) — Explorer prompt (DeepSeek-adapted)
- `.reasonix/prompts/v1/reviewer.md` (NEW) — Reviewer prompt
- `.reasonix/prompts/v1/verifier.md` (NEW) — Verifier prompt
- `.reasonix/prompts/v1/planner.md` (NEW) — Planner prompt
- `.reasonix/prompts/v1/executor.md` (NEW) — Executor prompt
- `.reasonix/prompts/v1/deepseek_notes.md` (NEW) — Shared DeepSeek adaptations
- `.reasonix/prompts/eval/explorer.json` (NEW) — Explorer eval scenarios
- `.reasonix/prompts/eval/reviewer.json` (NEW)
- `.reasonix/prompts/eval/verifier.json` (NEW)
- `.reasonix/prompts/eval/planner.json` (NEW)
- `.reasonix/prompts/eval/executor.json` (NEW)
- `internal/boot/boot.go` — modify: load prompts from `.reasonix/prompts/v1/` instead of hardcoded constants
- `internal/cli/prompt.go` (NEW) — `reasonix prompt eval` command

## Acceptance
- [ ] `reasonix prompt eval explorer` runs 4 scenarios, reports pass/fail, cost, and transcript
- [ ] All 5 roles have ≥90% pass rate on their scenarios before shipping
- [ ] Prompt files are plain markdown, user-editable
- [ ] Prompt loading falls back to built-in defaults if no custom file exists
- [ ] Eval harness works with mock providers for deterministic CI testing
- [ ] Eval results archived to `.reasonix/prompts/results/` with timestamps
- [ ] Prompt CHANGELOG tracks every iteration
