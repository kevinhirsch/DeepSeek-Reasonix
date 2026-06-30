Feature: Model-Specific Behavior Divergence
  As a developer using both DeepSeek and Anthropic providers
  I want provider-specific behavior to be handled transparently
  So that the architecture works correctly regardless of which model runs where

  Background:
    Given reasonix is configured with both a DeepSeek provider and an Anthropic provider
    And subagent routing can target either provider

  # ── System prompt injection strategy ──────────────────────────────

  @divergence-system-prompt
  Scenario: DeepSeek subagent receives role prompt in user message
    Given the resolved subagent provider is DeepSeek
    When a subagent session is created with role "explore"
    Then Messages[0] has role "user" and contains the explorer prompt
    And session.System is empty or contains only non-instruction text
    And the model does not receive a system-role instruction block

  @divergence-system-prompt
  Scenario: Anthropic subagent receives role prompt in system message
    Given the resolved subagent provider is Anthropic
    When a subagent session is created with role "verify"
    Then Messages[0] has role "system" and contains the verifier prompt
    And the first user message is the task prompt only

  @divergence-system-prompt
  Scenario: Role prompt injection strategy determined at session creation
    Given the SubagentSpec includes a SystemPrompt field
    And the provider is not yet resolved when the spec is built
    When the session is created by runSubSession
    Then the injection strategy (system vs user message) is chosen based on the resolved provider
    And the spec's SystemPrompt is used appropriately regardless of strategy

  # ── Reasoning content handling ────────────────────────────────────

  @divergence-reasoning-roundtrip
  Scenario: DeepSeek reasoning_content preserved on tool-call turns
    Given a DeepSeek subagent makes a tool call
    And the assistant turn has reasoning_content present
    When the next request is built
    Then reasoning_content IS included in the assistant message
    And content IS included in the assistant message
    And tool_calls ARE included in the assistant message

  @divergence-reasoning-roundtrip
  Scenario: DeepSeek reasoning_content stripped on no-tool turns
    Given a DeepSeek subagent produces a final answer with no tool calls
    And the assistant turn has reasoning_content present
    When the next request is built (if conversation continues)
    Then reasoning_content is NOT included
    And only content is included

  @divergence-reasoning-roundtrip
  Scenario: Anthropic thinking blocks signed and round-tripped
    Given an Anthropic subagent with thinking: "adaptive" makes a tool call
    And the assistant turn has thinking with a signature
    When the next request is built
    Then the thinking block IS included with its signature intact
    And the thinking block precedes the tool_use block

  @divergence-reasoning-roundtrip
  Scenario: Non-DeepSeek providers do not emit reasoning_content in request
    Given a non-DeepSeek provider receives a session with ReasoningContent on messages
    When the provider builds the API request
    Then ReasoningContent is stripped from all messages
    And the API does not receive the field at all

  # ── Effort mapping ────────────────────────────────────────────────

  @divergence-effort
  Scenario: DeepSeek maps "low" and "medium" to actual "high" on the wire
    Given a workflow stage specifies effort: "medium" for a DeepSeek subagent
    When the provider builds the API request
    Then the reasoning_effort parameter sent to DeepSeek is "high"
    And a debug log notes the mapping: "effort 'medium' mapped to 'high' for DeepSeek"

  @divergence-effort
  Scenario: Anthropic sends effort values directly without mapping
    Given a workflow stage specifies effort: "medium" for an Anthropic subagent
    When the provider builds the API request
    Then the output_config.effort sent to Anthropic is "medium"
    And no mapping or debug log is emitted

  @divergence-effort
  Scenario: DeepSeek rejects "xhigh" mapping from Anthropic users
    Given a user configured effort: "xhigh" (familiar from Anthropic)
    And the resolved provider is DeepSeek
    When the provider validates the effort value
    Then "xhigh" is mapped to "max"
    And the request sent to DeepSeek uses "max"

  # ── Temperature divergence ────────────────────────────────────────

  @divergence-temp
  Scenario: DeepSeek subagent temperature raised from zero
    Given configured temperature is 0.0
    When a DeepSeek subagent is constructed
    Then effective temperature is 0.6

  @divergence-temp
  Scenario: Anthropic subagent temperature left at zero
    Given configured temperature is 0.0
    When an Anthropic subagent is constructed
    Then effective temperature is 0.0
    And Anthropic does not receive a temperature parameter at all (it rejects sampling params)

  @divergence-temp
  Scenario: User-configured non-zero temperature preserved for DeepSeek
    Given the user configured temperature: 0.8
    When a DeepSeek subagent is constructed
    Then effective temperature is 0.8 (user's choice, not overridden)

  # ── Cognitive loop detection ──────────────────────────────────────

  @divergence-loop
  Scenario: DeepSeek subagent monitored for cognitive loops
    Given a DeepSeek subagent is running
    When the stream handler receives each assistant turn
    Then deepseekCognitiveLoopDetected() is called
    And the three detection patterns are checked

  @divergence-loop
  Scenario: Anthropic subagent NOT monitored for DeepSeek-specific loops
    Given an Anthropic subagent is running
    When the stream handler receives each assistant turn
    Then deepseekCognitiveLoopDetected() is NOT called
    And Anthropic's own stop_reason: "refusal" handling is active instead

  # ── Token counting divergence ─────────────────────────────────────

  @divergence-tokens
  Scenario: DeepSeek tokenizer produces different counts than Anthropic
    Given the same 1000-character prompt
    When counted against DeepSeek V4-Pro
    And counted against Claude Opus 4.8
    Then the two counts may differ by up to 35%
    And the context window tracking uses the provider-specific count
    And compaction triggers are based on the correct count

  @divergence-tokens
  Scenario: Subagent context window uses provider-specific value
    Given a DeepSeek provider with context_window: 1000000
    And an Anthropic provider with context_window: 1000000
    When a subagent is constructed on each provider
    Then each subagent's contextWindow is 1000000
    And compaction ratios are computed against the correct window

  # ── Max tokens divergence ─────────────────────────────────────────

  @divergence-max-tokens
  Scenario: Anthropic max_tokens defaults to 32768
    Given an Anthropic provider with no explicit max_tokens
    When the API request is built
    Then max_tokens is set to 32768

  @divergence-max-tokens
  Scenario: DeepSeek max_tokens uses provider default
    Given a DeepSeek provider with no explicit max_tokens
    When the API request is built
    Then max_tokens uses the DeepSeek default (typically much larger)

  # ── FE: Divergence Status Indicators ──────────────────────────────

  @divergence-fe-status
  Scenario: Status line shows provider family with icon
    Given the active provider is DeepSeek
    When the status line renders
    Then the provider family is shown with a DeepSeek icon/indicator
    And the model name is shown alongside

  @divergence-fe-status
  Scenario: Provider switch notice includes behavioral differences
    Given the user switches from Anthropic to DeepSeek
    When the model switch confirmation renders
    Then a notice card shows behavioral differences:
      - "System prompts → User messages (DeepSeek trained without system prompts)"
      - "Effort levels → Only high/max (low/medium aliased to high)"
      - "Temperature → Recommended 0.6 (DeepSeek can loop at 0.0)"
    And a "Got it" button dismisses

  @divergence-fe-effort
  Scenario: Effort mapper notice visible when aliasing occurs
    Given effort "low" is configured for a DeepSeek provider
    When reasonix starts
    Then a one-time notice fires in the transcript: "DeepSeek maps 'low' to 'high'. Only two real levels: high and max."
    And the notice has a "Don't show again" option

  @divergence-fe-reasoning
  Scenario: DeepSeek reasoning display shows thinking tag
    Given a DeepSeek subagent produced reasoning_content
    When the reasoning block renders in the desktop transcript
    Then a small "[DeepSeek Thinking]" tag is shown on the collapsed reasoning block
    And the Anthropic reasoning blocks show "[Claude Thinking]" instead
    And the tags distinguish provider-specific reasoning formats

  @divergence-fe-cache
  Scenario: Cache hit display differentiates auto-cache from manual cache
    Given DeepSeek auto-cache produced a cache hit
    When the context panel renders
    Then the cache hit shows "[auto]" next to the token count
    And a tooltip explains: "DeepSeek auto-caches — no breakpoints needed"
    Given Anthropic manual cache produced a hit
    Then the cache hit shows "[manual]" with breakpoint info
