# Definition of Done

Every deliverable must satisfy all three tiers. No exceptions.

---

## Tier 1: Supported (Functional Correctness)

- [ ] All Gherkin scenarios pass (`.reasonix/features/*.feature`)
- [ ] Provider coverage: DeepSeek V4-Flash, DeepSeek V4-Pro, Anthropic Claude Opus 4.8
- [ ] Platform coverage: macOS (Seatbelt), Linux (bubblewrap/Docker), Windows (unconfined fallback)
- [ ] Agent type coverage: single-model Agent, two-model Coordinator
- [ ] `go test -race ./internal/agent/... ./internal/jobs/... ./internal/worker/...` is clean
- [ ] `go test -count=1 ./internal/...` is green
- [ ] Every error path has a test (tool returns error → agent feeds back → model self-corrects)
- [ ] Every cancellation path has a test (context cancelled → cleanup → no goroutine leak)
- [ ] Every API contract has a test (see `features/contract-tests.feature`)

---

## Tier 2: Bug-Free (Reliability)

- [ ] 24-hour stress test: spawn 1000 subagents, zero crashes, zero goroutine leaks, zero orphaned Docker containers
- [ ] Memory: steady-state RSS < 2× baseline after 100 sequential workflow runs
- [ ] `runtime.ReadMemStats()` shows zero monotonic heap growth across 50 workflow runs
- [ ] Network partition: remote worker offline → parent subagent creation fails with clear error within 5s, no hang
- [ ] Remote worker SIGKILL: all containers cleaned up, work items marked failed, no queue corruption
- [ ] Disk: subagent transcripts compacted, orphaned metadata cleaned, session dir never exceeds 1GB
- [ ] DeepSeek loop detector integration: false positive rate < 2% on legitimate multi-turn reasoning
- [ ] Storm breaker integration: catches every (tool, error) repetition loop within 5 iterations
- [ ] Concurrent workflow spawn: 10 parallel `workflow` calls with 10 subagents each = 100 goroutines, no deadlock
- [ ] `send_to_subagent` delivery: message always arrives before subagent's next tool-call round (no ordering bug)
- [ ] Replay determinism: same workflow spec run twice produces structurally identical event trees

---

## Tier 3: Delightful (User Experience)

### Background Tasks Panel
- [ ] Panel renders in < 16ms (single frame budget at 60fps)
- [ ] First render after subagent spawn: panel appears within 200ms (not next polling interval)
- [ ] Tool count updates live — user sees counter increment within 500ms of tool call
- [ ] Reasoning tail updates live — user sees last 200 chars of subagent thinking streamed
- [ ] Completed tasks collapse with a 300ms ease-out animation (not jarring instant disappear)
- [ ] `s` key opens a mini input bar at panel bottom with placeholder: "Message to <subagent-name>..."
- [ ] Sending empty message: input bar flashes red briefly, does not dispatch
- [ ] `Enter` on completed task shows final result inline (no modal, just expands in place)
- [ ] Panel height auto-sizes: 1 task = 3 lines, 5 tasks = 7 lines, 15 tasks = viewport-capped at 12 lines + scroll
- [ ] Status line badge pulses briefly (200ms) when a new background task starts
- [ ] Remote indicator 🌐 has a tooltip on hover/select: "Running on build-server (Ubuntu 24.04, 4/8 CPUs)"
- [ ] Failed tasks show error in red, with `Enter` to expand full traceback

### Workflow Visualization
- [ ] Running workflow shows stage progress bar: `find [=====>    ] 2/3 · verify [=>        ] 1/9 · report [pending]`
- [ ] Pipeline mode: items visibly flow through stages — user sees "auth.go" appear in verify while "middleware.go" is still in find
- [ ] Barrier mode: clear "Waiting for: middleware.go (find)" message while blocked
- [ ] Loop-until-dry: shows round counter: "Round 3 — 2 new findings · 0 dry rounds"
- [ ] Voting result: shows vote breakdown inline: ✓✓✗ (2/3 confirmed)
- [ ] Cancelled workflow: graceful "Workflow cancelled — 4 completed, 3 skipped" summary

### Remote Worker UX
- [ ] Bootstrap script prints progress with spinners: `⠋ Detecting OS...`, `⠙ Installing Docker...`, `✔ Docker ready`
- [ ] Bootstrap script shows resource summary before starting: "This machine: 8 CPUs, 32GB RAM, 200GB disk. Max containers: 6."
- [ ] Bootstrap script validates API keys by making one cheap API call before writing config
- [ ] Settings panel: remotes table with columns: Name | Status | Load | OS | Uptime
- [ ] Status dot: 🟢 green pulse (online, idle), 🟡 yellow pulse (online, all containers busy), 🔴 red (offline), ⚪ gray (never connected)
- [ ] Offline remote: shows "Last seen: 2 hours ago" with reconnect countdown
- [ ] `reasonix remote test <name>` command: sends a ping subagent, verifies round-trip, reports latency

### Error UX
- [ ] Every user-facing error has: what happened, why, what to do next
- [ ] Example: "Subagent 'auth-review' failed: DeepSeek API returned 429 (rate limited). Retrying in 30s."
- [ ] Example: "Remote 'build-server' unreachable: connection refused. Check that the worker is running: `ssh build-server systemctl status reasonix-worker`"
- [ ] Never show raw Go stack traces to the user — only in `--debug` mode
- [ ] Spinner never freezes — if a subagent is stuck, show "⚠ stalled (2m 15s since last activity)" instead

### Tokenomic Delight
- [ ] Status line shows session cost: "$0.03 this session · $0.47 today"
- [ ] Background panel shows per-subagent cost: "2.3K tokens ($0.002)"
- [ ] Hover on cost shows breakdown: "Input: 1.2K ($0.001) · Output: 0.8K ($0.001) · Cache hit: 0.3K ($0.000)"
- [ ] Cost changes from green ($) to yellow ($$) to red ($$$) as session total grows
- [ ] First run after config: show estimated cost for typical task: "A typical code review costs ~$0.03"

### Accessibility
- [ ] All status encoding is dual: color + icon + text (no color-only information)
- [ ] Panel works at 80-column terminal width — labels truncate, not wrap
- [ ] No animation for users with `prefers-reduced-motion` (detect via terminal, or config flag)
- [ ] High contrast mode: `theme_style = "high-contrast"` uses block characters instead of Unicode icons

### Keyboard
- [ ] `Ctrl+B` toggles background panel visibility
- [ ] `j`/`k` navigate subagent list
- [ ] `Enter` expand/collapse selected subagent
- [ ] `s` send message to selected subagent
- [ ] `k` kill selected subagent (with confirmation)
- [ ] `Ctrl+J`/`Ctrl+K` in send-message mode: submit / cancel
- [ ] `Esc` closes peek, cancels send, returns focus to chat input
- [ ] Tab order: chat input → background panel → status line → chat input

---

## Tier 4: Token-Efficient (Cost Architecture)

> Every commit must preserve or improve reasonix's token savings mechanisms. The ~99% cost advantage over pure-Anthropic workflows is the product's core architecture, not a side effect. This tier gates all changes.

### A. Cache Integrity (The Single Biggest Lever)

- [ ] **Prefix stability audit:** Every change that touches `internal/boot/boot.go` system prompt assembly must pass a byte-stability smoke test — the rendered prefix (tools → system → first message) before and after the change must be diffed. If the prefix changes by more than 10% of bytes: flag, explain why, confirm cache miss is acceptable.
- [ ] **No dynamic content in cacheable prefix:** `datetime.now()`, `uuid4()`, `os.Hostname()`, `os.Getenv("USER")`, and any other runtime-varying value is forbidden in the system prompt, tool definitions, or any content before the last `cache_control` breakpoint. Static-only.
- [ ] **Tool schema stability:** Adding, removing, or reordering tools invalidates the entire cache. Tool changes must be committed with a `CACHE: explain` comment in the commit body documenting the expected cache miss and its estimated cost.
- [ ] **Model switch guard:** Changing the default model invalidates all caches. A model ID change must not ship without a `CACHE: model switch` comment and a one-time cost estimate.
- [ ] **Cache hit rate regression test:** After any change touching provider or boot packages, run 3 identical requests and verify `cache_read_input_tokens > 0` on requests 2 and 3. Zero reads = silent cache invalidator in the codebase.
- [ ] **DeepSeek auto-cache exploitation:** Verify that `cache_read_input_tokens` or `prompt_cache_hit_tokens` appear in the usage response for DeepSeek providers after the second identical-prefix request. If the field is absent, auto-caching is not engaged.

### B. Model Tiering (Cheapest Model That Can Do the Job)

- [ ] **Effort-to-model mapping enforced:** The effort calibration table in the executor prompt must route: explore/read_only → V4-Flash (high), review/plan → V4-Flash (high), verify/security → V4-Pro (max), execute → V4-Flash or V4-Pro based on task complexity per the table.
- [ ] **No pro-for-flash routing:** A subagent whose role is "explore" or whose effort is "high" must never resolve to V4-Pro unless V4-Flash is unavailable AND the user is warned. The reverse is also enforced: verify/security_review must never silently downgrade to Flash.
- [ ] **Subagent count cap per workflow:** A single workflow must not spawn more than `max_total_tasks` subagents (default 50). Exceeding this cap produces an error, not silent execution. The cap is user-configurable but never unbounded.
- [ ] **Dedup-before-verify:** When a workflow stage produces findings, duplicate claims (same file, same line, same type) must be collapsed before spawning verification subagents. One finding = one set of verifier subagents, not N.
- [ ] **Orchestrator token budget awareness:** The parent agent's system prompt must include: "Prefer the cheapest model that can do each subagent's job. Flash for exploration and review. Pro only for adversarial verification, security review, and complex multi-file execution."
- [ ] **Orchestrator should never do subagent work inline:** If the orchestrator starts reading files or grepping instead of delegating to an explorer subagent, it's burning pro-tier tokens for flash-tier work. A notice must fire when the orchestrator makes >3 read-only tool calls in a turn without spawning a subagent.

### C. Token Waste Prevention

- [ ] **Silence default enforced:** The system prompt quality block must include the silence-default rules. Regression test: a subagent performing 3 sequential read_file calls emits zero narrative text between them.
- [ ] **Reasoning content stripped on non-tool turns:** DeepSeek `reasoning_content` must be present on tool-call turns (API requirement) and absent on final-answer turns (token waste). Test: verify request body does not contain `reasoning_content` for the last assistant message in a non-tool turn.
- [ ] **No reasoning re-upload to non-DeepSeek providers:** Anthropic and other non-DeepSeek provider requests must never carry `reasoning_content`. It's a DeepSeek-specific field; sending it to Anthropic wastes tokens or causes errors.
- [ ] **Tool output capped:** `maxToolOutputBytes` (32KB) is enforced on all tool results entering the model's context. Regression test: a 5MB read_file call is truncated, not passed through.
- [ ] **Compaction fires before context exhaustion:** A subagent's context must never hit 100%. Compaction must trigger at `compactRatio` (default 0.8) and force at `compactForceRatio` (default 0.9). Test: verify that a session with 64K window compacts before model sees >57.6K tokens.
- [ ] **Anti-narration gate:** The promise detector must flag "I'll now..." / "Let me check..." / "Looking at..." text between tool calls. These phrases waste tokens without producing value.
- [ ] **Anti-overengineering gate:** A change that touches >3 files when the task named ≤1 file must be flagged with a notice. The model must justify multi-file changes. Test: "Fix nil pointer in auth.go:42" must not touch middleware.go without explicit reason.

### D. DeepSeek-Specific Token Protections

- [ ] **Temperature 0.6 for subagents:** Every DeepSeek subagent must run at temperature ≥0.6 unless user explicitly overrode. Running at 0.0 causes infinite reasoning loops that burn max_tokens with no output. Regression test: verify subagent construction with DeepSeek provider + temp 0.0 → effective temp is 0.6.
- [ ] **Cognitive loop kills within budget:** When the cognitive loop detector fires, the breaker message must be injected before the subagent exhausts its remaining `max_tokens`. Test: a subagent with 4K max_tokens that loops must be broken by token ~3K, not after exhaustion.
- [ ] **Storm breaker kills within budget:** Same as above — the existing storm breaker must inject its breaker before the subagent exhausts `max_tokens`.
- [ ] **DeepSeek effort aliasing is explicit:** When the user configures `low` or `medium` effort, a notice must fire: "DeepSeek maps 'low' to 'high'. There are only two real levels: high and max." The mapping must not be silent.

### E. Measurable Regression Gates

- [ ] **Cache hit rate benchmark:** A standard "audit auth.go" task run 3 times must show cache hit rate ≥80% on requests 2 and 3. Measured as `cache_read_input_tokens / (cache_read_input_tokens + input_tokens)`. If <80%, the last change that touched the provider or boot package is the regression.
- [ ] **Token-per-task benchmark:** A standard "review auth.go for bugs" task must not exceed baseline token consumption by >20%. Baseline: 8K input, 2K output, 0.5K reasoning. Each commit touching agent, provider, or tool packages must re-baseline.
- [ ] **Cost-per-workflow ceiling:** A standard 3-finder + 9-verifier security audit workflow must cost ≤$0.10 at published DeepSeek rates. If >$0.10, investigate: are verifiers spawning needlessly? Is dedup failing? Is the orchestrator doing flash work on pro?
- [ ] **Subagent spawn efficiency:** A workflow with `max_total_tasks: 50` and 3 items across 2 stages must not spawn >3 (stage-1) + (3 × findings × 3 skeptics) subagents. Over-spawning is a regression. Test: count actual subagent executions vs expected from the spec.
- [ ] **No silent cache invalidation on config change:** Changing `default_model`, `effort`, `temperature`, or any provider entry must produce a log line: "Cache invalidated: <field> changed from <old> to <new>". If the line is absent but the prefix changed, it's a bug.

### F. UX Cost Transparency

- [ ] **Session cost in status line:** Always visible. Updates after every turn. Format: "$0.03 this session".
- [ ] **Per-subagent cost in background panel:** Running total for each subagent. Format: "2.3K tok ($0.002)".
- [ ] **Cost breakdown on hover/select:** Input tokens × input rate, output tokens × output rate, cache hit tokens × cache rate, cache miss tokens × input rate. Sum = total.
- [ ] **Cost color scaling:** Green (<$0.10 session), yellow ($0.10-$1.00), red (>$1.00). Thresholds configurable.
- [ ] **Daily cost tracker:** Persisted across sessions. Shown in status line as "$0.47 today". Resets at midnight local time.
- [ ] **Workflow cost estimate before execution:** When the model composes a workflow, the tool estimates total cost before spawning. If estimate >$0.50, prompt user to confirm. Format: "This workflow will spawn ~15 subagents. Estimated cost: $0.04-0.08. Proceed?"
- [ ] **First-run cost education:** On first agent turn after config, inject into system prompt: "Your configured models cost approximately: [pricing table]. A typical subagent costs $0.001-0.005. Choose the cheapest model adequate for each task."
- [ ] **Cost regression alert:** If a session's cost-per-turn exceeds the 7-day rolling average by >3×, a notice fires: "This session is running 3× above your average cost. Check: are subagents routing to Pro unnecessarily? Is caching engaged?"

### G. Pre-Commit Tokenomic Gate

- [ ] Every commit touching `internal/boot/`, `internal/provider/`, `internal/agent/`, or `internal/tool/` must answer these in the commit body:
  - `CACHE:` Will this change invalidate the prompt cache? If yes, what's the one-time cost?
  - `MODEL:` Does this change which model runs for any subagent role? If yes, what's the cost delta?
  - `TOKENS:` Does this change increase or decrease tokens per typical task? Estimate the delta.
  - `REGRESSION:` Did the token-per-task and cache-hit-rate benchmarks pass?
- [ ] A commit that answers `TOKENS: +15%, REGRESSION: skipped` is rejectable by code review. Token regressions require justification or a follow-up issue to fix.

---

## Tier 5: Regression-Free (Preserving the Existing Product)

> Every additive feature must pass the full regression matrix. No new feature ships with a known regression against existing functionality. The 104 tests in `REGRESSION_MATRIX.md` define the contract — any of them failing is a blocking bug.

### A. Architecture Invariants (Ship-Blocking)

- [ ] **R0.1** Single `CGO_ENABLED=0` static binary still builds and runs on all 6 targets
- [ ] **R0.2** BurntSushi/toml remains the only external dependency — `go list -m all | wc -l` unchanged
- [ ] **R0.3** Config resolution order (flag > project > user > default) unchanged
- [ ] **R0.4** No API key stored in config files — static analysis: zero occurrences of key material in `.toml` or `.json` config
- [ ] **R0.5** Zero monotonic heap growth across 50-iteration stress test — unchanged from baseline
- [ ] **R0.6** `go test -race ./...` is clean on every package

### B. CLI Surface (Release-Blocking)

- [ ] **R1.1** `reasonix` launches TUI, renders markdown, accepts input
- [ ] **R1.2** `reasonix run "prompt"` completes single-turn, non-zero exit on error
- [ ] **R1.3** `reasonix serve --addr :8080` starts HTTP+SSE, all auth modes work
- [ ] **R1.4** `reasonix complete` generates valid shell completion
- [ ] **R1.5** `reasonix resume` loads saved session, continues conversation
- [ ] **R1.6** `reasonix rewind` restores workspace to checkpoint
- [ ] **R1.7** `reasonix branch` creates branch, switches context
- [ ] **R1.8** `reasonix doctor` diagnostic report completes without panic
- [ ] **R1.9** `reasonix --model NAME` flag overrides config default_model

### C. TUI Components (Release-Blocking)

- [ ] **R2.1** Markdown renderer: headers, lists, code blocks, CJK text all render correctly
- [ ] **R2.2** Diff view: colored additions/removals with line numbers
- [ ] **R2.3** Model switcher: `/model` lists configured models, switches active
- [ ] **R2.4** Theme engine: dark/light/auto, all theme_styles render
- [ ] **R2.5** Status line: shows model, tokens, cost, context gauge — all populated with data
- [ ] **R2.6** Tool cards: bash, read_file, write_file, edit_file rendered with correct data
- [ ] **R2.7** Transcript view: scrollable, message ordering correct
- [ ] **R2.8** Chat input: multi-line, paste, @ file references all work
- [ ] **R2.9** Slash command picker: `/` opens menu, fuzzy matching works
- [ ] **R2.10** Resume picker: lists sessions, Enter resumes to correct session

### D. Agent Core (Release-Blocking)

- [ ] **R3.1** Agent loop: user prompt → tool calls → final answer end-to-end
- [ ] **R3.2** Plan mode: toggle bans writer tools, toggle restores — no cache invalidation
- [ ] **R3.3** Coordinator: planner proposes → executor carries out — two separate sessions
- [ ] **R3.4** Compaction: long session auto-compacts, context stays under window
- [ ] **R3.5** Storm breaker: 6 identical (tool, error) loops → breaker message injected
- [ ] **R3.6** Stream recovery: interrupted SSE → partial text preserved → retry succeeds
- [ ] **R3.7** Final-answer readiness: unfinished todo → blocked → nudge message injected
- [ ] **R3.8** Max steps guard: hit maxSteps → pause notice, not crash, session saveable
- [ ] **R3.9** Context tracking: usage → context gauge updates → compaction triggers at ratio
- [ ] **R3.10** Reasoning language: zh/en/auto respected in output

### E. Provider Surface (Release-Blocking)

- [ ] **R4.1** Anthropic: basic message, streaming, tool use, thinking all functional
- [ ] **R4.2** Anthropic: cache_control placement correct, cache hits verified
- [ ] **R4.3** Anthropic: signed thinking blocks round-tripped correctly
- [ ] **R4.4** Anthropic: image input (base64 PNG, URL) works
- [ ] **R4.5** Anthropic: stop_reason handling (end_turn, tool_use, max_tokens, refusal)
- [ ] **R4.9** OpenAI/DeepSeek: reasoning_content round-trip — stripped on no-tool turns, preserved on tool-call turns
- [ ] **R4.10** OpenAI/MiniMax: thinking.type=adaptive, no reasoning_effort
- [ ] **R4.11** OpenAI/generic: vanilla reasoning_effort scale, chat_url override
- [ ] **R4.13** OpenAI: model fetching via /models probe works
- [ ] **R4.16** All providers: count_tokens returns provider-specific count
- [ ] **R4.17** All providers: retry on 429/5xx with exponential backoff
- [ ] **R4.18** All providers: HTTP_PROXY/HTTPS_PROXY honored
- [ ] **R4.20** All providers: model resolution (provider/model, bare model, provider default)

### F. Built-in Tools (Fix-Before-Merge)

- [ ] **R5.1-R5.20** All 18 built-in tools: basic functionality regression-tested per `REGRESSION_MATRIX.md` Tier 5
- [ ] bash: foreground + background + bg_output + kill + wait work end-to-end
- [ ] Edit tools: edit_file multi_edit delete_range delete_symbol — all variant paths tested
- [ ] Search tools: glob grep ls code_index — pattern matching, filters, limits respected
- [ ] web_fetch: URL fetch, SSRF protection, proxy support
- [ ] todo_write + complete_step: task creation, status tracking, evidence verification

### G. Sandbox + Permissions (Fix-Before-Merge)

- [ ] **R6.1** macOS Seatbelt confinement active and enforced
- [ ] **R6.2** Linux bubblewrap confinement active and enforced
- [ ] **R6.4** Permission rules: allow/ask/deny produce correct decisions
- [ ] **R6.5** bash_readonly: plan-mode safe commands only
- [ ] **R6.6** Guardian: review request produces assessment event

### H. Hooks (Fix-Before-Merge)

- [ ] **R7.1-R7.11** All 11 hook event types fire at correct lifecycle points
- [ ] PreToolUse: exit 2 blocks tool, exit 0 allows
- [ ] PostLLMCall: reasoning translation hook applied correctly
- [ ] PreCompact: hook stdout injected as guidance

### I. Skills (Fix-Before-Merge)

- [ ] **R8.1-R8.7** All 7 built-in skills resolve and execute
- [ ] init: detects project type (go/ts/py/rs), writes correct AGENTS.md
- [ ] explore/research/review/security-review: subagent spawns with correct tool scope
- [ ] test: detects runner, runs, reports failures without altering config
- [ ] install-capability: plans before applying, riskLevel respected

### J. MCP + Memory + Checkpoint (Fix-Before-Merge)

- [ ] **R10.1-R10.4** MCP stdio and HTTP transports functional
- [ ] **R9.1-R9.2** Memory loaded into prompt, cache-stable
- [ ] **R9.3-R9.4** Checkpoint: snapshot saved, workspace rewound correctly

### K. Serve + Bot + Desktop (Fix-Before-Merge)

- [ ] **R11.1-R11.6** HTTP server + bot integrations functional
- [ ] **R12.1-R12.6** Desktop: multi-tab, workspace isolation, persistence, crash recovery

### L. Cross-Cutting (Fix-Before-Merge)

- [ ] **R13.1-R13.3** i18n: en/zh/zh-tw strings resolve correctly
- [ ] **R13.4-R13.6** Config: credential store, migration, .env loading
- [ ] **R13.7** Output styles: explanatory/learning/concise/custom apply correctly

### M. Regression Test Execution Rules

- [ ] **Full suite** runs on every PR touching `internal/` — gate blocks merge on any Tier 0-4 failure
- [ ] **Quick suite** (R0 + R3 + R4 + R5) runs on every commit — gate blocks commit on Tier 0 failure
- [ ] **Stress suite** (R0.5 + R0.6 + 100-task workflow) runs nightly — alerts on regression
- [ ] **Cross-platform suite** (R6.1-R6.3 + R12) runs on merge to main — macOS + Linux + Windows
- [ ] Every regression failure cites the specific R-ID and links to `REGRESSION_MATRIX.md`
- [ ] A PR that fixes a regression must add a regression test for that regression so it cannot recur
- [ ] The regression matrix itself is versioned — new features add rows, removed features deprecate rows
- [ ] A quarterly audit compares the matrix against the live codebase to detect undocumented features or dead entries
