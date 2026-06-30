# Performance Budget

> Latency, memory, and responsiveness thresholds. Every feature must stay within budget.

## TUI Responsiveness

| Metric | Budget | Measured By |
|---|---|---|
| Panel render time | <16ms (60fps) | `time.Since(start)` in render path |
| First panel appear after subagent spawn | <200ms | Event emission to first render |
| Tool count update latency | <500ms | Tool call completion to UI update |
| Status line update latency | <100ms | Event emission to status line render |
| Chat input keystroke to echo | <16ms | Every keystroke |
| Slash picker open latency | <50ms | `/` press to menu render |
| @ file picker open latency | <100ms | `@` press to menu render |
| Model switch latency | <200ms | Selection to status line update |

## Agent Performance

| Metric | Budget | Measured By |
|---|---|---|
| Subagent spawn overhead | <100ms | Goroutine start to first API request |
| Steer message delivery | <50ms | `Steer()` call to message in queue |
| Compaction pass | <2s | Start to session.Replace() |
| Final readiness check | <500ms | Start to gate decision |
| Promise detection | <10ms | Regex match on final answer text |
| Claim verification | <50ms | Evidence ledger cross-reference |
| Workflow stage transition | <200ms | Last subagent complete to next stage spawn |

## Memory

| Metric | Budget | Measured By |
|---|---|---|
| Steady-state RSS | <2× baseline after 100 workflows | `runtime.ReadMemStats()` |
| Monotonic heap growth | 0 over 50 workflows | `memstats.HeapAlloc` trend |
| Subagent transcript in memory | <1MB per subagent | Message slice size |
| Workflow manifest in memory | <10KB per workflow | Struct size |
| Job view cache | <1KB per job | View struct size |

## Network

| Metric | Budget | Measured By |
|---|---|---|
| Worker health check | <5s timeout | HTTP client timeout |
| Work queue long poll | 30s timeout | HTTP client timeout |
| Clone progress update interval | 500ms | Poll timer |
| Remote event stream buffer | 1000 events | Channel capacity |
| Clone to remote: overhead | <2s (command round-trip) | Timestamp diff |

## Startup

| Metric | Budget | Measured By |
|---|---|---|
| Cold start to TUI visible | <1s | `time.Since(main)` |
| Warm start (config cached) | <500ms | No provider health checks |
| Crash recovery scan | <2s for 50 workflows | Manifest scan + status check |
| Config load + validation | <100ms | TOML parse + schema check |

## Token Budgets (from Tier 4 DoD)

| Metric | Budget |
|---|---|
| Cache hit rate | ≥80% on runs 2-3 |
| Token-per-task regression | ≤20% from baseline |
| Security audit cost ceiling | ≤$0.10 at DeepSeek rates |
| Subagent spawn efficiency | Within 10% of expected count |
