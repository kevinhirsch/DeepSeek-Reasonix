# 21 — Developer Experience Compensations (19-25)

**Phase:** Quality (ships after Phase 3-5) | **Effort:** 10.5 days total | **Priority:** P2

## What's Included

| # | Name | Effort |
|---|---|---|
| 19 | Adversarial Self-Testing | 2 days |
| 20 | Conversational Codebase Q&A | 1.5 days |
| 21 | Automated Technical Debt Manager | 2 days |
| 22 | Automated PR Stacking | 2 days |
| 23 | Git History Intelligence | 1 day |
| 24 | Automated Release Notes | 0.5 day |
| 25 | Learning Path Generator | 1.5 days |

## Design

### 19: Adversarial Self-Testing
After feature implementation: spawns tester subagent (Pro, max) whose goal is to BREAK the code. Writes tests exploiting edge cases. Executor fixes. Loop until tester finds nothing.

### 20: Conversational Codebase Q&A
`/ask` command opens focused Q&A panel. Natural language questions about codebase. Maintains conversation context. Uses KB for instant answers, spawns explorers when needed.

### 21: Automated Technical Debt Manager
Weekly scan of changed files. Scores: complexity, coupling, coverage, docs, duplication. Prioritized fix list with concrete plans and time estimates.

### 22: Automated PR Stacking
Decomposes large features into stacked PRs (<500 lines each). Each PR independently testable. Implements in dependency order. Creates GitHub PRs.

### 23: Git History Intelligence
Conversational git history. Semantic understanding of commits. Finds when features were introduced, not just keyword matches.

### 24: Automated Release Notes
Reads git log since last tag. Classifies commits (feature/fix/refactor/docs/breaking). Generates RELEASE_NOTES.md.

### 25: Learning Path Generator
Generates personalized learning path for new codebase contributors. Architecture analysis → dependency graph → ordered learning steps with practice tasks.

## Acceptance
- [ ] Each compensation works with mock providers
- [ ] Each independently toggleable in config
- [ ] Cost tracked per compensation
- [ ] No perceptible latency on user interactions
- [ ] All UI surfaces covered in `compensations-beyond.feature`
