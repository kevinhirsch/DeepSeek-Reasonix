Feature: Iterative Quality Improvement for Live Model Interaction
  As a developer calibrating subagent prompts against non-deterministic LLMs
  I want statistically rigorous quality gates that account for output variance
  So that every shipped prompt is proven reliable, not just lucky on one run

  Background:
    Given reasonix has the prompt evaluation framework from Issue #18
    And statistical calibration gates are enforced by the shipping algorithm

  # ── Phase A: Within-Scenario Reliability ─────────────────────────

  @quality-statistical-reliability
  Scenario: Prompt passes all scenarios across 5 runs and is marked reliable
    Given the explorer prompt is being calibrated
    And 4 scenarios exist in the calibration set
    When each scenario is run 5 times
    And all 4 scenarios achieve ≥4/5 pass rate
    Then the prompt passes Phase A
    And the result shows per-scenario pass rates: "3/4 scenarios at 5/5, 1/4 at 4/5"

  @quality-statistical-reliability
  Scenario: Prompt fails a scenario at 2/5 and is rejected
    Given the reviewer prompt scores 2/5 on the "negative-claim-is-evidenced" scenario
    When Phase A completes
    Then the prompt is rejected with: "Phase A FAILED: 'negative-claim-is-evidenced' pass rate 0.40 < 0.80 threshold"
    And the failing runs' transcripts are available for inspection
    And the prompt returns to iteration

  @quality-statistical-reliability
  Scenario: Borderline scenario at 3/5 triggers a warning but does not fail
    Given the verifier prompt scores 3/5 on "refute-with-counterevidence"
    When Phase A completes
    Then the prompt is flagged: "BORDERLINE: 'refute-with-counterevidence' at 0.60 — within threshold but brittle"
    And the prompt proceeds to Phase B with the warning attached

  # ── Phase B: Cross-Scenario Generalization ───────────────────────

  @quality-generalization
  Scenario: Held-out scenarios validate generalization
    Given the explorer prompt achieved 95% on the calibration set (4 scenarios)
    And 1 held-out scenario was reserved
    When the held-out scenario is run once
    And it passes
    Then the prompt passes Phase B
    And the result shows: "Held-out: 1/1 (100%) — within 10% of calibration (95%) → generalizes"

  @quality-generalization
  Scenario: Held-out failure indicates overfitting
    Given the explorer prompt achieved 100% on calibration (4 scenarios)
    And 1 held-out scenario was reserved
    When the held-out scenario fails
    Then the prompt is rejected with: "Phase B FAILED: held-out 0/1 (0%) vs calibration 1.0 — difference 1.0 > 0.10 threshold → overfit"
    And the prompt must be reworked to remove scenario-specific cues

  # ── Phase C: Inter-Run Variance Bounding ──────────────────────────

  @quality-variance
  Scenario: Low-variance prompt passes Phase C
    Given the explorer prompt had its worst scenario at 4/5
    When 10 additional runs are executed on that scenario
    And the variance of pass/fail across all 15 runs (5+10) is 0.12
    Then the prompt passes Phase C
    And the result shows: "Variance: 0.12 < 0.16 threshold — prompt is stable"

  @quality-variance
  Scenario: High-variance prompt fails Phase C
    Given the verifier prompt had its worst scenario at 3/5
    When 10 additional runs are executed
    And the variance across all 15 runs is 0.22
    Then the prompt is rejected with: "Phase C FAILED: variance 0.22 > 0.16 threshold — prompt is brittle. Add explicit structure or counter-examples."
    And the alternating pass/fail pattern is highlighted in the results

  # ── Phase D: Adversarial Probing ─────────────────────────────────

  @quality-adversarial
  Scenario: Prompt handles ambiguous input correctly
    Given the explorer prompt is being probed adversarially
    When the input is "Look at the auth stuff" (vague, underspecified)
    Then the subagent asks a clarifying question OR scopes its own search and explains
    And the subagent does NOT guess and silently search a wrong scope

  @quality-adversarial
  Scenario: Prompt handles impossible scope correctly
    Given the explorer prompt is being probed adversarially
    When the input is "Review every file in the project" (impossible scope)
    Then the subagent scope-bounds itself: "This project has 500 files. I'll review the 10 most recently modified."
    And the subagent does NOT attempt to read all 500 files

  @quality-adversarial
  Scenario: Prompt refuses scope creep
    Given the explorer prompt is being probed adversarially
    When the input is "Find all callers of X. Also while you're at it, fix any bugs you notice."
    Then the subagent handles the find task AND explicitly states: "Bug fixing is outside my scope as an explorer. I've noted potential issues but not modified any code."
    And the subagent does NOT silently start editing files

  @quality-adversarial
  Scenario: Failing any adversarial probe blocks shipping
    Given the explorer prompt passed Phases A, B, and C
    And it failed the "impossible scope" adversarial probe (attempted to read all files)
    When the shipping gate evaluates
    Then the prompt is REJECTED regardless of calibration scores
    And the rejection message states: "Adversarial probe 'impossible scope' failed — prompt does not scope-bound. Fix before shipping."

  # ── Phase E: Model Version Drift Detection ───────────────────────

  @quality-drift
  Scenario: Weekly drift check detects no degradation
    Given the explorer prompt was shipped with pass rates: 5/5, 5/5, 4/5, 5/5
    When the weekly drift check runs one randomly selected scenario
    And it scores 5/5
    Then no alert is raised
    And the result is logged: "Drift check: explorer/find-single-caller → 5/5 ✓"

  @quality-drift
  Scenario: Weekly drift check detects regression
    Given the shipped explorer prompt had 5/5 on "find-single-caller"
    When the weekly drift check runs "find-single-caller" and scores 2/5
    Then an alert is raised: "DRIFT DETECTED: explorer/find-single-caller dropped from 5/5 (shipped) to 2/5 (current)"
    And the prompt is marked "degraded" in the running instance
    And subagents using this prompt show a ⚠ indicator in the background panel

  @quality-drift
  Scenario: Degradation response protocol triggers full recalibration
    Given a drift alert was raised for the explorer prompt
    When the response protocol activates
    Then within the same day, full Phase A runs on the explorer role
    And if degradation is confirmed, Phase G (human review) is triggered
    And within 48 hours, either a recalibrated prompt ships or the previous good version is rolled back

  # ── Phase F: A/B Testing ─────────────────────────────────────────

  @quality-ab-test
  Scenario: Improved prompt wins A/B comparison
    Given the current explorer prompt v2 has pass rate 80% (4/5 avg)
    And the candidate v3 has pass rate 95% (4.75/5 avg)
    When A/B testing runs both versions 5 times each
    Then v3 is confirmed as an improvement: "v3: 0.95 vs v2: 0.80 — +0.15 improvement"
    And v3 is accepted for Phase G

  @quality-ab-test
  Scenario: Neutral change is rejected
    Given v2 and v3 have identical 80% pass rates across 5 runs
    And v3 is 200 tokens longer than v2
    When A/B testing completes
    Then v3 is rejected: "No improvement (+0.00). Preferring v2 (shorter — saves 200 tokens/turn)."
    And v3 is discarded

  @quality-ab-test
  Scenario: Regression is caught
    Given v2 has 90% pass rate and v3 has 60%
    When A/B testing runs
    Then v3 is rejected: "Regression: v3 0.60 vs v2 0.90 — -0.30. Discarding v3."
    And v3 results are archived for reference

  # ── Phase G: Human-in-the-Loop Spot Check ────────────────────────

  @quality-human-review
  Scenario: All transcripts pass human review and prompt ships
    Given the explorer prompt passed all automated phases
    When 2 transcripts are sampled and reviewed by a human
    And both are scored "ship"
    Then the prompt ships
    And the CHANGELOG records: "v3 shipped — 95% pass rate, 2/2 human review"

  @quality-human-review
  Scenario: One transcript scored fix-major blocks shipping
    Given the verifier prompt passed automated phases
    When 2 transcripts are reviewed
    And one is scored "fix-major" (claims "confirmed" but evidence contradicts)
    Then the prompt is REJECTED
    And the rejection message includes the specific transcript and the issue
    And the prompt returns to iteration with a note: "Fix: verifier confirms claims on contradictory evidence"

  @quality-human-review
  Scenario: Two fix-minor scores at threshold
    Given the explorer prompt had 2 transcripts scored "fix-minor" (minor communication issues)
    When the human review gate evaluates
    Then the prompt is rejected: "2 fix-minor scores — threshold is ≤2. Fix minor issues and re-submit."
    And the specific issues are listed for the prompt author

  @quality-human-review
  Scenario: DeepSeek-specific issues caught in human review
    Given a DeepSeek explorer transcript
    When the human reviews it
    And the reasoning_content is 3× longer than the answer (cognitive bloat)
    Then this is scored "fix-minor" with note: "Excessive reasoning — 3× the answer length"
    And the prompt author adds: "Keep chain-of-thought under 50% of your final answer length"

  # ── Shipping Algorithm ────────────────────────────────────────────

  @quality-full-gate
  Scenario: Prompt passes all phases and ships
    Given a prompt passes Phase A (≥80% per scenario × 5 runs)
    And passes Phase B (held-out within 10%)
    And passes Phase C (variance < 0.16)
    And passes Phase D (all 3 adversarial probes)
    And passes Phase G (human review: all ship or ≤2 fix-minor)
    When the shipping algorithm evaluates
    Then the output is: "SHIP: explorer v3 — calibration 95%, held-out 100%, variance 0.08, adversarial 3/3, human 2/2 ship"
    And the prompt is written to .reasonix/prompts/v1/explorer.md
    And the CHANGELOG is updated
    And the running reasonix instance loads the new prompt

  @quality-full-gate
  Scenario: Prompt fails any phase and is blocked
    Given a prompt passes Phase A but fails Phase D
    When the shipping algorithm evaluates
    Then the output is: "BLOCKED: explorer v4 — Phase D adversarial 'scope creep' failed"
    And the prompt is NOT written to the prompts directory
    And the reasonix instance continues using the current shipped prompt
    And the failure is logged with a link to the failing transcript

  # ── FE: Quality Dashboard ─────────────────────────────────────────

  @quality-fe-dashboard
  Scenario: Prompt quality dashboard shows all roles at a glance
    When the user opens Settings → Prompts
    Then a dashboard shows per role: current version, shipped date, calibration pass rate, last drift check result, status
    And roles with degraded prompts show ⚠
    And roles with healthy prompts show ✓

  @quality-fe-calibration-progress
  Scenario: Calibration run shows phase-by-phase progress
    Given a full calibration is running for the explorer prompt
    When the progress renders
    Then it shows: "Phase A: ⠋ Running 20 evaluations (4 scenarios × 5 runs)... 15/20 complete"
    And each phase updates as it completes: "Phase A ✓ · Phase B ✓ · Phase C ⠋ · Phase D (pending) · Phase G (pending)"
    And the estimated remaining time is shown

  @quality-fe-drift-alert
  Scenario: Drift alert renders in the status line
    Given the explorer prompt has degraded
    When the status line renders
    Then it shows: "⚠ Explorer prompt degraded (2/5 vs 5/5 baseline)"
    And clicking opens the prompt dashboard
    And the alert color is yellow (not red — prompts still work, just degraded)
