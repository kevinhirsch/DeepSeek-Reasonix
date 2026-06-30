Feature: Tokenomic Efficiency
  As a developer who chose reasonix for its cost advantages
  I want every commit to preserve or improve token efficiency
  So that the ~99% cost advantage over pure-Anthropic workflows is never silently eroded

  Background:
    Given reasonix has DeepSeek V4-Flash ($0.14/M in) and V4-Pro ($0.435/M in) configured
    And the tokenomic gates are enforced by pre-commit checks and benchmark regressions

  # ── Cache Integrity ──────────────────────────────────────────────

  @token-cache-prefix-stability
  Scenario: System prompt change does not silently invalidate cache
    Given the system prompt is 8,000 tokens and was cache-stable across 50 previous turns
    When a developer adds a 200-token line to the quality block in internal/boot/boot.go
    Then the rendered prefix before and after the change is diff'd
    And the diff shows exactly the 200 added tokens and nothing else shifted
    And the commit message includes "CACHE: +200 tokens in system prompt — cache invalidated, one-time cost ~$0.008"

  @token-cache-prefix-stability
  Scenario: Dynamic content rejected from cacheable prefix
    Given a developer attempts to add `os.Hostname()` to the system prompt
    When the prefix stability audit runs
    Then the change is flagged: "dynamic content in cacheable prefix"
    And the commit is blocked until the dynamic content is moved after the last cache_control breakpoint

  @token-cache-prefix-stability
  Scenario: Tool reordering detected and documented
    Given a developer adds a new tool at position 0 in the tools array
    When the prefix stability audit runs
    Then the change is flagged: "tool ordering changed — all tool schemas after position 0 are cache-missed"
    And the commit must include "CACHE: tool reordering — full cache invalidation, estimated cost $0.XX"

  @token-cache-hit-rate
  Scenario: Cache hit rate regression test passes after provider change
    Given the provider package was modified
    When the cache hit rate benchmark runs (3 identical "review auth.go" requests)
    Then request 1 shows cache_creation_input_tokens > 0 (cache written)
    And request 2 shows cache_read_input_tokens > 0 (cache read)
    And request 3 shows cache_read_input_tokens > 0 (cache read)
    And the hit rate on requests 2 and 3 is ≥ 80%

  @token-cache-hit-rate
  Scenario: Cache hit rate regression test fails loudly
    Given the provider package was modified
    When the cache hit rate benchmark runs
    And request 2 shows cache_read_input_tokens == 0
    Then the benchmark fails with: "CACHE REGRESSION: zero cache reads on second request. Silent invalidator present."
    And the failure names the file most recently changed in the provider package

  @token-cache-deepseek-auto
  Scenario: DeepSeek auto-cache is engaged
    Given the provider is DeepSeek V4-Flash
    When 2 identical "review auth.go" requests are sent
    Then the second request shows either cache_read_input_tokens > 0 OR prompt_cache_hit_tokens > 0
    And the total prompt tokens on the second request are ≤10% of the first request

  # ── Model Tiering ────────────────────────────────────────────────

  @token-tier-explorer-flash
  Scenario: Explorer subagent routes to Flash by default
    When a workflow spawns an explorer subagent with role "explore"
    Then the resolved model is V4-Flash
    And the effort is "high"
    And the subagent does NOT run on V4-Pro

  @token-tier-verifier-pro
  Scenario: Verifier subagent routes to Pro by default
    When a workflow spawns a verify subagent with role "verify"
    Then the resolved model is V4-Pro
    And the effort is "max"
    And the subagent does NOT run on V4-Flash

  @token-tier-fallback
  Scenario: Explorer warns if Pro is the only available model
    Given V4-Flash is unavailable (API key not configured)
    When a workflow spawns an explorer subagent
    Then a warning is emitted: "Explorer subagent falling back to V4-Pro — Flash is unavailable. This will cost ~3× more."
    And the subagent runs on V4-Pro

  @token-tier-dedup
  Scenario: Duplicate findings collapsed before verification
    Given a finder stage produced 10 findings
    And 3 findings reference the same file and line (duplicate claims from different finders)
    When the dedup-before-verify step runs
    Then 8 unique findings proceed to verification
    And 24 verifier subagents are spawned (8 × 3 skeptics)
    And NOT 30 verifier subagents (which would verify duplicates)

  @token-tier-orchestrator-delegation
  Scenario: Orchestrator warned when doing flash-tier work inline
    Given the orchestrator is running on V4-Pro
    When the orchestrator makes 4 read-only tool calls without spawning a subagent
    Then a notice is emitted: "You're burning pro-tier tokens on flash-tier work. Delegate exploration to a subagent."
    And on the 5th read-only call, the notice escalates to a warning

  @token-tier-orchestrator-delegation
  Scenario: Orchestrator legitimately uses read-only tools
    Given the orchestrator needs to verify a small claim about a single file
    When the orchestrator makes 2 read_file calls and produces an answer
    Then no warning is emitted (below the 3-call threshold)

  @token-tier-cost-estimate
  Scenario: Workflow cost estimated before execution
    Given a workflow would spawn 3 finders (Flash), 9 verifiers (Pro), and 1 synthesizer (Flash)
    When the workflow tool computes the estimate
    Then the estimate is $0.04-0.08
    And if the estimate exceeds $0.50, the user is prompted to confirm

  # ── Token Waste Prevention ────────────────────────────────────────

  @token-waste-silence
  Scenario: Subagent emits zero narrative between sequential reads
    Given a subagent is reading auth.go, middleware.go, and handlers.go in sequence
    When all 3 read_file calls complete
    Then the assistant messages between the reads contain ONLY tool_calls
    And zero text content is emitted
    And the total output tokens for this phase are from tool results, not narration

  @token-waste-reasoning-strip
  Scenario: DeepSeek reasoning_content stripped on final-answer turn
    Given a DeepSeek subagent has produced a final answer with no tool calls
    And the assistant turn has reasoning_content: "chain of thought..."
    When the next request is built (if conversation continues)
    Then the request body does NOT contain reasoning_content for that assistant message
    And the field is absent, not empty-string

  @token-waste-reasoning-preserve
  Scenario: DeepSeek reasoning_content preserved on tool-call turn
    Given a DeepSeek subagent made a tool call
    And the assistant turn has reasoning_content: "I need to check..."
    When the next request is built
    Then the request body DOES contain reasoning_content for that assistant message
    And the reasoning_content value matches the original

  @token-waste-reasoning-non-deepseek
  Scenario: reasoning_content never sent to non-DeepSeek providers
    Given the provider is Anthropic Claude
    And the session contains assistant turns with ReasoningContent set
    When the provider builds the API request
    Then no reasoning_content field appears anywhere in the request body

  @token-waste-tool-output-cap
  Scenario: Large tool output truncated before entering context
    Given a subagent reads a 5MB log file
    When the tool output is processed
    Then only the first 32KB (maxToolOutputBytes) enters the model's context
    And a truncation notice is appended: "[truncated — 5MB total, showing first 32KB]"

  @token-waste-compaction
  Scenario: Compaction fires before context exhaustion
    Given a subagent session has a 64K token window
    And the session has reached 52K tokens (81% of window)
    When the next turn is about to start
    Then auto-compaction fires
    And the compacted context is ≤32K tokens (50% of window)
    And the subagent continues without context exhaustion

  @token-waste-compaction
  Scenario: Force compaction fires at high-water mark
    Given a subagent session has reached 58K tokens (90% of window)
    When the next turn starts
    Then force compaction fires immediately
    And the compacted context is ≤32K tokens
    And a notice is emitted: "Force compaction triggered at 90% context usage"

  @token-waste-narration
  Scenario: "I'll now" narration flagged between tool calls
    Given a subagent emits "I'll now read the configuration file" between tool calls
    When the anti-narration check runs
    Then a notice is fired: "Narration detected — 'I'll now' wastes tokens without producing value"
    And the message is stripped before the next model request

  @token-waste-overengineering
  Scenario: Multi-file change beyond task scope flagged
    Given the task is "Fix nil pointer in auth.go:42"
    And the agent has modified auth.go, middleware.go, handlers.go, and types.go
    When the anti-overengineering gate runs
    Then the 3 files beyond auth.go are flagged with a notice
    And the agent is asked to justify: "Task requested fix in auth.go only. 3 additional files changed. Justify or revert."

  # ── DeepSeek-Specific Token Protections ───────────────────────────

  @token-deepseek-temp
  Scenario: DeepSeek subagent temperature prevents loop waste
    Given a DeepSeek subagent is configured with temperature 0.0
    When the subagent is constructed
    Then effective temperature becomes 0.6
    And a one-time info notice fires
    And the subagent does not enter an infinite reasoning loop that exhausts max_tokens

  @token-deepseek-loop
  Scenario: Cognitive loop killed before token exhaustion
    Given a DeepSeek subagent with max_tokens: 4096
    And the subagent has entered a cognitive loop burning reasoning tokens
    When the cognitive loop detector fires at token ~3000
    Then the breaker message is injected immediately (not after the turn completes)
    And the subagent either breaks the loop within 500 additional tokens or is failed
    And the subagent does NOT consume all 4096 tokens in loop

  @token-deepseek-storm
  Scenario: Storm breaker killed before token exhaustion
    Given a subagent has made 5 identical failing tool calls consuming 3500 tokens
    When the storm breaker fires on call 6
    Then the breaker message is injected
    And if the model still fails, the subagent is terminated before exhausting max_tokens

  @token-deepseek-effort-alias
  Scenario: DeepSeek effort aliasing is explicit
    Given a user configured effort: "low" for a DeepSeek provider
    When the provider validates the effort
    Then an info notice fires: "DeepSeek maps 'low' to 'high'. Only two real levels: high and max."
    And the effective effort is "high"

  # ── Measurable Regression Gates ───────────────────────────────────

  @token-benchmark-hit-rate
  Scenario: Cache hit rate benchmark gates provider changes
    Given the "audit auth.go" benchmark task
    When run 3 times with identical prefix after a provider change
    Then the hit rate on runs 2 and 3 is ≥ 80%
    And the benchmark result is recorded in the commit message under "REGRESSION: pass"

  @token-benchmark-hit-rate
  Scenario: Cache hit rate below threshold blocks merge
    Given the "audit auth.go" benchmark task
    When run 3 times and the hit rate on run 2 is 35%
    Then the CI pipeline fails with: "CACHE REGRESSION: hit rate 35% (threshold 80%)"
    And the failure message names the files most likely responsible

  @token-benchmark-tokens-per-task
  Scenario: Token-per-task benchmark gates agent changes
    Given the baseline "review auth.go" task consumes 8K input + 2K output + 0.5K reasoning
    When the task is run after an agent package change
    Then token consumption is within 20% of baseline
    And the result is recorded as "TOKENS: +5% (within threshold)"

  @token-benchmark-tokens-per-task
  Scenario: Token-per-task regression caught
    Given the "review auth.go" task baseline is 10.5K tokens
    When the task consumes 15K tokens after a change (43% increase)
    Then the benchmark fails with: "TOKEN REGRESSION: +43% (threshold 20%)"
    And the commit is flagged for review

  @token-benchmark-cost-ceiling
  Scenario: Security audit cost ceiling enforced
    Given the standard 3-finder + 9-verifier + 1-synthesizer security audit workflow
    When the workflow completes
    Then total cost is ≤ $0.10 at published DeepSeek rates
    And if >$0.10, the result includes a breakdown showing which stage exceeded expectations

  @token-benchmark-spawn-efficiency
  Scenario: Workflow spawn count matches expected
    Given a workflow spec with 3 items, 2 stages, and fan_out: 3 for stage 2
    And stage 1 produced 5 findings (after dedup)
    When the workflow completes
    Then at most 3 + (5 × 3) + 1 = 19 subagents were spawned
    And the spawn count is logged as "SPAWN: 19 (expected ≤19)"

  @token-benchmark-cache-invalidation-log
  Scenario: Config change that invalidates cache produces log line
    Given the user changes default_model from "deepseek-v4-flash" to "deepseek-v4-pro"
    When the config is loaded
    Then a log line fires: "Cache invalidated: default_model changed from deepseek-v4-flash to deepseek-v4-pro"
    And the log level is INFO

  @token-benchmark-cache-invalidation-log
  Scenario: Config change that does NOT invalidate cache produces no log
    Given the user changes ui.theme from "dark" to "light"
    When the config is loaded
    Then no cache invalidation log line fires

  # ── Pre-Commit Gate ───────────────────────────────────────────────

  @token-precommit
  Scenario: Commit to boot package requires cache/model/token/regression fields
    Given a developer commits a change to internal/boot/boot.go
    When the pre-commit hook runs
    And the commit message does not contain "CACHE:", "MODEL:", "TOKENS:", and "REGRESSION:"
    Then the commit is rejected
    And the rejection message explains: "Commits to boot/ must include CACHE:, MODEL:, TOKENS:, and REGRESSION: fields"

  @token-precommit
  Scenario: Commit to non-token-sensitive package skips gate
    Given a developer commits a change to internal/cli/theme.go (UI-only)
    When the pre-commit hook runs
    Then the tokenomic gate is skipped
    And the commit is not rejected for missing CACHE:/MODEL:/TOKENS:/REGRESSION: fields

  @token-precommit
  Scenario: Token regression with justification is accepted
    Given a developer commits a change that increases token-per-task by 25%
    And the commit message includes "TOKENS: +25% — adding quality gate adds ~2K tokens to system prompt, which gates prevent ~8K of waste per turn"
    And the commit message includes "REGRESSION: pass — re-baselined, new baseline is 12.5K"
    Then the commit is accepted

  # ── FE: Tokenomic Rendering ──────────────────────────────────────

  @token-fe-cost-alert
  Scenario: Cost exceeds 3× average triggers alert in status line
    Given the 7-day rolling average cost is $0.05/session
    And the current session cost is now $0.18 (3.6× average)
    When the status line renders
    Then the cost indicator shows "⚠ $0.18" in yellow
    And clicking it shows: "3.6× your average. Check: are subagents routing to Pro unnecessarily? Is caching engaged?"
    And a "View cost breakdown" link opens the context panel

  @token-fe-tier-routing-alert
  Scenario: Model tier routing mismatch renders warning
    Given an explorer subagent is about to run on V4-Pro instead of the recommended V4-Flash
    When the spawn notice renders
    Then a yellow card shows: "⚠ Explorer routing to Pro ($0.435/M) instead of Flash ($0.14/M)"
    And the card shows estimated cost difference: "~$0.009 extra for this subagent"
    And a "Switch to Flash" button is shown if Flash is available
    And a "Continue with Pro" button accepts the overage

  @token-fe-cache-status
  Scenario: Cache status indicator in status line
    Given the current session has 85% cache hit rate
    When the status line renders
    Then the context gauge shows "▕████████░░▏ 85% hit"
    And the bar color is green (>80%)
    And hovering shows: "Cache: 120K hits / 22K misses · Estimated savings: $0.43 this session"
    Given the hit rate drops to 45%
    Then the bar color is yellow
    And hovering shows: "Cache degradation detected. Check: is the system prompt stable? Are tools being reordered?"

  @token-fe-precommit-failure
  Scenario: Pre-commit token gate failure shows actionable error
    Given a developer committed without CACHE: field
    When the pre-commit hook rejects the commit
    Then the error shows: "COMMIT REJECTED: Missing CACHE: field"
    And the error shows the required format: "CACHE: <impact description> · one-time cost estimate"
    And the error shows an example: "CACHE: +200 tokens in quality prompt — cache invalidated, one-time cost ~$0.008"
    And the commit is not created
