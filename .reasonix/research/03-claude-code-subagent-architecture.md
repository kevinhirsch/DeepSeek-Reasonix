# Claude Code Subagent Architecture Research

**Date:** 2026-06-30
**Source:** Deep-research workflow (wlgtkpwgl) — 101 agents, 1,284 tool uses, 3,391,245 tokens

## Verified Findings

### Finding 1: Strict Role Specialization with Isolated Context Windows
**Confidence: HIGH**

Claude Code subagents start with a fresh context window that excludes:
- Parent conversation history
- Previously invoked skills
- Previously read files

Non-fork subagents receive only: their own system prompt, a delegation task message, CLAUDE.md files, git status snapshot.

Explore and Plan agents further omit CLAUDE.md and git status to reduce latency/cost.

The BUILD-AND-FIND research protocol (Lin 2026, arXiv 2605.06136) formalizes this with exactly two roles having strictly separated I/O contracts.

**Sources:** code.claude.com/docs/en/sub-agents, arXiv 2605.06136, Piebald-AI/claude-code-system-prompts

### Finding 2: Two-Tier Effort Calibration
**Confidence: HIGH**

- **Session-level effort:** low/medium/high/xhigh/max, overridable per-subagent via frontmatter
- **Explore-specific thoroughness:** quick/medium/very thorough
- Effort metrics gated behind accuracy: interpreted only when recovery succeeds reliably
- BUILD-AND-FIND protocol: effort comparison within finder-task cells only where recovery succeeds

**Sources:** code.claude.com/docs/en/sub-agents, arXiv 2605.06136

### Finding 3: Adversarial Verification with Three-State Classification
**Confidence: HIGH**

- **Classification:** confirmed / plausible / refuted
- **Precision-biased mode (medium effort):** one-vote verification, three finder angles
- **Recall-biased mode (high effort):** "treats realistic uncertain findings as plausible unless code refutes them"
- **Verifier identity is explicitly adversarial:** "Your job is not to confirm the implementation works — it's to try to break it"
- **Mandatory probes:** concurrency, boundary conditions, error paths, input validation

**Sources:** Piebald-AI/claude-code-system-prompts (parts 4-7), repowise-dev/claude-code-prompts, DeepWiki verification specialist prompt

### Finding 4: Plan Agent 5-Phase Workflow
**Confidence: MEDIUM**

1. Understanding — spawn Explore agents in parallel for codebase exploration
2. Design — architecture and approach
3. Review — self-review the plan
4. Final Plan — produce the deliverable
5. Exit — confirm completion

**Sources:** Piebald-AI/claude-code-system-prompts

### Finding 5: Task Budgets as Hard Guardrails
**Confidence: MEDIUM**

- Server-side task budgets: minimum 20,000 tokens
- `task_budget` parameter in `output_config` tells model its remaining token ceiling
- Distinct from `max_tokens` — model-aware budget vs enforced request-level ceiling
- Opus 4.7/4.8 and Fable 5 only (Claude-specific, not portable to DeepSeek)

**Sources:** Claude API documentation (shared/model-migration.md → Task Budgets)

### Finding 6: Sagent Arbitrary-Tree Coordination Topology
**Confidence: MEDIUM**

Sagent demonstrates an alternative coordination model where any agent can recursively spawn children AND message peers laterally, not just report up to parent. This is a v2 enhancement pattern for reasonix.

**Sources:** Sagent documentation (specific URL not captured in workflow output)

## Gaps Not Confirmed

- No verified claims about DeepSeek-specific subagent adaptations in existing tools
- No verified claims about inter-agent communication protocols within Claude Code itself
- No verified claims about the exact format of SendMessage wire protocol
