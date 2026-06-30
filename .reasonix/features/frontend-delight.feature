Feature: Frontend Delightfulness
  As a developer using reasonix daily
  I want the TUI to feel fast, responsive, and informative
  So that I enjoy using it and never wonder "what's happening right now"

  Background:
    Given reasonix TUI is running at 80 columns
    And the background tasks panel, status line, and chat are all visible

  # ── Perceived performance ─────────────────────────────────────────

  @fe-perf-panel-render
  Scenario: Panel renders in a single frame
    Given 5 background subagents are running
    When the panel renders
    Then the render completes in under 16ms
    And no screen flicker is visible
    And text in the chat area does not shift or jump

  @fe-perf-spawn-latency
  Scenario: Panel appears immediately when first subagent spawns
    Given the panel is hidden (no background tasks)
    When the first background subagent is spawned
    Then the panel appears within 200ms
    And the chat area smoothly shrinks to accommodate it
    And the status line count updates in the same frame

  @fe-perf-tool-count
  Scenario: Tool count updates without full panel redraw
    Given a subagent is showing "3 calls"
    When the subagent makes its 4th tool call
    Then only the tool count text updates (not the entire panel)
    And the update is visible within 500ms

  @fe-perf-reasoning-stream
  Scenario: Reasoning tail streams character by character
    Given a subagent is streaming reasoning "I need to check the authentication flow..."
    When characters arrive at 30ms intervals
    Then the reasoning tail in the panel updates at 30ms intervals
    And the text smoothly appends (no jarring full-string replacement)
    And the last 200 chars are always visible

  @fe-perf-scroll
  Scenario: Scrolling through 15 subagents is smooth
    Given 15 subagents are displayed in the panel
    When the user holds 'j' to scroll down
    Then each 'j' press moves the selection by one row
    And the scroll is instant (no debounce, no animation lag)

  # ── Animation ─────────────────────────────────────────────────────

  @fe-anim-collapse
  Scenario: Completed task collapses with ease-out
    Given a subagent has just completed
    When the collapse timer fires after 2 seconds
    Then the row smoothly shrinks from full height to 1 line
    And the animation duration is 300ms
    And the animation uses an ease-out curve (fast start, slow end)
    And rows below slide up to fill the gap

  @fe-anim-appear
  Scenario: New task appears with fade-in
    Given the panel is visible
    When a new subagent is spawned
    Then the row fades in over 150ms
    And existing rows slide down by one row height
    And the slide animation uses an ease-out curve

  @fe-anim-status-pulse
  Scenario: Status line badge pulses on new task
    Given the status line shows "[2 running · 1 done]"
    When a new subagent starts
    Then the badge "[3 running · 1 done]" pulses once (background color flashes)
    And the pulse duration is 200ms
    And after the pulse, the badge returns to its normal color

  @fe-anim-reduced-motion
  Scenario: Animations disabled for reduced motion preference
    Given the user has set `ui.reduced_motion = true` in config
    When any animation would play
    Then the state change happens instantly (0ms)
    And no easing curves are applied

  # ── Color and theming ─────────────────────────────────────────────

  @fe-theme-status-icons
  Scenario: Status icons are visible on dark themes
    Given the terminal uses a dark background
    When the panel renders
    Then ● (green) has sufficient contrast against dark background
    And ◐ (yellow) has sufficient contrast against dark background
    And ✓ (green) has sufficient contrast against dark background
    And ✗ (red) has sufficient contrast against dark background
    And ○ (gray) is visible but muted (not invisible)

  @fe-theme-status-icons
  Scenario: Status icons are visible on light themes
    Given the terminal uses a light background
    When the panel renders
    Then ● (green) has sufficient contrast against light background
    And text colors are darker/more saturated to maintain readability

  @fe-theme-high-contrast
  Scenario: High contrast mode uses ASCII fallbacks
    Given `theme_style = "high-contrast"`
    When the panel renders
    Then instead of ●, it shows [RUN]
    And instead of ✓, it shows [OK]
    And instead of ✗, it shows [ERR]
    And instead of ○, it shows [---]
    And instead of 🌐, it shows [REMOTE]
    And all text is high-contrast black/white only

  @fe-theme-token-cost
  Scenario: Cost color scales with session total
    Given the session has used $0.03 in tokens
    When the status line renders the cost
    Then "$0.03" is shown in green
    Given the session has used $0.50
    Then "$0.50" is shown in yellow
    Given the session has used $2.00
    Then "$2.00" is shown in red

  # ── Responsive layout ─────────────────────────────────────────────

  @fe-layout-narrow
  Scenario: Panel works at 80 columns
    Given the terminal is 80 columns wide
    When the panel renders with a subagent label "comprehensive-security-audit-of-authentication-module"
    Then the label is truncated to fit: "comprehensive-security-aud..."
    And no text wraps
    And no horizontal scroll is needed

  @fe-layout-ultra-narrow
  Scenario: Panel degrades gracefully at 60 columns
    Given the terminal is 60 columns wide
    When the panel renders
    Then the layout switches to single-line-per-task mode
    And each task shows only: icon + short label + status
    And tool count and reasoning tail are hidden
    And selecting a task expands it to show details

  @fe-layout-wide
  Scenario: Panel uses available width at 120 columns
    Given the terminal is 120 columns wide
    When the panel renders
    Then the reasoning tail shows 400 characters instead of 200
    And the model/effort column is always visible (not hidden behind "more")

  @fe-layout-auto-hide
  Scenario: Panel auto-hides and chat reclaims space smoothly
    Given the panel is visible
    When the last subagent completes and its collapse delay passes
    Then the panel disappears
    And the chat area expands to fill the reclaimed space
    And the expansion is instant (no animation — just relayout)
    And the chat scroll position is preserved

  # ── Send message UX ───────────────────────────────────────────────

  @fe-send-message
  Scenario: Send message input bar appears at panel bottom
    Given the panel is visible and "auth-review" is selected
    When the user presses 's'
    Then a mini input bar appears below the selected subagent row (or at panel bottom)
    And the placeholder text is "Message to auth-review..."
    And the chat input is temporarily disabled

  @fe-send-message
  Scenario: Send message with content
    Given the send message input bar is open with text "Also check the OAuth callback"
    When the user presses Ctrl+J
    Then the message is dispatched via send_to_subagent
    And the input bar closes
    And a brief confirmation shows: "✓ Sent to auth-review"
    And the confirmation fades after 2 seconds
    And chat input is re-enabled

  @fe-send-message
  Scenario: Cancel send message
    Given the send message input bar is open
    When the user presses Escape or Ctrl+K
    Then the input bar closes without dispatching
    And chat input is re-enabled

  @fe-send-message
  Scenario: Empty message rejected with visual feedback
    Given the send message input bar is open and empty
    When the user presses Ctrl+J
    Then the input bar border flashes red for 200ms
    And no message is dispatched
    And the input bar stays open

  # ── Peek panel UX ─────────────────────────────────────────────────

  @fe-peek
  Scenario: Peek shows subagent output inline
    Given "auth-review" is selected and has produced 30 lines of output
    When the user presses Enter
    Then the row expands in-place to show the last 20 lines of output
    And the expansion is instant (no animation)
    And a footer line shows: "Showing last 20 of 30 lines · Enter to collapse · s to send message"

  @fe-peek
  Scenario: Peek auto-scrolls with new output
    Given the peek panel is open on a running subagent
    When new output arrives
    Then the peek panel auto-scrolls to show the newest line
    And the footer updates: "Showing last 20 of 35 lines"

  @fe-peek
  Scenario: Closing peek restores collapsed row
    Given the peek panel is open
    When the user presses Enter again
    Then the row collapses back to its normal height
    And the scroll position in the panel is preserved

  # ── Information density ───────────────────────────────────────────

  @fe-density
  Scenario: Each subagent row shows exactly the right information
    Given a running subagent "verify:sql-injection" on remote "build-server"
    And it has made 5 calls, last was "read_file"
    And its reasoning tail is "Checking parameterization at line 89..."
    When the panel renders the row
    Then the row shows:
      - 🌐 (remote indicator)
      - ● (status icon — running)
      - "verify:sql-injection" (label)
      - "[running]"
      - "5 calls"
      - "last: read_file"
      - model: "v4-pro"
      - effort: "max"
      - cost: "$0.01"
      - reasoning: "Checking parameterization at line 89..."

  @fe-density
  Scenario: Information hierarchy is visually clear
    Given the panel is rendering
    Then the label is the most prominent text (bold or bright)
    And the status is second most prominent (colored icon)
    And metadata (calls, last tool, model) is normal weight
    And reasoning tail is dimmed/italic

  # ── Error state UX ────────────────────────────────────────────────

  @fe-error
  Scenario: Failed subagent shows actionable error
    Given a subagent failed with "DeepSeek API returned 429 (rate limited)"
    When the panel renders
    Then the error is shown in red
    And below the error: "Retrying in 30s..."
    And a countdown timer updates every second

  @fe-error
  Scenario: Subagent stuck shows stall warning
    Given a subagent has had no activity for 2 minutes
    When the panel renders
    Then the row shows ⚠ (warning icon)
    And the status text changes to "stalled (2m 15s)"
    And the elapsed time updates live

  @fe-error
  Scenario: Never show raw stack traces in normal mode
    Given a subagent panicked with a Go stack trace
    When the panel renders in normal (non-debug) mode
    Then only the panic message is shown: "Internal error: panic in subagent"
    And the stack trace is NOT shown
    And a hint is shown: "Run with --debug to see full trace"

  @fe-error
  Scenario: Debug mode shows full error details
    Given reasonix is running with --debug
    And a subagent failed with a Go stack trace
    When the user expands the failed subagent
    Then the full stack trace is visible
    And line numbers are clickable (if terminal supports OSC 8 hyperlinks)

  # ── First-run experience ──────────────────────────────────────────

  @fe-onboarding
  Scenario: First subagent spawn shows helpful hint
    Given the user has never spawned a background subagent before
    When the first background subagent starts
    Then a one-time notice appears: "💡 Background tasks run here. Press Ctrl+B to toggle this panel. Press s to send messages to running subagents."
    And the notice does not appear on subsequent spawns

  @fe-onboarding
  Scenario: Cost estimate shown before first subagent spawn
    Given the user has configured DeepSeek V4-Flash
    And no subagents have run yet
    When the user is about to spawn a code review subagent
    Then the system prompt includes a cost estimate: "A typical subagent costs ~$0.001-0.005. You can set max_total_cost in config."
