# 16 — Claude Code Behavioral Quality Adaptation

**Phase:** Quality | **Effort:** 1 day | **Priority:** P0

## Problem

reasonix produces functionally correct code but lacks the "thoughtfulness" and "carefulness" of Claude Code. The agent over-engineers, over-narrates, makes unverified claims, and ends with promises instead of completed work.

## Design

Adapt Claude Code's five quality pillars into reasonix:

### 1. System prompt quality block

Add ~45 lines of behavioral rules to `internal/boot/boot.go`. Covers: anti-overengineering, evidence-grounded claims, silence default, outcome-first communication, anti-patterns.

### 2. Three behavioral gates in `internal/agent/agent.go`

- **Promise detector**: Regex "I'll do X" at end of final answer → inject: "execute that work NOW"
- **Claim verifier**: Cross-reference file:line claims against evidence.Ledger → flag unverified
- **Completeness check**: Match action verbs in request against answer → flag missed verbs

### 3. DeepSeek-specific quality tuning

Counter-steering for DeepSeek's tendencies: over-engineering, over-narration, repetition.

## Files
- `internal/boot/boot.go` — add quality block to sysPrompt
- `internal/agent/promise_detector.go` (NEW) — promise detection
- `internal/agent/agent.go` — wire gates into finalReadinessCheck()

## Acceptance
- [ ] System prompt includes all 5 quality pillars
- [ ] Agent rejects final answer ending in "I'll..." → injects action directive
- [ ] Agent flags claims about files not read this turn
- [ ] Agent checks whether all requested actions were completed
- [ ] DeepSeek-specific notes appended for DeepSeek providers
- [ ] Quality gates add <500ms to turn completion
