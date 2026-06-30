# CLAUDE.md — kevinhirsch/DeepSeek-Reasonix

## Fork Identity

This is **`kevinhirsch/DeepSeek-Reasonix`** — a fork of [`esengine/DeepSeek-Reasonix`](https://github.com/esengine/DeepSeek-Reasonix) (25,480★, 1,566 forks).

- **Upstream:** `esengine/DeepSeek-Reasonix` (MIT license)
- **Fork:** `kevinhirsch/DeepSeek-Reasonix` (public, MIT)
- **Branch:** `main-v2` (Go rewrite, identical to upstream default branch)
- **Remote:** `origin` = fork, `upstream` = esengine

### Why a fork

The upstream receives 60+ PRs/day with ~10 merges/day. 133 open PRs at any given moment. Our work — 25 architectural issues spanning subagent messaging, remote workers, workflow engines, compensations, and stress-test resilience — is a systematic rebuild, not a series of small fixes. It would sit in review purgatory upstream.

The strategy: **implement independently, cherry-pick useful upstream fixes, ship fast.**

## Roadmap

7 phases, dependency-ordered. See `.reasonix/ROADMAP.md` for full detail.

| Phase | Issues | Duration |
|---|---|---|
| 1: Quality Foundation | #18, #16, #1, #2 | 3.5 days |
| 2: Subagent Foundation | #3, #4, #6 | 2.5 days |
| 3: Workflow Engine | #5, #8 | 3.5 days |
| 4: Visibility | #7, #17 | 2.5 days |
| 5: Remote Environments | #9–#15 | 9.5 days |
| 6: Quality Compensations | #19, #20, #21 | — |
| 7: Resilience & Gap Closures | #22, #23, #24, #25 | — |

**Critical path:** Phase 1 → Phase 2 → Phase 3 → Phase 5
**Parallelizable:** Phase 4 runs alongside Phase 2-3. Phase 6-7 items spread across.

## Issue Tracking

All 25 issues live at `https://github.com/kevinhirsch/DeepSeek-Reasonix/issues`. Each has:
- Phase label (`phase:intelligence`, `phase:remote`, etc.)
- Priority label (`priority:p0` through `priority:p2`)
- Full Problem / Design / Files / Acceptance sections
- Harvest audit comments on impacted issues (#1, #3, #4, #8, #18)

Dependencies are documented in `.reasonix/DEPENDENCY_GRAPH.md`.

## Upstream Harvesting

We selectively cherry-pick high-value upstream PRs. Full process in `.github/harvesting.md`.

### Quick harvest

```bash
# Evaluate a PR
gh pr view <N> --repo esengine/DeepSeek-Reasonix --json files,additions

# Fetch for conflict test
git fetch upstream refs/pull/<N>/head:pr-upstream/<N>

# If clean (no conflicts in our changed files):
git cherry-pick -x pr-upstream/<N>

# Amend with attribution
git commit --amend -m "$(git log -1 --format=%B) \nCo-Authored-By: <gh-handle>"

# Squash into harvest commit (preferred for bulk)
```

### Harvested so far (2 commits, 11 PRs)

| Commit | Upstream PRs |
|---|---|
| `dd66dc92` | #5461, #5564, #5589, #5643, #5660 |
| `0ebba5f8` | #4864, #5055, #5105, #5212, #5362, #5415 |

### Pending harvest inventory
- #5276 (BTW TUI — when starting issue #7)
- #5299 (prompt catalog — when starting issue #18)
- #5405 (token tracking — when starting issue #6)
- #4881 (search_sessions — when starting issue #20)

### Harvest principles
- **Tier 1:** Direct cherry-picks — bug fixes, hardening, validation. Safe, no design decisions.
- **Tier 2:** Infrastructure harvest — take the data layer, rewrite the surface API.
- **Tier 3:** Conceptual — read, learn, write fresh for our architecture.
- **Attribute:** Every harvest commit includes `Co-Authored-By` trailers.
- **Never harvest:** Werewolf games, heavily conflicted refactors, desktop-only polish.

## Clone & Setup Recipe

```bash
git clone https://github.com/kevinhirsch/DeepSeek-Reasonix.git
cd DeepSeek-Reasonix
git remote add upstream https://github.com/esengine/DeepSeek-Reasonix.git

# Pull harvest tracking data (refs + notes)
git config --local --add remote.origin.fetch "+refs/harvest/*:refs/harvest/*"
git config --local --add remote.origin.fetch "+refs/notes/*:refs/notes/*"
git fetch origin

# Restore harvest inventory note
git notes --ref=upstream-harvest show HEAD
```

## Documentation Inventory

### Roadmap & Planning
- `.reasonix/ROADMAP.md` — 7-phase implementation plan with durations
- `.reasonix/DEPENDENCY_GRAPH.md` — issue dependency graph and critical path
- `.reasonix/MIGRATION_PATH.md` — v1.0→v2.0 migration strategy

### Quality Gates
- `.reasonix/DEFINITION_OF_DONE.md` — 5 tiers, 189 checklist items. Every deliverable must satisfy all tiers.

### Design Documents (by issue)

| Doc | Issues | Purpose |
|---|---|---|
| `.reasonix/PROMPT_CALIBRATION.md` | #18 | 7-part prompt structure, evaluation harness, calibration thresholds |
| `.reasonix/QUALITY_ARCHITECTURE.md` | #16 | 5 quality pillars, behavioral gates, DeepSeek-specific tuning |
| `.reasonix/QUALITY_IMPROVEMENT_PLAN.md` | #16 | Quality improvement tracking |
| `.reasonix/STRUCTURAL_COMPENSATION.md` | #19 | Compensations 1-10: speculative execution, cross-validation, post-processing |
| `.reasonix/COMPENSATIONS_BEYOND.md` | #20 | Compensations 11-18: guardian, idle pre-computation, dependency watchdog |
| `.reasonix/COMPENSATIONS_FINAL.md` | #21 | Compensations 19-25: adversarial self-testing, KB, tech debt, release notes |
| `.reasonix/COMPENSATION_UI_WIRING.md` | #19-#21 | UI surface wiring for all compensations |
| `.reasonix/STRESS_TEST.md` | #22 | Architecture stress test findings across 8 categories |
| `.reasonix/GAP_ANALYSIS.md` | #22-#25 | Gap analysis across all phases |
| `.reasonix/OPERATIONAL_GAPS.md` | #23 | 15 operational gap closures (O1-O15) |
| `.reasonix/FINAL_GAPS.md` | #25 | 10 final gap closures (F1-F10) |
| `.reasonix/FIDELITY_BRIDGE.md` | All phases | Strategy for bridging DeepSeek↔Claude Code behavioral quality |

### Architecture & Design Reference
- `.reasonix/STRATEGY_COMPARISON.md` — archival: original design explorations
- `.reasonix/PERSISTENCE_DESIGN.md` — session, transcript, and state persistence design
- `.reasonix/PACKAGE_DEPENDENCIES.md` — Go package dependency map
- `.reasonix/CONFIG_SCHEMA_ADDITIONS.md` — reasonix.toml schema extensions
- `.reasonix/MOCK_PROVIDER_DESIGN.md` — mock provider design for deterministic testing
- `.reasonix/PERFORMANCE_BUDGET.md` — latency and token budgets per operation
- `.reasonix/REGRESSION_MATRIX.md` — cross-version regression test matrix
- `.reasonix/THREAT_MODEL.md` — security threat model for new tools

### Implementation Reference
- `.reasonix/ERROR_CODES.md` — error code taxonomy
- `.reasonix/CLI_FLAGS.md` — CLI flag registry
- `.reasonix/UI_TRACEABILITY.md` — UI surface to Gherkin scenario mapping
- `.reasonix/UNKNOWNS.md` — known unknowns and open decisions

### Research
- `.reasonix/research/01-deepseek-reasoning-patterns.md` — DeepSeek cognitive behavior patterns
- `.reasonix/research/02-subagent-ux-patterns.md` — subagent monitoring UX from 20+ CLI tools
- `.reasonix/research/03-claude-code-subagent-architecture.md` — Claude Code's subagent internals
- `.reasonix/research/04-reasonix-codebase-audit.md` — full codebase architecture audit
- `.reasonix/research/05-tokenomics-reference.md` — pricing, tokens-per-turn, cost models
- `.reasonix/research/README.md` — research archive index

### Specifications
- `.reasonix/issues/*.md` — 25 issue specifications (Problem, Design, Files, Acceptance)
- `.reasonix/features/*.feature` — 49 Gherkin feature files (1,174 scenarios)
- `docs/GUIDE.md` — user guide
- `docs/SPEC.md` — architecture specification
- `.github/harvesting.md` — upstream PR harvesting procedure and 24-PR inventory

## Communication

- Issues: `https://github.com/kevinhirsch/DeepSeek-Reasonix/issues`
- Upstream: `https://github.com/esengine/DeepSeek-Reasonix` (MIT, 25K★)
- All harvested code is MIT-licensed, compatible with our fork
