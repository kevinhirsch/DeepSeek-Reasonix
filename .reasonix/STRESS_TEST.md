# Architecture Stress Test — Finding What Breaks

> What happens when everything is active simultaneously and pushed to its limits?

---

## Category 1: Compositional Explosions

### 1A: Multi-Compensation Token Multiplier

**Scenario:** User asks "Implement rate limiting for the API." All compensations active.

```
Pre-mortem (Comp 9):    1 explorer → $0.001
Self-play ×3 (Comp 14): 3 executors × $0.005 → $0.015
Post-processing (Comp 3): 3 passes per executor output → 9 passes × $0.0005 = $0.0045
Semantic diff (Comp 10):  1 pass → $0.0005
Cross-validate (Comp 2):  Flash+Pro on judge output → $0.004
Adversarial test (Comp 19): 1 tester + 1 fix loop → $0.008
Context janitor (Comp 8): 1 pass → $0.001
Ambient guardian (Comp 11): checks per file save during implementation → $0.003
Impact analysis (Comp 18): triggered by signature changes → $0.002

Total for ONE feature: $0.039
At Anthropic rates: $2.08 (53×)
```

**Gap:** No scenario tests the case where ALL compensations fire simultaneously. Each compensation was tested in isolation. We need a "maximum compensation blast radius" stress test.

### 1B: Competing Subagent Spawn Limits

**Scenario:** Workflow spawns 27 verifiers. Idle pre-compute spawns 5 explorers. Ambient guardian spawns 2 checkers. Context janitor spawns. Semantic diff spawns. Total: 36+ concurrent subagents.

**Gap:** `max_total_tasks` (default 50) is a per-workflow cap, not a system-wide cap. No global subagent concurrency limit. At 36 concurrent Flash subagents, you hit DeepSeek's rate limit of 10 req/s and 32 subagents queue behind each other.

### 1C: Compensation Cost Cascade

**Scenario:** Comp 9 (pre-mortem) predicts a risk → that risk materializes → Comp 1 (speculative) runs extra verifiers → Comp 3 (post-process) checks them all → Comp 8 (janitor) compacts the result. The compensations compound not just additively, but by TRIGGERING each other.

**Gap:** No scenario for "compensation A's output triggers compensation B, which triggers C." The cost model assumes additive costs, but in practice these trigger chains could multiply.

---

## Category 2: Timing & Ordering

### 2A: Save During Active Workflow

**Scenario:** User saves a file (triggers Ambient Guardian + Impact Analysis) WHILE a security audit workflow is running (27 parallel subagents). The Guardian and Impact Analysis are subagents that compete with the workflow for rate limit slots.

**Gap:** No priority model for subagent spawning. Should the Guardian queue behind the workflow's verifiers? Should it preempt them (it's a single quick check vs 27 long verifiers)? No answer in the current design.

### 2B: Idle Detection Race

**Scenario:** User stops typing for 1.9 seconds, then types one character, then stops again. The idle timer (2 second threshold) nearly fires, resets, nearly fires, resets... generating subagent spawn/despawn churn.

**Gap:** No debounce or hysteresis on the idle detection. A user who reads-and-occasionally-scrolls could trigger dozens of spawn/cancel cycles.

### 2C: Drift Detection During Calibration

**Scenario:** Weekly drift check (Phase E) detects the explorer prompt degraded from 5/5 to 2/5. It triggers recalibration. While recalibration is running, users are still using the degraded prompt. How long are they exposed?

**Gap:** No "instant rollback" mechanism. The self-healing library (Comp 7) evaluates variants and selects the best, but this takes ~$0.18 and ~30 seconds. During those 30 seconds, every explorer subagent is degraded.

---

## Category 3: State Corruption

### 3A: Knowledge Base Poisoning

**Scenario:** An explorer subagent returns a confident-sounding but WRONG analysis of the auth module. The analysis is embedded and stored in the KB. Every future question about auth retrieves this wrong answer. Over time, the KB accumulates incorrect analyses and becomes a liability.

**Gap:** No verification that KB entries are correct before storage. No staleness scoring beyond "was the file modified since analysis." An analysis can be factually wrong but not stale by git log.

### 3B: Confidence Profile Decay

**Scenario:** A verifier has 94% accuracy on Go (50 samples). DeepSeek releases a model update that silently changes the verifier's behavior. The 94% profile is now stale — the actual accuracy on the new model might be 70%. But the profile still says 94% until enough new samples accumulate.

**Gap:** No confidence profile expiration on model version bump. No decay function. A profile built on model v4 is applied to model v5 with no adjustment.

### 3C: Cross-Session Drift Invalidation

**Scenario:** User has 200 KB entries built over 3 months. DeepSeek updates their model. The KB entries were all produced by the OLD model's explorer subagents. The new model explores differently — different search patterns, different conclusions. The KB is now a museum of how the old model thought about the codebase.

**Gap:** KB entries are not tagged with the model version that produced them. On model bump, all entries appear equally valid. Some may be systematically wrong on the new model.

---

## Category 4: Provider Assumption Violations

### 4A: DeepSeek Price Increase

**Scenario:** DeepSeek raises V4-Flash from $0.14/M to $0.28/M — still cheap, but now compensations cost twice as much. The annual budget goes from $141 to $282. Is this still viable? At what price point does each compensation become uneconomical?

**Gap:** No cost-effectiveness threshold per compensation. No automatic "pause if cost exceeds X" per compensation. No alert on price change.

### 4B: DeepSeek Feature Removal

**Scenario:** DeepSeek removes the auto-cache feature (or changes its behavior). Our caching advantage evaporates. Cache hit rates drop from 99% to 0%. Every request costs the full input price. Our tokenomics model assumes 80%+ cache hit rates for the cost projections.

**Gap:** No cache degradation detection beyond the basic "hit rate < 80%" check. No scenario for "what if auto-cache stops working entirely." No fallback behavior.

### 4C: DeepSeek Model Behavior Change

**Scenario:** DeepSeek's V4-Flash model gets an update that makes it over-narrate MORE than before. The conciseness post-processing pass (Comp 3) was calibrated against the old behavior. Now it's under-powered — stripping 20% when it needs to strip 40%. Subagent output quality degrades silently.

**Gap:** Conciseness pass effectiveness is never recalibrated after the initial calibration. No drift detection on "how much narration is the model producing now vs at calibration time."

---

## Category 5: Resource Exhaustion

### 5A: Subagent Transcript Accumulation

**Scenario:** User runs reasonix for 8 hours/day, spawning 100 subagents/day. Each subagent transcript is ~50KB. After 30 days: 3,000 transcripts × 50KB = 150MB. After 6 months: 900MB. After a year: 1.8GB of subagent transcripts on disk.

**Gap:** No transcript retention policy. No auto-cleanup beyond the 7-day workflow TTL. `maxToolOutputBytes` caps individual tool outputs but doesn't cap accumulated transcript storage.

### 5B: Knowledge Base Growth

**Scenario:** Cross-session KB (Comp 16) stores embeddings for every subagent output. At 1,000 entries, the embedding index is ~50MB. At 10,000 entries, ~500MB. At 100,000 entries, ~5GB. The KB grows unboundedly.

**Gap:** No KB size limit. No eviction policy. No deduplication of semantically identical entries (same analysis produced twice).

### 5C: Confidence Profile Storage

**Scenario:** Confidence calibration (Comp 4) tracks per-(role, model, language) accuracy. With 5 roles × 3 models × 10 languages, that's 150 profiles. Each profile needs at least 30 samples for statistical significance = 4,500 subagent runs to warm up.

**Gap:** Cold start: profiles need 30+ samples to be meaningful. Before that, routing is no better than default. No time-to-significance estimate in the UI.

---

## Category 6: User Workflow Gaps

### 6A: "Undo" for Subagent Work

**Scenario:** User spawns a workflow that modifies 5 files. User disagrees with one change. How do they revert just that change? Checkpoint/rewind exists for the whole workspace, but not per-subagent or per-finding.

**Gap:** No granular undo. Checkpoint snapshots the entire workspace. There's no "revert the changes from subagent X only."

### 6B: "Explain This Finding" Deep Dive

**Scenario:** A security review produces a finding: "SQL injection in db.go:108." User wants to understand WHY the verifier thinks this. They can peek at the verifier transcript (subagent debugging, Feature U5), but the finding itself should have a one-click path from "what" to "why" to "how to fix."

**Gap:** The "Why?" button exists in the subagent debugging feature, but there's no "Fix this" button that spawns an executor with the finding context pre-loaded.

### 6C: Session Fork

**Scenario:** User is mid-session with a complex investigation. They want to try a different approach WITHOUT losing the current state. They need to fork the session: "continue from here, but in a new tab, with this alternative approach."

**Gap:** Session fork exists for subagent transcripts (continue_from, fork_from) but not for full user sessions. A user can't say "fork this conversation and try a different approach."

### 6D: "What Did I Change?" Audit

**Scenario:** User has been working with reasonix for 2 hours. Multiple subagents have made changes across 12 files. User wants a summary: "What changed, by whom (which subagent), and why?"

**Gap:** git diff shows changes but doesn't attribute them to subagents. No "audit trail" view that shows: "Subagent 'fix-sql-injection' changed db.go:108-112. Reason: parameterize user input to prevent SQL injection."

---

## Category 7: Edge Cases in the Quality System

### 7A: The "Calibrated to the Test" Problem

**Scenario:** Our 17 eval scenarios for prompt calibration become the de facto target. Prompt authors (human or LLM) optimize for the scenarios, not for real-world performance. The prompt achieves 95% on eval scenarios but performs poorly on real user queries.

**Gap:** Held-out scenarios (Phase B) mitigate this partially, but 1 held-out scenario per role with a single run is not enough statistical power to detect overfitting. 4 calibration + 1 held-out is not an 80/20 split — it's more like 80/20 in scenario count but ~95/5 in statistical power because the held-out set has N=1.

### 7B: The "Always Degraded" Scenario

**Scenario:** Drift check (Phase E) detects degradation weekly. Each time, it triggers recalibration. A new prompt variant is selected. Two weeks later, it degrades again. The cycle repeats. The user sees perpetual degradation notices. Eventually they ignore them.

**Gap:** No "degradation fatigue" handling. After N degradation-recalibration cycles within M weeks, the system should escalate differently — perhaps notifying that this role seems fundamentally unstable on the current model version.

### 7C: "Goodhart's Law" in Evaluation

**Scenario:** The verifier expectation `output_contains: "file:"` passes if the output contains the literal string "file:" anywhere. A verifier that says "see attached file: /dev/null" passes. The expectation is gamed.

**Gap:** Our eval expectations are surface-level. `output_contains` patterns match substrings, not semantic content. A subagent could learn to pattern-match the expectations without actually doing good work.

---

## Category 8: DeepSeek-Specific Failure Modes We Missed

### 8A: DeepSeek "Hallucinated Tool Output"

**Scenario:** DeepSeek sometimes generates tool output in its reasoning_content — it imagines what `grep` would return without actually calling `grep`. The cognitive loop detector catches some patterns, but not this specific one: the model writes plausible-but-fabricated grep output in reasoning, then references it as evidence.

**Gap:** No cross-check that cited tool output actually matches recorded tool results. The claim verifier (Comp 3, correctness pass) checks whether claims reference files that were READ, but doesn't verify that the content of tool output cited in claims matches the actual recorded tool output.

### 8B: DeepSeek "Silent Language Switch"

**Scenario:** A user configured language: "en." Mid-session, DeepSeek's reasoning switches to Chinese for a complex technical explanation. The final answer is in English, but the reasoning_content is in Chinese. The post-processing passes see only the final answer.

**Gap:** No language consistency check across reasoning_content and content. If reasoning switches languages, the subagent might have reasoned incorrectly (or at least, the user can't inspect the reasoning).

---

## What We Found: 24 New Gaps Across 8 Categories

| Category | Gaps | Severity |
|---|---|---|
| Compositional Explosions | 3 | High — real user scenarios |
| Timing & Ordering | 3 | Medium — race conditions |
| State Corruption | 3 | High — silent degradation |
| Provider Assumptions | 3 | Medium — outside our control |
| Resource Exhaustion | 3 | Medium — accumulates over time |
| User Workflow Gaps | 4 | High — missing UX |
| Quality System Edge Cases | 3 | Medium — evaluation validity |
| DeepSeek-Specific Misses | 2 | High — model behavior |
