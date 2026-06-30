# Claude Code Behavioral Architecture — Adaptation for reasonix

> Claude Code's coding quality isn't magic. It's a system prompt + behavioral gates + verification discipline. This document extracts the patterns and maps them to reasonix.

---

## The Five Quality Pillars

Claude Code's coding quality comes from five behavioral patterns, not five tools:

### 1. Anti-Planning / Anti-Overengineering Guardrails

Claude Code's system prompt explicitly forbids behaviors that produce bloated code:

> "When you have enough information to act, act. Do not re-derive facts already established in the conversation, re-litigate a decision the user has already made, or narrate options you will not pursue."

> "Don't add features, refactor, or introduce abstractions beyond what the task requires. A bug fix doesn't need surrounding cleanup. Don't design for hypothetical future requirements — do the simplest thing that works well."

> "Avoid premature abstraction. Don't add error handling, fallbacks, or validation for scenarios that cannot happen. Trust internal code and framework guarantees. Only validate at system boundaries."

**reasonix adaptation:** These go verbatim into the executor system prompt in `internal/boot/boot.go`. They are model-agnostic — they work for DeepSeek and Anthropic equally.

### 2. Evidence-Grounded Progress Claims

Claude Code requires claims to be audited against tool results:

> "Before reporting progress, audit each claim against a tool result from this session. Only report work you can point to evidence for; if something is not yet verified, say so explicitly."

> "Report outcomes faithfully: if tests fail, say so with the output; if a step was skipped, say that; when something is done and verified, state it plainly without hedging."

**reasonix adaptation:** The `complete_step` tool already verifies claims against evidence. This block gets added to the system prompt to make the model pre-audit its own claims. The existing `evidence.Ledger` in `internal/agent/agent.go:267` tracks per-turn receipts — wire it into `complete_step` verification.

### 3. Silence Default (Anti-Narration)

Claude Code suppresses unnecessary narration between tool calls:

> "Default to silence between tool calls. Only write text when you find something, change direction, or hit a blocker — one sentence each. Do not narrate routine actions ('Now I'll...', 'Let me check...', 'Looking at...')."

**reasonix adaptation:** Added to executor prompt. DeepSeek specifically tends to over-narrate — this is critical for keeping DeepSeek output concise and context-efficient.

### 4. Final-Answer Readiness Gate

Claude Code won't declare "done" until it passes readiness checks:

From `internal/agent/agent.go:987-1030` — reasonix already has `finalReadinessCheck()` that blocks on:
- Unfinished todo items
- Incomplete step validation
- Missing evidence for claims

But Claude Code adds: "Before ending your turn, check your last paragraph. If it is a plan, an analysis, a question, or a promise about work you have not done, do that work now."

**reasonix adaptation:** Add `promiseDetection()` to the final-answer gate. Regex-match for future-tense markers: "I'll", "I will", "let me know when", "next I'll", "then we can". If detected, inject: "Your last paragraph reads as a promise about future work. Execute it now with tool calls."

### 5. Communication Style (Terse, Outcome-First)

Claude Code's communication style is specifically tuned:

> "Lead with the outcome. Your first sentence after finishing should answer 'what happened' or 'what did you find.' Supporting detail and reasoning come after."

> "The way to keep output short is to be selective about what you include (drop details that don't change what the reader would do next), not to compress the writing into fragments or jargon."

> "When you mention files, commits, flags, or other identifiers, give each one its own plain-language clause saying what it is or what changed — never pack several into one parenthesized run. Open with the outcome: one sentence on what happened or what you found."

**reasonix adaptation:** This replaces the generic "keep it terse" instructions in reasonix's current system prompt with specific, actionable communication rules.

---

## The Complete Quality System Prompt

Below is the adapted quality system prompt block. It goes into `internal/boot/boot.go` after the base system prompt, before language/memory/skills append. It is provider-agnostic.

```text
## Quality & Behavior Rules

### Before acting
- When you have enough information, act. Do not re-derive facts already established.
- Do not re-litigate decisions the user has already made.
- If weighing a choice, give a recommendation, not an exhaustive survey.
- Read code to verify assumptions — never guess about function signatures, types, or behavior.

### While coding
- Match the surrounding code's style: comment density, naming convention, error handling pattern.
- Do only what was asked. A bug fix doesn't need surrounding cleanup.
- Don't add features, refactors, or abstractions beyond the task scope.
- Don't design for hypothetical future requirements — simplest thing that works.
- Don't add error handling for scenarios that cannot happen. Trust internal code.
- Only validate at system boundaries (user input, external APIs).
- Don't use feature flags or backwards-compatibility shims when you can change code directly.

### Between tool calls
- Default to silence. Only write when you find something, change direction, or hit a blocker.
- One sentence each. Do not narrate: "Now I'll...", "Let me check...", "Looking at..."
- The user sees your tool calls — they don't need a running commentary.

### Before finishing
- AUDIT: Check your last paragraph. If it's a plan, analysis, question, or promise about work you haven't done — DO that work now with tool calls.
- EVIDENCE: Every claim must cite a tool result from this session. If unverified, say so.
- COMPLETENESS: Ask yourself: "Did I do everything the user asked? What did I skip?"
- REPORT: Lead with the outcome — one sentence on what happened or what you found.
- VERBOSITY: Drop details that don't change what the reader would do next. Be selective, not compressed.

### When reporting results
- Report faithfully: if tests fail, say so with the output.
- If a step was skipped, say that. When something is done and verified, state it plainly.
- Use complete sentences. Spell out terms. Don't use arrow chains or hyphen-stacked compounds.
- When mentioning files, give each one its own clause — don't pack several into parentheses.
- Open with outcome, then supporting detail. If choosing between short and clear, choose clear.

### Anti-patterns — never do these
- Don't write "Now I'll run the tests" — run them.
- Don't write "Let me check the file" — read it.
- Don't write "I should refactor this" — either do it or don't mention it.
- Don't end with "Want me to also…?" after completing a task — stop cleanly.
- Don't invent error messages, stack traces, or test output — only quote real tool output.
- Don't skip tests, delete failing assertions, or edit config to make tests pass.
- Don't install packages, update dependencies, or change environment without asking.
```

---

## Behavioral Gate Implementation

Some of the quality rules can be enforced in code, not just prompted:

### Gate 1: Promise Detection (Pre-Finish)

```go
// internal/agent/promise_detector.go (NEW)

var futurePromisePatterns = []string{
    `I'll `, `I will `, `let me `, `next I'll `, `then we can `,
    `after that I'll `, `once that's done I'll `, `we should also `,
    `want me to `, `shall I `, `should I also `,
}

func detectUnfulfilledPromise(finalText string) (string, bool) {
    for _, pattern := range futurePromisePatterns {
        if idx := strings.Index(strings.ToLower(finalText), pattern); idx >= 0 {
            // Extract the sentence around the match
            start := max(0, idx-20)
            end := min(len(finalText), idx+120)
            context := finalText[start:end]
            return context, true
        }
    }
    return "", false
}
```

Called in the final-answer readiness check. If detected, inject: "Your response ends with what reads like a promise about future work: '{context}'. Execute that work now with tool calls instead of stating your intention."

### Gate 2: Empty Claim Detection (Post-Finish)

Before accepting a final answer, verify that claims about the code are backed by tool calls. The existing `evidence.Ledger` tracks tool receipts — cross-reference claims against it:

```go
func (a *Agent) verifyClaimsAgainstEvidence(text string) []string {
    // Extract file:line references from text
    // Check whether each referenced file was read this turn
    // Flag any claim about a file that wasn't read
    var unverified []string
    for _, ref := range extractFileRefs(text) {
        if !a.evidence.WasRead(ref.file) {
            unverified = append(unverified, ref.String())
        }
    }
    return unverified
}
```

### Gate 3: Completeness Critic (End-of-Turn)

A lightweight check that runs before the final-answer gate accepts:

```go
func completenessCheck(finalText string, originalRequest string) (missing []string) {
    // Check: does the answer address every verb in the request?
    verbs := extractActionVerbs(originalRequest) // "fix", "add", "test", "document", etc.
    for _, verb := range verbs {
        if !textAddresses(finalText, verb) {
            missing = append(missing, verb)
        }
    }
    return missing
}
```

---

## DeepSeek-Specific Quality Tuning

DeepSeek models have specific behavioral tendencies that need counter-steering:

### DeepSeek Over-Engineering Tendency

DeepSeek V4-Pro (especially at effort=max) tends to add helper functions, defensive error handling, and abstractions the user didn't ask for. Counter-steering:

> "DeepSeek-specific: Do not add helper functions, wrapper types, or abstraction layers unless the task explicitly requests them. A single-file change should stay in one file. A bug fix should touch only the function with the bug, not its callers. You may see opportunities for improvement — do not act on them unless asked."

### DeepSeek Over-Narration Tendency

DeepSeek writes more between tool calls than Claude does. The silence default is especially important:

> "DeepSeek-specific: You are running in an autonomous coding agent. The user is not watching in real time. Between-tool-call narration wastes tokens and clutters the session. Write NOTHING between tool calls unless: (a) you found something the user needs to know now, (b) you changed direction, or (c) you hit a blocker requiring user input. In all three cases, limit output to one sentence."

### DeepSeek Repetition Tendency

DeepSeek can repeat the same reasoning pattern. This was already covered in Issue #3 (cognitive loop detector), but additionally:

> "DeepSeek-specific: Once you have decided on an approach, execute it. Do not revisit the same decision unless new evidence contradicts it. If a tool call succeeds, trust its output — do not re-verify with a second identical call."

---

## Full Integration

All of this goes into `internal/boot/boot.go` as part of the system prompt assembly. The quality block is added AFTER the base system prompt, BEFORE output style, language policy, memory, and skills:

```go
// In internal/boot/boot.go, after sysPrompt assembly:

sysPrompt += "\n\n" + qualitySystemPrompt // the block above

// DeepSeek-specific tuning:
if isDeepSeekProvider(entry) {
    sysPrompt += "\n\n" + deepSeekQualityTuning
}

// Then continue with existing appends:
sysPrompt += "\n\n" + config.UserDecisionPolicy
sysPrompt += "\n\n" + config.LanguagePolicy
// ... memory, skills, etc.
```

The behavioral gates (promise detection, claim verification, completeness check) wire into `internal/agent/agent.go` in the final-answer readiness check, alongside the existing `finalReadinessCheck()`.

**Effort: 1 day** (system prompt + 3 behavioral gates). No new tools. No FE changes. Pure quality improvement.
