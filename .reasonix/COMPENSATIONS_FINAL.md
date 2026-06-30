# Final Compensations: Single-User, High-Value, Buildable Now

> No org infra. No multi-user. No training pipelines. Just things that make one developer's reasonix genuinely better than anything else.

---

## 19: Adversarial Self-Testing — 2 days

**What it does:** When reasonix implements a feature, it spawns a tester subagent whose explicit goal is to BREAK the implementation. The tester is rewarded (in prompt instructions) for finding bugs. When it finds one, the executor fixes it. Loop until the tester can't find anything.

**Why it's different from verify:** Verify checks a CLAIM against evidence. Self-testing checks CODE against behavior. It writes tests designed to find edge cases the executor didn't think of.

**Mechanism:**
```
After feature implementation:
    1. Spawn tester subagent (Pro, max effort)
       "You are an adversarial tester. Your job is to BREAK this implementation.
        Write tests that exploit edge cases, race conditions, nil inputs, boundary values.
        For every test that fails, you win. Report the failure with exact reproduction steps."
    2. Tester spawns, writes tests, runs them, reports failures
    3. For each failure: executor fixes, re-runs
    4. Loop until tester returns "No failures found after 3 rounds"
    
    Cost: ~$0.008 per self-test round (tester + executor). Typically 1-2 rounds.
```

**UI:**
- Transcript: "⚔️ Self-testing: tester found 2 edge cases. Fixing..."
- Status: "⚔️ Self-test round 2/3 — tester found 0 new failures"
- Final: "✓ Self-test passed — tester found 2 bugs (both fixed). Feature is battle-tested."

---

## 20: Conversational Codebase Q&A — 1.5 days

**What it does:** On top of the cross-session knowledge base (Comp 16), a conversational interface that lets you ask natural language questions about your codebase and get cited answers. Not search — conversation.

**Why it's different from explore:** Explore returns findings. This returns ANSWERS, in conversation, with memory of what you just asked. "How does auth work?" → answer. "What about the middleware part?" → understands "the middleware part" refers to auth middleware from the last question.

**Mechanism:**
```
User: "How does payment processing work?"
    1. KB check: 2 prior analyses of payment module exist
    2. If KB answers: conversational synthesis with citations
    3. If not: spawn explorer, index result, answer conversationally
    
User: "What happens when it fails?"
    4. Context: previous question was about payment processing
    5. KB check: payment error handling was analyzed last week
    6. Answer: "Payment failures trigger the retry queue in payment/retry.go:45.
       After 3 retries, PaymentFailed event is emitted..."
    
    Each answer costs $0.001-0.005 depending on whether KB has it or needs fresh exploration.
    Conversation context persists within the Q&A session.
```

**UI:**
- `/ask` command opens a focused Q&A panel (not the main chat — separate context)
- Each answer shows: when the underlying analysis was done, whether it's fresh or cached
- "Ask a follow-up" input stays open for conversation threading
- Answers cite specific files:lines that are clickable

---

## 21: Automatic Technical Debt Manager — 2 days

**What it does:** Weekly, reasonix scans the codebase and produces a prioritized technical debt list with concrete fix plans. Not just "this file is messy" — "this function has 4 responsibilities, extract 2 into helpers, estimated 30 minutes, would reduce coupling score from 0.7 to 0.3."

**Mechanism:**
```
Weekly cron:
    1. Scan all source files changed in the past week
    2. For each, spawn a reviewer subagent with a "technical debt" lens:
       "Rate this file on: complexity, coupling, test coverage, documentation, duplication.
        For each issue: provide a concrete fix plan with estimated effort and impact."
    3. Synthesize into a prioritized debt list
    4. Each item includes: file, issue, concrete fix, estimated time, impact score
    
    Cost: ~$0.02 per weekly scan (reviewer subagents for changed files)
```

**UI:**
- `/debt` shows the current debt list, sorted by impact/effort ratio
- Each item: "auth.go — extract token validation (30 min, impact: high)"
- "Fix this" button spawns an executor to implement the fix
- Debt trend: "Technical debt score: 6.2/10 · ↓ from 7.1 last week"
- History: tracks debt over time as a sparkline

---

## 22: Automated PR Stacking — 2 days

**What it does:** When a feature is too big for one coherent PR, reasonix automatically decomposes it into a stack of dependent PRs, implements each one in order, and creates the PRs.

**Mechanism:**
```
When a feature request is too large:
    1. Planner subagent: "Decompose this feature into a stack of independent, reviewable PRs.
        Each PR must be <500 lines. Each PR must pass tests independently.
        Order them by dependency: PR 1 must merge before PR 2 can work."
    2. For each PR in order:
        a. Create git worktree from previous PR's branch
        b. Execute the PR's changes
        c. Run tests
        d. If tests pass → commit, create GitHub PR with description
        e. Next PR stacks on this branch
    
    Cost: ~$0.03 for full decomposition + implementation (planner + N executors)
```

**UI:**
- "📚 PR Stack: auth-refactor" with numbered stack visualization
- Each PR: title, status, line count, test status
- PR 1 shows "Merged ✓", PR 2 shows "Ready for review", PR 3 shows "Waiting on PR 2"
- "Create all PRs" button, or per-PR review before creation
- Auto-generated PR descriptions with context, changes, and test results

---

## 23: Git History Intelligence — 1 day

**What it does:** You can ask questions ABOUT your git history. "When did we add the rate limiter?" "Who changed the auth middleware last?" "Show me every commit that touched the payment module in the past month."

**Why it's different from git log:** git log shows commits. This UNDERSTANDS commits. "Show me when the rate limiter was added" → it finds the commit where the rate limiting LOGIC was introduced, not just the first commit that mentions "rate" in the message.

**Mechanism:**
```
User: "When did we add rate limiting?"
    1. grep "rate" in git log → 47 commits
    2. Spawn explorer subagent: "These 47 commits mention 'rate'. Which one INTRODUCED the rate limiting feature?
        Look at the diffs. The introduction commit will add the core rate limiting logic,
        not just modify configuration or fix bugs."
    3. Explorer returns: "abc1234 (March 15) — added rate_limiter.go with token bucket implementation.
        Subsequent commits were config changes and bug fixes."
    
    Cost: $0.001 per history query
```

**UI:**
- `/git ask "when was the rate limiter added"` — conversational interface
- Answer: commit hash (clickable → GitHub), date, author, files changed, diff summary
- Follow-up: "Show me the original implementation" → opens the file at that commit
- History search bar in the workspace panel

---

## 24: Automated Release Notes — 0.5 day

**What it does:** When you're ready to tag a release, reasonix reads the git log since the last tag, categorizes every commit by type (feature, fix, refactor, docs, breaking), and generates a release notes file.

**Mechanism:**
```
User: "Generate release notes for v2.1.0"
    1. Find last tag (v2.0.0)
    2. Get all commits between v2.0.0 and HEAD
    3. Spawn classifier subagent: "Categorize each commit: feature, fix, refactor, docs, breaking.
        For features and fixes: write a one-line user-facing description.
        For breaking changes: flag prominently."
    4. Generate RELEASE_NOTES.md with categorized, described changes
    
    Cost: $0.002 for classification (typically 20-50 commits)
```

**UI:**
- `/release v2.1.0` generates and opens the release notes
- Preview with sections: Features, Bug Fixes, Breaking Changes, Internal
- "Edit" button for manual adjustments
- "Publish" creates the GitHub release with the generated notes

---

## 25: Learning Path Generator — 1.5 days

**What it does:** When you're new to a codebase, reasonix generates a personalized learning path. "I'm new to this project. What should I learn first?" → Analyzes the architecture, identifies the core modules and their dependencies, generates a learning order.

**Mechanism:**
```
User: "I'm new to this codebase. Where do I start?"
    1. Architecture analysis subagent: identifies core modules, entry points, dependency graph
    2. Learning path subagent: "Given this architecture, generate a learning path.
        Order: entry point → core data models → key abstractions → peripheral modules.
        For each step: the files to read, the concepts to understand, and a small task to practice."
    3. Learning path saved. Progress tracked.
    
    Cost: $0.005 for initial path generation
```

**UI:**
- `/learn` shows the learning path as a checklist
- Each step: "1. Read main.go and cmd/ — understand the entry point. Task: add a --verbose flag."
- Steps marked ✓ as completed
- Progress bar: "3/7 steps · next: understand the auth middleware"
- Path regenerates when architecture changes significantly

---

## Cost Summary

| # | What | Daily | Monthly | Annual |
|---|---|---|---|---|
| 19 | Adversarial Self-Testing | $0.02 (3 features tested) | $0.60 | $7.30 |
| 20 | Conversational Q&A | $0.01 (10 questions) | $0.30 | $3.65 |
| 21 | Tech Debt Manager | — (weekly) | $0.08 | $1.04 |
| 22 | Automated PR Stacking | $0.03 (1 stack) | $0.90 | $10.95 |
| 23 | Git History Intelligence | $0.005 (5 queries) | $0.15 | $1.83 |
| 24 | Automated Release Notes | — (per release) | $0.02 | $0.24 |
| 25 | Learning Path Generator | — (per codebase) | $0.02 | $0.24 |

**Total annual cost for compensations 19-25: ~$25/year.**

---

## The Complete Picture: 25 Compensations

1-10: Structural Quality (close the gap with Anthropic)
11-13: Continuous Background Intelligence (always on)
14-15: Ensemble Intelligence (N>1 always)
16-18: Compounding Learning + Pre-Emptive Intelligence (gets better over time)
19-25: Developer Experience (makes the tool genuinely delightful)

**Total annual cost for all 25 compensations: ~$141/year.**
**At Anthropic rates: ~$7,000+/year. Fifty times more.**

Anthropic couldn't build this product. Not because they lack the engineering talent — but because every one of these compensations multiplies token spend by 2-50×, and their tokens cost 50× more. The economics prevent them from implementing always-on intelligence. We can.
