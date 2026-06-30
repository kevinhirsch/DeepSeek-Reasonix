# Prompt Calibration Strategy

> We can't copy Claude Code's prompts — they're Anthropic-proprietary and Claude-tuned.
> We CAN encode Claude Code's behavioral patterns into DeepSeek-adapted prompts
> and calibrate them through systematic evaluation.

---

## The Strategy: Encode Patterns, Not Copy Text

Claude Code's behavioral patterns are well-understood from the research. We don't need their exact prompts — we need to encode the SAME BEHAVIORAL CONTRACT into DeepSeek-adapted text.

| Claude Code Pattern | DeepSeek Adaptation |
|---|---|
| System prompt instructions | User message instructions (DeepSeek degrades on system prompts) |
| "Default to silence between tool calls" | Same, but stronger — DeepSeek over-narrates |
| "Your job is to try to break it" (verifier) | Same adversarial framing, works cross-model |
| "Report every finding" (reviewer) | Same, but add "do not self-censor" (DeepSeek sometimes filters) |
| "Lead with the outcome" | Same, but add explicit structure example (DeepSeek follows examples well) |
| "Don't add features beyond what was asked" | Stronger: add negation examples ("do NOT extract a helper", "do NOT add error handling") |
| Effort levels: low/medium/high/xhigh/max | Map to DeepSeek's binary: high/max with explicit behavioral differences documented |
| No temperature/top_p (rejected by Claude 4.x) | temperature 0.6 mandatory for subagents, documented in prompt |

## Prompt Evaluation Framework

Every prompt must be evaluated before shipping. This is a DEVELOPMENT prerequisite, not a CI gate.

```
reasonix prompt eval <role> [--model <name>] [--scenarios <file>]
```

### Evaluation Harness

```go
// internal/prompt/eval.go (new package)

type PromptEval struct {
    Role     string
    Prompt   string
    Model    string
    Effort   string
}

type EvalScenario struct {
    Name        string
    Input       string   // task description
    Expectations []Expectation
}

type Expectation struct {
    Kind    string   // "tool_used", "tool_not_used", "output_contains", "output_not_contains",
                    // "output_matches", "stops_within_n_calls", "no_narration"
    Value   string
    Count   int      // for stops_within_n_calls
}

type EvalResult struct {
    Scenario    string
    Passed      bool
    Failures    []string
    TokensIn    int
    TokensOut   int
    ReasoningTokens int
    Duration    time.Duration
    Transcript  string
}
```

### Standard Evaluation Scenarios

```
.reasonix/prompts/eval/
├── explorer.json       Explorer role scenarios
├── reviewer.json       Reviewer role scenarios
├── verifier.json       Verifier role scenarios
├── planner.json        Planner role scenarios
└── executor.json       Executor role scenarios
```

Example `explorer.json`:
```json
{
  "scenarios": [
    {
      "name": "find-single-caller",
      "input": "Find all callers of AuthMiddleware in this codebase",
      "expectations": [
        {"kind": "tool_used", "value": "grep"},
        {"kind": "stops_within_n_calls", "count": 10},
        {"kind": "output_contains", "value": "file:"},
        {"kind": "output_not_contains", "value": "I recommend"},
        {"kind": "no_narration"}
      ]
    },
    {
      "name": "negative-claim-is-evidenced",
      "input": "Is there a rate limiter in this project?",
      "expectations": [
        {"kind": "output_contains", "value": "searched for"},
        {"kind": "output_not_contains", "value": "does not exist"}
      ]
    },
    {
      "name": "broad-first-search-then-read",
      "input": "How does authentication work in this project?",
      "expectations": [
        {"kind": "tool_used", "value": "grep"},
        {"kind": "tool_used", "value": "read_file"},
        {"kind": "stops_within_n_calls", "count": 15},
        {"kind": "output_contains", "value": "file:"}
      ]
    },
    {
      "name": "no-narration-between-reads",
      "input": "Read auth.go, middleware.go, and handlers.go and summarize each",
      "expectations": [
        {"kind": "no_narration"},
        {"kind": "output_contains", "value": "auth.go"},
        {"kind": "output_contains", "value": "middleware.go"},
        {"kind": "output_contains", "value": "handlers.go"}
      ]
    }
  ]
}
```

### Evaluation Workflow

```
1. Load prompt variant → 2. Run all scenarios → 3. Score pass rate → 4. Iterate prompt → 5. Re-evaluate
```

**Calibration threshold:** ≥90% pass rate across all scenarios before the prompt ships.

## Prompt Iteration Loop

```
┌─────────────────────────────────────────────────────────────────┐
│  WRITE: Draft prompt based on behavioral contract                │
│    ↓                                                            │
│  EVAL: Run against scenario suite ($0.05-$0.10 per full eval)   │
│    ↓                                                            │
│  ANALYZE: Which expectations failed? Why?                        │
│    ↓                                                            │
│  TWEAK: Adjust prompt wording. Add counter-examples.            │
│    ↓                                                            │
│  RE-EVAL: Did pass rate improve?                                │
│    ↓                                                            │
│  SHIP: ≥90% pass rate → commit prompt to repo                   │
└─────────────────────────────────────────────────────────────────┘
```

## Prompt Baseline Reference

Every shipped prompt is versioned and stored:

```
.reasonix/prompts/
├── v1/
│   ├── explorer.md      Explorer system prompt (user message format)
│   ├── reviewer.md      Reviewer prompt
│   ├── verifier.md      Verifier prompt
│   ├── planner.md       Planner prompt
│   ├── executor.md      Executor prompt
│   └── deepseek_notes.md DeepSeek-specific adaptations
├── eval/                 Evaluation scenarios per role
│   ├── explorer.json
│   ├── reviewer.json
│   ├── verifier.json
│   ├── planner.json
│   └── executor.json
├── results/              Historical eval results
│   └── v1-2026-0701.json
└── CHANGELOG.md          Prompt iteration history
```

## DeepSeek Prompt Architecture

Every DeepSeek subagent prompt follows this structure:

```
1. ROLE IDENTITY (1-2 lines)
   "You are a <role> subagent. Your job is <one sentence>."

2. BEHAVIORAL RULES (bullet list, 5-8 items)
   Concrete, demonstrable rules. Each rule has a counter-example.
   "✓ Read broadly first, then read deeply.  ✗ Do not read every file."

3. TOOL GUIDANCE (2-3 lines)
   Which tools to prefer, which to avoid, and when.
   "Prefer grep over glob for finding callers. Use read_file to verify."

4. OUTPUT FORMAT (2-3 lines)
   Exact structure expected. Include a template.
   "Return: one paragraph conclusion, then bulleted findings with file:line."

5. STOP CONDITIONS (2-3 lines)
   When to stop. Token budget awareness.
   "Stop after 12 tool calls or when you can answer confidently."

6. DEEPSEEK-SPECIFIC NOTES (appended for DeepSeek providers only)
   "You are running on DeepSeek. Keep reasoning brief — it's billable."
   "If you feel uncertain, make a decision rather than cycling."
   "Do not re-verify successful tool calls."

7. COUNTER-EXAMPLES (appended for complex roles)
   "Example: For 'find all callers of X', grep for X, read the 3-5 most relevant files, stop."
   "Anti-example: Do NOT read every file that imports the package."
```

## Example: Complete Explorer Prompt (DeepSeek-Adapted, v1 Draft)

```
You are an explorer subagent. Your job is to sweep the codebase and return concrete
findings with file:line references, not summaries or opinions.

── Behavioral Rules ──
✓ Search broadly first (grep, glob, ls), then read the 3-8 most relevant files.
  ✗ Do not read every matching file — be selective.
✓ Return concrete references: file path, line number, symbol name.
  ✗ Do not return paragraphs of prose without file:line citations.
✓ Stop when you have enough evidence to answer the question.
  ✗ Do not keep searching for marginal improvements after you can answer.
✓ If the search space is too large, report what you covered and what remains.
  ✗ Do not silently return incomplete results.
✓ Do not propose fixes or evaluate code quality.
  ✗ Do not say "you should refactor this" or "consider using a different pattern."

── Tool Guidance ──
Start with grep or code_index to find candidates, then read_file to verify.
Use glob only for filename patterns, not for content search.
Limit read_file to the relevant line ranges when the file is large.

── Output Format ──
One paragraph conclusion. Then bullet list: `- file:line — one-sentence finding`
Example:
"The project has 3 callers of AuthMiddleware across 2 files:
- middleware.go:42 — called in the HTTP handler chain
- router.go:108 — registered in route setup
- router.go:156 — registered for admin routes"

── Stop Conditions ──
Stop after 15 tool calls OR when you can answer with confidence.
If 15 calls isn't enough, return what you found plus: "Remaining: [what's unexplored]."

── DeepSeek Notes ──
Your thinking is always on and billable as prompt input. Keep CoT brief.
If you're cycling between options, pick one and act. Re-decide only on new evidence.
Trust successful tool output — don't re-verify.

── Counter-Examples ──
"Find all callers of AuthMiddleware":
  ✓ grep "AuthMiddleware" → read_file middleware.go:38-50, router.go:105-160 → stop.
  ✗ read_file on every .go file in the project.
"How does auth work in this project?":
  ✓ grep "auth\|Auth\|login\|token" → glob "**/auth*" → read top 5 matches → stop.
  ✗ Start reading files from main.go and follow every import chain.
```

## Prompt Calibration Budget

| Role | Scenarios | Est. tokens per eval run | Est. cost per run | Runs to calibrate | Total calibration cost |
|---|---|---|---|---|---|
| Explorer | 4 | ~12K in, ~3K out | ~$0.003 | 5-8 iterations | ~$0.02 |
| Reviewer | 3 | ~15K in, ~4K out | ~$0.004 | 5-8 iterations | ~$0.03 |
| Verifier | 4 | ~10K in, ~2K out | ~$0.003 | 5-8 iterations | ~$0.02 |
| Planner | 3 | ~12K in, ~3K out | ~$0.003 | 5-8 iterations | ~$0.02 |
| Executor | 3 | ~15K in, ~6K out | ~$0.005 | 5-8 iterations | ~$0.03 |
| **Total** | **17** | | | | **~$0.12** |

Twelve cents to calibrate all five subagent roles to 90%+ pass rate. The eval framework itself is reusable — future prompt iterations use the same scenarios at the same cost.
