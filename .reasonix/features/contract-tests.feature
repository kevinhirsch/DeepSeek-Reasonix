Feature: Component Contract Tests
  As a developer modifying one component
  I want contracts between components to be verified
  So that changes to the task tool don't silently break the workflow tool

  # ── TaskTool → SubagentMessenger contract ─────────────────────────

  @contract-task-messenger
  Scenario: Background subagent registered on spawn
    Given a TaskTool with transcripts and messenger wired
    When task.Execute() is called with run_in_background: true
    Then the subagent run's Ref is registered with the messenger
    And the job ID (if any) is mapped to the ref

  @contract-task-messenger
  Scenario: Foreground subagent registered on spawn
    Given a TaskTool with messenger wired
    When task.Execute() is called with no run_in_background
    Then the subagent is registered with the messenger for the duration of execution
    And the subagent is unregistered when Execute() returns

  @contract-task-messenger
  Scenario: Subagent unregistered on completion
    Given a background subagent registered with the messenger
    When the subagent's job completes
    Then the subagent is removed from the messenger registry
    And the job ID mapping is removed

  @contract-task-messenger
  Scenario: Subagent unregistered on Job cancellation
    Given a background subagent registered with the messenger
    When the job is cancelled via kill_shell
    Then the subagent is removed from the messenger registry

  # ── WorkflowTool → TaskTool contract ──────────────────────────────

  @contract-workflow-task
  Scenario: Workflow stage spawns subagents via TaskTool.runSubSession
    Given a WorkflowTool with a TaskTool reference
    When a stage spawns subagents
    Then each subagent is spawned via TaskTool.runSubSession()
    And the subagent receives the correct role prompt for its stage role
    And the subagent receives the correct effort for its stage config

  @contract-workflow-task
  Scenario: Workflow passes model and effort to TaskTool
    Given a workflow stage specifies model: "deepseek-v4-pro" and effort: "max"
    When the stage spawns a subagent
    Then TaskTool.effectiveProfile() receives model="deepseek-v4-pro" and effort="max"
    And the subagent's resolved provider is DeepSeek V4-Pro

  @contract-workflow-task
  Scenario: Workflow fan_out creates N independent TaskTool calls
    Given a workflow stage with fan_out: 5
    When the stage processes one item
    Then TaskTool.runSubSession() is called 5 times
    And each call receives the same prompt but gets a unique tool call ID
    And all 5 calls run concurrently

  # ── SubagentStore → Session contract ──────────────────────────────

  @contract-store-session
  Scenario: PrepareFresh creates a new Session with the spec's system prompt
    Given a SubagentSpec with SystemPrompt: "You are an explorer"
    When SubagentStore.PrepareFresh() is called
    Then the returned SubagentRun.Session has a system message with the prompt
    And the session has exactly one message (the system prompt)

  @contract-store-session
  Scenario: PrepareContinue loads prior session messages
    Given a prior subagent run completed with 10 messages
    When SubagentStore.PrepareContinue() is called with that ref
    Then the returned SubagentRun.Session has all 10 messages
    And the session's Messages are a copy (not a reference to the stored data)

  @contract-store-session
  Scenario: MarkRunning persists the running status
    Given a fresh SubagentRun
    When SubagentStore.MarkRunning() is called
    Then the metadata file on disk shows status: "running"
    And the run is released

  @contract-store-session
  Scenario: SaveCompleted writes transcript and completed metadata
    Given a SubagentRun that ran to completion with 15 messages
    When SubagentStore.SaveCompleted() is called
    Then the transcript file contains 15 JSONL entries
    And the metadata file shows status: "completed" and updated_at is recent
    And the run is released

  # ── Provider → Agent contract ────────────────────────────────────

  @contract-provider-agent
  Scenario: Provider returns Chunk stream that Agent consumes
    Given a mock provider that emits: reasoning chunk, text chunk, tool_call_start, tool_call (complete), done
    When Agent.stream() reads the channel
    Then it accumulates reasoning into the reasoning variable
    And it accumulates text into the text variable
    And it accumulates the complete ToolCall into the calls slice
    And it returns when ChunkDone is received

  @contract-provider-agent
  Scenario: Provider stream interruption is recoverable
    Given a mock provider that emits half a response and closes the channel with an error
    When Agent.stream() reads the channel
    Then it returns the partial text and reasoning accumulated so far
    And it returns interrupted=true
    And the agent attempts stream recovery (if under maxStreamRecoveries)

  @contract-provider-agent
  Scenario: Agent feeds tool results back to provider in correct format
    Given the agent has executed 3 tool calls and has results
    When it builds the next provider.Request
    Then the request contains an assistant message with 3 tool_use blocks
    And the request contains a user message with 3 tool_result blocks
    And each tool_result.tool_call_id matches its corresponding tool_use.id

  @contract-provider-agent
  Scenario: DeepSeek provider reports reasoning_tokens in usage
    Given a DeepSeek provider
    When a turn completes
    Then usage includes completion_tokens_details.reasoning_tokens
    And the agent surfaces reasoning_tokens in the Usage event

  @contract-provider-agent
  Scenario: Anthropic provider reports cache tokens in usage
    Given an Anthropic provider
    When a turn completes with a cache hit
    Then usage includes cache_read_input_tokens > 0
    And the agent surfaces cache tokens in the Usage event
    And SessionCache accumulates hit/miss for the status line
