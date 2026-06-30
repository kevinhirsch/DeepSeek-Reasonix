# DeepSeek Reasoning Model Behavior Research

**Date:** 2026-06-30
**Source:** Research agent (a0404d327786e4182) — 24 web fetch/source operations, 37,891 tokens

## Key Findings

### reasoning_content Field Behavior

- DeepSeek R1 base **does not support function calling**
- R1-0528 added function calling (BFCL: 93.25%)
- For models supporting tool calls with thinking: `reasoning_content` is separate from `content`
- **Critical:** When tool calls occurred, `reasoning_content` MUST be passed back in subsequent requests or API returns 400
- When no tool calls occurred, `reasoning_content` should be STRIPPED
- Known bug: LangChain ChatOpenAI wrapper drops `reasoning_content` → breaks multi-turn tool loops
- R1 quirk: sometimes writes tool calls into `reasoning_content` text instead of structured `tool_calls` slot

### reasoning_effort Levels

- Only **two real levels**: `"high"` (default) and `"max"`
- Aliases: `"low"` → `"high"`, `"medium"` → `"high"`, `"xhigh"` → `"max"`
- Applied via `reasoning_effort` or `output_config.effort`
- For older `deepseek-reasoner`, thinking always on — no toggle

### Known Failure Modes

1. **Infinite reasoning loops** — repeats identical reasoning, exhausts max_tokens at low temperatures
2. **Confusion/cognitive loops** — "tricky", "confused", "ambiguous" cycles
3. **Multi-turn tool-call breaks** — `reasoning_content` dropped by wrappers → 400
4. **Multi-turn dialogue degradation** — after several turns, responses become mechanical self-explanation (GitHub issue #1125)
5. **Malformed JSON in reasoning** — JSON fragments inside `<think>` blocks when requesting structured output

### Context Caching: DeepSeek vs Anthropic

| Aspect | DeepSeek | Anthropic |
|---|---|---|
| Activation | Automatic | Explicit breakpoints |
| Write cost | None | 1.25x (5m) / 2x (1h) |
| Read discount | ~99% off | ~90% off |
| Guarantee | Best-effort | Deterministic |
| Lifetime | Hours to days | 5 min / 1 hour |

### Prompting Best Practices

- **No system prompts** — R1 trained without them, put instructions in user messages
- **Temperature: 0.6** (0.5-0.7 range) — too low causes infinite repetition
- **Avoid "think step by step"** — R1 does CoT automatically
- **Avoid few-shot examples** — consistently degrade R1 performance
- **Structured prompt framework**: Role → Task → Input → Output spec → References (only if essential)

### Model Selection Decision Matrix

| Use Case | Model | Reason |
|---|---|---|
| Simple Q&A, creative writing | V4-Flash (non-thinking) | Fast, cheap |
| Complex math, hard coding | V4-Pro (thinking, effort=max) | Deepest reasoning |
| Agentic tool-calling + reasoning | V4-Pro (thinking enabled) | Only model with full interleaving |
| Budget agentic coding | Hybrid: R1-0528 (planning) + V4-Flash (execution) | Separates reasoning from execution |

### Sources
- api-docs.deepseek.com/guides/reasoning_model
- api-docs.deepseek.com/guides/thinking_mode
- api-docs.deepseek.com/guides/kv_cache
- api-docs.deepseek.com/quick_start/pricing
- github.com/deepseek-ai/DeepSeek-V3/issues/1125
- github.com/deepseek-ai/DeepSeek-R1/issues/93
- sitepoint.com/deepseek-r1-troubleshooting-guide
- ar5iv.labs.arxiv.org/html/2512.12895 (Reasoning Model Loops paper)
- fireworks.ai/blog/deepseek-models
- qbitai.com/2025/02/254182.html (official DeepSeek recommendations)
- github.com/anomalyco/opencode/pull/25110
- github.com/0xPlaygrounds/rig/issues/1434
