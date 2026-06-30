# 08 — Effort Calibration Table

**Phase:** Monitoring | **Effort:** 0.5 day | **Priority:** P2

## Problem

Model has no guidance on which effort to use per subagent role. May use max for simple exploration or high for verification.

## Design

Add table to executor system prompt in `internal/boot/boot.go`:

| Role | Effort | Why |
|---|---|---|
| explore, read_only | high | DeepSeek high is already capable |
| plan | high | Architecture doesn't need deeper |
| review | high | Coverage > depth per finding |
| verify | max | Must thoroughly attempt refutation |
| security_review | max | Adversarial thinking needed |
| complex write | max | Multi-file changes |
| simple write | high | Single-file; overthinking adds latency |

DeepSeek: only high/max. low/medium→high, xhigh→max.

## Files
- `internal/boot/boot.go` — add table to sysPrompt

## Acceptance
- [ ] Table in executor's system prompt
- [ ] Model uses correct effort per role
- [ ] Provider-aware (DeepSeek binary, Anthropic full range)
