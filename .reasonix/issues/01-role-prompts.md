# 01 — Role-Specific Subagent Prompts

**Phase:** Intelligence | **Effort:** 1 day | **Priority:** P0

## Problem

Every subagent gets the same 3-line prompt. Verifier doesn't know to be adversarial. Explorer doesn't know to prioritize breadth.

## Design

New file `internal/agent/subagent_prompts.go` with:
- `DefaultExplorerPrompt` — breadth over depth, concrete file:line references
- `DefaultReviewerPrompt` — report everything, don't self-filter
- `DefaultVerifierPrompt` — adversarial: "Your job is to try to break it." CONFIRMED/PLAUSIBLE/REFUTED
- `DefaultPlannerPrompt` — numbered testable steps with alternatives
- `DefaultExecutorPrompt` — follow the plan, no extras
- `DeepSeekCommonNotes` — cognitive loops, token cost, stop-early

Key decision: Inject via **first user message**, not system prompt. DeepSeek R1 trained without system prompts.

## Files
- `internal/agent/subagent_prompts.go` (NEW)
- `internal/agent/task.go` — prepend role prompt to user message
- `internal/boot/boot.go` — wire `RolePrompt(role, isDeepSeek)`

## Acceptance
- [ ] Explorer returns file:line references, not summaries
- [ ] Verifier uses adversarial framing with CONFIRMED/PLAUSIBLE/REFUTED
- [ ] All prompts via user message when provider is DeepSeek
