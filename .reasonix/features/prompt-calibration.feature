Feature: Prompt Calibration & Evaluation
  As a developer shipping subagent prompts
  I want to evaluate prompts against standard scenarios and calibrate them to ≥90% pass rate
  So that every shipped prompt is proven effective before any user sees it

  Background:
    Given reasonix has a prompt evaluation framework in internal/prompt/eval.go
    And standard eval scenarios exist in .reasonix/prompts/eval/

  # ── Evaluation Harness ───────────────────────────────────────────

  @prompt-eval-basic
  Scenario: Run eval for a single role and get pass/fail results
    When the user runs "reasonix prompt eval explorer"
    Then all 4 explorer scenarios run against the live DeepSeek API
    And each scenario reports: name, passed/failed, expectations met/unmet, tokens used, duration
    And a summary shows: "3/4 passed (75%) · $0.003 · 12.4s"
    And failed scenarios show which expectations failed

  @prompt-eval-basic
  Scenario: Run eval with mock provider for CI
    When the user runs "reasonix prompt eval explorer --mock"
    Then the mock provider simulates DeepSeek responses
    And the eval completes without network access
    And results are deterministic (same run → same output)

  @prompt-eval-basic
  Scenario: Run eval for a specific scenario
    When the user runs "reasonix prompt eval explorer --scenario find-single-caller"
    Then only that scenario runs
    And the output includes the full subagent transcript

  @prompt-eval-basic
  Scenario: Run eval and archive results
    When the user runs "reasonix prompt eval explorer --archive"
    Then results are written to .reasonix/prompts/results/v1-YYYY-MMDD-HHMM.json
    And the archive includes: timestamp, model, effort, all scenario results, prompt version

  # ── Expectation Types ────────────────────────────────────────────

  @prompt-eval-expectations
  Scenario: tool_used expectation validates tool was called
    Given a scenario expects {"kind": "tool_used", "value": "grep"}
    When the subagent calls grep during the run
    Then the expectation passes
    When the subagent never calls grep
    Then the expectation fails with: "Expected tool 'grep' was never used"

  @prompt-eval-expectations
  Scenario: tool_not_used expectation validates tool was not called
    Given a scenario expects {"kind": "tool_not_used", "value": "write_file"}
    When the subagent never calls write_file
    Then the expectation passes
    When the subagent calls write_file
    Then the expectation fails with: "Forbidden tool 'write_file' was used"

  @prompt-eval-expectations
  Scenario: output_contains expectation validates text in final answer
    Given a scenario expects {"kind": "output_contains", "value": "file:"}
    When the final answer contains "auth.go:42 — nil pointer dereference"
    Then the expectation passes (contains "file:")

  @prompt-eval-expectations
  Scenario: output_not_contains expectation validates text absent
    Given a scenario expects {"kind": "output_not_contains", "value": "I recommend"}
    When the final answer does not contain "I recommend"
    Then the expectation passes
    When the final answer says "I recommend refactoring this"
    Then the expectation fails with: "Forbidden text 'I recommend' found in output"

  @prompt-eval-expectations
  Scenario: stops_within_n_calls enforces tool call budget
    Given a scenario expects {"kind": "stops_within_n_calls", "count": 10}
    When the subagent stops after 8 tool calls
    Then the expectation passes
    When the subagent makes 12 tool calls
    Then the expectation fails with: "Exceeded max calls: 12 > 10"

  @prompt-eval-expectations
  Scenario: no_narration expectation validates silence between tool calls
    Given a scenario expects {"kind": "no_narration"}
    When the subagent makes 3 read_file calls with zero text between them
    Then the expectation passes
    When the subagent writes "Now I'll read the next file" between reads
    Then the expectation fails with: "Narration detected: 'Now I'll read the next file' between tool calls"

  @prompt-eval-expectations
  Scenario: output_matches validates against regex
    Given a scenario expects {"kind": "output_matches", "value": ".*\\.go:\\d+.*"}
    When the final answer contains "auth.go:42"
    Then the expectation passes

  # ── Calibration Workflow ─────────────────────────────────────────

  @prompt-eval-calibration
  Scenario: Calibration loop reports iteration-over-iteration improvement
    Given the explorer prompt is at v2 with 75% pass rate
    When the user edits the prompt to v3 and runs eval
    Then the results show: "v3: 4/4 (100%) · ↑ from v2 (75%)"
    And the improvement is highlighted in green

  @prompt-eval-calibration
  Scenario: Calibration threshold gates shipping
    Given the reviewer prompt has 67% pass rate
    When a developer attempts to mark it as "shipped"
    Then the gate blocks: "Reviewer prompt: 2/3 passed (67%). Threshold is 90%. Fix failing scenarios before shipping."
    And the failing scenarios are listed

  @prompt-eval-calibration
  Scenario: Shipping a prompt with ≥90% pass rate
    Given the explorer prompt has 100% pass rate across 4 scenarios
    When the developer marks it as shipped
    Then the prompt is copied to .reasonix/prompts/v1/explorer.md
    And a CHANGELOG entry is appended
    And the shipped prompt is loaded by reasonix on next start

  @prompt-eval-calibration
  Scenario: Prompt version rollback
    Given v3 of the explorer prompt was shipped
    And v3 is found to have a regression (users report missed findings)
    When the developer runs "reasonix prompt rollback explorer --to v2"
    Then v2 is restored as the active prompt
    And v3 eval results are annotated with the rollback reason
    And the CHANGELOG records the rollback

  # ── FE: Prompt Eval UI ───────────────────────────────────────────

  @prompt-eval-fe
  Scenario: Eval results render as a pass/fail table
    When "reasonix prompt eval explorer" completes
    Then a table renders with columns: Scenario, Status, Expectations Met, Tokens, Duration
    And passed rows are green, failed rows are red
    And the summary row shows overall pass rate and cost

  @prompt-eval-fe
  Scenario: Failed expectation renders with context
    Given a scenario failed because "no_narration" was violated
    When the results render
    Then the failure shows: "✗ no_narration — Narration detected: 'Now I'll read the config file'"
    And the surrounding transcript lines are shown for context
    And the prompt text that should have prevented this is highlighted

  @prompt-eval-fe
  Scenario: Prompt diff view when iterating
    Given the user edited the prompt from v2 to v3
    When the eval results render
    Then a "View diff from v2" link is available
    And the diff shows the exact lines changed between versions
    And the eval result delta is shown next to the diff

  @prompt-eval-fe
  Scenario: Prompt eval shows cost per iteration
    When eval results render
    Then each scenario run shows: "12.4s · $0.0008 · 15K in, 3K out, 2K reasoning"
    And the total cost across all runs is shown in the summary
    And a running total is shown during execution: "Running scenario 2/4... $0.0012 so far"

  @prompt-eval-fe
  Scenario: Prompt eval progress renders during run
    When "reasonix prompt eval explorer" is running
    Then a progress bar shows: "Evaluating explorer: [██░░░░] 2/4 scenarios · find-single-caller ✓ · negative-claim..."
    And each scenario updates live as it completes
    And the status icon changes from spinner to ✓ or ✗
