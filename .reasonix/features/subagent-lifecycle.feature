Feature: Subagent Lifecycle State Machine
  As a developer reasoning about subagent behavior
  I want every subagent state transition to be well-defined and observable
  So that error handling, cancellation, and recovery are deterministic

  Background:
    Given reasonix has SubagentStore and jobs.Manager wired

  # ── State machine ──────────────────────────────────────────────
  # Created → Running → Idle → Running → Idle → ... → Completed
  #                                                   → Failed
  #                                                   → Killed
  #                                                   → Interrupted
  # Running → Idle (with stop_reason: requires_action) → Running (after user.custom_tool_result)
  # Any state → Interrupted (on context cancellation)

  @lifecycle-happy
  Scenario: Foreground subagent completes normally
    Given a task subagent is spawned with prompt "Find callers of X"
    When the subagent runs and returns a final answer without error
    Then the subagent status transitions:
      | from       | to        | trigger                              |
      | created    | running   | Run() called                         |
      | running    | completed | model returned final answer          |
    And the result contains the model's final answer
    And the subagent is released from SubagentStore

  @lifecycle-happy
  Scenario: Background subagent runs across turns
    Given a task subagent is spawned with run_in_background: true
    When the subagent starts
    Then its status is "running"
    And a Job is registered in jobs.Manager
    And the parent receives "Started background task '...' (job_id)"
    When the subagent completes on a later turn
    Then a Notice event is emitted: "background task completed"
    And DrainCompletedNote surfaces the result to the model

  @lifecycle-tool-error
  Scenario: Subagent fails on tool execution error
    Given a subagent calls a tool that returns an error
    And the model cannot self-correct after 3 attempts
    When the subagent run loop exits with an error
    Then the subagent status is "failed"
    And the error is returned to the parent
    And the subagent transcript is saved with status "failed"

  @lifecycle-storm-breaker
  Scenario: Subagent stopped by storm breaker
    Given a subagent has made 5 consecutive calls to "bash" with the same error
    When the storm breaker fires on the 6th attempt
    Then the subagent receives the breaker message
    And if the model still fails after the breaker, the subagent status is "failed"
    And the failure message includes "loop guard"

  @lifecycle-cognitive-loop
  Scenario: DeepSeek subagent stopped by cognitive loop detector
    Given a DeepSeek subagent has produced 3 turns with escalating uncertainty and no tool calls
    When the cognitive loop detector fires
    Then a breaker message is injected
    And if the model produces one more identical reasoning turn, the subagent status is "failed"
    And the failure message includes "cognitive loop detected"

  @lifecycle-cancellation
  Scenario: Context cancellation interrupts running subagent
    Given a subagent is running with tool calls in progress
    When the parent context is cancelled
    Then the subagent's Run() returns context.Canceled
    And the subagent status is "interrupted"
    And any in-flight tool calls are cancelled
    And no orphaned goroutines remain

  @lifecycle-cancellation
  Scenario: Background subagent survives parent turn cancellation
    Given a background subagent is running via jobs.Manager
    When the spawning turn's context is cancelled
    Then the background subagent continues running
    And the subagent's own context (from jobs.Manager) is not cancelled

  @lifecycle-max-steps
  Scenario: Subagent hits max_steps limit gracefully
    Given a subagent is configured with max_steps: 5
    And the subagent makes 5 tool calls without producing a final answer
    When the 6th tool call would be made
    Then the subagent returns a paused notice: "paused after 5 tool-call rounds"
    And the subagent status is "completed" (work saved, can be continued)
    And the notice includes the max_steps key so the model can continue

  @lifecycle-compaction
  Scenario: Subagent compacts its own context mid-run
    Given a subagent has a context window of 64000 tokens
    And the subagent's session approaches 80% of the window
    When auto-compaction fires
    Then older messages are summarized into a compaction block
    And the recent tail is preserved verbatim
    And the subagent continues running with the compacted context

  @lifecycle-continue
  Scenario: Subagent continued from prior transcript
    Given a subagent was spawned with "Run tests on auth.go" and completed
    And the result included "Subagent reference: sa_abc123"
    When a new subagent is spawned with continue_from: "sa_abc123" and prompt "Re-run tests after the last fix"
    Then the new subagent loads the prior session transcript
    And the model sees the prior conversation as history
    And the new subagent runs with the same tool scope and model

  @lifecycle-continue
  Scenario: Continue from cross-conversation ref forks into current conversation
    Given subagent ref "sa_abc123" belongs to an ancestor conversation
    When a subagent is spawned with continue_from: "sa_abc123"
    Then a copy is created owned by the current conversation
    And the original transcript is not modified
    And the result includes "Forked from: sa_abc123"

  @lifecycle-fork
  Scenario: Read-only subagent forked for parallel research
    Given a read_only_task subagent "sa_research" completed research on auth.go
    When a new read_only_task is spawned with fork_from: "sa_research" and prompt "Extend research to middleware.go"
    Then the new subagent inherits the prior conversation history
    And the forked subagent runs independently without affecting the original
    And both transcripts are independently saveable

  @lifecycle-cleanup
  Scenario: Stale running refs cleaned up on SubagentStore creation
    Given a previous process crashed leaving subagent "sa_stale" marked as "running"
    When a new SubagentStore is created for the same session directory
    Then "sa_stale" is marked as "interrupted"
    And a warning notice is emitted about the interrupted subagent

  @lifecycle-cleanup
  Scenario: Subagent artifacts reconciled on controller boot
    Given metadata files exist without corresponding session files
    When the controller boots and reconciles cleanup
    Then orphaned metadata files are deleted
    And a notice is emitted for each cleaned artifact

  # ── FE: Lifecycle Status Indicators ────────────────────────────────

  @lifecycle-fe-status-icons
  Scenario: Background panel shows correct icon for each lifecycle state
    Given subagents in states: running, idle(waiting), completed, failed, interrupted, killed
    When the background panel renders
    Then running shows ● green with spinner
    And waiting shows ◐ yellow
    And completed shows ✓ green
    And failed shows ✗ red
    And interrupted shows ⚡ yellow with "interrupted" label
    And killed shows ⊘ gray with "killed" label

  @lifecycle-fe-status-transition
  Scenario: Status icon transitions smoothly on state change
    Given a subagent transitions from running to completed
    When the background panel renders the transition
    Then the ● spinner stops and morphs into ✓
    And the morph animation is 300ms
    And the row briefly highlights green before settling

  @lifecycle-fe-pause-notice
  Scenario: Max steps pause renders as resumable card
    Given a subagent hit max_steps: 5 and paused
    When the notice renders
    Then a card shows: "⏸ Paused after 5 tool-call rounds — work saved"
    And a "Continue" button is shown
    And a "Increase limit" button opens max_steps config
    And the subagent status in the panel shows "paused" with ⏸ icon

  @lifecycle-fe-crash-interrupted
  Scenario: Interrupted subagent shows recovery option
    Given a subagent was interrupted by a crash
    When the user resumes the session
    Then the interrupted subagent shows "⚠ Interrupted by crash"
    And a "Resume subagent" button is available
    And resuming continues from the last persisted turn


  @lifecycle-transcript
  Scenario: Subagent transcript saved on completion
    Given a subagent with a persisted SubagentStore completed successfully
    When the subagent releases
    Then a JSONL transcript is written to the subagents directory
    And a metadata file is written with status "completed"
    And both files are atomically renamed into place

  @lifecycle-transcript
  Scenario: Subagent transcript saved on failure
    Given a subagent failed with error "bash: command not found"
    When the subagent releases
    Then the transcript is saved with status "failed"
    And the metadata includes the error message
