Feature: Regression — Agent Core Loop
  As a developer protecting reasonix's existing agent behavior
  I want no additive feature to break the agent loop, plan mode, compaction, or recovery
  So that the existing coding agent works identically before and after new features

  Background:
    Given reasonix has a fully configured agent with tools

  # ── Agent Loop ────────────────────────────────────────────────────

  @regression-agent-loop
  Scenario: Agent completes a simple turn end-to-end
    Given an agent with read_file and grep tools
    When the user asks "What files are in the current directory?"
    Then the agent makes at least one tool call
    And the agent produces a final answer referencing the tool output
    And the turn completes with stop_reason "end_turn" or "stop"

  @regression-agent-loop
  Scenario: Agent handles multi-turn tool use
    Given an agent with bash and read_file tools
    When the user asks "Run ls, then read the first Go file you find"
    Then the agent makes a bash call, gets output, makes a read_file call
    And the final answer references content from both tool calls

  @regression-agent-loop
  Scenario: Agent handles tool errors gracefully
    Given an agent with a tool that returns an error
    When the tool call fails
    Then the error is fed back to the model
    And the model self-corrects or asks for clarification
    And the agent loop does not crash

  @regression-agent-loop
  Scenario: Agent accumulates tool results across turns
    Given the agent has made 3 turns each with tool calls
    When the session is inspected
    Then Messages contains user → assistant(tool_calls) → user(tool_results) alternating
    And all tool_use IDs match their corresponding tool_result tool_use_ids

  # ── Plan Mode ─────────────────────────────────────────────────────

  @regression-plan-mode
  Scenario: Plan mode blocks write tools
    Given plan mode is enabled
    When the model calls write_file
    Then the tool is blocked with a "plan mode" reason
    And the model receives the block message
    And the model can still call read_file

  @regression-plan-mode
  Scenario: Plan mode toggle does not invalidate cache
    Given plan mode is toggled from off to on
    When the next API request is built
    Then the system prompt is unchanged
    And the tool schemas are unchanged
    And only the execute-time gate changes behavior

  @regression-plan-mode
  Scenario: Plan mode toggled off restores write tools
    Given plan mode was enabled, blocking writes
    When plan mode is disabled
    Then write_file calls execute normally
    And files are modified

  # ── Coordinator (Two-Model) ───────────────────────────────────────

  @regression-coordinator
  Scenario: Coordinator runs planner then executor
    Given a Coordinator with a planner model and an executor model
    When a task "Add a README to the project" is submitted
    Then the planner session receives the task first
    And the planner produces a plan using read-only tools
    And the executor session receives the plan and executes it
    And the planner and executor sessions are separate (no message leak)

  @regression-coordinator
  Scenario: Coordinator skips planner for trivial tasks
    Given a Coordinator with shouldPlan filtering out greetings
    When the user says "Hello"
    Then the planner is skipped
    And the executor responds directly

  # ── Compaction ────────────────────────────────────────────────────

  @regression-compaction
  Scenario: Compaction fires when context approaches ratio
    Given a session with context_window: 8000
    And the prompt reaches 6500 tokens (81%)
    When the next turn starts
    Then auto-compaction fires
    And older messages are summarized
    And the recent tail is preserved verbatim
    And the compacted context is under the target ratio

  @regression-compaction
  Scenario: Compaction preserves critical information
    Given a session where the user stated "My name is Alice" early on
    And compaction fires
    When the user later asks "What's my name?"
    Then the compaction summary contains "User's name is Alice"
    And the agent correctly answers "Alice"

  @regression-compaction
  Scenario: Compaction does not loop infinitely
    Given compaction fires
    And the compacted context is still above the ratio
    When compaction fires again
    Then it does not fire a third time if consecutive compactions exceed limit
    And the compactStuck latch engages
    And a notice is emitted: "auto-compaction paused"

  # ── Storm Breaker ─────────────────────────────────────────────────

  @regression-storm-breaker
  Scenario: Storm breaker detects repeating failures
    Given the model calls "bash" with an invalid command 6 times
    And each call returns the same error
    When the storm breaker threshold is reached
    Then a breaker message is injected into the tool result
    And a notice is emitted: "loop guard: bash failed 6× the same way"

  @regression-storm-breaker
  Scenario: Storm breaker resets on successful call
    Given the model had 3 failed bash calls
    When the 4th call succeeds
    Then the storm counter resets
    And no breaker message is injected

  @regression-storm-breaker
  Scenario: Storm breaker does not trigger on different errors
    Given the model has 3 bash failures with "command not found"
    When the 4th call fails with "permission denied"
    Then the storm signature changes
    And the counter resets

  # ── Stream Recovery ───────────────────────────────────────────────

  @regression-stream-recovery
  Scenario: Stream recovery preserves partial text
    Given a stream is interrupted mid-response
    When the stream handler detects the interruption
    Then the partial text accumulated so far is preserved
    And the partial reasoning is preserved
    And a recovery message is appended: "Your previous response was interrupted..."
    And the agent retries the API call

  @regression-stream-recovery
  Scenario: Stream recovery does not exceed max attempts
    Given maxStreamRecoveries is 3
    And the stream has been interrupted 4 times
    When the 4th recovery would be attempted
    Then the agent returns the error instead of retrying
    And the error message indicates stream recovery exhausted

  # ── Max Steps ─────────────────────────────────────────────────────

  @regression-max-steps
  Scenario: Max steps guard pauses agent gracefully
    Given maxSteps is 5
    And the agent makes 5 tool calls without producing a final answer
    When the 6th iteration would begin
    Then a pause notice is emitted: "paused after 5 tool-call rounds"
    And the session is saved (resumable)
    And the agent does not crash or hang

  @regression-max-steps
  Scenario: Unlimited max steps (0) never pauses
    Given maxSteps is 0
    And the agent makes 50 tool calls
    When each call returns a result
    Then the agent continues without pause
    And no max-steps notice is emitted

  # ── Final-Answer Readiness ────────────────────────────────────────

  @regression-readiness
  Scenario: Unfinished todo blocks final answer
    Given the agent created a todo with 3 items
    And only 2 items are completed
    When the agent attempts to produce a final answer
    Then the readiness check flags the uncompleted item
    And the agent is nudged to complete or update it

  @regression-readiness
  Scenario: All todos done allows final answer
    Given the agent created a todo with 2 items
    And both items are completed via complete_step
    When the agent produces a final answer
    Then the readiness check passes
    And the turn is accepted
