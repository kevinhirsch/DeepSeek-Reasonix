# 23 — Operational Gap Closures

**Phase:** Resilience | **Effort:** 8 days total | **Priority:** P1

## What's Included

| # | Gap | Effort |
|---|---|---|
| O1 | CI/CD Pipeline | 1 day |
| O2 | Release Versioning | 0.5 day |
| O3 | Backward Compat Testing | 0.5 day |
| O4 | Performance Regression CI | 0.5 day |
| O5 | Security Audit of New Tools | 1 day |
| O6 | License/Dependency Audit | 0.5 day |
| O7 | Corrupted State Recovery | 1 day |
| O8 | Force Quit / Sleep Recovery | 0.5 day |
| O9 | Disk Space Exhaustion | 0.5 day |
| O10 | Concurrent Config Modification | 0.5 day |
| O11 | Symlink/TOCTOU Sandbox Audit | 0.5 day |
| O12 | Multi-Monitor / DPI | 0.5 day |
| O13 | Terminal Edge Cases | 0.5 day |
| O14 | Factory Reset | 0.5 day |
| O15 | Network Partition Mid-Op | 0.5 day |

## Design Per Gap

### O1: CI/CD Pipeline — 1 day
New Makefile targets: `test-short`, `test-full`, `test-live`, `test-stress`, `build-worker`, `cross-worker`. GHA workflow: short on commit, full on PR, live nightly, stress weekly. Coverage gates: FE scenarios touched → corresponding Go test required. i18n coverage check.

### O2: Release Versioning — 0.5 day
Update `docs/RELEASING.md`. Semantic versioning: v2.0 for all new features. Changelog format: Features, Breaking Changes, Migration Notes, Deprecations. Auto-generated release notes from merged PRs (dogfood Comp 24).

### O3: Backward Compat Testing — 0.5 day
Cross-version Gherkin: v1.0 session transcript loads in v2.0 (with normalization). v2.0 new fields rejected gracefully in v1.0. Config migration tested with real v1.0 config samples.

### O4: Performance Regression CI — 0.5 day
Benchmark suite using `go test -bench`. Baseline stored. CI gate: PR fails if any benchmark regresses >10%. Dashboard: historical performance trend.

### O5: Security Audit — 1 day
Per-tool review: web_search (SSRF), monitor (command injection), send_notification (XSS in message), knowledge_base (data at rest), webhook (URL validation). Checklist in `SECURITY.md`.

### O6: Dependency Audit — 0.5 day
CI gate: `go list -m all` must be single-line (BurntSushi/toml only). Any new dependency → PR blocked unless explicitly justified in commit body with `DEP: <package> <justification>`.

### O7: Corrupted State Recovery — 1 day
`reasonix reset --keep-config` (reset KB, prompts, profiles, telemetry — keep config). `reasonix reset --everything` (full factory). Corruption detection on boot: validate JSON, check checksums, attempt repair. Fall back to defaults if unrecoverable.

### O8: Force Quit Recovery — 0.5 day
Gherkin for SIGKILL, sleep/wake, battery death mid-write. All persistence uses atomic write (temp + rename) — verify in test. Verify no partial writes survive force quit.

### O9: Disk Space Exhaustion — 0.5 day
Disk monitor: check every 5 minutes. Warning at 90%, degraded mode at 95% (pause transcript writes, evict KB, stop telemetry). Critical at 98% (read-only mode, notify user). Recovery when space freed.

### O10: Concurrent Config — 0.5 day
File watcher on reasonix.toml. Auto-reload on external change (with notice). Settings panel reads before write to avoid clobbering. `reasonix doctor` detects external changes.

### O11: Symlink/TOCTOU — 0.5 day
Sandbox test: subagent cannot create symlink outside WriteRoots. TOCTOU: path validated, then replaced with symlink before write — must be caught. Git hook injection: `.git/hooks/` in ForbidReadRoots by default.

### O12: Multi-Monitor/DPI — 0.5 day
Desktop test matrix: 100%/125%/150%/200% DPI. Monitor plug/unplug. Window move between DPI scales. Full screen on HiDPI.

### O13: Terminal Edge Cases — 0.5 day
TUI test matrix: 40-col minimum, 240-col maximum. TERM=dumb fallback. 200ms latency simulation. tmux nested keybinding conflict detection. CJK double-width character alignment.

### O14: Factory Reset — 0.5 day
CLI: `reasonix reset --keep-config`, `reasonix reset --everything`. Confirmation prompt with list of what will be deleted. Backup created before reset. Post-reset: first-run setup flow.

### O15: Network Partition — 0.5 day
Mid-operation network loss scenarios. All should surface clear errors within timeout, not hang. Subagent transcripts locally saved regardless. KB operations local only — unaffected.

## Files
- `Makefile` — new targets
- `.github/workflows/` — new CI jobs
- `docs/RELEASING.md` — update
- `SECURITY.md` — per-tool review
- `internal/disk/monitor.go` (NEW)
- `internal/config/watcher.go` (NEW)
- `internal/cli/reset.go` (NEW)
- `internal/sandbox/symlink_test.go` (NEW tests)
- `internal/sandbox/toctou_test.go` (NEW tests)
- Various test files for cross-version, terminal, DPI

## Acceptance
- [ ] CI pipeline runs all test tiers on correct cadence
- [ ] Cross-version session tests pass in both directions
- [ ] Performance regression CI gate blocks on >10% regression
- [ ] Security audit checklist completed for all new tools
- [ ] Dependency audit gate blocks new deps without justification
- [ ] Factory reset recovers from all corrupted state scenarios
- [ ] Force quit / sleep recovery verified via atomic write tests
- [ ] Disk monitor fires warnings at correct thresholds
- [ ] Config watcher detects external changes and reloads
- [ ] Sandbox blocks symlink escape and TOCTOU
- [ ] Desktop renders correctly at all DPI scales
- [ ] TUI works at 40-col minimum and 240-col maximum
