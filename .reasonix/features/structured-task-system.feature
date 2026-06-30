Feature: Structured Output & Task System
  As a developer using reasonix for code review
  I want structured findings, typed task tracking, and worktree isolation
  So that reviews produce machine-actionable results and parallel tasks don't conflict

  Background:
    Given reasonix has report_findings, task_get, task_output, and enter_worktree tools
    And the structured task system is wired into the agent loop

  # ── ReportFindings Tool ───────────────────────────────────────────

  @gap-report-findings
  Scenario: Code review produces structured findings list
    When the agent calls report_findings with level "high" and findings:
      [
        {file: "auth.go", line: 42, summary: "nil pointer dereference", failure_scenario: "config is nil → crash at line 42"},
        {file: "auth.go", line: 108, summary: "missing authz check", failure_scenario: "unauthenticated user → admin endpoint accessible"}
      ]
    Then the tool validates that all required fields are present
    And the findings are emitted as a structured event the UI renders as a typed list
    And each finding is clickable (file:line)

  @gap-report-findings
  Scenario: Empty findings returns success
    When the agent calls report_findings with level "low" and findings: []
    Then the tool returns success with an empty list
    And the UI shows "No findings at this level"

  @gap-report-findings
  Scenario: Missing required field returns error
    When the agent calls report_findings with a finding missing the "summary" field
    Then the tool returns error: "finding at index 0: summary is required"

  @gap-report-findings
  Scenario: Severity-ordered findings render correctly
    Given 4 findings with verdicts: CONFIRMED (critical), CONFIRMED (high), PLAUSIBLE (medium), REFUTED
    When the UI renders the findings
    Then only CONFIRMED findings are shown by default
    And REFUTED findings are hidden behind a toggle: "1 refuted finding"

  @gap-report-findings
  Scenario: ReportFindings with outcome tracking
    Given a review produced 3 findings
    And the parent agent applied fixes for all 3
    When report_findings is called again with outcome: "fixed" on each
    Then the UI shows all 3 as "✓ Fixed"

  # ── Structured Task System ────────────────────────────────────────

  @gap-task-get
  Scenario: TaskGet retrieves task by ID with full context
    Given a task with ID "1" was created via todo_write with description "Fix auth bug"
    When the agent calls task_get with task_id "1"
    Then the result includes: subject, description, status, blockedBy, blocks
    And the result includes the full original task description

  @gap-task-list
  Scenario: TaskList shows all tasks with status
    Given 3 tasks: pending(auth fix), in_progress(db migration), completed(readme update)
    When the agent calls task_list
    Then all 3 tasks are returned with their IDs and statuses
    And completed tasks are visually distinct

  @gap-task-block
  Scenario: Task blocks on another task
    Given task "2" depends on task "1"
    When the agent calls task_get on task "2"
    Then task "2" shows blockedBy: ["1"]
    And the agent is instructed: "Task 2 cannot start until Task 1 completes"

  @gap-task-output
  Scenario: TaskOutput retrieves completed task result
    Given task "1" completed with output "Fixed nil pointer in auth.go:42 — all tests pass"
    When the agent calls task_output with task_id "1"
    Then the output is returned
    And the status is "completed"

  @gap-task-output-running
  Scenario: TaskOutput on running task returns current status
    Given task "1" is in_progress
    When the agent calls task_output with block: false
    Then the output shows current partial result
    And the status is "in_progress"

  @gap-task-output-ui
  Scenario: Completed task result renders as expandable card
    Given task "1" completed with a 500-character output
    When the task result renders in the task panel
    Then the card shows the first 120 characters with "..." and an "Expand" button
    And clicking "Expand" shows the full output
    And clicking "Collapse" returns to the truncated view

  @gap-task-output-ui
  Scenario: Failed task result renders error in red with traceback toggle
    Given task "1" failed with error "bash: command not found: nonexistent" and a stack trace
    When the task result renders
    Then the card shows the error message in red
    And a "Show traceback" toggle is available
    And clicking the toggle reveals/collapses the full stack trace
    And the stack trace is rendered in monospace

  @gap-task-output-ui
  Scenario: Task panel shows live task count badge
    Given 3 tasks are pending, 2 in_progress, 1 completed
    When the status bar or task panel header renders
    Then a badge shows "3 pending · 2 active · 1 done"
    And the badge colors match status (pending=gray, active=yellow, done=green)

  @gap-task-update
  Scenario: TaskUpdate changes task status
    Given task "1" is pending
    When the agent calls task_update with task_id "1" and status "in_progress"
    Then task "1" shows status "in_progress"
    And the active form spinner updates to match

  @gap-task-delete
  Scenario: TaskDelete removes task
    Given task "3" is completed
    When the agent calls task_update with task_id "3" and status "deleted"
    Then task "3" no longer appears in task_list

  @gap-task-dependency-auto-unblock
  Scenario: Completing a task unblocks dependents
    Given task "2" is blockedBy: ["1"]
    And task "1" is marked completed
    Then task "2" shows blockedBy: []
    And the agent is notified: "Task 2 is now unblocked and available"

  # ── EnterWorktree ─────────────────────────────────────────────────

  @gap-worktree-create
  Scenario: EnterWorktree creates isolated workspace
    Given the current repo is clean at branch "main"
    When the agent calls enter_worktree with name "review-auth"
    Then a new git worktree is created at .reasonix/worktrees/review-auth
    And the worktree has its own branch "reasonix/review-auth"
    And the worktree directory exists and is writable

  @gap-worktree-subagent
  Scenario: Subagent runs in worktree
    Given a worktree "review-auth" was created
    When a subagent is spawned with workspace: ".reasonix/worktrees/review-auth"
    Then the subagent's sandbox WriteRoots include the worktree path
    And the subagent's cwd is the worktree path
    And file writes in the subagent go to the worktree, not the main workspace

  @gap-worktree-exit-keep
  Scenario: ExitWorktree with action keep preserves changes
    Given a worktree "review-auth" with uncommitted changes
    When the agent calls exit_worktree with action "keep"
    Then the worktree directory remains on disk
    And the branch "reasonix/review-auth" is not deleted
    And the main workspace returns to its original branch

  @gap-worktree-exit-remove
  Scenario: ExitWorktree with action remove cleans up
    Given a worktree "review-auth" has no uncommitted changes
    When the agent calls exit_worktree with action "remove"
    Then the worktree directory is deleted
    And the branch "reasonix/review-auth" is deleted
    And the main workspace returns to its original branch

  @gap-worktree-exit-remove-dirty
  Scenario: ExitWorktree remove refuses dirty worktree
    Given a worktree "review-auth" has uncommitted changes
    When the agent calls exit_worktree with action "remove"
    Then the tool returns error listing the dirty files
    And the worktree is NOT deleted
    And a hint is shown: "Use discard_changes: true to force removal"

  @gap-worktree-concurrent
  Scenario: Two worktrees can coexist
    Given worktree "review-auth" exists
    When the agent creates worktree "review-db"
    Then both worktrees exist simultaneously
    And subagents can run concurrently in each without file conflicts
