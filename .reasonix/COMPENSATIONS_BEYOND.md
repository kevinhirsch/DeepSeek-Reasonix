# Beyond Structural: Capabilities Anthropic Cannot Afford

> Structural compensations 1-10 close the quality gap. These go beyond — they're capabilities that are ONLY economically viable at 1/50th the token cost. Anthropic literally cannot build these without losing money on every user.

---

## Class I: Continuous Background Intelligence (Always-On Agents)

### Compensation 11: Ambient Code Guardian — 2 days

**What it does:** A permanently-running background subagent (Flash, high effort) that watches every file change. Not triggered by user request — always on. When it detects a likely bug, security issue, or pattern violation, it proactively flags it BEFORE the user discovers it in production.

**Mechanism:**
```
On file save:
    1. Guardian receives the diff
    2. Runs a lightweight check: "Does this change introduce any of: nil dereference, race condition, SQL injection, path traversal, missing error handling, broken test?"
    3. If yes → emits a proactive notice: "⚠ Guardian: possible nil dereference in auth.go:42 — config is nil on first call"
    4. If no → silent
    
Cost: ~$0.0003 per save. For 100 saves/day: $0.03/day.
```

**Why Anthropic can't:** A permanently-running security reviewer at Opus rates costs $2-5/day per user. At 100K users, that's $200-500K/day in inference costs for a feature that produces value on <5% of saves.

### Compensation 12: Idle-Time Pre-Computation — 2 days

**What it does:** When reasonix is idle (no user input for 2 minutes), it spawns a fleet of explorer subagents to pre-compute understanding of the codebase. By the time the user asks a question, the answer is already computed.

**Mechanism:**
```
On idle detection:
    1. Priority queue of unexplored code areas (most-referenced first)
    2. Spawn explorer subagents (Flash, high effort, n=2 for consensus)
    3. Each explorer answers: callers of X, architecture of Y, data flow through Z
    4. Results stored in a local knowledge base (SQLite + embeddings)
    5. When user asks a question → check knowledge base first → instant answer if pre-computed
    
    Idle budget: max 10 subagents per idle session, max $0.01 per idle session
    Stop on user activity (any keystroke)
```

**Why Anthropic can't:** Burning tokens while the user isn't even using the product. At their rates, idle pre-computation would cost $0.50-2.50 per idle session. At 10 idle sessions per day per user, that's $5-25/day. Unsustainable.

### Compensation 13: Continuous Dependency Watchdog — 1.5 days

**What it does:** Weekly, spawns subagents to check every dependency for updates. For each update, clones the repo into a git worktree, applies the update, runs the test suite, and reports: "Safe to merge" or "Breaks 3 tests."

**Mechanism:**
```
Weekly cron:
    1. Fetch latest versions of all dependencies
    2. For each outdated dependency:
        a. Spawn worktree (EnterWorktree)
        b. Apply update (bash)
        c. Run test suite (bash, background)
        d. Subagent analyzes results: "All 42 tests pass → safe" or "3 tests fail → details below"
    3. Report: "3 dependencies outdated. 2 safe to update. 1 breaks tests."
    
    Cost: ~$0.01 per dependency checked. For 10 deps: $0.10/week.
```

**Why Anthropic can't:** Running the full test suite inside a subagent context for every dependency update. Each check costs $0.50-2.00 at Anthropic rates. For 10 dependencies: $5-20/week per project.

---

## Class II: Ensemble Intelligence (N>1 Always)

### Compensation 14: Self-Play Code Generation — 2 days

**What it does:** When asked to implement a feature, spawn 3 independent executor subagents. Each implements the SAME thing from scratch, with different approaches. Then spawn a judge subagent that picks the best implementation and synthesizes ideas from the runners-up.

**Mechanism:**
```
For a feature implementation request:
    1. Pre-mortem: predict failure modes ($0.001)
    2. Spawn 3 executors with the SAME task but SEEDED with different approach hints:
        A: "Prefer simplicity — fewest lines, fewest files"
        B: "Prefer clarity — most readable, best-named"
        C: "Prefer performance — fastest, least memory"
    3. All 3 run independently
    4. Judge subagent (Pro, max effort): "Compare these 3 implementations. Pick the best. Graft the best ideas from the runners-up into the winner."
    5. Final output: the synthesized implementation
    
    Cost: Pre-mortem ($0.001) + 3 executors ($0.005 each) + judge ($0.003) = $0.019
    Anthropic cost: $0.05 + 3×$0.25 + $0.15 = $0.95
```

**Why Anthropic can't:** 3 parallel Opus executors for a single feature request would cost $0.75 in inference. Nobody would enable this. But at $0.015 for 3 Flash executors, it's automatic.

### Compensation 15: Ensemble Code Review — 1 day

**What it does:** When code is ready to commit, spawn 3 reviewer subagents with different review philosophies. Synthesize into one comprehensive review.

**Mechanism:**
```
On pre-commit:
    1. Security reviewer: "Find injection, auth bypass, secrets, path traversal"
    2. Correctness reviewer: "Find nil deref, race conditions, off-by-one, logic errors"
    3. Performance reviewer: "Find N+1 queries, unnecessary allocations, blocking IO"
    4. Synthesizer: "Merge all findings, deduplicate, rank by severity"
    
    Cost: 3 reviewers ($0.002 each) + synthesizer ($0.001) = $0.007
    Anthropic cost: 3×$0.10 + $0.05 = $0.35
```

---

## Class III: Compounding Learning (Gets Better Over Time)

### Compensation 16: Cross-Session Knowledge Base — 3 days

**What it does:** Everything reasonix learns about the codebase across ALL sessions is indexed into an embeddings store. When the user asks a question, reasonix retrieves relevant prior knowledge before spawning new subagents. This means the 100th question about the codebase costs the same as the 1st — but gets answered instantly from the knowledge base.

**Mechanism:**
```
Per session:
    Every subagent finding, every architecture analysis, every bug pattern → embedded and stored
    
Cross-session retrieval:
    User asks: "How does authentication work?"
    1. Query knowledge base → "3 prior analyses of auth exist (2 weeks ago, 1 month ago)"
    2. Retrieve most recent + most comprehensive
    3. Spawn a synthesis subagent: "These 3 prior analyses cover auth. The user asked how it works. Synthesize. Only spawn new exploration if the prior analyses are insufficient."
    4. If KB answers the question → instant, $0.001 (synthesis only)
    5. If KB is stale → full exploration, $0.005, KB updated
    
    Knowledge base size: ~10MB per 100 subagent outputs (embeddings are compact)
    Stored locally in .reasonix/knowledge/
```

**Why Anthropic can't:** Building the knowledge base requires subagent runs that produce indexable output. At their rates, 100 subagent runs to seed the KB costs $5-10. At our rates, $0.10-0.30. The compounding value is identical — the acquisition cost is 50× different.

### Compensation 17: Bug Pattern Recognition — 1.5 days

**What it does:** When a bug is found and fixed, the pattern is extracted, embedded, and stored. When new code is written that matches a known bug pattern, reasonix proactively flags it BEFORE commit.

**Mechanism:**
```
On bug fix:
    1. Extract the bug pattern: what was wrong, what fixed it, what the code looked like before
    2. Embed the pattern
    3. Store: pattern + fix + context (language, framework, file type)
    
On new code:
    1. Diff is checked against known bug patterns
    2. If similarity > 0.85 → "⚠ This code matches a pattern that caused a bug 3 weeks ago in handlers.go:108. The fix then was: [show fix]."
    
    Pattern library grows organically. After 6 months, ~200 patterns covering the user's specific codebase.
```

**Why Anthropic can't:** Pattern extraction and matching require LLM calls on every save. At their rates, this doubles the effective cost of every coding session. At our rates, it's noise.

---

## Class IV: Pre-Emptive Intelligence (Answers Before Questions)

### Compensation 18: Impact Analysis Pre-Computation — 1.5 days

**What it does:** When you change a function signature, before you even ask, reasonix has already analyzed every caller and knows EXACTLY what will break, where, and how to fix it.

**Mechanism:**
```
On save of a function signature change:
    1. Detect changed signature (not just body edit)
    2. Spawn explorer: "Find every call site of <changed function>. For each, report: file:line, current args, whether this call will break with the new signature."
    3. Store results
    
    When user asks "will this break anything?" → instant answer, pre-computed
    When user asks "fix all callers" → pre-computed call site list feeds into executor
    
    Cost: $0.002 per signature change
    Anthropic cost: $0.10 per signature change
```

---

## Cost Summary

| # | Compensation | Cost/use | Anthropic cost | Daily budget | Annual cost |
|---|---|---|---|---|---|
| 11 | Ambient Guardian | $0.0003/save | $0.015/save | $0.03 (100 saves) | $10.95 |
| 12 | Idle Pre-Computation | $0.01/idle session | $0.50/idle | $0.03 (3 idle) | $10.95 |
| 13 | Dependency Watchdog | $0.10/week | $5.00/week | — | $5.20 |
| 14 | Self-Play Code Gen | $0.019/feature | $0.95/feature | $0.10 (5 features) | $36.50 |
| 15 | Ensemble Review | $0.007/commit | $0.35/commit | $0.04 (6 commits) | $14.60 |
| 16 | Cross-Session KB | $0.001/query | $0.05/query | $0.02 (20 queries) | $7.30 |
| 17 | Bug Pattern Recog | $0.0005/save | $0.025/save | $0.03 (60 saves) | $10.95 |
| 18 | Impact Analysis | $0.002/sig change | $0.10/sig change | $0.01 (5 changes) | $3.65 |

**Total annual cost for all 8 compensations: ~$100/year.** At Anthropic rates: ~$5,000/year. Fifty times more. Nobody would build these at Anthropic's cost. At ours, they're automatic.

---

## The Full Picture: 18 Compensations

| # | Name | Class |
|---|---|---|
| 1 | Speculative Execution | Structural Quality |
| 2 | Multi-Model Cross-Validation | Structural Quality |
| 3 | Post-Processing Passes | Structural Quality |
| 4 | Confidence Calibration | Structural Quality |
| 5 | Prompt Regression Bisection | Structural Quality |
| 6 | Consensus Weighting | Structural Quality |
| 7 | Self-Healing Prompt Library | Structural Quality |
| 8 | Context Janitor | Structural Quality |
| 9 | Pre-Mortem Analysis | Structural Quality |
| 10 | Semantic Diff | Structural Quality |
| 11 | Ambient Code Guardian | Continuous Background |
| 12 | Idle-Time Pre-Computation | Continuous Background |
| 13 | Dependency Watchdog | Continuous Background |
| 14 | Self-Play Code Generation | Ensemble Intelligence |
| 15 | Ensemble Code Review | Ensemble Intelligence |
| 16 | Cross-Session Knowledge Base | Compounding Learning |
| 17 | Bug Pattern Recognition | Compounding Learning |
| 18 | Impact Analysis Pre-Computation | Pre-Emptive Intelligence |

**Total annual cost for all 18 compensations: ~$116/year.**
**At Anthropic rates: ~$5,000+/year.**
**The economic moat isn't the features. It's that we can afford to run them.**
