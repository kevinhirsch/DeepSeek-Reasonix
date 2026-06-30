# Implementation Roadmap

> Ordered by dependency. Each phase unblocks the next. Nothing in a phase may ship until its dependencies are complete.

## Phase 1: Quality Foundation (Week 1 — 3.5 days)

Must ship before any subagent work. Every subsequent phase inherits these behaviors.

| # | Issue | Effort | Depends On |
|---|---|---|---|
| 18 | Prompt Calibration Framework | 2 days | — |
| 16 | Quality Architecture (prompt + gates) | 1 day | 18 |
| 01 | Role-Specific Subagent Prompts | 1 day | 18 |
| 02 | DeepSeek Temperature Override | 0.5 day | — |

**Milestone:** All subagent prompts calibrated to ≥90% pass rate. Agent produces Claude Code-quality output with anti-overengineering, silence default, promise detection.

## Phase 2: Subagent Foundation (Week 1-2 — 2.5 days)

Core subagent infrastructure. Every subsequent phase uses these primitives.

| # | Issue | Effort | Depends On |
|---|---|---|---|
| 03 | Cognitive Loop Detector | 1 day | Phase 1 |
| 04 | Subagent Messenger + send_to_subagent | 1 day | Phase 1 |
| 06 | Enriched Job View | 0.5 day | Phase 1 |

**Milestone:** Subagents can communicate. Parent can steer. Loop detection active. Jobs have live state.

## Phase 3: Workflow Engine (Week 2-3 — 3.5 days)

Dynamic multi-stage workflows. Depends on subagent primitives from Phase 2.

| # | Issue | Effort | Depends On |
|---|---|---|---|
| 05 | Workflow Tool (pipeline/barrier/loop-until-dry) | 3 days | Phase 2 |
| 08 | Effort Calibration Table | 0.5 day | Phase 1 |

**Milestone:** Model can compose find→verify→report workflows in one tool call.

## Phase 4: Visibility (Week 3-4 — 2.5 days)

User-visible monitoring. Depends on job views from Phase 2.

| # | Issue | Effort | Depends On |
|---|---|---|---|
| 07 | Background Tasks TUI Panel | 2 days | Phase 2, 6 |
| 17 | Repo Connection & Auto-Setup | 3 days | — |

**Milestone:** User sees subagent status live. Repos auto-connect.

## Phase 5: Remote Environments (Week 4-6 — 9.5 days)

Remote sandbox execution. Depends on workflow engine from Phase 3.

| # | Issue | Effort | Depends On |
|---|---|---|---|
| 09 | Remote Bootstrap Command | 1 day | Phase 1 |
| 10 | Worker Package | 3 days | Phase 1 |
| 11 | Worker Binary | 0.5 day | 10 |
| 12 | Work Queue Endpoints | 1.5 days | 10 |
| 13 | Docker Sandbox Engine | 2 days | Phase 1 |
| 14 | Remotes Config + Settings Panel | 1.5 days | Phase 1 |
| 15 | Remote Target in Tools | 1 day | Phase 3, 10 |

**Milestone:** Subagents run on remote build servers with full isolation.

## Phase 6: Tool Gaps (Week 6-8 — 13 days)

New tools and skills. Many are independent of each other.

| # | Gap | Effort | Depends On |
|---|---|---|---|
| G1 | web_search tool | 1 day | — |
| G2 | deep-research skill | 0.5 day | Phase 3, G1 |
| G3 | Monitor tool | 1 day | — |
| G4 | ReportFindings tool | 0.5 day | — |
| G5 | Cron + ScheduleWakeup | 2 days | — |
| G6 | PushNotification (proactive) | 1.5 day | — |
| G7 | Structured Task System | 1.5 day | — |
| G8 | EnterWorktree | 1.5 day | — |
| G9 | Skills (code-review, verify, simplify, run, loop, review-PR, update-config, permissions, keybindings) | 2.5 days | Phase 3 |
| G10 | AskUserQuestion enhancement | 1 day | — |

**Milestone:** Full Claude Code tool parity. All skill gaps closed.

## Phase 7: Resilience & Polish (Week 8-10 — spread across)

Cross-cutting concerns. Many are addressed incrementally.

| # | Concern | Effort | Depends On |
|---|---|---|---|
| U2 | Model Fallback / Degradation | 2 days | Phase 1 |
| U1 | Workflow Crash Recovery | 2.5 days | Phase 3 |
| U3 | Rate Limit Backpressure | 1.5 day | Phase 2 |
| U6 | Remote Worker Security | 2 days | Phase 5 |
| U12 | Testing Strategy (mock providers) | 2 days | Phase 1 |
| U5 | Subagent Debugging | 1.5 day | Phase 4 |
| U7 | Config Migration | 1 day | Phase 1 |
| U13 | i18n of New Features | 1 day | All phases |
| U14 | Graceful Degradation | 1 day | Phase 4 |
| U4 | Session Portability | 1.5 day | Phase 3 |
| U8 | Observability / Telemetry | 1.5 day | Phase 4 |
| U15 | Documentation | Ongoing | All phases |
| U9 | Accessibility | 1.5 day | Phase 4 |
| U10 | Mobile / Off-Device | 2 days | Phase 5 |
| U11 | Plugin API | 2 days | v2 |

## Summary

| Phase | Duration | Issues | Cumulative |
|---|---|---|---|
| 1: Quality Foundation | 2.5 days | 3 | 2.5 |
| 2: Subagent Foundation | 2.5 days | 3 | 5.0 |
| 3: Workflow Engine | 3.5 days | 2 | 8.5 |
| 4: Visibility | 2.5 days | 2 | 11.0 |
| 5: Remote Environments | 9.5 days | 7 | 20.5 |
| 6: Tool Gaps | 13 days | 10 | 33.5 |
| 7: Resilience & Polish | 24.5 days (spread) | 15 | — |

**Critical path:** Phase 1 → Phase 2 → Phase 3 → Phase 5 → Phase 6
**Parallelizable:** Phase 4 (Visibility) runs alongside Phase 2-3. Phase 7 items run spread across all phases.
