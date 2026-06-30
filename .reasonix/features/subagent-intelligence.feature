Feature: Subagent Intelligence
  As a developer using reasonix
  I want subagents to use role-specific system prompts and DeepSeek-aware behavior
  So that explorers find more, verifiers catch more false claims, and subagents don't loop forever

  Background:
    Given reasonix is configured with a DeepSeek V4 provider
    And the executor agent has access to task and parallel_tasks tools

  @01-role-prompts
  Scenario: Explorer subagent returns concrete file references
    When the model spawns an "explore" subagent with the prompt "Find all callers of AuthMiddleware"
    Then the subagent's system prompt contains breadth-over-depth guidance
    And the subagent's system prompt instructs it to return file:line references
    And the subagent's system prompt does NOT instruct it to evaluate or fix code
    And the subagent returns at least one finding with a specific file path and line number

  @01-role-prompts
  Scenario: Verifier subagent defaults to adversarial refutation
    When the model spawns a "verify" subagent with the prompt "Verify: nil pointer in auth.go:42"
    Then the subagent's system prompt opens with "Your job is not to confirm the implementation works — it's to try to break it"
    And the subagent's system prompt requires returning CONFIRMED, PLAUSIBLE, or REFUTED
    And the subagent's system prompt lists mandatory probes: concurrency, boundary, error paths, input validation

  @01-role-prompts
  Scenario: Reviewer subagent reports all findings without self-filtering
    When the model spawns a "review" subagent on internal/auth/auth.go
    Then the subagent's system prompt instructs it to report every issue including uncertain ones
    And the subagent's system prompt states a separate verification step will filter findings

  @01-role-prompts
  Scenario: Role prompt injects via user message for DeepSeek providers
    Given the resolved provider is DeepSeek
    When a subagent session is created with role "explore"
    Then the role prompt appears in the first user message
    And the system prompt is empty
    And the DeepSeekCommonNotes are appended to the user message

  @01-role-prompts
  Scenario: Role prompt uses system message for non-DeepSeek providers
    Given the resolved provider is Anthropic Claude
    When a subagent session is created with role "verify"
    Then the role prompt appears in the system message
    And no DeepSeek-specific notes are present

  # ── FE: Role & Temp Indicators ────────────────────────────────────

  @01-fe-role-indicator
  Scenario: Background panel shows subagent role icon
    Given an explorer subagent is running
    When the background panel renders
    Then the subagent label shows a role icon: 🔍 for explorer, 🔬 for reviewer, ⚔️ for verifier, 📋 for planner, ⚙️ for executor
    And hovering the icon shows a tooltip: "Explorer — breadth over depth, read-only"

  @01-fe-temp-notice
  Scenario: Temperature override notice renders once per session
    Given a DeepSeek subagent was constructed with temp override from 0.0 to 0.6
    When the notice fires
    Then it shows: "ℹ️ Subagent temperature adjusted to 0.6 for DeepSeek compatibility"
    And the notice has a "Learn more" link to DeepSeek temperature documentation
    And the notice does not fire for subsequent subagents in the same session

  @03-fe-loop-detected
  Scenario: Cognitive loop detected renders warning with loop details
    Given the loop detector fired on a DeepSeek subagent
    When the warning renders in the background panel
    Then the subagent row shows ⚠ with status "looping"
    And the reasoning tail shows "⚠ Cognitive loop detected — breaker injected"
    And a "Kill subagent" button is prominent
    And a "Let it recover" option is available

  @03-fe-loop-recovery
  Scenario: Loop recovery renders success notice
    Given a subagent was in a cognitive loop
    And the breaker message successfully broke the loop
    When the subagent resumes normal work
    Then a notice renders: "✓ Subagent recovered from loop — continuing"
    And the ⚠ indicator is removed
    And the status returns to normal running

  @02-deepseek-temp
  Scenario: DeepSeek subagent temperature overridden from zero
    Given the configured temperature is 0.0
    And the resolved subagent provider is DeepSeek
    When the subagent is constructed
    Then its effective temperature is 0.6
    And an info notice is emitted: "Subagent temperature adjusted to 0.6 for DeepSeek compatibility"

  @02-deepseek-temp
  Scenario: Non-DeepSeek subagent temperature unchanged
    Given the configured temperature is 0.0
    And the resolved subagent provider is Anthropic Claude
    When the subagent is constructed
    Then its effective temperature is 0.0
    And no temperature override notice is emitted

  @02-deepseek-temp
  Scenario: Parent orchestrator temperature not affected by override
    Given the configured temperature is 0.0
    And the resolved provider is DeepSeek
    When the parent orchestrator turn runs
    Then its effective temperature is the configured value, not 0.6

  @03-cognitive-loop
  Scenario: Uncertainty escalation detected and broken
    Given a DeepSeek subagent has produced two consecutive turns
    And each turn's reasoning contains at least 3 occurrences of "tricky", "confused", or "ambiguous"
    And neither turn produced a tool call
    When the third turn's reasoning is received
    Then a cognitive loop warning is emitted
    And a breaker message is injected: "Stop reasoning and act"

  @03-cognitive-loop
  Scenario: Identical reasoning prefix detected
    Given a DeepSeek subagent has produced a turn with reasoning starting with "I need to first understand the structure"
    When the next turn's reasoning also starts with "I need to first understand the structure"
    Then a cognitive loop warning is emitted
    And a breaker message is injected

  @03-cognitive-loop
  Scenario: Mechanical self-explanation without action detected
    Given a DeepSeek subagent has produced a turn ending with "I will now run grep to find"
    And the turn produced no tool calls
    When the next turn also starts with "I will now" and produces no tool calls
    Then a cognitive loop warning is emitted
    And a breaker message is injected

  @03-cognitive-loop
  Scenario: Legitimate multi-step reasoning not flagged
    Given a DeepSeek subagent is working through a complex refactor
    And it has produced three turns each with tool calls
    When the fourth turn's reasoning contains "uncertain" once in context of a specific API behavior
    Then no cognitive loop warning is emitted

  @03-cognitive-loop
  Scenario: Loop detector does not arm for non-DeepSeek providers
    Given the resolved provider is Anthropic Claude
    And the subagent produces text containing "tricky" three times with no tool calls
    Then no cognitive loop warning is emitted
