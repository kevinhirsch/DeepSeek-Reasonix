Feature: Background Task Monitoring
  As a developer using reasonix
  I want to see live subagent status, reasoning, and progress
  So that I know what background tasks are doing without attaching to each one

  Background:
    Given reasonix TUI is running with the background tasks panel integrated
    And the executor has spawned finders, verifiers, and a report subagent

  # ── Enriched Job View ─────────────────────────────────────────────

  @06-view-tool-count
  Scenario: Live tool count and last tool displayed
    Given a "review:auth" subagent is running
    When the subagent calls grep, then read_file, then grep again
    Then the panel shows "3 calls" and "last: grep"

  @06-view-reasoning
  Scenario: Live reasoning tail displayed
    Given a "verify:sql-injection" subagent is running
    And its most recent reasoning is "Checking the query builder at line 142... no, this path is protected by the ORM"
    Then the panel shows the last 200 characters of that reasoning

  @06-view-tokens
  Scenario: Token usage accumulated across subagent turns
    Given a "review:handlers" subagent has completed 3 turns
    And it consumed 1200 input and 800 output tokens
    Then the panel shows "2.0K tokens"

  @06-view-preview
  Scenario: Result preview shows first line of output
    Given a completed "review:handlers" subagent returned "3 bugs found: nil pointer in handler.go:42, race condition in cache.go:108, missing validation in input.go:15"
    Then the panel shows "3 bugs found: nil pointer in handler.go:42..."

  @06-view-empty-preview
  Scenario: Empty result shows no preview
    Given a completed subagent returned an empty string
    Then no result preview line is shown

  @06-view-truncated-preview
  Scenario: Very long first line is truncated
    Given a completed subagent's first line of output is 500 characters
    Then the preview shows the first 80 characters followed by "..."

  @06-view-dependency
  Scenario: Dependency graph shown for pending tasks
    Given a report subagent depends on verify:sql-injection and verify:auth-bypass
    And both verifiers are running
    When the panel renders
    Then the report row shows "depends on: verify:sql-injection, verify:auth-bypass"

  @06-view-model
  Scenario: Model and effort shown when different from default
    Given a verify subagent is running with model "deepseek-v4-pro" and effort "max"
    And the default subagent model is "deepseek-v4-flash" with effort "high"
    Then the panel shows the model and effort next to the subagent label

  # ── Panel rendering ───────────────────────────────────────────────

  @07-panel-render
  Scenario: Panel renders running, completed, and pending tasks
    Given subagents with states: running(auth-review), running(sql-verify), done(api-review), pending(report)
    When the background tasks panel renders
    Then "auth-review" shows ● green with live tool count and reasoning
    And "sql-verify" shows ● green with live tool count and reasoning
    And "api-review" shows ✓ green with token count and result preview
    And "report" shows ○ gray with dependency info

  @07-panel-render
  Scenario: Failed task shows error
    Given a subagent failed with "bash: command not found: nonexistent-tool"
    When the panel renders
    Then the subagent row shows ✗ red
    And the error message is shown instead of result preview

  @07-panel-render
  Scenario: Waiting task shows indicator
    Given a subagent is blocked on a tool approval
    When the panel renders
    Then the subagent row shows ◐ yellow
    And the status text is "waiting for approval"

  @07-panel-remote
  Scenario: Remote subagent shows 🌐 indicator
    Given a subagent is running on remote worker "build-server"
    When the panel renders
    Then the subagent row shows 🌐
    And hovering/selecting shows the remote name

  @07-panel-collapse
  Scenario: Completed tasks auto-collapse after delay
    Given a subagent has just completed
    When 2 seconds pass
    Then the subagent row collapses to: "✓ review:handlers [done] 8 calls 2.3K tokens"

  @07-panel-autohide
  Scenario: Panel auto-hides when no background tasks exist
    Given all background tasks have completed and their collapse delay has elapsed
    When the next render tick fires
    Then the background tasks panel is not rendered
    And the chat area reclaims the space

  @07-panel-statusline
  Scenario: Status line shows collapsed count
    Given 3 subagents are running and 2 have completed
    When the status line renders
    Then it shows "[3 running · 2 done]"

  @07-panel-statusline
  Scenario: Status line shows zero when idle
    Given no background tasks are active
    When the status line renders
    Then no background task count is shown

  @07-panel-statusline
  Scenario: Status line includes remote count
    Given 2 local and 1 remote subagents are running
    When the status line renders
    Then it shows "[2 local · 1 remote]"

  # ── Panel interactions ─────────────────────────────────────────────

  @07-panel-send
  Scenario: User sends message to running subagent via panel
    Given the panel is visible and "auth-review" is selected
    When the user presses 's' and types "Also check the rate limiter"
    Then a send_to_subagent call is dispatched with the message
    And the subagent receives it as a steer

  @07-panel-peek
  Scenario: User peeks at subagent output
    Given a running subagent "auth-review" is selected
    When the user presses Enter
    Then a peek panel slides in showing the last 20 lines of output
    And the panel shows the longest-running tool call with duration
    And pressing Escape closes the peek

  @07-panel-kill
  Scenario: User kills a running subagent
    Given a running subagent "auth-review" is selected
    When the user presses 'k'
    Then a confirmation prompt appears: "Kill 'auth-review'?"
    And on confirmation, kill_shell is dispatched for the subagent's job
    And the subagent row shows ✗ killed

  # ── Panel edge cases ──────────────────────────────────────────────

  @07-panel-many-tasks
  Scenario: Panel handles 20+ concurrent subagents
    Given 20 subagents are running simultaneously
    When the panel renders
    Then only the first 10 are shown (viewport height)
    And the header shows "(20 running · 0 done)"
    And scrolling (j/k) reveals additional subagents

  @07-panel-long-label
  Scenario: Very long subagent label is truncated
    Given a subagent has label: "comprehensive-security-audit-of-authentication-module"
    When the panel renders
    Then the label is shown as "comprehensive-security-audit-of-..."
    And hovering/selecting shows the full label

  @07-panel-unicode
  Scenario: Unicode in subagent output renders correctly
    Given a subagent's reasoning contains CJK characters: "正在检查认证模块"
    When the panel renders the reasoning tail
    Then the characters display correctly (not mojibake)
    And the truncation length is character-aware (not cutting mid-codepoint)

  # ── Effort calibration ─────────────────────────────────────────────

  @08-effort-explore
  Scenario: Model uses correct effort for explorer subagents
    When the executor spawns an explorer subagent
    Then the subagent runs at effort "high"
    And the model's reasoning (visible in the orchestrator turn) references the calibration table

  @08-effort-verify
  Scenario: Model uses max effort for verification
    When the executor spawns a verify subagent
    Then the subagent runs at effort "max"

  @08-effort-security
  Scenario: Model uses max effort for security review
    When the executor spawns a security_review subagent
    Then the subagent runs at effort "max"

  @08-effort-deepseek
  Scenario: DeepSeek shows only high and max in calibration table
    Given the resolved provider is DeepSeek
    When the effort calibration table is rendered in the system prompt
    Then it shows only "high" and "max" levels
    And a note explains: "DeepSeek reasoning_effort: low and medium → high, xhigh → max"

  @08-effort-anthropic
  Scenario: Anthropic shows full effort range
    Given the resolved provider is Anthropic
    When the effort calibration table is rendered
    Then it shows: low, medium, high, xhigh, max
    And each role has an appropriate level from the full range
