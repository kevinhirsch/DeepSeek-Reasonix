# 05 — workflow Tool (Pipeline, Barrier, Loop-Until-Dry)

**Phase:** Workflows | **Effort:** 3 days | **Priority:** P1

## Problem

Model can't compose multi-stage subagent strategies dynamically. "Find bugs → verify each → synthesize" requires multiple sequential tool calls, burning context.

## Design

New file `internal/agent/workflow.go` with `WorkflowTool`. Declarative spec:

```json
{
  "strategy": "pipeline | barrier | loop_until_dry",
  "stages": [{name, role, prompt_template, items_from, fan_out, voting, model, effort}],
  "items": ["input strings"],
  "dry_rounds": 2
}
```

Strategies:
- **pipeline** — items flow independently through stages (default)
- **barrier** — all finish stage N before N+1 starts
- **loop_until_dry** — keep spawning until K rounds with nothing new

Verification modes:
- `mode: "precision"` (medium effort) — 3 finder angles, 1-vote verify
- `mode: "recall"` (high effort, max on DeepSeek) — recall-biased, treats uncertain as plausible

Reuses `TaskTool.runSubSession()` — orchestration layer, not new engine.

## Files
- `internal/agent/workflow.go` (NEW)
- `internal/boot/boot.go` — register workflow tool

## Acceptance
- [ ] Pipeline: items progress independently, no barrier
- [ ] Barrier: all finish stage N before N+1
- [ ] Loop-until-dry: stops after N dry rounds
- [ ] fan_out: N parallel per item
- [ ] voting: threshold-based confirmation
- [ ] Precision + recall verification modes
- [ ] Events nested under workflow call
- [ ] Max total tasks cap enforced
