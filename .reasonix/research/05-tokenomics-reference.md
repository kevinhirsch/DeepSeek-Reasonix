# Tokenomics Reference Data

**Date:** 2026-06-30

## DeepSeek Pricing (V4 Models, mid-2026)

| Model | Input (cache miss) | Input (cache hit) | Output | Cache Discount |
|---|---|---|---|---|
| V4-Flash | $0.14/M | $0.0028/M | $0.28/M | ~98% off |
| V4-Pro | $0.435/M | $0.003625/M | $0.87/M | ~99% off |

## Anthropic Pricing (Current Models, mid-2026)

| Model | Input | Output |
|---|---|---|
| Claude Fable 5 | $10.00/M | $50.00/M |
| Claude Opus 4.8 | $5.00/M | $25.00/M |
| Claude Opus 4.7 | $5.00/M | $25.00/M |
| Claude Sonnet 4.6 | $3.00/M | $15.00/M |
| Claude Haiku 4.5 | $1.00/M | $5.00/M |

Anthropic cache: reads ~0.1× base input, writes 1.25× (5m TTL) or 2× (1h TTL).

## Projected Tokenomics for reasonix Workflow

"Audit auth module for security" task with 3 finders + 9 verifiers + 1 synthesizer:

| Stage | Model | Calls | Est. Tokens | Cache | Cost |
|---|---|---|---|---|---|
| Orchestrator | V4-Pro | 2 turns | ~20K | Prefix cached | $0.008 |
| Finders (3×) | V4-Flash | 3 subagents | ~45K | System cached | $0.006 |
| Verifiers (9×) | V4-Pro | 27 subagents | ~216K | System cached | $0.034 |
| Synthesizer | V4-Flash | 1 subagent | ~12K | System cached | $0.002 |
| **Total** | | **33 calls** | **~293K** | | **~$0.05** |

Same task on Claude Code at Opus 4.8 rates: ~$7-15.

**reasonix cost advantage: ~99% over pure-Anthropic for subagent workloads.**
