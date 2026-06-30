# Compensation UI Wiring Audit

> Every compensation → does it need UI? → where? → covered?

## Comp 1: Speculative Execution
- **Needs UI:** Yes — user must see when speculation is active, what consensus was reached
- **Where:** Background panel (subagent count indicator "🔍×3"), transcript (merged result card showing confirmed/likely/possible tiers)
- **Covered:** `structural-compensation.feature` @comp-speculative-basic, @comp-fe-panel

## Comp 2: Multi-Model Cross-Validation
- **Needs UI:** Yes — dissent between models must be visible
- **Where:** Transcript (finding annotations showing "⚠ Flash dissent"), tooltip on hover showing Flash's rationale
- **Covered:** `structural-compensation.feature` @comp-cross-validate, @comp-fe-cross-validate

## Comp 3: Post-Processing Passes
- **Needs UI:** Yes — annotations on subagent output, cost tracking
- **Where:** Transcript ([UNVERIFIED], [MISSING] tags), context panel (compensation cost breakdown)
- **Covered:** `structural-compensation.feature` @comp-postprocess-*, @comp-fe-cost-breakdown

## Comp 4: Confidence Calibration
- **Needs UI:** Yes — routing decisions, accuracy dashboards
- **Where:** Background panel (routing notices), Settings → Compensations (accuracy profiles per role/language/model)
- **Covered:** `structural-compensation.feature` @comp-confidence-*, Settings panel in desktop regression tests

## Comp 5: Prompt Regression Bisection
- **Needs UI:** Yes — bisection results, rollback confirmation
- **Where:** Settings → Prompts (bisection log, before/after pass rates), status line alert on degradation
- **Covered:** `prompt-calibration.feature` @prompt-eval-fe, `quality-improvement.feature` @quality-fe-drift-alert

## Comp 6: Consensus Weighting
- **Needs UI:** Yes — weighted vs unweighted vote display
- **Where:** Transcript (finding detail showing both counts: "CONFIRMED 2/3 (weighted 2.63 vs 0.78)")
- **Covered:** `structural-compensation.feature` @comp-weighted-voting

## Comp 7: Self-Healing Prompt Library
- **Needs UI:** Minimal — automatic, user just sees result
- **Where:** Settings → Prompts (library dashboard, auto-select log), status line notice on model bump
- **Covered:** `structural-compensation.feature` @comp-prompt-library

## Comp 8: Context Janitor
- **Needs UI:** Yes — savings tracking, before/after comparison
- **Where:** Context panel (janitor activity log: "3 passes today · 1,200 tokens saved · $0.004 net savings"), transcript (compacted sections marked)
- **Covered:** `structural-compensation.feature` @comp-janitor

## Comp 9: Pre-Mortem Analysis
- **Needs UI:** Yes — predictions vs actuals comparison
- **Where:** Transcript (pre-mortem predictions card before workflow), workflow results card (post-mortem: "Predicted 3 risks. 1 materialized.")
- **Covered:** `structural-compensation.feature` @comp-premortem

## Comp 10: Semantic Diff
- **Needs UI:** Yes — structured diff output
- **Where:** Transcript (tiered findings card: 12 CONFIRMED · 7 LIKELY · 4 POSSIBLE · 2 CONTRADICTIONS), each tier expandable
- **Covered:** `structural-compensation.feature` @comp-semantic-diff

## Comp 11: Ambient Guardian
- **Needs UI:** Yes — proactive notices, cost indicator in status line
- **Where:** Status line (🛡️ dimmed indicator with daily cost), transcript (guardian notice card on detection), do-not-disturb queue
- **Covered:** `compensations-beyond.feature` @comp-ambient-guardian

## Comp 12: Idle-Time Pre-Computation
- **Needs UI:** Yes — idle activity indicator, instant answer badge
- **Where:** Status line (💤 idle pre-compute indicator), transcript (⚡ instant answer badge on pre-computed responses), knowledge base stat card
- **Covered:** `compensations-beyond.feature` @comp-idle-prefetch

## Comp 13: Dependency Watchdog
- **Needs UI:** Yes — weekly report card, safe-to-merge indicators
- **Where:** Transcript (weekly report card), status line (📦 on report day), per-dependency action buttons
- **Covered:** `compensations-beyond.feature` @comp-dependency-watchdog

## Comp 14: Self-Play Code Generation
- **Needs UI:** Yes — comparison card, approach metrics
- **Where:** Transcript (comparison card: 3 implementations with metrics, winner highlighted), "View diff" between any two
- **Covered:** `compensations-beyond.feature` @comp-self-play

## Comp 15: Ensemble Code Review
- **Needs UI:** Yes — color-coded findings by reviewer type
- **Where:** Transcript (findings grouped by type with color: red=security, blue=correctness, yellow=performance), summary counts
- **Covered:** `compensations-beyond.feature` @comp-ensemble-review

## Comp 16: Cross-Session Knowledge Base
- **Needs UI:** Yes — instant answer badge, KB stats
- **Where:** Transcript (⚡ instant answer badge), Settings → Knowledge Base (stats: entries, size, hit rate, savings), search interface
- **Covered:** `compensations-beyond.feature` @comp-knowledge-base

## Comp 17: Bug Pattern Recognition
- **Needs UI:** Yes — pattern match alert, pattern library stats
- **Where:** Status line (pattern match notice on save), Settings → Patterns (library stats: patterns learned, catches, precision), pattern detail cards
- **Covered:** `compensations-beyond.feature` @comp-bug-pattern

## Comp 18: Impact Analysis Pre-Computation
- **Needs UI:** Yes — call-site table, fix-all button
- **Where:** Transcript (call-site table: file, line, status color), "Fix all" button for trivial fixes
- **Covered:** `compensations-beyond.feature` @comp-impact-analysis

## Global Compensation UI

- **Compensations Dashboard:** Settings → Compensations. All 18 listed with status, cost, value. Grouped by class. Toggle per compensation. `compensations-beyond.feature` @comp-dashboard
- **Status Bar Aggregate:** ⚙️ N compensations active · $X.XX today · est. $Y.YY saved. `compensations-beyond.feature` @comp-status-bar

## Verdict

**All 18 compensations have fully specified UI coverage.** No compensation is invisible to the user. Every compensation answers: where it renders, what the user sees, and how the user interacts with it (toggle, dismiss, act).
