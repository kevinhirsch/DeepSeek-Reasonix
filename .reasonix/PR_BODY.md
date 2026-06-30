# reasonix v2.0 — DeepSeek-Native Architecture

> **This PR documents 42 commits implementing 25 architectural issues across 7 phases.**
> It is a release summary — individual components can be cherry-picked.
> 279 files changed, 42,076 insertions.

---

## Overview

This fork implements a systematic architectural rebuild of reasonix, optimized for DeepSeek models. The work spans 7 phases (Quality Foundation → Resilience), produces 4 build artifacts (CLI, worker daemon, desktop app, NSIS installer), and passes 17/17 prompt evaluation scenarios at 100%.

## Build Artifacts

| Artifact | Size | Build Command |
|---|---|---|
| reasonix.exe | 31.5 MB | `go build ./cmd/reasonix` |
| reasonix-worker.exe | 9.5 MB | `go build ./cmd/reasonix-worker` |
| reasonix-desktop.exe | 44.9 MB | `wails build` |
| reasonix-desktop-installer.exe | 15.6 MB | `wails build -nsis` |

## Phase 1: Quality Foundation (Issues #18, #16, #1, #2)

- **Prompt Calibration Framework** — `internal/prompt/eval.go`, 17 scenarios across 5 roles. File-based prompt loading from `.reasonix/prompts/v1/`. CLI: `reasonix prompt eval <role> [--live]` with auto-generated mock fallback.
- **Quality Architecture** — `internal/boot/boot.go`: ~45-line quality system prompt block (anti-overengineering, evidence-grounded claims, silence default, outcome-first). 3 behavioral gates in `internal/agent/agent.go`: promise detector, claim verifier, completeness check. DeepSeek-specific tuning for over-engineering, over-narration, repetition.
- **Role-Specific Subagent Prompts** — `internal/agent/subagent_prompts.go`: Explorer, Reviewer, Verifier, Planner, Executor prompts via first-user-message injection for DeepSeek. `DeepSeekCommonNotes` for cognitive loop counter-steering.
- **DeepSeek Temperature Override** — `internal/boot/boot.go`: `deepSeekSubagentTemperature()` overrides 0.0→0.6 for DeepSeek subagents only. Parent turn unaffected. Info notice on first override.

## Phase 2: Subagent Foundation (Issues #3, #4, #6)

- **Cognitive Loop Detector** — `internal/agent/cognitive_loop.go`: 3 detection patterns (uncertainty escalation, identical reasoning prefix, mechanical self-explanation). Breaker injection on detection. DeepSeek-only gating via `IsDeepSeek` on `Agent.Options`.
- **Subagent Messenger** — `internal/agent/subagent_message.go`: `SubagentMessenger` with thread-safe register/unregister/send. `SendToSubagentTool` model-facing tool. Context plumbing via `callContext`.
- **Enriched Job Views** — `internal/jobs/jobs.go`: 8 new fields on `View` (Model, Effort, ToolCalls, LastTool, LastReasoning, Usage, DependsOn, Blocks, ResultPreview). `UpdateView()` setter for live TUI updates.

## Phase 3: Workflow Engine (Issues #5, #8)

- **Workflow Tool** — `internal/agent/workflow.go`: Pipeline, Barrier, Loop-until-dry strategies. Voting (majority/unanimous). Precision/recall verification modes. Max-tasks cap enforcement.
- **Effort Calibration Table** — `internal/boot/boot.go`: Markdown table in executor system prompt mapping roles→effort. Provider-aware (DeepSeek binary noted).

## Phase 4: Visibility (Issues #7, #17)

- **Background Tasks TUI Panel** — `internal/cli/background_panel.go`: Inline terminal panel with status icons, live tool count, keyboard navigation (j/k/Enter/s/Esc). Auto-collapse after 3s. Ctrl+T toggle.
- **Git Repository Connection** — `internal/git/git.go`: Clone, fetch, branch list, commit info, project type detection, agent docs detection. `internal/github/github.go`: OAuth device flow, repo listing, search. `desktop/frontend/src/components/RepoPicker.tsx`: React frontend picker.

## Phase 5: Remote Environments (Issues #9-#15)

- **Remote Bootstrap** — `internal/cli/remote.go`: Self-contained shell script generator (~200 lines).
- **Worker Package** — `internal/worker/`: config.go, worker.go, poller.go, executor.go, streamer.go, health.go.
- **Worker Binary** — `cmd/reasonix-worker/main.go`: Headless daemon entry point.
- **Work Queue Endpoints** — `internal/serve/workqueue.go`: REST API with bearer auth.
- **Docker Sandbox** — `internal/sandbox/docker.go`: Container lifecycle with resource limits, API key injection, timeout.
- **Remotes Config** — `internal/config/remote.go`: `RemoteEntry` with validation and token resolution.
- **Remote Target** — Target field on task/workflow tool schemas.

## Phase 6: Quality Compensations (Issues #19-#21)

- **Speculative Execution** — `internal/agent/speculative.go`: N-way consensus merging.
- **Cross-Validation** — `internal/agent/cross_validate.go`: Flash/Pro agreement check.
- **Post-Processing** — `internal/agent/postprocess.go`: Correctness/conciseness/completeness passes.
- **Confidence Calibration** — `internal/agent/confidence.go`: Per-(role,model) accuracy profiles.
- **Self-Play + Ensemble Review** — `internal/agent/selfplay.go`, `internal/agent/ensemble_review.go`.
- **Knowledge Base** — `internal/knowledge/`: store.go, query.go, verify.go, eviction.go.
- **Prompt Bisection + Library** — `internal/prompt/bisect.go`, `internal/prompt/library.go`.
- **Idle Precomputation** — `internal/idle/precompute.go`: 2-min idle with hysteresis.

## Phase 7: Resilience (Issues #22-#25)

- **Dependency Watchdog** — `internal/watchdog/dependency.go`: Weekly go module check.
- **Disk Monitor** — `internal/disk/monitor.go`: Warn 90%, degraded 95%, critical 98%.
- **Config Watcher** — `internal/config/watcher.go`: stats-based file change detection.
- **Factory Reset** — `internal/cli/reset.go`: `reasonix reset --keep-config` and `--everything`.
- **Key Rotation Detector** — `internal/provider/key_rotation.go`: 3+ consecutive 401s triggers prompt.
- **Injection Guard** — `internal/agent/injection_guard.go`: Pattern detection for MCP/file content.
- **Export Redaction** — `internal/export/redact.go`: Regex-based secret stripping.
- **Timezone Store** — `internal/schedule/timezone.go`: IANA timezone with TZ env fallback.
- **Impact Analyzer** — `internal/impact/analyze.go`: Call site analysis on signature changes.
- **Pattern Recognizer** — `internal/pattern/recognize.go`: Bug pattern learning + pre-commit scan.

## Desktop Frontend (12 new files)

| Component | Purpose |
|---|---|
| `RepoPicker.tsx` | GitHub repo browser with clone/open actions |
| `RepoSetupProgress.tsx` | Stage-by-stage clone progress |
| `BackgroundTasksPanel.tsx` | Live subagent status with keyboard nav |
| `RemotesPanel.tsx` | Remote worker management |
| `WorkflowCard.tsx` | Multi-stage workflow visualization |
| `SpeculativeResultsCard.tsx` | Consensus breakdown |
| `VotingBreakdown.tsx` | Animated voting bar chart |
| `WorkflowPanel.tsx` | Tabbed master panel |
| `accessibility.ts` | Focus trap, reduced motion, ARIA labels |
| `accessibility.css` | WCAG ratios, forced-colors, color-blind indicators |
| `animations.css` | 6 keyframes matching Fidelity Bridge spec |
| `colors.css` | Dark/light theme with documented contrast ratios |

## Upstream PRs Harvested

11 PRs cherry-picked with Co-Authored-By attribution:
`#5461 #5564 #5589 #5643 #5660 #4864 #5055 #5105 #5212 #5362 #5415`

Includes the `loopguard.go` / `curator.go` / `selfevolve.go` modules from upstream PR #5415.

## Verification

```
go build ./cmd/reasonix        — PASS
go build ./cmd/reasonix-worker  — PASS
wails build -nsis              — PASS
go test (60+ packages)         — 0 build failures
reasonix prompt eval (5 roles) — 17/17 scenarios at 100%
```

## Cherry-Pick Guide

Individual components that can be extracted with minimal dependencies:
- **cognitive_loop.go** — DeepSeek loop detection (standalone, no other changes needed)
- **workflow.go** — Multi-stage workflow engine (depends on task.go subagent dispatch)
- **worker/** — Remote execution daemon (standalone package)
- **prompt/eval.go** — Calibration framework (standalone package)
- **quality blocks in boot.go** — System prompt improvements (copy-paste constants)
- **Any frontend .tsx file** — Self-contained React components

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
