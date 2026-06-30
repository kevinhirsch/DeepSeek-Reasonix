Feature: Dynamic Workflows
  As a developer auditing code
  I want the model to compose multi-stage subagent workflows dynamically
  So that "find bugs → verify → report" happens in one tool call with structured results

  Background:
    Given reasonix has the workflow tool registered
    And DeepSeek V4-Flash is configured for finders
    And DeepSeek V4-Pro is configured for verifiers

  @05-pipeline
  Scenario: Pipeline strategy runs stages independently
    When the model calls workflow with:
      """
      strategy: pipeline
      stages:
        - name: find, role: review, prompt_template: "Review ${item} for bugs"
        - name: verify, role: verify, prompt_template: "Adversarially verify: ${prev}", items_from: prev, fan_out: 3
        - name: report, role: execute, prompt_template: "Synthesize: ${results}", items_from: prev
      items: ["auth.go", "middleware.go"]
      """
    Then finders for auth.go and middleware.go start concurrently
    And each finding flows into the verify stage independently as soon as it completes
    And stage-3 synthesizer starts as soon as all verifications complete
    And each subagent's events are nested under the workflow call

  @05-barrier
  Scenario: Barrier strategy waits for all items before next stage
    When the model calls workflow with strategy "barrier" and 3 stages
    Then no stage-2 subagent starts until all stage-1 subagents complete
    And no stage-3 subagent starts until all stage-2 subagents complete

  @05-loop-until-dry
  Scenario: Loop-until-dry stops after two dry rounds
    When the model calls workflow with:
      """
      strategy: loop_until_dry
      dry_rounds: 2
      stages:
        - name: find, role: explore, prompt_template: "Find undocumented public functions in ${item}"
      items: ["pkg/"]
      """
    And round 1 finds 5 undocumented functions
    And round 2 finds 2 new functions (not seen in round 1)
    And round 3 finds 0 new functions
    And round 4 finds 0 new functions
    Then the workflow stops after round 4
    And the result contains all 7 unique findings

  @05-voting
  Scenario: Adversarial verification with fan-out and voting
    When the model calls workflow with:
      """
      stages:
        - name: verify, role: verify, prompt_template: "Verify: ${prev}", items_from: prev, fan_out: 3, voting: {required: 2}
      """
    And the fan-out spawns 3 skeptic subagents for one finding
    And 2 skeptics return CONFIRMED, 1 returns REFUTED
    Then the finding is marked CONFIRMED (2 ≥ threshold)
    And the dissenting REFUTED is included in the output

  @05-voting
  Scenario: Finding rejected when votes below threshold
    When fan_out: 3 and voting: {required: 2}
    And only 1 skeptic returns CONFIRMED, 2 return REFUTED
    Then the finding is marked REFUTED

  @05-precision-mode
  Scenario: Precision verification uses three finder angles and one-vote
    When the model calls workflow with verification mode "precision"
    Then exactly one verifier is spawned per finding
    And the verifier runs at medium effort
    And findings are classified as CONFIRMED or REFUTED (no PLAUSIBLE)

  @05-recall-mode
  Scenario: Recall verification uses recall-biased classification
    When the model calls workflow with verification mode "recall"
    Then verifiers run at high effort
    And findings that are uncertain but not refuted are classified as PLAUSIBLE
    And PLAUSIBLE findings are included in the output alongside CONFIRMED

  @05-max-tasks
  Scenario: Max total tasks cap enforced
    When the model calls workflow with max_total_tasks: 10
    And the workflow would spawn 15 subagents across all stages
    Then only 10 subagents are spawned
    And the remaining items are reported as unprocessed

  # ── FE: Workflow Visualization ────────────────────────────────────

  @05-fe-workflow-card
  Scenario: Workflow card shows stage progress as a mini bar chart
    Given a 3-stage workflow with find (3/3 done), verify (5/9 running), report (pending)
    When the workflow card renders in the transcript
    Then a progress bar shows: "find ████████████ 3/3 · verify ██████░░░░░░ 5/9 · report ░░░░░░░░░░░░ pending"
    And each bar segment is colored by stage status: green (done), blue (running), gray (pending)
    And the card header shows: "Workflow: security-audit · pipeline · 3 stages · 15 subagents"

  @05-fe-workflow-voting-card
  Scenario: Voting results render inline with visual breakdown
    Given a finding received 2 CONFIRMED votes and 1 REFUTED from 3 skeptics
    When the finding renders in the workflow output
    Then the verdict is shown with the vote breakdown: "CONFIRMED — ✓✓✗ (2/3)"
    And the 3 skeptic subagent refs are shown
    And clicking a skeptic opens its transcript in peek mode

  @05-fe-workflow-pipeline-flow
  Scenario: Pipeline mode shows items visually flowing
    Given a pipeline workflow with 3 items flowing from find → verify
    When item "auth.go" completes finding and spawns 3 verifiers
    Then the workflow card updates to show "auth.go → verify (3 verifiers spawned)"
    And the transition is animated: a brief highlight on "auth.go" as it moves stages

  @05-fe-workflow-cancel-ui
  Scenario: Cancelling workflow shows confirmation and graceful shutdown
    Given a workflow is running with 8 of 15 subagents active
    When the user clicks "Cancel workflow"
    Then a confirmation dialog appears: "Cancel 'security-audit'? 8 subagents will be stopped. Completed results will be saved."
    And confirming shows: "Cancelling... waiting for subagents to stop..."
    And the final state shows: "Workflow cancelled — 4 stages completed, 8 subagents stopped, results saved"
    And a "Resume" button is available
