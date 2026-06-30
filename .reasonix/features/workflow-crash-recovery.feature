Feature: Workflow Crash Recovery
  As a developer running multi-stage workflows
  I want workflow state to survive crashes, restarts, and interruptions
  So that a $0.05 security audit that crashes at 80% doesn't waste $0.04 in tokens

  Background:
    Given reasonix has workflow state persistence enabled
    And workflow results are saved to disk after each stage completes

  # ── State Persistence ────────────────────────────────────────────

  @crash-persist-stage
  Scenario: Completed stage results saved to disk immediately
    Given a 3-stage workflow (find → verify → report) is running
    When the find stage completes with 5 findings from 3 finder subagents
    Then the 5 findings are persisted to .reasonix/workflows/<id>/stage-1.json
    And the file is written atomically (not partial)
    And the persist happens before any verify subagent spawns

  @crash-persist-stage
  Scenario: Stage results include subagent transcripts
    Given a finder subagent "finder-2" completed and returned 3 findings
    When the stage result is persisted
    Then the persisted file includes the 3 findings
    And the file includes finder-2's subagent ref for transcript retrieval
    And the file includes token usage for finder-2
    And the file includes the stage completion timestamp

  @crash-persist-stage
  Scenario: In-progress stage state is NOT persisted mid-stage
    Given the find stage has 2 of 3 finders completed, 1 still running
    When a crash occurs
    Then the persisted state shows the find stage as "in_progress"
    And only the 2 completed finders' results are persisted
    And the 1 running finder is marked "interrupted"

  # ── Workflow Manifest ─────────────────────────────────────────────

  @crash-manifest
  Scenario: Workflow manifest tracks overall state
    Given a workflow "wf_abc" is created with 3 stages and 5 items
    When the workflow starts
    Then a manifest file .reasonix/workflows/wf_abc/manifest.json is created
    And the manifest includes: workflow ID, strategy, stage definitions, items, created_at, status="running"
    And the manifest updates status to "completed" when all stages finish
    And the manifest is written atomically

  @crash-manifest
  Scenario: Manifest survives power loss during write
    Given the manifest is being updated from "running" to "completed"
    When power loss occurs mid-write
    Then the manifest file is either the old "running" version or the new "completed" version
    And the file is never corrupt or partial
    And the atomic write guarantee holds

  # ── Resume Detection ──────────────────────────────────────────────

  @crash-resume-detect
  Scenario: Controller detects interrupted workflow on boot
    Given a workflow "wf_abc" has manifest status "running"
    And the workflow was started 15 minutes ago
    And the parent session that spawned it is no longer active
    When the controller boots
    Then the workflow is detected as interrupted
    And a notice is emitted: "Interrupted workflow detected: 'security-audit' (wf_abc). Stage 2 of 3 — 5 findings need verification. Resume?"

  @crash-resume-detect
  Scenario: Recently-running workflow not flagged on quick restart
    Given a workflow "wf_abc" has manifest status "running"
    And the workflow was started 3 seconds ago (quick restart after a config change)
    When the controller boots
    Then the workflow is detected
    And a notice is emitted: "Workflow 'security-audit' was in progress. Resuming..."
    And the resume is automatic (not prompted) because the subagent processes are likely still alive

  @crash-resume-detect
  Scenario: Stale workflow running status cleaned up
    Given a workflow "wf_abc" has manifest status "running"
    And the workflow was started 7 days ago
    And all subagent refs point to transcripts that are marked "interrupted"
    When the controller boots
    Then the workflow is marked "abandoned"
    And the manifest is updated with abandoned_at timestamp
    And no resume prompt is shown

  # ── Resume Execution ──────────────────────────────────────────────

  @crash-resume-execute
  Scenario: Resume from last completed stage
    Given a workflow "wf_abc" crashed after stage 1 (find) completed
    And stage 1 produced 5 findings persisted to disk
    When the user confirms resume
    Then stage 1 is skipped (results loaded from disk)
    And stage 2 (verify) starts with the 5 persisted findings
    And the 3 original finder subagents are NOT re-spawned
    And no tokens are wasted re-running completed work

  @crash-resume-execute
  Scenario: Resume from partially-completed stage
    Given a workflow "wf_abc" crashed during stage 1 with 2 of 3 finders completed
    When the user confirms resume
    Then only the 1 incomplete finder is re-spawned
    And the 2 completed finders' results are loaded from disk
    And the stage completes when the 3rd finder finishes
    And the token waste from the crash is limited to the 1 interrupted finder

  @crash-resume-execute
  Scenario: Resume with new items added since crash
    Given a workflow "wf_abc" crashed after stage 1 with items ["auth.go", "middleware.go"]
    And the user wants to add "handlers.go" on resume
    When the user confirms resume with additional items
    Then stage 1 runs only for "handlers.go" (original items are loaded from disk)
    And stage 2 gets all 3 items' findings

  @crash-resume-execute
  Scenario: Resume with different model/effort
    Given a workflow crashed with model "deepseek-v4-flash"
    And the user's config now has a different default model
    When the user confirms resume
    Then the remaining stages use the ORIGINAL model (workflow was designed for Flash)
    And a notice warns: "Workflow was started with deepseek-v4-flash. Model unchanged for consistency."
    And the user can override with --model flag if desired

  # ── Idempotency ───────────────────────────────────────────────────

  @crash-idempotent
  Scenario: Re-running a completed stage is safe
    Given stage 1 of workflow "wf_abc" completed and was persisted
    When the stage is somehow re-executed (resume bug, manual trigger)
    Then the re-executed result replaces the previous persisted result
    And the new result is used for stage 2
    And no state corruption occurs from the double-write

  @crash-idempotent
  Scenario: Stage results are deterministic for same inputs
    Given stage 1 completed with 5 findings
    When the exact same stage 1 is re-run with the same items and model
    Then the findings are structurally identical (same file:line locations)
    And content may vary (LLM non-determinism) but structure is identical
    And the new result is accepted as a valid replacement

  # ── Partial Failure Recovery ──────────────────────────────────────

  @crash-partial-failure
  Scenario: One finder failed before crash, resume re-spawns only that finder
    Given stage 1 had 3 finders: F1 (completed, 3 findings), F2 (completed, 2 findings), F3 (failed, error)
    And the workflow crashed
    When the user resumes
    Then F1 and F2 results are loaded from disk
    And F3 is re-spawned
    And the stage completes when F3 finishes (or fails again)

  @crash-partial-failure
  Scenario: Verifier results preserved through crash
    Given stage 2 (verify) had 9 verifier subagents for 3 findings
    And 5 verifiers completed (2 confirmed, 2 refuted, 1 plausible) before crash
    When the user resumes
    Then the 5 completed verifier results are loaded
    And the 4 incomplete verifiers are re-spawned
    And voting aggregates ALL 9 results (5 loaded + 4 new)
    And the final verdict is correct

  @crash-partial-failure
  Scenario: All verifiers for one finding completed, zero for another
    Given stage 2 had verifiers for finding-X (all 3 completed) and finding-Y (0 of 3 completed)
    And the workflow crashed
    When the user resumes
    Then finding-X is already verified (results loaded, no re-spawn)
    And finding-Y gets 3 new verifiers
    And the stage shows progress: "Verifying: 1/2 findings done, 3/6 verifiers complete"

  # ── Cleanup ───────────────────────────────────────────────────────

  @crash-cleanup
  Scenario: Completed workflow directory cleaned after TTL
    Given a workflow completed 8 days ago
    And the workflow TTL is 7 days
    When the controller performs periodic cleanup
    Then the workflow directory is removed
    And only the workflow summary is retained (in session transcript)

  @crash-cleanup
  Scenario: Abandoned workflow directory cleaned
    Given a workflow was abandoned 1 day ago
    When the controller performs cleanup
    Then the workflow directory is removed
    And subagent transcripts for the workflow are archived

  @crash-cleanup
  Scenario: Active workflow protected from cleanup
    Given a workflow is currently running
    When the controller performs cleanup
    Then the workflow directory is NOT touched
    And the manifest is NOT removed

  # ── Token Accounting Through Crash ─────────────────────────────────

  @crash-tokens
  Scenario: Token usage preserved across crash and resume
    Given a workflow consumed 15,000 tokens before crashing
    When the workflow is resumed and completes
    Then the final token report includes both pre-crash and post-resume tokens
    And the cost breakdown separates pre-crash and post-resume
    And the total is correct: pre_crash_tokens + resume_tokens

  @crash-tokens
  Scenario: Crash waste is surfaced in token report
    Given a workflow crashed mid-verification
    And 2 verifier subagents were interrupted and their partial work is unbillable
    When the final token report is generated
    Then the report includes: "Tokens: 15,000 (completed) + 1,200 (interrupted, unbilled)"
    And the interrupted tokens are NOT counted in the cost
    And a note explains: "Interrupted subagent work is not billed by DeepSeek for pre-output refusals"

  # ── UI for Crash Recovery ─────────────────────────────────────────

  @crash-ui
  Scenario: Resume prompt renders with workflow details
    Given a workflow "security-audit" was interrupted at stage 2 of 3
    When the resume prompt renders
    Then the prompt shows: workflow name, current stage, completed stages, findings so far
    And a "Resume" button is prominent
    And a "Discard" button is secondary
    And a "View details" link shows the full manifest

  @crash-ui
  Scenario: Resume progress reconstructs UI state
    Given a workflow is resumed after crash
    When the workflow continues
    Then the background tasks panel shows the workflow with correct stage progress
    And the stage progress bar shows the pre-crash completed items as already done
    And the panel does NOT show them as "running" or "pending"
    And new subagents appear as they spawn

  @crash-ui
  Scenario: Multiple interrupted workflows listed in priority order
    Given 3 workflows were interrupted: A (80% done, 2 min ago), B (30% done, 1 hour ago), C (5% done, 6 hours ago)
    When the resume prompt renders
    Then workflows are ordered: A, B, C (highest completion first)
    And the user can select which to resume
    And a "Resume All" button is available

  @crash-ui
  Scenario: Crash recovery panel in settings
    When the user opens settings → Workflows
    Then a "Crash Recovery" section shows: all interrupted workflows, their status, age, and a Resume/Discard action per workflow
    And a "Clean up all abandoned" button removes workflows older than TTL
