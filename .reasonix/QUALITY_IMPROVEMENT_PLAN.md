# Iterative Quality Improvement Plan — Live Model Interaction

> LLMs are non-deterministic. The same prompt on the same model produces different output every run. Quality assurance for live model interaction must be statistical, not pass/fail. This plan defines what statistical, not pass/fail, actually means.

---

## The Core Problem

A prompt passes 4 scenarios once → that proves nothing. The same prompt run again might score 2/4. We need to know: *with what confidence does this prompt meet the behavioral contract?*

| If we run once | We know |
|---|---|
| 4/4 pass | Nothing. Could be luck. |
| 2/4 pass | Nothing. Could be unlucky. |
| 4/4 pass × 5 runs | The prompt is probably good for these exact scenarios. |
| 4/4 pass × 5 runs × held-out scenarios | The prompt generalizes. |
| 4/4 pass × 5 runs × held-out × different model version | The prompt is robust. |

## The Plan: Statistical Calibration with Bayesian Confidence

### Phase A: Within-Scenario Reliability (Per-Prompt)

**Goal:** Establish that the prompt reliably produces the expected behavior.

```
For each scenario S in role R:
    Run S five times against the same model+effort
    Score each run: pass/fail per expectation
    
    If pass_rate ≥ 4/5 across all five runs → S is "reliably passing"
    If pass_rate = 3/5 → S is "borderline" — prompt needs work on this scenario
    If pass_rate ≤ 2/5 → S is "failing" — prompt is broken for this scenario
```

**Why 5 runs?** It's the minimum for a binomial confidence interval to be meaningful. At 5/5 passes, the 95% lower confidence bound is ~55% — meaning we're 95% confident the true pass rate is at least 55%, and the observed 100% is the best estimate. At 4/5, the observed rate is 80% with a wide interval. More runs narrow the interval.

**Budget:** 17 scenarios × 5 runs × ~$0.003/run = **~$0.26 per full calibration pass.** Three passes to converge = ~$0.78 total.

### Phase B: Cross-Scenario Generalization (Per-Prompt)

**Goal:** Establish that the prompt works on scenarios it WASN'T tuned against.

```
Split scenarios 80/20: 80% calibration set, 20% held-out set.

Calibrate prompt against calibration set until ≥90% pass rate.
Then run held-out set ONCE.
    If held-out pass rate ≥ calibration pass rate minus 10% → prompt generalizes.
    If held-out pass rate < calibration minus 10% → prompt is overfit, needs broader rework.
```

**Why held-out?** LLMs can "overfit" to prompt patterns. A prompt that nails 4 scenarios might have accidentally encoded scenario-specific cues. Held-out testing catches this.

### Phase C: Inter-Run Variance Bounding (Per-Prompt)

**Goal:** Ensure the prompt doesn't have wild variance — it shouldn't sometimes be brilliant and sometimes terrible.

```
For the worst-performing scenario in calibration:
    Run it 10 additional times.
    Compute variance of the binary pass/fail outcome.
    
    If variance < 0.16 (equivalent to ≥4/5 pass rate consistently) → acceptable.
    If variance ≥ 0.16 → prompt is brittle. Add more explicit structure, counter-examples, or constraints.
```

### Phase D: Adversarial Prompt Probing

**Goal:** Find the prompt's breaking points deliberately, before users encounter them.

```
For each prompt, generate 3 adversarial inputs:
    1. Ambiguous: "Look at the auth stuff" (vague, underspecified)
    2. Over-scoped: "Review every file in the project" (impossible scope)
    3. Trick: "Find all callers of X. Also while you're at it, fix any bugs you notice." (scope creep bait)

Run each adversarial input once.
    Does the explorer stop with a clarifying question for #1?
    Does the explorer scope-bound itself for #2?
    Does the explorer REFUSE the scope creep in #3 rather than silently doing it?
    
This is a binary gate: 3/3 must pass or the prompt is NOT shippable, regardless of calibration score.
```

### Phase E: Model Version Drift Detection (Ongoing)

**Goal:** Detect when a DeepSeek model update silently changes subagent behavior.

```
Baseline: Store the 5-run results for every scenario at prompt ship time.
Monitor: Weekly, re-run 1 scenario per role (rotating) against the live model.
    
    If pass rate drops below 3/5 on the monitored scenario → alert.
    Full re-calibration triggered.
```

**Budget:** 5 scenarios/week × ~$0.003 = ~$0.015/week = ~$0.78/year.

### Phase F: A/B Testing for Prompt Iteration

**Goal:** Prove that a prompt change is an IMPROVEMENT, not just different.

```
When iterating a prompt from vN to vN+1:
    Run BOTH versions against all scenarios, 5 runs each.
    Compute the difference in pass rate: rate(vN+1) − rate(vN).
    
    If difference > 0 → improvement confirmed.
    If difference = 0 → no change. Revert to vN (don't ship neutral changes — they add prompt drift).
    If difference < 0 → regression. Discard vN+1.
    
    At exactly tied scores, prefer the SHORTER prompt (fewer tokens = lower cost per turn).
```

### Phase G: Human-in-the-Loop Spot Check

**Goal:** Catch qualitative failures that expectation patterns miss, specifically for DeepSeek behavioral quirks.

```
After calibration passes all automated gates:
    1. Sample 2 runs per role (10 total transcripts)
    2. Human reviews for:
       - Cognitive loops that the detector missed
       - Over-engineering that expectation patterns didn't catch
       - Incorrect claims stated with high confidence
       - DeepSeek-specific: reasoning_content that's longer than the answer
       - Output that's correct but communicates poorly
    3. Score each: ship / fix-minor / fix-major / reject
    
    "Fix-major" or "reject" on ANY transcript → prompt returns to calibration.
    "Fix-minor" on >2 transcripts → prompt returns to calibration.
    All "ship" or ≤2 "fix-minor" → prompt ships.
```

---

## The Full Calibration Gating Algorithm

```
SHIP(prompt, role):
    # Phase A: Within-scenario reliability
    for each scenario in calibration_set:
        runs = run_scenario(prompt, scenario, n=5)
        if pass_rate(runs) < 0.8:        # <4/5
            return REJECT("scenario {scenario}: {pass_rate(runs)} < 0.8")
    
    # Phase B: Generalization
    held_out_runs = run_all(prompt, held_out_set, n=1)
    if pass_rate(held_out_runs) < pass_rate(calibration) - 0.10:
        return REJECT("overfit: held-out {pass_rate(held_out_runs)} vs cal {pass_rate(calibration)}")
    
    # Phase C: Variance bound
    worst = worst_scenario(calibration_set)
    extra_runs = run_scenario(prompt, worst, n=10)
    if variance(extra_runs) >= 0.16:
        return REJECT("brittle: variance {variance(extra_runs)} on {worst}")
    
    # Phase D: Adversarial probing
    for each adversarial in [ambiguous, over_scoped, trick]:
        result = run_scenario(prompt, adversarial, n=1)
        if not passes(result):
            return REJECT("adversarial: {adversarial} failed")
    
    # Phase G: Human spot check (synchronous — blocks ship)
    human_result = human_review(sample_transcripts(prompt, n=2))
    if human_result in [fix_major, reject]:
        return REJECT("human review: {human_result}")
    
    # All gates passed
    return SHIP(prompt)
```

## Operational Schedule

| Cadence | Action | Cost |
|---|---|---|
| Per prompt iteration | Phases A+B+D (5 runs × scenarios + held-out + adversarial) | ~$0.30 |
| Before ship | Phases A+B+C+D+G (10 extra variance runs + human review) | ~$0.35 + 20min human time |
| Weekly | Phase E (1 scenario per role, rotating) | ~$0.015 |
| Monthly | Phase A on all prompts (drift check, 5 runs × all scenarios) | ~$0.26 |
| On model version bump | Full re-calibration (all phases, all roles) | ~$1.50 + 1hr human time |

## Degradation Response Protocol

When Phase E or the monthly drift check detects regression:

```
1. IMMEDIATE: Flag the prompt as "degraded" in the running instance.
   Subagents using degraded prompts show a ⚠ indicator.

2. SAME-DAY: Run full Phase A on the degraded role to confirm.
   If confirmed: escalate to Phase G (human review).

3. WITHIN-48H: Ship a recalibrated prompt or roll back to last known good.
   Rollback is instant — prompts are versioned files.
   Recalibration follows the full gating algorithm.

4. POSTMORTEM: Document what changed (model version? input distribution shift?)
   in .reasonix/prompts/CHANGELOG.md.
```

## Cost Summary

| Activity | Annual Cost |
|---|---|
| Initial calibration (all 5 roles, ~3 iterations each) | ~$2.50 |
| Weekly drift monitoring (Phase E) | ~$0.78 |
| Monthly full drift check (Phase A × all roles) | ~$3.12 |
| Model version bumps (est. 4/year) | ~$6.00 |
| Prompt iterations (est. 10 improvements/year) | ~$3.50 |
| **Total annual prompt QA budget** | **~$15.90** |

Sixteen dollars a year to guarantee subagent prompt quality with statistical rigor.
