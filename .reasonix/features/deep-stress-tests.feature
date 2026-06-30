Feature: Deep Stress Tests — State, Resources, Provider, User Flows
  As a developer who uses reasonix every day over months
  I want the system to handle accumulated state, provider changes, and real user workflows
  So that reasonix doesn't silently degrade the longer I use it

  Background:
    Given reasonix has been used daily for 6 months
    And 25 compensations have been active throughout

  # ── Cat 3C: Cross-Session Drift ───────────────────────────────────

  @stress-kb-model-drift
  Scenario: KB entries tagged with producing model version
    Given 200 KB entries exist from model "deepseek-v4-flash (2026-03)"
    And the current model is "deepseek-v4-flash (2026-07)"
    When the KB serves an entry
    Then the entry shows: "⏱️ Analyzed March 2026 (v4-flash Mar) · Current model: July 2026"
    And entries from significantly different model versions show a model-mismatch warning
    And the user can filter KB results to "current model only"

  @stress-kb-model-drift
  Scenario: Stale KB entries are re-verified on access
    Given a KB entry is from a model version 4 months old
    When the user asks a question that matches this entry
    Then a lightweight freshness check runs (Flash, $0.0003): "Is this analysis still accurate given the current code?"
    And if yes → served with "✓ Verified current"
    And if no → fresh analysis spawned, KB updated, old entry deprecated

  # ── Cat 4A: Provider Price Change ─────────────────────────────────

  @stress-provider-price
  Scenario: Provider price increase triggers compensation review
    Given DeepSeek Flash input price increased from $0.14/M to $0.28/M
    When the next compensation cost tracking cycle runs
    Then each compensation's daily cost is recalculated at the new rate
    And compensations exceeding their cost-effectiveness threshold are flagged
    And a notice shows: "Provider price change: 2 compensations now exceed cost threshold. Review in Settings."
    And the user is not forced to disable — just notified

  @stress-provider-price
  Scenario: Cost-effectiveness threshold is configurable per compensation
    Given the Ambient Guardian costs $0.03/day at current prices
    And the user set its max daily cost to $0.05
    When prices double
    Then the Guardian now costs $0.06/day — exceeding the threshold
    And the Guardian auto-pauses with notice: "Ambient Guardian paused — daily cost $0.06 exceeds threshold $0.05"
    And the user can raise the threshold or keep it paused

  @stress-provider-price
  Scenario: Massive price spike triggers emergency pause
    Given DeepSeek raises prices 10× overnight
    When the next API response includes the new pricing (or the first bill shock hits)
    Then ALL compensations auto-pause
    And a prominent notice shows: "Provider price increased 10×. All compensations paused. Review in Settings → Compensations."
    And the user must explicitly re-enable them

  # ── Cat 4B: Cache Removal ────────────────────────────────────────

  @stress-cache-removed
  Scenario: Auto-cache stops working — detected and alerted
    Given DeepSeek silently removes auto-caching
    When 3 consecutive requests show zero cache hits
    Then a cache degradation alert fires: "⚠ Auto-cache may be unavailable — 0% hit rate across last 3 requests"
    And if the next 10 requests also show zero hits
    Then the alert escalates: "🔴 Auto-cache appears disabled. Token costs increased ~50×. Review provider status."
    And cost projections are updated to assume zero cache

  @stress-cache-removed
  Scenario: Cache-dependent compensations adjust behavior
    Given auto-cache is confirmed unavailable
    When compensations that relied on high cache hit rates run
    Then they operate in "degraded cache" mode
    And compensation cost estimates are updated
    And the compensations dashboard shows which ones are in degraded mode

  # ── Cat 4C: Model Behavior Drift ──────────────────────────────────

  @stress-model-behavior-drift
  Scenario: Narration inflation detected and conciseness pass adjusts
    Given the model is producing 35% narration (baseline was 15%)
    When the weekly conciseness check runs
    Then the drift is detected: "Narration ratio: 0.35 vs baseline 0.15 — +133%"
    And the conciseness post-processing target is auto-adjusted: "Now targeting 35% reduction (was 20%)"
    And if auto-adjustment fails to restore target narration ratio, the prompt is flagged for recalibration

  # ── Cat 5A: Transcript Accumulation ───────────────────────────────

  @stress-transcript-retention
  Scenario: Transcript rotation prevents unbounded disk growth
    Given 5,000 subagent transcripts exist consuming 250MB
    And the retention policy is: keep 30 days or 1,000 latest, whichever is larger
    When the retention policy runs
    Then transcripts older than 30 days are deleted
    And at least the 1,000 most recent are preserved
    And the cleanup log shows: "Transcript rotation: deleted 2,100 old transcripts · 2,900 retained · 145MB freed"

  @stress-transcript-retention
  Scenario: Transcripts referenced by workflows are preserved
    Given a transcript from 45 days ago is referenced by a workflow that is still in the user's session history
    When retention cleanup runs
    Then the referenced transcript is preserved
    And the retention policy respects workflow references

  # ── Cat 5B: Knowledge Base Growth ─────────────────────────────────

  @stress-kb-growth
  Scenario: KB size limit enforced with intelligent eviction
    Given the KB has grown to 500MB (limit is 500MB)
    When a new entry would exceed the limit
    Then the least-valuable entry is evicted
    And value is scored by: recency, access frequency, confidence, uniqueness
    And the eviction notice shows: "KB at limit. Evicted 3 entries (oldest, least-accessed, lowest confidence). 497MB."
    And entries referenced by recent conversations are protected from eviction

  @stress-kb-dedup
  Scenario: Semantically identical entries are deduplicated
    Given two analyses of "how auth middleware works" were produced 2 weeks apart
    And they are semantically identical (>0.95 similarity)
    When the second is stored
    Then it replaces the first with updated timestamp
    And a note shows: "Updated: auth middleware analysis (same conclusions, refreshed 2026-07-15)"
    And the dedup saves storage and prevents stale entries from persisting

  # ── Cat 6A: Granular Undo ─────────────────────────────────────────

  @stress-granular-undo
  Scenario: Undo changes from a specific subagent
    Given subagents A, B, and C each modified different files
    When the user wants to undo only subagent B's changes
    Then the subagent transcript panel shows: "Files changed: handlers.go (lines 45-52), types.go (line 12)"
    And an "Undo these changes" button reverts only those files/lines
    And subagent A and C's changes are untouched

  @stress-granular-undo
  Scenario: Undo across overlapping changes handled safely
    Given subagent A edited auth.go lines 40-50
    And subagent B edited auth.go lines 45-55 (overlapping)
    When the user undoes subagent A's changes
    Then only A's unique lines (40-44) are reverted
    And the overlapping lines (45-50) show a warning: "⚠ Also modified by subagent B — partial revert may cause conflicts"
    And the user can preview the revert before applying

  # ── Cat 6B: "Fix This" One-Click ─────────────────────────────────

  @stress-fix-this
  Scenario: Finding has one-click fix button
    Given a security review produced "SQL injection in db.go:108"
    When the finding renders
    Then a "Fix this" button is shown
    And clicking it spawns an executor subagent with the finding context
    And the executor prompt includes: the finding, the file, the line, the suggested fix direction
    And the executor runs in a worktree to isolate the fix

  # ── Cat 6C: Session Fork ──────────────────────────────────────────

  @stress-session-fork
  Scenario: Fork session into new tab
    Given a session with 15 messages and an active investigation
    When the user clicks "Fork session"
    Then a new tab opens with a copy of the session up to the fork point
    And the original session continues independently
    And the forked session gets a new session ID
    And the fork point is marked in both transcripts: "Session forked here — diverged at 2026-07-01 14:22"

  # ── Cat 6D: Change Audit Trail ────────────────────────────────────

  @stress-audit-trail
  Scenario: Audit trail shows what changed, by which subagent, and why
    Given 5 subagents have modified 12 files over 2 hours
    When the user opens the audit trail
    Then a timeline shows: each change with subagent name, file, lines, timestamp, and rationale
    And clicking a change shows the diff
    And the audit trail is searchable by file or subagent

  # ── Cat 7A: Evaluation Overfitting ────────────────────────────────

  @stress-eval-overfitting
  Scenario: Held-out statistical power is adequate
    Given 4 calibration scenarios and 1 held-out scenario
    When the held-out power is checked
    Then the statistical power of 1 held-out run is flagged as insufficient
    And the system recommends: "Held-out set has low statistical power (N=1). Consider increasing to 3 held-out scenarios with 3 runs each for meaningful generalization testing."
    And the recommendation is logged but does not block shipping (it's advisory)

  # ── Cat 7B: Degradation Fatigue ───────────────────────────────────

  @stress-degradation-fatigue
  Scenario: Repeated degradation cycles trigger escalation
    Given the explorer prompt has degraded and been recalibrated 4 times in 4 weeks
    When the 5th degradation is detected
    Then the notice escalates: "Explorer prompt degraded 5 times in 4 weeks. This role may be fundamentally unstable on the current model version. Consider: switching to a different model for this role, or accepting degraded performance."
    And the usual auto-recalibrate flow is replaced with a user decision prompt
    And the user can choose: "Keep recalibrating automatically" or "Pin to specific model version" or "Accept degraded"

  # ── Cat 8A: Hallucinated Tool Output ─────────────────────────────

  @stress-hallucinated-output
  Scenario: Tool output cross-referenced against recorded results
    Given a subagent claims "grep returned 3 matches in handlers.go"
    And the actual grep output shows 2 matches
    When the post-processing correctness pass (Comp 3) runs
    Then the discrepancy is flagged: "⚠ Claimed 3 grep matches in handlers.go — actual tool output shows 2"
    And the claim confidence is downgraded
    And the original tool output is cited alongside the claim

  @stress-hallucinated-output
  Scenario: Claim references tool output with different content
    Given a subagent claims "read_file showed the function returns an error"
    And the actual read_file output shows a function that returns (result, nil)
    When the correctness pass cross-references
    Then the claim is flagged: "⚠ Claim contradicts tool output: read_file shows (result, nil) return, not error"
    And the finding is re-verified

  # ── Cat 7C: Goodhart's Law / Eval Gaming ──────────────────────────

  @stress-goodhart
  Scenario: Surface-level expectation match is distinguished from semantic match
    Given a verifier scenario expects {"kind": "output_contains", "value": "file:"}
    When a subagent output contains "see attached file: /dev/null" (surface match only)
    Then the surface-level expectation passes
    But a semantic check runs: "Is there an actual file path with a line number in the output?"
    And the semantic check fails: "No real file:line reference found"
    And the eval report shows: "⚠ Surface pass / semantic fail — possible expectation gaming"
    And the finding is flagged for eval scenario improvement

  @stress-goodhart
  Scenario: Eval scenarios are periodically refreshed to prevent gaming
    Given the explorer eval scenarios were last updated 90 days ago
    When the quarterly eval refresh check runs
    Then a notice shows: "Eval scenarios are 90 days old. Consider adding new scenarios to prevent overfitting."
    And the prompt calibration results show an "eval age" warning
    And new scenarios can be suggested by analyzing common failure patterns in live subagent runs

  @stress-goodhart
  Scenario: Expectation diversity is measured per scenario set
    Given the 4 explorer eval scenarios
    When the expectation diversity is calculated
    Then the report shows: "3/4 scenarios use output_contains · 1/4 uses no_narration · 0/4 use output_matches"
    And a warning fires if >50% of expectations in a scenario set are the same type
    And the warning suggests: "Add semantic expectations (output_matches with regex) to reduce surface-level pass risk"

  # ── Cat 5C: Confidence Profile Cold Start ────────────────────────

  @stress-profile-cold-start
  Scenario: Profile shows time-to-significance estimate
    Given a new accuracy profile for "Explorer · Flash · Python" has 3 samples
    When the profile renders in the dashboard
    Then it shows: "Collecting... 3/30 samples (10%) · est. significant in ~2 weeks at current rate"
    And the profile is flagged: "⚠ Insufficient samples — routing uses default, not profile"
    And the profile weight is 0 (not used for routing) until 30 samples accumulate

  @stress-profile-cold-start
  Scenario: Profile becomes active at significance threshold
    Given a profile has accumulated 30 samples
    When the 30th sample is stored
    Then a notice shows: "✓ Explorer · Flash · Python profile is now active (30 samples)"
    And the profile weight transitions from 0 to 1.0
    And routing starts using the profile
    And the dashboard shows: "Active · 30 samples · 87% accuracy"

  @stress-profile-cold-start
  Scenario: Profile confidence intervals narrow with more samples
    Given a profile has 30 samples with 87% accuracy
    When the profile stats render
    Then the confidence interval is shown: "87% ± 12% (30 samples)"
    Given 100 samples with 89% accuracy
    Then the interval narrows: "89% ± 7% (100 samples)"
    Given 500 samples with 90% accuracy
    Then the interval narrows further: "90% ± 3% (500 samples)"

  # ── Cat 8B: Silent Language Switch ────────────────────────────────

  @stress-language-switch
  Scenario: Reasoning language switch detected
    Given the configured reasoning language is "en"
    And a subagent's reasoning_content switches to Chinese mid-stream
    When the reasoning is stored
    Then a notice fires: "⚠ Reasoning language switched to zh during subagent execution"
    And the notice is informational, not blocking
    And the user can view the non-English reasoning or set reasoning_language to "auto"
