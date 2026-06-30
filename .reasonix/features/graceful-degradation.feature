Feature: Graceful Degradation & Progressive Disclosure
  As a new reasonix user
  I want the product to feel simple until I opt into complexity
  So that I'm not overwhelmed by features I haven't configured yet

  Background:
    Given reasonix has progressive disclosure in the UI
    And features appear as they become relevant

  # ── Empty States ──────────────────────────────────────────────────

  @graceful-empty-remotes
  Scenario: No remotes configured shows helpful empty state
    Given the user has no [[remotes]] in config
    When the user opens the remotes panel
    Then a message is shown: "No remote workers configured. Remote workers let you run subagents on build servers for resource-intensive tasks."
    And a "Set up a remote worker" button is shown
    And the button links to documentation

  @graceful-empty-workflows
  Scenario: No workflows running shows subtle indicator
    Given no workflows are active
    When the background panel renders
    Then the panel does NOT show an empty state — it auto-hides entirely
    And no "No workflows running" placeholder is shown

  @graceful-empty-scheduled
  Scenario: No scheduled tasks shows helpful empty state
    Given no cron jobs exist
    When the user opens the scheduled tasks view
    Then a message is shown: "No scheduled tasks. Schedule recurring work with /loop or schedule_task."
    And examples are shown: "Every morning: 'Summarize overnight CI failures'. Every hour: 'Check for dependency updates'."

  # ── Feature Discovery ─────────────────────────────────────────────

  @graceful-discovery-subagent
  Scenario: Subagent workflow suggested after repeated manual usage
    Given the user has spawned 5 manual task subagents across 3 sessions
    When the 6th task subagent is spawned
    Then a one-time notice appears: "💡 You've been spawning subagents manually. Try the workflow tool to compose multi-stage pipelines."
    And the notice links to /help workflows

  @graceful-discovery-remote
  Scenario: Remote worker suggested when subagent is resource-heavy
    Given the user is on a machine with ≤4 CPU cores
    And a subagent has been running for >5 minutes
    When the subagent completes
    Then a notice appears: "💡 Long-running subagents can run on a remote build server to free your machine. Set up a remote worker."
    And the notice does not appear more than once per day

  @graceful-discovery-effort
  Scenario: Effort suggestion after repeated verification failures
    Given the user has run 3 verify subagents at effort "high" that all returned PLAUSIBLE
    When the 4th verify subagent is spawned
    Then a notice suggests: "💡 Your verifiers keep returning PLAUSIBLE at high effort. Try effort 'max' for more definitive results."
    And the notice links to /help subagents#effort-calibration

  # ── UI Complexity Budget ──────────────────────────────────────────

  @graceful-complexity
  Scenario: First-run UI shows only essential panels
    Given the user is running reasonix for the first time
    When the TUI renders
    Then only the chat area, composer, and status line are visible
    And the background panel, remotes panel, and scheduled tasks are hidden
    And they appear only when relevant

  @graceful-complexity
  Scenario: Advanced features hidden behind discoverable toggle
    Given the user has not used advanced features
    When the slash menu opens
    Then basic commands are shown first: /help, /model, /clear, /export
    And advanced commands are grouped under an "Advanced" section: /workflow, /remotes, /schedule, /metrics
    And the Advanced section has a note: "These features unlock as you use reasonix more"

  @graceful-complexity
  Scenario: Settings panel shows basic/advanced toggle
    When the user opens settings
    Then basic settings are shown by default: Model, Theme, Language
    And an "Advanced" toggle reveals: Subagent Models, Effort Calibration, Fallback Chains, Rate Limits, Telemetry
    And the toggle remembers state across sessions

  # ── Degraded Mode ─────────────────────────────────────────────────

  @graceful-degraded
  Scenario: Degraded mode when no API keys configured
    Given no provider API keys are set
    When reasonix starts
    Then the UI loads with all panels accessible
    And the composer shows: "Configure an API key to start. Run reasonix setup or set DEEPSEEK_API_KEY."
    And all documentation (/help) is fully accessible
    And settings can be changed

  @graceful-degraded
  Scenario: Degraded mode when one of multiple providers is down
    Given DeepSeek Pro is down but Flash and Anthropic are up
    When reasonix runs
    Then all features work normally
    And the status line shows Pro as degraded
    And subagents that need Pro show the fallback prompt
    And subagents that use Flash or Anthropic are unaffected
