# 25 — Final Gap Closures (Third Pass)

**Phase:** Resilience | **Effort:** 5 days total | **Priority:** P2

## Gaps

| # | Gap | Effort |
|---|---|---|
| F1 | Proxy coverage for new network paths | 0.5 day |
| F2 | MCP output as prompt injection vector | 0.5 day |
| F3 | API key rotation detection | 0.5 day |
| F4 | Session export redaction depth | 0.5 day |
| F5 | Compaction at extreme session lengths | 1 day |
| F6 | Graceful degradation when compensation fails | 0.5 day |
| F7 | Timezone handling for scheduling | 0.5 day |
| F8 | Subagent spawn during active compaction | 0.5 day |
| F9 | Prompt injection via file content | 0.5 day |
| F10 | Warm start vs cold start UX | 0.5 day |

## Design Per Gap

### F1: Proxy for New Network Paths
All new outbound HTTP calls must use `netclient.NewHTTPClient(proxySpec)`. Audit: GitHub OAuth, web_search, clone remote, bootstrap download, mobile webhook. CI gate: any new `http.Get` or `http.Client{}` without proxy is a lint error.

### F2: MCP Output Injection Guard
MCP tool results pass through a lightweight content check before entering the agent's context. Pattern match for known injection markers: "ignore previous instructions", "you are now", "system:", "<system-reminder>". Flagged output gets a warning annotation but still enters context (false positive risk). Non-blocking.

### F3: Key Rotation Detection
If >3 consecutive requests across any provider return 401: emit "API key appears invalid. Update in credential store?" with a link to the relevant env var or credential store entry. Distinguish from 403 (permissions) and 401 (auth). Pause subagent spawning for that provider until key is updated.

### F4: Export Redaction
`/export` runs a redaction pass: strip `Authorization: *`, `x-api-key: *`, `api_key=*`, git remote URLs with userinfo, `~/home/user/` patterns. User can preview redactions before export. `--no-redact` flag for full export.

### F5: Extreme Session Compaction
Stress test: 8-hour session simulation with 500+ messages and 50+ subagents. Verifies: compaction doesn't loop, GC pressure stays bounded, subagent context windows don't exhaust on spawn. Compaction summary prompt tested with 500-message inputs.

### F6: Compensation Failure Isolation
Compensation execution wrapped in recover(). Failure → original output returned unmodified + notice: "⚠ Correctness pass failed — original output preserved." No compensation failure blocks the parent task.

### F7: Timezone
Scheduled tasks store timezone explicitly (IANA). `reasonix serve` requires `--timezone` flag. Remote workers report their timezone. Desktop uses OS timezone. Migration: existing tasks tagged with local timezone on upgrade.

### F8: Subagent Spawn Context Bounded
Before spawning a subagent: check parent session token count. If >80% of context window: compact parent first, then spawn. Subagent starts with ≤50% context utilization.

### F9: File Content Injection Guard
`read_file` output passes through same content check as MCP output (F2). No content blocking — annotation only. Non-blocking because any file could trigger it (including legitimate documentation about prompt injection).

### F10: Startup UX
Cold start: spinner with phase text ("Loading config...", "Checking providers...", "Ready."). First ever start: setup wizard. Crash recovery: "Recovering..." with session count. Update available: "Downloading reasonix v2.1..." with progress bar.

## Files
- `internal/netclient/proxy_audit_test.go` (NEW)
- `internal/agent/injection_guard.go` (NEW)
- `internal/provider/key_rotation.go` (NEW)
- `internal/export/redact.go` (NEW)
- `internal/agent/compact_stress_test.go` (NEW)
- `internal/agent/compensation_safety.go` (NEW)
- `internal/schedule/timezone.go` (NEW)
- `internal/cli/startup.go` (NEW or modify)

## Acceptance
- [ ] All new HTTP calls audited for proxy support
- [ ] MCP output flagged on injection patterns, not blocked
- [ ] 3+ consecutive 401s triggers key rotation prompt
- [ ] Export redacts secrets, previewable
- [ ] 500-message session compacts without loop, subagents spawn under context limit
- [ ] Compensation crash returns original output + notice
- [ ] Scheduled tasks store IANA timezone
- [ ] File content flagged on injection patterns, not blocked
- [ ] Startup shows phase progress, not blank screen
