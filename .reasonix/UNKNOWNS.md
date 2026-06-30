# Things We Haven't Thought Of

> This file exists because every architecture has blind spots. These are the gaps.
> Each section identifies: what's missing, why it matters, and the severity if unaddressed.

---

## 1. Workflow Crash Recovery

**What we have:** Desktop crash recovery (R12.5), session save on close (R12.4), session resume (R11.5).

**What we don't have:** If a workflow is mid-execution and reasonix crashes/restarts — 3 of 10 finders completed, 2 of their findings are flowing into verifiers — what happens on restart?

**Why it matters:** A $0.05 security audit that crashes at 80% completion is $0.04 of wasted tokens and zero useful output. Over a year of daily use, that's real money.

**Severity:** High. Workflows are long-running by design. A crash during a 30-subagent workflow is not an edge case — it's an expected failure mode.

**What needs design:**
- Workflow state persistence: save completed stage results to disk after each stage
- Resume from last completed stage on restart
- Subagent transcripts survive crash (they already do via SubagentStore)
- The orchestrator must detect "this workflow was in progress" on boot and offer resume
- Idempotency: re-running a completed stage produces identical results, or at minimum doesn't corrupt state

---

## 2. Model Fallback / Degradation

**What we have:** Model resolution (R4.20), provider retry on 429/5xx (R4.17).

**What we don't have:** If DeepSeek API is down for 30 minutes, what happens? Does the system automatically fail over to Anthropic? Does it warn about cost? What if Anthropic is ALSO down?

**Why it matters:** API outages happen. DeepSeek has had multi-hour outages. Without fallback, reasonix is a brick during those windows. With silent fallback to Anthropic, a user could unknowingly burn $5 instead of $0.05.

**Severity:** High. This is the difference between "reasonix is down, I'll use something else" and "reasonix handled it, I barely noticed."

**What needs design:**
- Provider health check on boot and periodically
- Fallback chain per role: "explorer → Flash, if Flash down → Pro (with warning), if Pro down → Anthropic Haiku (with cost warning)"
- Fallback must be explicit: "DeepSeek Flash is down. Falling back to DeepSeek Pro. This will cost ~3× more. Continue? [y/N]"
- Never silently cross providers without user consent
- Circuit breaker: if a provider returns 5xx for >N consecutive requests, stop trying it for M minutes

---

## 3. Rate Limit Backpressure

**What we have:** Provider retry on 429 (R4.17).

**What we don't have:** A workflow spawns 27 verifiers simultaneously. DeepSeek rate limit is, say, 10 requests/second. What happens? Does the workflow queue the excess? Does it crash? Do the verifiers all retry in lockstep, creating a thundering herd?

**Why it matters:** 27 parallel subagents hitting the same API endpoint is a self-DOS. Without backpressure, you get: 10 succeed, 17 get 429, 17 retry simultaneously, 10 succeed, 7 get 429, 7 retry... each retry burns tokens on retried requests.

**Severity:** Medium. The workflow completes eventually, but slowly and wastefully.

**What needs design:**
- Provider-level rate limit awareness: the provider client tracks remaining quota from response headers
- Workflow spawn limiter: don't fire 27 simultaneous requests — batch them to the provider's known rate limit
- Jitter in retry backoff (the retry logic exists but check whether it jitters)
- The workflow tool should accept a `max_concurrent` per stage
- DeepSeek rate limit headers should be parsed and exposed in the background panel

---

## 4. Session Portability / Export

**What we have:** Session persistence to disk, session resume.

**What we don't have:** Export workflow results as a Markdown report. Share a session with another developer. Import a session from another machine.

**Why it matters:** The output of a security audit is valuable. A user who spends $0.05 finding 3 real bugs wants to paste those findings into a PR, share them with a teammate, or archive them for compliance.

**Severity:** Medium. Doesn't block functionality, but limits the value of the output.

**What needs design:**
- `/export` command: exports current session as Markdown with findings, file references, and timestamps
- `/export --format json` for machine consumption
- Session sharing: export includes enough metadata for another reasonix instance to import and continue
- Workflow output is already structured (report_findings) — export should be one command

---

## 5. Subagent Debugging / Deep Trace

**What we have:** Background panel shows reasoning tail and tool count. Peek panel shows last 20 lines of output.

**What we don't have:** When a verifier subagent returns REFUTED for a claim that's actually true, why did it make that mistake? The user needs to see the verifier's full reasoning, all its tool calls, and what evidence it saw.

**Why it matters:** False refutations are expensive — they drop real bugs from the report. Without debugging, the user either trusts the system (and misses bugs) or manually re-verifies everything (defeating the purpose).

**Severity:** Medium. Power users will demand this.

**What needs design:**
- `--debug` mode shows full subagent transcripts, not just tails
- Per-subagent "Inspect" view: full message log, every tool call with input/output, reasoning blocks
- "Why did you flag this?" — clicking a finding shows the verifier's exact reasoning chain
- Subagent transcript search: grep across all subagent transcripts for a keyword
- Subagent diff: compare two verifier transcripts to understand divergent conclusions

---

## 6. Security Model for Remote Workers

**What we have:** Auth token for work queue (Issue #12). Docker container isolation (Issue #13). API keys via env vars (Issue #10).

**What we don't have:**
- **Worker impersonation:** What stops a compromised machine from registering as "build-server" and claiming work items?
- **Work item interception:** What stops a man-in-the-middle from reading work item contents (which include the prompt and potentially sensitive code context)?
- **Token scoping:** The remote worker gets API keys to call DeepSeek. What stops a compromised worker from using those keys for non-reasonix purposes (crypto mining via the API, data exfiltration)?
- **Replay attacks:** Can an attacker replay a captured work-item-claim request?

**Why it matters:** Remote workers run on infrastructure the user doesn't fully control. A compromised build server shouldn't be able to impersonate, intercept, or exfiltrate.

**Severity:** High for anyone using remote workers in production.

**What needs design:**
- Workers authenticate with unique per-worker tokens, not a shared token
- Work queue uses TLS with certificate pinning
- API keys on the worker are scoped: the worker can only call the specific models needed for its assigned subagents, with spending limits
- Work item payloads are encrypted with the worker's public key (so only the intended worker can decrypt)
- Work item claims are signed and timestamped to prevent replay

---

## 7. Configuration Migration Path

**What we have:** Config loading with merge, v0.x → v1.0 migration.

**What we don't have:** If a user has a carefully tuned v1.0 `reasonix.toml` and installs v1.1 with all these new features, does their config still work? Are new config sections added with sensible defaults? Are deprecated fields handled?

**Why it matters:** Breaking a user's config on upgrade is a fast way to lose users. The config surface is growing significantly: `[[remotes]]`, `[agent.subagent_models]`, `[agent.subagent_efforts]`, workflow settings, schedule settings, notification push settings.

**Severity:** Medium. Backward compatibility is a product quality signal.

**What needs design:**
- Every new config section has a `config_version` bump and a migration function
- Unknown fields in config should warn, not error (forward compatibility)
- `reasonix doctor` should detect config that references removed/renamed fields
- A `--dry-run` mode that validates config without starting the agent

---

## 8. Telemetry / Observability of the System Itself

**What we have:** Desktop telemetry (anonymous launch ping), cost tracking, token tracking.

**What we don't have:** System-level metrics. What's the average workflow success rate? Which subagent role fails most often? What's the p50/p95/p99 latency for a finder subagent? How often does the cognitive loop detector fire? What's the actual cache hit rate in production across all users?

**Why it matters:** Without telemetry, every improvement is guesswork. "We think the cognitive loop detector helps" vs "The loop detector fires on 3% of turns and prevents $0.12 of waste per fire — $4.30 saved per user per month."

**Severity:** Low for MVP, high for sustained improvement.

**What needs design:**
- Opt-in telemetry for subagent metrics (explicit consent, not default)
- Metrics: workflow spawn count, stage success rate, role-specific latency, loop detector fires, cache hit rate, cost per workflow type
- Zero content telemetry — counts and durations only, never prompts or outputs
- Local dashboard: `/metrics` command shows your personal stats without sending anywhere
- Privacy: all telemetry is aggregated and anonymized before leaving the machine

---

## 9. Accessibility

**What we have:** High contrast mode, reduced motion option, dual color+icon+text encoding.

**What we don't have:**
- Screen reader support for the desktop app (ARIA labels, focus management, semantic HTML)
- Screen reader support for the TUI (terminal screen readers need specific ANSI patterns)
- Keyboard-only navigation for EVERY interaction (not just the main ones)
- Focus indicators visible at all times
- Configurable text size / zoom
- Audio cues as an alternative to visual status changes

**Why it matters:** Accessibility is not optional for a professional tool. Developers with visual impairments, motor disabilities, or neurodivergence need to use this tool too.

**Severity:** Medium. Legal requirement in some jurisdictions, ethical requirement everywhere.

**What needs design:**
- Desktop: ARIA labels on every interactive element, focus trap in modals, skip links, semantic heading hierarchy
- TUI: compatibility with terminal screen readers (speak status changes, read tool output on demand)
- Keyboard audit: every mouse action has a keyboard equivalent, documented
- `accessibility` config section: screen_reader_mode, audio_cues, focus_style, text_scale

---

## 10. Mobile / Off-Device Interaction

**What we have:** Desktop notifications, proactive push notifications via webhook.

**What we don't have:** Approve a tool from your phone. Check workflow status from a mobile browser. Receive a push notification when a 30-minute workflow completes.

**Why it matters:** Remote workflows run on build servers. The user spawns a $0.05 security audit, walks away from their desk, and wants to know when it's done — or approve a high-risk tool call that came up mid-audit.

**Severity:** Low for MVP, medium for production deployment.

**What needs design:**
- The `reasonix serve` HTTP server already exists — a lightweight mobile web view showing active workflows + approve/deny
- Push notifications via webhook to mobile (Firebase, APNs, or a simple webhook the user configures)
- The `send_notification` tool already exists — it just needs a mobile delivery channel

---

## 11. Plugin API for New Capabilities

**What we have:** MCP tool integration (plugin stdio + HTTP transports), skill system.

**What we don't have:** Can a plugin provide a subagent role? Can a plugin provide a workflow stage type? Can a plugin register a new sandbox engine?

**Why it matters:** The architecture is config-driven and plugin-driven by design (Spec §1). If workflow stages, verifier roles, and sandbox engines are hardcoded, the architecture betrays its own principles.

**Severity:** Low for MVP. The built-in roles and stages cover the 80% case.

**What needs design:**
- Subagent role registration: a plugin can provide a "custom-role" with its own system prompt and tool scope
- Workflow stage types: a plugin can register a stage handler (e.g., a custom verification strategy)
- Sandbox engine registration: like `provider.Register("openai", ...)`, a `sandbox.Register("docker", ...)` or `sandbox.Register("podman", ...)`
- These are v2 concerns, but the interfaces should be designed now so the internal code doesn't hardcode strings

---

## 12. Testing Strategy for the Full System

**What we have:** 651 Gherkin scenarios, `go test -race` clean, contract tests, concurrency tests.

**What we don't have:**
- Mock provider that simulates DeepSeek behavior (including reasoning_content, cognitive loops, 429s)
- Mock provider that simulates Anthropic behavior (thinking blocks, refusals, cache hits)
- Deterministic workflow replay tests (same seed → same output)
- Chaos testing (kill a worker mid-execution, kill a subagent, partition the network)
- Performance benchmarks in CI (not just correctness)
- What runs on every commit vs every PR vs nightly?

**Why it matters:** 651 scenarios against real APIs costs real money and takes real time. Without mock providers, the test suite is either expensive (real API calls) or incomplete (no provider behavior coverage).

**Severity:** High for development velocity. Every change needs confidence it doesn't break DeepSeek-specific behavior.

**What needs design:**
- `internal/agent/testutil/mock_provider.go` already exists — extend it to simulate DeepSeek-specific behaviors
- Mock provider modes: "normal", "looping" (simulates cognitive loop), "rate-limited" (simulates 429s), "slow" (simulates high latency)
- Test tiers: `go test -short` (unit tests, no network), `go test ./...` (integration, mock providers), `go test -tags=live` (real API calls, nightly only)
- Workflow replay: record a real workflow's tool calls, replay them deterministically in test
- CI pipeline: short tests on every commit, full tests on PR, live tests nightly

---

## 13. Internationalization of New Features

**What we have:** en/zh/zh-tw for existing UI surface.

**What we don't have:** All new error messages, tool descriptions, skill bodies, quality prompt blocks, effort calibration tables, and workflow stage descriptions in zh/zh-tw. Every new string added in this architecture needs i18n.

**Why it matters:** The user base is significantly Chinese-speaking (README is bilingual, bot integrations are Chinese platforms). Shipping new features in English-only is a regression for the primary audience.

**Severity:** Medium. i18n is the kind of thing that's cheap to do as you go and expensive to retrofit.

**What needs design:**
- Every new user-facing string has an i18n key from day one
- Skill bodies (explore, review, verify, etc.) need zh translations
- Quality prompt block needs zh translation
- Effort calibration table needs zh rendering
- Tool descriptions for all new tools need zh entries in `messages_zh.go` and `messages_zh_tw.go`
- Regression test: R13.1-R13.3 should cover the new surface, not just the old

---

## 14. Graceful Degradation When Features Are Unconfigured

**What we have:** Defaults for most config values.

**What we don't have:** What does the experience look like if a user has NONE of the new features configured? No remotes. No subagent model overrides. No cron jobs. No custom skills beyond built-ins. Does the UI show empty states or hide irrelevant panels?

**Why it matters:** A new user installing reasonix for the first time should not see a complex UI full of "No X configured" empty states. The product should feel simple until the user opts into complexity.

**Severity:** Medium. First-run experience determines retention.

**What needs design:**
- Progressive disclosure in the UI: panels and features appear as they become relevant
- Empty states are helpful, not dead: "No remote workers configured. [Set one up]" not "Remotes: 0"
- The background panel auto-hides (already designed in Issue #7)
- Workflow tool only appears in the slash menu after the first subagent is spawned
- Feature discovery: after the 3rd manual `task` call, suggest: "💡 You've been spawning subagents manually. Try /workflow for multi-stage automation."

---

## 15. Documentation

**What we have:** 651 Gherkin scenarios, 16 issue specs, architecture docs. Zero user-facing documentation.

**What we don't have:** A user manual. API docs for plugin authors. Migration guide from v1.0. Example workflows. Screenshots of the new UI. A "Getting Started with Workflows" guide.

**Why it matters:** The best architecture in the world is useless if nobody knows how to use it.

**Severity:** High for adoption. Documentation is the product for developer tools.

**What needs design:**
- `docs/WORKFLOWS.md` — how to use the workflow tool, with 5 worked examples
- `docs/SUBAGENTS.md` — subagent roles, effort calibration, model tiering
- `docs/REMOTE.md` — setting up remote workers, security considerations
- `docs/MIGRATING.md` — migrating from v1.0, config changes
- In-app help: `/help workflows` shows workflow documentation in the TUI
- Each built-in skill has a `--help` mode: `/explore --help` shows the explorer's system prompt and tool scope

---

## Summary: Severity Ranking

> **All 15 items now have full BDD coverage.** Each maps to a dedicated `.feature` file with Gherkin scenarios.

| # | Unknown | Feature File | Scenarios | Severity | Action |
|---|---|---|---|---|---|
| 2 | Model fallback / degradation | `model-fallback.feature` | 18 | **Critical** | Design before Phase 1 ships |
| 6 | Remote worker security | `remote-worker-security.feature` | 17 | **Critical** | Design before Phase 5 ships |
| 1 | Workflow crash recovery | `workflow-crash-recovery.feature` | 23 | **High** | Design as part of Issue #5 |
| 12 | Testing strategy | `testing-strategy.feature` | 17 | **High** | Mock provider before any code |
| 15 | Documentation | `documentation.feature` | 15 | **High** | Write alongside implementation |
| 3 | Rate limit backpressure | `rate-limit-backpressure.feature` | 12 | **Medium** | Design as part of Issue #5 |
| 5 | Subagent debugging | `subagent-debugging.feature` | 12 | **Medium** | Add to Issue #7 (panel) |
| 7 | Config migration | `config-migration.feature` | 14 | **Medium** | Add to regression matrix |
| 13 | i18n of new features | `internationalization.feature` | 11 | **Medium** | Strings from day one |
| 14 | Graceful degradation | `graceful-degradation.feature` | 12 | **Medium** | Design empty states |
| 4 | Session portability | `session-portability.feature` | 16 | **Low** | Post-MVP |
| 8 | Telemetry / observability | `observability.feature` | 15 | **Low** | Post-MVP |
| 9 | Accessibility | `accessibility.feature` | 16 | **Low** | Incremental |
| 10 | Mobile / off-device | `mobile-interaction.feature` | 9 | **Low** | Post-MVP |
| 11 | Plugin API | `plugin-extensibility.feature` | 11 | **Low** | v2 |

**15 items → 15 feature files → 218 new scenarios. Full BDD/TDD loop closed.**
