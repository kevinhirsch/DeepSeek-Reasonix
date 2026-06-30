Feature: Compositional Edge Cases — When Compensations Compose
  As a developer running reasonix with all compensations active
  I want the system to handle multi-compensation interactions without cost explosion or deadlock
  So that turning everything on is safe, not risky

  Background:
    Given all 25 compensations are active
    And a workflow is running with multiple stages

  # ── Cat 1: Compositional Explosions ───────────────────────────────

  @stress-compositional-blast
  Scenario: All compensations fire simultaneously without cost explosion
    Given a user requests "Implement rate limiting for the API"
    And compensations 1-10 and 14-19 are active
    When the feature is implemented
    Then the total compensation token cost is reported
    And no individual compensation fires more than once for the same trigger
    And the compensation chain depth is tracked and visible
    And if a compensation triggers another compensation, the chain is logged
    And the total cost is within 3× of the base implementation cost

  @stress-compositional-trigger-chain
  Scenario: Compensation trigger chain is detected and bounded
    Given Comp 9 (pre-mortem) predicted a risk
    And that risk materialized, triggering Comp 1 (speculative verification)
    And Comp 1 results triggered Comp 3 (post-processing)
    When Comp 3 completes
    Then the trigger chain depth is 3
    And no further compensations fire from Comp 3's output
    And a log entry shows: "Comp chain: pre-mortem → speculative → post-process (depth 3)"

  @stress-compositional-trigger-chain
  Scenario: Trigger chain is capped at max depth
    Given a trigger chain has reached depth 5 (the configured max)
    When a compensation would trigger another at depth 6
    Then the trigger is suppressed
    And a notice shows: "Compensation chain capped at depth 5 — 'semantic_diff' suppressed"

  # ── Cat 1B: Subagent Concurrency ─────────────────────────────────

  @stress-concurrency-global-limit
  Scenario: Global subagent limit prevents rate-limit stampede
    Given 30 subagents are queueing for the same provider
    And the provider rate limit is 10 requests/second
    When the global subagent limiter engages
    Then at most 8 subagents run concurrently (headroom below rate limit)
    And remaining subagents queue
    And the queue order is: workflow subagents first, then compensations, then ambient
    And the background panel shows: "8 running · 22 queued (releasing at 8/s)"

  @stress-concurrency-priority
  Scenario: User-initiated subagents preempt background compensations
    Given 5 idle pre-compute subagents are running (Comp 12)
    And the user spawns a workflow that needs 10 subagents
    When the workflow spawns
    Then 5 idle pre-compute subagents are paused
    And workflow subagents use the freed slots
    And idle pre-compute resumes when workflow spawns complete

  @stress-concurrency-competing
  Scenario: Ambient guardian queues behind active workflow
    Given a workflow has 20 subagents queued
    And the user saves a file (triggers Ambient Guardian)
    When the Guardian would spawn
    Then the Guardian subagent queues behind the workflow subagents
    And the status line shows: "🛡️ Guardian queued (position 21)"
    And if the Guardian doesn't spawn within 30 seconds, it times out and retries on next save

  # ── Cat 2A: Save During Active Workflow ──────────────────────────

  @stress-save-during-workflow
  Scenario: File save during active workflow is handled gracefully
    Given a security audit workflow is running 27 verifiers
    When the user saves a file with a function signature change
    Then the Impact Analysis (Comp 18) queues, doesn't preempt
    And the Ambient Guardian (Comp 11) queues at lower priority
    And both complete after the workflow's critical path subagents
    And the user is NOT blocked waiting for either analysis

  # ── Cat 2B: Idle Detection ────────────────────────────────────────

  @stress-idle-hysteresis
  Scenario: Idle detection with hysteresis prevents spawn churn
    Given the idle timer is 120 seconds
    And the user is reading and occasionally scrolling (activity every 30-60 seconds)
    When the idle timer would fire
    Then it does NOT fire because activity occurred within the window
    And the timer resets cleanly
    And no subagents are spawned or cancelled due to idle churn

  @stress-idle-hysteresis
  Scenario: Genuinely idle triggers pre-computation
    Given no user activity for 130 seconds
    When the idle timer fires
    Then pre-computation subagents spawn
    And the status line shows: "💤 Idle — pre-computing"

  @stress-idle-hysteresis
  Scenario: Pre-computation pauses immediately on user activity
    Given idle pre-computation is running
    When the user presses any key
    Then running pre-compute subagents are NOT killed (they complete)
    And no new pre-compute subagents spawn
    And the idle timer resets

  # ── Cat 2C: Degradation Exposure Window ──────────────────────────

  @stress-degradation-window
  Scenario: Degraded prompt rolls back instantly while recalibration runs
    Given drift detection found explorer prompt degraded from 5/5 to 2/5
    When the degradation is confirmed
    Then the prompt is IMMEDIATELY rolled back to the last known good version
    And rollback happens in <1 second (file swap, no API calls needed)
    And recalibration begins in the background
    And the status line shows: "⚠ Explorer prompt rolled back to v2 — recalibrating"
    And user subagents use the rolled-back prompt during recalibration

  @stress-degradation-window
  Scenario: No known-good version triggers fallback behavior
    Given drift is detected and no prior version scored above threshold
    When the rollback would occur
    Then a notice shows: "No known-good prompt version available. Explorer subagents may be degraded."
    And explorer subagents get extra post-processing passes (Comp 3 correctness) to compensate
    And recalibration is prioritized (runs immediately, not queued)

  # ── Cat 3A: Knowledge Base Poisoning ──────────────────────────────

  @stress-kb-poisoning
  Scenario: Factually wrong KB entry is detected and flagged
    Given a KB entry claims "auth middleware validates JWT tokens"
    And a new analysis of the auth middleware finds it uses session cookies, not JWT
    When the KB serves the old entry and the answer contradicts a fresh analysis
    Then the KB entry is flagged: "⚠ Possibly stale — contradicts fresh analysis from <date>"
    And the user sees the contradictory evidence
    And the KB entry confidence is downgraded

  @stress-kb-poisoning
  Scenario: KB entry verified by cross-validation before storage
    Given a new analysis is about to be stored in the KB
    And cross-validation (Comp 2) is active
    When the analysis is produced
    Then a second explorer (different model or effort) independently checks the key claims
    And if they agree → stored with confidence "high"
    And if they disagree → stored with confidence "low" and the disagreement noted
    And uncorroborated entries are flagged in KB search results

  # ── Cat 3B: Confidence Profile Decay ──────────────────────────────

  @stress-profile-decay
  Scenario: Confidence profile expires on model version bump
    Given a verifier has 94% accuracy on Go (50 samples, built on DeepSeek V4-Flash)
    And DeepSeek releases a new model version
    When the profile is queried after the model bump
    Then the profile is flagged: "⚠ Pre-update profile — accuracy may have changed"
    And the profile weight is halved until 10 new samples accumulate on the new model
    And the UI shows both: "Pre-update: 94% (50 samples) · Post-update: collecting..."
