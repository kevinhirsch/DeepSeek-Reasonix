# 03 — DeepSeek Cognitive Loop Detector

**Phase:** Intelligence | **Effort:** 1 day | **Priority:** P0

## Problem

`stormBreaker` catches (tool, error) loops but misses DeepSeek cognitive loops: repeating reasoning, "tricky"/"confused", mechanical self-explanation without action.

## Design

Add `deepseekCognitiveLoopDetected()` in stream handler after each assistant turn. Three patterns:
1. Uncertainty escalation — "tricky"/"confused"/"ambiguous" count rising across turns
2. Identical reasoning prefix — first 200 chars match previous
3. Mechanical self-explanation — consecutive "I need to"/"Let me think" without tool calls

On detection: emit warning, inject breaker: "Stop reasoning and act. Make a concrete decision."

## Files
- `internal/agent/agent.go` — add detection, wire after ~line 978

## Acceptance
- [ ] Detects all 3 loop patterns
- [ ] Breaker message actually breaks loops
- [ ] No false positives on legitimate reasoning
- [ ] Only arms for DeepSeek providers
