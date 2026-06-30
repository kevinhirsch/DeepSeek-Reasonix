Feature: Structural Compensation — Cost Advantage as Quality Surface
  As a developer using reasonix on DeepSeek
  I want the system to use DeepSeek's 50× cost advantage to do quality-enhancing work Anthropic can't afford
  So that output quality matches or exceeds Claude Code despite model capability differences

  Background:
    Given reasonix has structural compensations 1-10 enabled
    And DeepSeek Flash costs $0.14/M input and DeepSeek Pro costs $0.435/M input

  # ── Comp 1: Speculative Execution ─────────────────────────────────

  @comp-speculative-basic
  Scenario: Three explorers find more than one
    When the agent calls explore_speculative with query "Find all callers of AuthMiddleware" and n=3
    Then 3 independent explorer subagents spawn with the same prompt
    And each explores independently (different search paths due to LLM non-determinism)
    And results are merged: deduplicated file:line findings
    And findings found by 2+/3 subagents are marked "confirmed"
    And findings found by only 1/3 are marked "uncertain"
    And the merged result includes the union of all unique findings

  @comp-speculative-basic
  Scenario: Speculative execution with varying n
    When the agent calls explore_speculative with n=5
    Then 5 explorers spawn
    And findings found by 3+/5 are confirmed
    And a note shows: "5 explorers found 12 unique findings (8 confirmed by ≥3, 4 uncertain)"

  @comp-speculative-cost
  Scenario: Speculative execution reports cost vs single explorer
    Given a single explorer costs ~$0.001
    When speculating with n=3 costs ~$0.003
    Then the result includes: "Cost: $0.003 (3× single explorer). Found 2 findings a single explorer missed."

  # ── Comp 2: Multi-Model Cross-Validation ──────────────────────────

  @comp-cross-validate
  Scenario: Flash and Pro agree on verdict
    Given a finding needs verification
    When cross-validate runs the verifier on Flash (effort=high) and Pro (effort=max)
    And both return CONFIRMED
    Then the verdict is CONFIRMED with confidence "high (cross-model agreement)"
    And the Pro output is returned as the authoritative result

  @comp-cross-validate
  Scenario: Flash and Pro disagree
    Given a finding needs verification
    When Flash returns CONFIRMED and Pro returns REFUTED
    Then the verdict is REFUTED (Pro wins as authoritative)
    And a dissent notice is included: "⚠ Flash dissent: Flash confirmed this finding. Pro refuted it."
    And the finding is flagged for human review

  @comp-cross-validate
  Scenario: Either model returns PLAUSIBLE
    Given either model returns PLAUSIBLE
    When the cross-validation gate evaluates
    Then the finding is re-verified with expanded evidence scope
    And a notice shows: "Re-verifying with broader scope — PLAUSIBLE from one model."

  # ── Comp 3: Post-Processing Passes ─────────────────────────────────

  @comp-postprocess-correctness
  Scenario: Correctness pass flags unsupported claims
    Given a subagent output claims "all 42 tests pass"
    But no test output was captured in the subagent's tool calls
    When the correctness pass runs
    Then the output is annotated: "[UNVERIFIED: 'all 42 tests pass' — no test output in tool calls]"
    And the annotation is visible in the parent's context

  @comp-postprocess-conciseness
  Scenario: Conciseness pass strips narration
    Given a subagent output is 800 tokens with 200 tokens of narration
    When the conciseness pass runs
    Then the output is reduced to ~640 tokens (20% reduction)
    And no factual content is removed
    And narration like "I then proceeded to examine the file" is stripped

  @comp-postprocess-completeness
  Scenario: Completeness pass flags missed parts
    Given the original task was "Find all callers of X and check Y"
    And the subagent output addresses "Find all callers of X" but not "check Y"
    When the completeness pass runs
    Then the output is annotated: "[MISSING: 'check Y' not addressed in this output]"
    And the parent agent can decide to re-run or supplement

  @comp-postprocess-cost
  Scenario: Post-processing cost is tracked per pass
    When all 3 passes run on a subagent output
    Then the cost is reported: "Post-processing: $0.0015 (correctness $0.0005 + conciseness $0.0005 + completeness $0.0005)"
    And the token savings from conciseness are reported: "Saved ~160 tokens downstream"

  # ── Comp 4: Confidence Calibration ─────────────────────────────────

  @comp-confidence-profile
  Scenario: Accuracy profile builds over time
    Given an explorer subagent has completed 50 Go exploration tasks
    And 46 were confirmed accurate by subsequent verification
    When the profile is queried
    Then the profile shows: "Explorer · DeepSeek Flash · Go: 92% accuracy (46/50 confirmed, last updated 2026-07-01)"
    And the profile is persisted to .reasonix/confidence/profiles.json

  @comp-confidence-routing
  Scenario: Subagent routed to most accurate profile
    Given a Go exploration task
    And profile shows: Flash/Go: 92% accuracy, Pro/Go: 88% accuracy
    When the explorer subagent is spawned
    Then it routes to Flash (92% > 88%)
    And a notice shows: "Routing explorer to Flash (92% Go accuracy vs Pro 88%)"

  @comp-confidence-cold-start
  Scenario: No profile yet falls back to default routing
    Given no accuracy profile exists for Python exploration
    When a Python exploration task is spawned
    Then the default routing is used (Flash, per effort calibration table)
    And a notice shows: "No accuracy profile for Python yet — using default routing. Profile will build over time."

  # ── Comp 5: Prompt Regression Bisection ───────────────────────────

  @comp-bisect
  Scenario: Bisection identifies the breaking prompt change
    Given explorer prompt degraded from v2 (5/5) to v5 (2/5)
    When bisection runs
    Then intermediate versions v3 and v4 are evaluated (5 runs each)
    And the results show: "v4 introduced regression — 'be thorough' caused over-exploration"
    And v5 is rolled back to v3 + v5's non-breaking improvements
    And the restored prompt scores 5/5

  @comp-bisect
  Scenario: Bisection cost is tracked
    Given bisection evaluated v3 and v4 (5 runs each across 4 scenarios)
    When the bisection completes
    Then the cost is reported: "Bisection: $0.06 (40 evaluations) — identified regression in v4 line 12"

  # ── Comp 6: Consensus Weighting ───────────────────────────────────

  @comp-weighted-voting
  Scenario: Weighted vote changes the consensus
    Given 3 verifiers voted on a Go concurrency finding
    And Verifier A (Go accuracy 94%) voted CONFIRMED
    And Verifier B (Go accuracy 78%) voted REFUTED
    And Verifier C (Go accuracy 91%) voted CONFIRMED
    When weighted voting computes the consensus
    Then the weighted score is CONFIRMED: 2.63 vs REFUTED: 0.78
    And the result shows both weighted and unweighted counts

  @comp-weighted-voting
  Scenario: Weighted vote is too close to call
    Given weighted scores are 2.1 vs 1.9 (within 0.3)
    When the consensus gate evaluates
    Then the finding is escalated: "Consensus too close (2.1 vs 1.9). Re-running with 2 additional skeptics."

  # ── Comp 7: Self-Healing Prompt Library ───────────────────────────

  @comp-prompt-library
  Scenario: Model version bump auto-selects best prompt
    Given DeepSeek releases a new model version
    And 3 prompt variants exist for the explorer role (v1: 85%, v2: 92%, v3: 89%)
    When the library evaluates all variants against the new model
    Then v2 is selected (highest pass rate at 92%)
    And the active prompt switches to v2 automatically
    And a notice shows: "Model version bump detected. Selected explorer v2 (92%) — best of 3 variants."

  @comp-prompt-library
  Scenario: No variant meets threshold triggers recalibration
    Given a model version bump occurred
    And the best variant scores 78% (below 80% threshold)
    When the library evaluates
    Then a notice shows: "No prompt variant meets threshold (best: 78%). Triggering recalibration."
    And the prompt calibration workflow (Phase A-G) is triggered

  # ── Comp 8: Context Janitor ────────────────────────────────────────

  @comp-janitor
  Scenario: Janitor compacts 5 turns semantically
    Given a session has accumulated 5 turns of exploration with 3 dead ends
    When the janitor subagent runs
    Then the output replaces the 5 turns with a compacted summary
    And dead ends (explored path, found nothing, changed direction) are dropped
    And key findings are preserved with citations
    And decisions that were reversed keep only the final decision
    And the compacted output is ≤40% of the original token count

  @comp-janitor
  Scenario: Janitor cost is net-positive
    Given the janitor costs $0.001 to run
    And it saves 800 tokens of context that would be carried forward
    And those 800 tokens would cost $0.0007 at DeepSeek rates if re-read
    When the net token savings are calculated
    Then the report shows: "Janitor: $0.001 spent, ~$0.005 saved (5 future turns × 800 fewer tokens read per turn)"
    And the net savings per janitor run is approximately $0.004

  # ── Comp 9: Pre-Mortem Analysis ────────────────────────────────────

  @comp-premortem
  Scenario: Pre-mortem catches a blind spot
    Given a security audit workflow is about to run on auth.go, middleware.go, handlers.go
    When the pre-mortem subagent runs
    And it predicts: "Verifiers might miss SQL injection in ORM-generated queries"
    Then the finder prompts are augmented with: "Pre-mortem: pay special attention to ORM-generated queries for SQL injection"
    And the pre-mortem output is logged for post-workflow review

  @comp-premortem
  Scenario: Pre-mortem finds no concerns
    Given a simple rename-refactor workflow
    When the pre-mortem subagent runs
    And it predicts no significant failure modes
    Then the output is: "Pre-mortem: no significant failure modes identified."
    And the workflow proceeds normally

  @comp-premortem
  Scenario: Pre-mortem suggestions are verifiable post-workflow
    Given a pre-mortem predicted "Verifiers might miss X"
    When the workflow completes
    Then a post-workflow check compares actual failures against pre-mortem predictions
    And the report shows: "Pre-mortem predicted 3 risks. 1 materialized (verifier missed X). 2 did not."

  # ── Comp 10: Semantic Diff ─────────────────────────────────────────

  @comp-semantic-diff
  Scenario: Semantic diff identifies agreements and disagreements
    Given 3 explorers returned findings on the same query
    When the semantic diff subagent runs
    Then agreements (all 3 found) are marked CONFIRMED
    And 2/3 agreements are marked LIKELY with the dissenter noted
    And solo findings (only 1 found) are marked POSSIBLE
    And direct contradictions (Agent A says X, Agent B says NOT X) are flagged ESCALATE

  @comp-semantic-diff
  Scenario: Semantic diff output is more useful than raw merge
    Given a raw merge would produce 25 undifferentiated findings
    When semantic diff processes them
    Then the output is structured: "12 CONFIRMED · 7 LIKELY · 4 POSSIBLE · 2 CONTRADICTIONS (escalated)"
    And the parent agent can prioritize confirmed findings

  # ── Configuration ─────────────────────────────────────────────────

  @comp-config
  Scenario: Each compensation is independently toggleable
    Given the user wants speculative execution but not cross-validation
    When the config is set:
      [compensations]
      speculative_execution = true
      cross_validate = false
      post_process = ["correctness"]
    Then only speculative execution and correctness post-processing run
    And cross-validation, conciseness, and completeness do not run

  @comp-config
  Scenario: All compensations disabled has zero overhead
    Given all compensations are disabled
    When a subagent runs
    Then no extra LLM calls are made
    And latency is identical to uncompensated mode
    And cost tracking shows zero compensation spend

  # ── FE ─────────────────────────────────────────────────────────────

  @comp-fe-panel
  Scenario: Compensation activity shown in background panel
    Given speculative execution is active with n=3
    When the background panel renders
    Then the subagent row shows: "explore:auth 🔍×3"
    And hovering shows: "Speculative execution: 3 explorers · 2/3 consensus threshold"
    And completed explorers show their merged result

  @comp-fe-cross-validate
  Scenario: Cross-validation dissent shown in findings
    Given a finding was cross-validated and Flash dissented
    When the finding renders
    Then it shows: "CONFIRMED (Pro) ⚠ Flash dissent"
    And clicking the dissent indicator shows Flash's rationale

  @comp-fe-cost-breakdown
  Scenario: Compensation cost shown in context panel
    When the user opens the context panel
    Then a "Compensations" section shows: total spent on compensations, per-compensation breakdown, estimated savings from compensations
    And the net shows: "Compensations: $0.04 spent · est. $0.18 saved from catches · net +$0.14 value"
