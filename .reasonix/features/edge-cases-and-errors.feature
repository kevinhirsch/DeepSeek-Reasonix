Feature: Edge Cases and Error Paths
  As a developer who pushes tools to their limits
  I want edge cases to fail gracefully with clear errors
  So that I can debug failures without reading the source code

  # ── Empty/null inputs ─────────────────────────────────────────────

  @edge-empty-prompt
  Scenario: Subagent spawned with empty prompt
    When task.Execute() is called with prompt: ""
    Then Execute returns error: "prompt is required"

  @edge-empty-items
  Scenario: Workflow with empty items array
    When the workflow tool is called with items: []
    Then it returns error: "at least one item is required"

  @edge-single-item
  Scenario: parallel_tasks with single task
    When parallel_tasks is called with 1 task
    Then it returns error: "parallel_tasks with a single task is equivalent to task; use task instead"

  @edge-empty-tool-whitelist
  Scenario: Subagent with empty tool whitelist gets all parent tools minus boundary
    When task.Execute() is called with tools: []
    Then the subagent receives all parent tools except meta tools (task, parallel_tasks, etc.)
    And bash is wrapped to foreground-only

  @edge-no-parent-sink
  Scenario: parallel_tasks with no event sink falls back to background jobs
    Given CallContext(ctx) returns no sink
    When parallel_tasks.Execute() is called
    Then runAsBackgroundJobs() is used instead of the event-based path
    And all tasks complete via the jobs manager

  @edge-read-only-no-tools
  Scenario: read_only_task with no read-only tools available
    Given the parent registry has only writer tools (write_file, edit_file)
    When read_only_task.Execute() is called
    Then it returns error: "read_only_task has no read-only tools available"

  # ── Continuation edge cases ───────────────────────────────────────

  @edge-continue-mutually-exclusive
  Scenario: continue_from and fork_from both specified
    When task.Execute() is called with continue_from: "sa_abc" and fork_from: "sa_def"
    Then it returns error: "continue_from and fork_from are mutually exclusive"

  @edge-continue-headless
  Scenario: Continuation in headless run without session
    Given the parent session is "" (headless run)
    When task.Execute() is called with continue_from: "sa_abc"
    Then it returns error: "subagent continuation requires a persisted session"

  @edge-continue-wrong-conversation
  Scenario: Continuation from ancestor conversation creates fork
    Given subagent "sa_abc" belongs to conversation "conv-1"
    And the current conversation is "conv-2"
    When task.Execute() is called with continue_from: "sa_abc"
    Then a copy is created under "conv-2"
    And the result includes "Forked from: sa_abc" with the fork explanation

  # ── Dependency edge cases ─────────────────────────────────────────

  @edge-circular-dependency
  Scenario: Circular dependency rejected
    When parallel_tasks is called with task-1 depends on task-2, task-2 depends on task-1
    Then validateParallelTaskItems returns an error
    And the error message mentions "circular dependency"

  @edge-self-dependency
  Scenario: Self-dependency rejected
    When parallel_tasks is called with task-1 depends on [1]
    Then validateParallelTaskItems returns an error
    And the error message mentions "self-dependency"

  @edge-out-of-range-dependency
  Scenario: Out of range dependency rejected
    When parallel_tasks is called with 3 tasks and task-0 depends on [5]
    Then validateParallelTaskItems returns an error
    And the error message mentions "invalid dependency index"

  # ── Max constraints ────────────────────────────────────────────────

  @edge-max-steps-zero
  Scenario: max_steps: 0 means unbounded
    Given a subagent is configured with max_steps: 0
    When the subagent runs
    Then there is no step limit
    And the subagent continues until the model produces a final answer or cancellation

  @edge-max-steps-one
  Scenario: max_steps: 1 allows only one tool round
    Given a subagent is configured with max_steps: 1
    And the model makes a tool call
    When the tool result is returned
    Then the subagent stops on the next iteration with a pause notice
    Even if the model wants to call another tool

  @edge-max-total-tasks
  Scenario: max_total_tasks exactly equals required tasks
    Given a workflow would spawn exactly 10 subagents
    And max_total_tasks is 10
    When the workflow runs
    Then all 10 subagents are spawned
    And the workflow completes normally

  @edge-max-total-tasks-plus-one
  Scenario: max_total_tasks exceeded by one
    Given a workflow would spawn 11 subagents
    And max_total_tasks is 10
    When the workflow runs
    Then exactly 10 subagents are spawned
    And 1 item is reported as unprocessed with reason "max_total_tasks"

  # ── Voting edge cases ─────────────────────────────────────────────

  @edge-voting-unanimous
  Scenario: All skeptics confirm
    Given fan_out: 3 and voting: {required: 2}
    And all 3 skeptics return CONFIRMED
    When the voting result is computed
    Then the finding is CONFIRMED
    And the result notes unanimous confirmation

  @edge-voting-tie
  Scenario: Tie vote with even number of skeptics
    Given fan_out: 4 and voting: {required: 2}
    And 2 skeptics return CONFIRMED, 2 return REFUTED
    When the voting result is computed
    Then the finding is CONFIRMED (tie goes to confirmation at exactly the threshold)
    And the result notes the split verdict

  @edge-voting-all-refuted
  Scenario: All skeptics refute
    Given fan_out: 3 and voting: {required: 2}
    And all 3 skeptics return REFUTED
    When the voting result is computed
    Then the finding is REFUTED
    And all 3 refutations are included in the result

  @edge-voting-single-skeptic
  Scenario: Single skeptic (fan_out: 1) with voting {required: 1}
    Given fan_out: 1 and voting: {required: 1}
    And the skeptic returns CONFIRMED
    When the voting result is computed
    Then the finding is CONFIRMED (single vote meets threshold)

  # ── Loop-until-dry edge cases ─────────────────────────────────────

  @edge-loop-zero-items
  Scenario: Loop-until-dry with no initial items
    When the workflow is called with strategy: loop_until_dry and items: []
    Then it returns error: "loop_until_dry requires at least one initial item"

  @edge-loop-first-round-dry
  Scenario: First round finds nothing
    Given a loop_until_dry workflow with dry_rounds: 1
    When the first finder round returns no new results
    Then the workflow stops after 1 dry round
    And returns an empty result set

  @edge-loop-all-duplicates
  Scenario: Every round finds only previously-seen results
    Given a loop_until_dry workflow with dry_rounds: 2
    And the finder keeps returning results that match the seen set
    When 3 rounds complete (1 initial + 2 dry)
    Then the workflow stops
    And returns only the results from round 1

  # ── Remote edge cases ─────────────────────────────────────────────

  @edge-remote-unknown
  Scenario: Target references unconfigured remote
    When task.Execute() is called with target: "nonexistent-remote"
    Then it returns error: "remote 'nonexistent-remote' is not configured"

  @edge-remote-concurrent-cap
  Scenario: Remote at concurrent capacity
    Given remote "build-server" has max_concurrent: 2
    And 2 subagents are already running on "build-server"
    When a third subagent targets "build-server"
    Then the work item is queued (not rejected)
    And the work item status shows "queued"
    And it is claimed when a slot frees up

  @edge-remote-auth-failure
  Scenario: Remote worker has invalid auth token
    Given a remote worker is configured with an expired auth token
    When the worker long-polls GET /v1/work/pending
    Then the server returns 401
    And the worker logs the error and retries after backoff
    And the worker does NOT crash

  @edge-bootstrap-unsupported-os
  Scenario: Bootstrap on unsupported OS
    When "reasonix remote bootstrap" targets an unsupported OS (Alpine, Arch, etc.)
    Then the generated script exits early with: "Unsupported OS. Please install Docker manually."
    And the exit code is 1

  # ── Sandbox edge cases ────────────────────────────────────────────

  @edge-sandbox-unavailable
  Scenario: Sandbox mode requested but OS support missing
    Given the platform has no Seatbelt or bubblewrap
    When a bash tool runs with sandbox mode "enforce"
    Then a warning is printed: "bash sandbox requested but unavailable on this platform"
    And the command runs unconfined

  @edge-sandbox-path-traversal
  Scenario: Subagent attempts to write outside workspace
    Given the sandbox has WriteRoots: ["/workspace"]
    When a subagent's write_file tool attempts to write to "/etc/passwd"
    Then the sandbox blocks the write
    And the tool returns error: "path escapes workspace root"

  @edge-sandbox-forbidden-read
  Scenario: Subagent attempts to read forbidden directory
    Given the sandbox has ForbidReadRoots: ["/etc/ssh"]
    When a subagent's read_file tool attempts to read "/etc/ssh/ssh_config"
    Then the sandbox blocks the read
    And the tool returns error: "path is in a forbidden directory"

  # ── Event ordering edge cases ─────────────────────────────────────

  @edge-event-ordering
  Scenario: Workflow events are properly nested
    Given a workflow with 2 stages, each with 2 items
    When the workflow runs
    Then every ToolDispatch event for a stage subagent has the workflow's call ID as ancestor
    And ToolResult events for stage subagents are emitted before the workflow's own ToolResult
    And no stage-2 event is emitted before all stage-1 ToolResult events

  @edge-event-ordering
  Scenario: Concurrent subagent events are interleaved correctly
    Given 3 finders running concurrently in a pipeline workflow
    When finder-1 emits a ToolDispatch for grep
    And finder-2 emits a ToolDispatch for read_file
    And finder-1 emits a ToolResult for grep
    And finder-2 emits a ToolResult for read_file
    Then all 4 events are emitted
    And each event's ParentID correctly identifies which finder it belongs to
    And no event is dropped

  # ── FE: Error Rendering ───────────────────────────────────────────

  @edge-fe-error-card
  Scenario: Every error renders as a structured card with action
    Given a subagent failed with "bash: command not found: non-existent-tool"
    When the error renders in the transcript
    Then a card is shown with: error icon (✗), error message in red, cause section, suggested action
    And the suggested action is: "Check the tool name. Available tools: bash, read_file, write_file, ..."
    And a "Retry" button is available if the error is retryable

  @edge-fe-rate-limit
  Scenario: Rate limit error shows retry countdown
    Given DeepSeek returned 429 with retry-after: 30
    When the error renders
    Then a card shows: "⏳ Rate limited — retrying in 30s..."
    And the countdown updates every second
    And a "Retry now" button skips the countdown
    And the background panel shows the subagent as "waiting (rate limited)"

  @edge-fe-timeout
  Scenario: Timeout error shows elapsed time and suggestion
    Given a subagent timed out after 30 minutes
    When the error renders
    Then a card shows: "⏰ Subagent timed out after 30m 00s"
    And the suggestion is: "Increase container_timeout_minutes in remote config, or split the task into smaller sub-tasks"
    And the partial output (if any) is available via "View partial output"

  @edge-fe-validation
  Scenario: Validation error shows exactly which field failed
    Given a tool call had invalid arguments
    When the validation error renders
    Then a card shows: "⚠ Invalid arguments: 'max_results' must be between 1 and 100 (got 500)"
    And the invalid value is highlighted
    And the valid range is shown

  @edge-fe-context-exhausted
  Scenario: Context window exhaustion renders with token breakdown
    Given a subagent exhausted its 64K context window
    When the error renders
    Then a card shows: "⚠ Context window exhausted (64,000/64,000 tokens)"
    And a breakdown shows: System prompt (8K), Messages (48K), Tool output (8K)
    And the suggestion is: "Enable compaction or increase context_window in config"
    And a "Compact now" button is available

  @edge-fe-sandbox-blocked
  Scenario: Sandbox blocked path renders with path and rule
    Given a subagent attempted to write to a forbidden directory
    When the error renders
    Then a card shows: "🛡️ Sandbox blocked: write to '/etc/passwd' denied"
    And the card shows which sandbox rule blocked it: "WriteRoots: [/workspace]"
    And the card shows the attempted path relative to the workspace
