# Operational Gaps — What We Missed

> Not features. Not scenarios. Operational reality — build, release, crash, corrupt, recover.

## 1. Build & CI/CD Pipeline — Undocumented

**What we have:** Makefile exists. `.goreleaser.yaml` exists. GitHub Actions presumably exist.

**What's missing:** No CI pipeline documented for the new feature set. The testing strategy (`testing-strategy.feature`) defines test tiers but doesn't define the CI pipeline that runs them.

**Gap:** How does CI change? New binaries (reasonix-worker), new test tiers (prompt calibration, stress tests), new coverage requirements (FE tests, i18n coverage). The Makefile needs new targets. `.goreleaser.yaml` needs the worker binary.

## 2. Release Versioning & Communication — Undocumented

**What we have:** `docs/RELEASING.md` for the existing release process. No versioning strategy for the new features.

**What's missing:** What's the version number? v2.0? v1.5? How do we communicate "this release adds workflows, remote workers, and 25 compensations" to users? What's the changelog format? How do we handle the migration notice on first run?

**Gap:** Release notes generation (Comp 24) handles project releases, not reasonix's own releases.

## 3. Backward Compatibility Testing — Undesigned

**What we have:** Migration path (`MIGRATION_PATH.md`) documents the config migration. Regression matrix covers existing features.

**What's missing:** Explicit cross-version tests. Can a v1.0 session transcript load in v2.0? (Should: yes, with normalization.) Can v2.0 session transcripts with new fields (workflow refs, compensation metadata) load in v1.0? (Should: no, with a clear error.) Are these tested?

**Gap:** Cross-version session compatibility is asserted in the migration doc but never tested in Gherkin.

## 4. Performance Regression Automation — Undesigned

**What we have:** Performance budget (`PERFORMANCE_BUDGET.md`) with specific thresholds.

**What's missing:** No automated benchmark suite. No CI job that runs the stress test suite and compares against baseline. No alert when a PR regresses latency by >10%.

**Gap:** The budget is documented but not enforced in CI.

## 5. Security Audit of New Features — Unaudited

**What we have:** Threat model for remote worker (`THREAT_MODEL.md`).

**What's missing:** Security review of the new tools:
- `web_search`: SSRF protection? (Client-side tool — needs same IP-blocking as `web_fetch`.)
- `repo_connection`: GitHub token storage? (Credential store — already documented, but needs verification.)
- Knowledge base: data at rest encryption? (Local filesystem — no encryption. Is this acceptable for code analysis data?)
- Webhook push notifications: endpoint validation? (User-configured URL — SSRF risk if attacker controls config.)
- `send_notification`: arbitrary message content? (User-initiated — fine, but verify it can't be triggered by subagent output injection.)

**Gap:** New tools increase the attack surface. Each needs a brief security review.

## 6. License & Dependency Audit — Unverified

**What we have:** Single dependency (BurntSushi/toml). Spec commitment to lean dependencies.

**What's missing:** New packages might add dependencies. The knowledge base needs embeddings — does that require a new dependency? The GitHub client needs OAuth — new dependency? Docker sandbox engine needs Docker SDK — new dependency?

**Gap:** Every new package's dependencies must be audited against the lean-dependency spec before merge.

## 7. Corrupted State Recovery — Undesigned

**What we have:** Crash recovery for workflows. Atomic writes for persistence.

**What's missing:** What happens when state IS corrupted despite atomic writes?
- KB SQLite database is corrupted — is there a repair/rebuild path?
- Prompt library file is manually edited to invalid JSON — does reasonix crash or fall back to built-in defaults?
- Confidence profile JSON is truncated mid-write — does the system detect and rebuild?
- Scheduled tasks JSON is corrupted — do all cron jobs silently disappear?

**Gap:** No "reset to factory defaults" or "rebuild corrupted state" mechanism beyond the existing migration backup.

## 8. Graceful Shutdown with In-Flight Operations — Partially Covered

**What we have:** Worker SIGTERM drains in-flight work. Session save on close.

**What's missing:**
- Force quit (SIGKILL) mid-KB-write — does atomic write protect this? (Yes, temp file + rename.) Verified?
- Force quit mid-prompt-calibration — does the partial calibration result corrupt the prompt library? (Should not — prompts are only updated on explicit ship.)
- Force quit mid-confidence-profile-update — does the partial write corrupt the profile?
- User closes laptop (sleep) during remote worker communication — does the connection recover?

**Gap:** Force quit / sleep scenarios not covered in Gherkin.

## 9. Disk Space Exhaustion — Undesigned

**What we have:** No disk space monitoring. KB has a size limit with eviction. Transcripts have retention.

**What's missing:** What happens when the disk is ACTUALLY full?
- KB eviction tries to write but gets ENOSPC
- Subagent transcript save gets ENOSPC
- Workflow manifest update gets ENOSPC
- Tool output capped at 32KB, but what if the tool can't even write 32KB?

**Gap:** No disk space monitoring. No "degraded mode" when disk is >95% full. No early warning.

## 10. Concurrent Config Modification — Undesigned

**What we have:** Config loaded at boot. Settings panel writes to config.

**What's missing:** What if the user edits reasonix.toml in an editor while reasonix is running?
- Does reasonix detect the change and reload? (Current behavior: no — config is loaded at boot.)
- Does the settings panel overwrite the user's manual edit on next save?
- Does `reasonix doctor` detect config changes since boot?

**Gap:** No file watcher on config. No reload-without-restart.

## 11. Symlink & TOCTOU Attacks Within Sandbox — Unaudited

**What we have:** Sandbox enforces WriteRoots and ForbidReadRoots at the OS level.

**What's missing:** Within the workspace (allowed root):
- Subagent creates a symlink from `/workspace/innocent` → `/etc/passwd`, then reads it
- Subagent creates a file, verifies it's within WriteRoots, then replaces it with a symlink before writing (TOCTOU)
- Subagent writes a malicious `.git/hooks/pre-commit` that executes outside the sandbox on the next git operation

**Gap:** The OS sandbox handles direct path escape, but symlink attacks within allowed roots are a different class. The existing sandbox tests cover path traversal but not symlink TOCTOU.

## 12. Multi-Monitor & DPI Edge Cases — Uncovered

**What we have:** Desktop app with window state management. Resizable drawers.

**What's missing:**
- User has 2 monitors with different DPI scaling (125% and 200%)
- User moves the window between monitors — does it render correctly?
- User unplugs an external monitor — does the window move to the remaining screen?
- Full screen on a 4K display with 200% scaling — are fonts readable?

**Gap:** Desktop regression tests don't cover multi-monitor or DPI scaling.

## 13. Terminal Edge Cases — Uncovered

**What we have:** TUI tests at 80 columns. Theme tests for dark/light/auto.

**What's missing:**
- Terminal <40 columns — does the TUI degrade gracefully or crash?
- Terminal with no color support (TERM=dumb) — does it fall back to plain text?
- SSH session with 200ms latency — does streaming feel responsive or laggy?
- tmux/screen nested sessions — do keybindings conflict?
- CJK terminal with double-width characters — does alignment break?

**Gap:** TUI regression tests cover 80-column behavior but not extreme terminal environments.

## 14. Factory Reset — Undesigned

**What we have:** No reset mechanism. Config migration creates backups.

**What's missing:** If everything is broken — corrupted KB, corrupted prompts, corrupted profiles, corrupted config — is there one command to reset to a clean state?

```
reasonix reset --keep-config    # Reset KB, prompts, profiles, telemetry — keep reasonix.toml
reasonix reset --everything     # Full factory reset
```

**Gap:** No recovery path for "everything is corrupted." The user's only option is manual deletion of `.reasonix/` subdirectories.

## 15. Network Partition During Critical Operations — Uncovered

**What we have:** Circuit breaker for provider outages. Offline detection for remotes.

**What's missing:** Mid-operation network loss:
- KB is writing an entry, network drops (KB is local — unaffected. But the subagent that produced the entry was mid-stream.)
- Prompt calibration is running live API evaluations, network drops — does it fail gracefully or hang?
- Confidence profile is being updated from a just-completed verification, network drops — the verification result is locally available (subagent transcript), but the profile update doesn't need network. Unaffected.
- Repo clone is 80% complete, network drops — git handles this. Does reasonix surface the partial failure clearly?

**Gap:** Mid-operation network loss scenarios not covered.

---

## Severity Summary

| # | Gap | Severity | Action |
|---|---|---|---|
| 7 | Corrupted state recovery | **Critical** | Design factory reset + corruption detection |
| 9 | Disk space exhaustion | **High** | Add disk monitoring + degraded mode |
| 11 | Symlink/TOCTOU in sandbox | **High** | Audit + test sandbox against these attacks |
| 1 | CI/CD pipeline | **High** | Document before Phase 1 implementation |
| 4 | Performance regression automation | **Medium** | Add benchmark CI job |
| 8 | Graceful shutdown edge cases | **Medium** | Add Gherkin for force quit/sleep |
| 5 | Security audit of new tools | **Medium** | Per-tool security review checklist |
| 6 | License/dependency audit | **Medium** | Gate in CI |
| 14 | Factory reset | **Medium** | Design + implement |
| 2 | Release versioning | **Low** | Update RELEASING.md |
| 3 | Backward compat testing | **Low** | Add cross-version Gherkin |
| 10 | Concurrent config modification | **Low** | File watcher or explicit reload command |
| 12 | Multi-monitor/DPI | **Low** | Desktop test matrix |
| 13 | Terminal edge cases | **Low** | TUI test matrix |
| 15 | Network partition mid-op | **Low** | Covered by existing retry/circuit breaker for most cases |
