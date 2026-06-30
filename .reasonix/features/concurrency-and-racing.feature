Feature: Concurrency and Race Conditions
  As a developer running parallel subagents
  I want concurrent operations to be deterministic and race-free
  So that parallel workflows are reliable under load

  Background:
    Given reasonix has SubagentMessenger, jobs.Manager, and workflow tool all active

  # ── Messenger concurrency ─────────────────────────────────────────

  @concurrency-messenger-register
  Scenario: Concurrent registration from parallel_tasks
    Given 50 subagents are spawned simultaneously via parallel_tasks
    When each subagent calls messenger.Register() in its own goroutine
    Then all 50 subagents appear in the messenger's registry
    And no data race is detected by the Go race detector
    And no subagent ref is silently lost

  @concurrency-messenger-send
  Scenario: Concurrent send during registration
    Given 10 subagents are being registered
    And the parent is simultaneously sending messages to 5 of them
    When all goroutines complete
    Then every registered subagent that was messaged received exactly one steer
    And no panics or nil pointer dereferences occur
    And no messages are delivered to the wrong subagent

  @concurrency-messenger-unregister
  Scenario: Unregister during message delivery
    Given a subagent is completing and unregistering
    And the parent is sending a message to that subagent simultaneously
    When both goroutines complete
    Then either the message is delivered (if send beat unregister) or the send returns "not found"
    And no panic occurs

  # ── Workflow concurrency ──────────────────────────────────────────

  @concurrency-workflow-pipeline
  Scenario: Pipeline items progress independently without artificial barriers
    Given a pipeline workflow with 10 items and 3 stages
    And stage-1 takes 1s per item, stage-2 takes 5s per item, stage-3 is instant
    When the workflow runs
    Then item-1 reaches stage-3 while item-2 is still in stage-2 and item-3 is still in stage-1
    And the total wall-clock time is approximately max(stage times) not sum(stage times)
    And no item's stage-2 waits for another item's stage-1

  @concurrency-workflow-barrier
  Scenario: Barrier waits for slowest item before proceeding
    Given a barrier workflow with 5 items in stage-1
    And item-3 takes 10 seconds while items 1,2,4,5 take 1 second each
    When the workflow runs
    Then no stage-2 subagent starts until item-3's stage-1 completes
    And the barrier time is approximately 10 seconds (the slowest item)

  @concurrency-workflow-cancellation
  Scenario: Cancelling workflow cancels all in-flight subagents
    Given a pipeline workflow with 20 items across 3 stages
    And 5 subagents are currently running
    When the workflow context is cancelled
    Then all 5 running subagents receive cancellation
    And pending subagents are marked as skipped
    And the workflow returns a partial result with completed items

  @concurrency-workflow-cancellation
  Scenario: Partial workflow result on cancellation
    Given a pipeline workflow was cancelled with 3 of 10 items completed
    When the workflow returns
    Then 3 items have results
    And 7 items are marked as skipped with reason "cancelled"
    And the result includes a summary: "3 completed, 7 skipped (cancelled)"

  @concurrency-workflow-max-simultaneous
  Scenario: Concurrency capped at system limit
    Given the system has a max concurrency cap of 8
    And a workflow would spawn 20 subagents in one stage
    When the stage begins
    Then at most 8 subagents run simultaneously
    And the remaining 12 queue and start as slots free up

  @concurrency-workflow-dependency
  Scenario: Dependency chain respects ordering
    Given 3 tasks with dependencies: task-3 depends on task-2 depends on task-1
    When the parallel_tasks tool runs
    Then task-1 starts immediately
    And task-2 starts only after task-1 completes
    And task-3 starts only after task-2 completes

  @concurrency-workflow-dependency
  Scenario: Diamond dependency resolves correctly
    Given 4 tasks: A and B have no deps, C depends on A, D depends on A and B
    When the parallel_tasks tool runs
    Then A and B start concurrently
    And C and D start only after A completes
    And D waits for B even though A completed (both deps must be satisfied)
    And all 4 complete successfully

  # ── Job manager concurrency ───────────────────────────────────────

  @concurrency-jobs-view
  Scenario: Job View read while job state updated
    Given a job is running and updating ToolCalls every 100ms
    And the TUI is reading Job.View at 500ms intervals
    When both run for 30 seconds
    Then no data race is detected
    And no partial/inconsistent View snapshot is observed

  @concurrency-jobs-stalled
  Scenario: Stalled job detected while other jobs run
    Given 3 background jobs are running
    And job-2 has had no activity for 5 minutes
    When the stalled warning check fires
    Then a warning notice is emitted: "background task 'job-2' may be stalled"
    And jobs 1 and 3 are not affected

  # ── Subagent store concurrency ────────────────────────────────────

  @concurrency-store-prepare
  Scenario: Concurrent PrepareFresh calls get unique refs
    Given two goroutines call SubagentStore.PrepareFresh() simultaneously
    When both complete
    Then each receives a unique subagent ref (sa_...)
    And neither ref collides
    And both runs are independently releasable

  @concurrency-store-load
  Scenario: Load during save does not see partial write
    Given a subagent transcript is being saved (written to temp file, atomically renamed)
    And another goroutine calls LoadMeta for the same ref
    When the load occurs mid-write
    Then the load either sees the old file or the complete new file
    And never a partial/corrupt file (atomic rename guarantees this)
