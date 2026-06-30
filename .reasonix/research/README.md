# Research Archive

> Preserved research data from the reasonix subagent architecture design session.
> Date: 2026-06-30

## Files

| File | Source | Description |
|---|---|---|
| `01-deepseek-reasoning-patterns.md` | Research agent (a0404d327786e4182) | DeepSeek API behavior, failure modes, prompting, context caching |
| `02-subagent-ux-patterns.md` | Research agent (aaaccb4b972dca323) | Claude Code Agent View, Agent Deck, terminal monitoring UX patterns |
| `03-claude-code-subagent-architecture.md` | Deep-research workflow (wlgtkpwgl) | Verified findings about Claude Code's subagent architecture |
| `04-reasonix-codebase-audit.md` | Direct codebase analysis | Complete package audit, built-in tools, skills, and infrastructure |
| `05-tokenomics-reference.md` | API docs + calculations | DeepSeek and Anthropic pricing, projected cost model for reasonix workflows |

## Key Decision Trace

1. **Role prompts go in user messages, not system prompts** — DeepSeek R1 was trained without system prompts. Research finding from 01-deepseek-reasoning-patterns.

2. **Temperature 0.6 for DeepSeek subagents** — Official DeepSeek recommendation; 0.0 causes infinite loops. From 01-deepseek-reasoning-patterns.

3. **Adversarial verification: CONFIRMED/PLAUSIBLE/REFUTED** — From Claude Code's verified pattern. From 03-claude-code-subagent-architecture.

4. **Status encoding: dual (state + liveness)** — From Agent Deck's proven pattern. From 02-subagent-ux-patterns.

5. **Inline panel preserves scrollback** — No major terminal agent uses full-screen TUI. From 02-subagent-ux-patterns.

6. **Auto-cache economics favor DeepSeek** — Free writes, ~99% hit discount. From 01-deepseek-reasoning-patterns and 05-tokenomics-reference.

7. **DeepSeek only has high/max real effort levels** — low/medium map to high, xhigh maps to max. From 01-deepseek-reasoning-patterns.

8. **Cognitive loop detection needs 3 patterns** — uncertainty escalation, identical reasoning prefix, mechanical self-explanation. From 01-deepseek-reasoning-patterns.

## Session Research Agents

- `a0404d327786e4182` — DeepSeek reasoning patterns (24 sources, 37.9K tokens)
- `aaaccb4b972dca323` — Subagent UX monitoring (19 sources, 37.3K tokens)
- `wlgtkpwgl` — Deep-research workflow (101 agents, 1,284 tool uses, 3.4M tokens)
