# UI Fidelity Bridge

> Gherkin says WHAT. Fidelity artifacts say EXACTLY.
> Every UI-heavy feature needs all three before implementation.

---

## 1. Component State Matrix

Every UI component → every possible state → what it looks like.

For the Background Tasks Panel:

```
State               | Icon    | Color  | Label           | Reasoning | Actions
────────────────────┼─────────┼────────┼─────────────────┼───────────┼────────
empty               | —       | —      | (panel hidden)  | —         | —
single_running      | ●       | green  | "auth-review"   | live tail | peek, send, kill
single_waiting      | ◐       | yellow | "auth-review"   | frozen    | peek, approve, deny
single_completed    | ✓       | green  | "auth-review"   | collapsed | peek, dismiss
single_failed       | ✗       | red    | "auth-review"   | collapsed | peek, retry, dismiss
single_killed       | ⊘       | gray   | "auth-review"   | collapsed | peek, dismiss
single_remote       | 🌐      | green  | "auth-review"   | live tail | peek, send, kill
multiple_mixed      | (per row) | (per) | (per subagent)  | (per)     | (per)
many_overflow       | ●...    | green  | "+12 more"      | collapsed | scroll
compensation_active | ⚙️      | dim    | "Guardian (3/h)"| summary   | dashboard
```

## 2. Pixel-Level Mockups

Not wireframes. Character-precise terminal renderings.

For the background panel at 80 columns:

```
┌─ Background Tasks (3 running · 2 done · $0.03) ────────────────────────┐
│                                                                         │
│  ●  explore:auth-module    [running]  12 calls  last: grep      $0.001  │
│     "Checking OAuth flow in middleware/auth.go... found 3 handlers..."  │
│                                                                         │
│  ◐  verify:sql-injection   [waiting]   5 calls  last: read_file $0.002  │
│     "Waiting for tool approval: bash — 'rm -rf /tmp/build'"             │
│                                                                         │
│  🌐 review:api-handlers    [running]   8 calls  last: bash      $0.003  │
│     "Running on build-server (Ubuntu 24.04, 4/8 CPUs) · 12ms latency"  │
│                                                                         │
│  ✓  plan:refactor          [done]      8 calls  2.3K tokens    $0.004  │
│  ✓  test:handlers          [done]      5 calls  0.8K tokens    $0.001  │
│                                                                         │
│  ○  report:security-audit  [pending]  depends on: verify:*              │
│                                                                         │
│  [s:send message] [Enter:peek] [k:kill] [j/k:navigate] [Esc:close]     │
└─────────────────────────────────────────────────────────────────────────┘
```

## 3. Responsive Breakpoints

Every layout at every supported width.

```
Terminal width   Layout
─────────────────────────────────────────────────────
<40 cols          Degraded: single-line indicators only, no panel
40-59 cols        Compact: 1 line per subagent, truncated labels, no reasoning tail
60-79 cols        Standard: 2 lines per subagent, 100-char reasoning tail
80-119 cols       Full: 2 lines per subagent, 200-char reasoning tail, model/effort visible
120+ cols         Wide: 2 lines per subagent, 400-char reasoning tail, cost, remote info
```

## 4. Animation & Timing Specification

Not "animates smoothly." Exact values.

```
Animation          Duration  Easing        Trigger
──────────────────────────────────────────────────
panel_appear       200ms     ease-out      first subagent spawns
row_insert         150ms     ease-out      new subagent added
row_collapse       300ms     ease-out      subagent completes + 2s delay
row_expand         150ms     ease-out      peek opened
icon_pulse         200ms     ease-in-out   status line count changes
shake_on_reject    300ms     spring(2,10)  empty message sent
reduced_motion     0ms       —             all animations instant
```

## 5. Color Palette with Contrast Ratios

Every color used in every theme, with verified contrast.

```
Element         Dark Theme              Light Theme             Contrast (dark/light)
─────────────────────────────────────────────────────────────────────────────────────
running_icon    #22C55E on #1A1B26      #16A34A on #FFFFFF     6.1:1 / 4.6:1
waiting_icon    #EAB308 on #1A1B26      #CA8A04 on #FFFFFF     8.2:1 / 4.8:1
failed_icon     #EF4444 on #1A1B26      #DC2626 on #FFFFFF     5.8:1 / 4.5:1
completed_icon  #22C55E on #1A1B26      #16A34A on #FFFFFF     6.1:1 / 4.6:1
pending_icon    #6B7280 on #1A1B26      #9CA3AF on #FFFFFF     4.5:1 / 3.0:1
reasoning_text  #9CA3AF on #1A1B26      #6B7280 on #FFFFFF     5.2:1 / 5.8:1
label_text      #F9FAFB on #1A1B26      #111827 on #FFFFFF     14.1:1 / 18.5:1
metadata_text   #6B7280 on #1A1B26      #6B7280 on #FFFFFF     4.5:1 / 5.8:1
panel_border    #374151 on #1A1B26      #D1D5DB on #FFFFFF     4.5:1 / 1.8:1
```

## 6. Keyboard Flow Diagram

Not a list of shortcuts. A state machine of what keys do what in what context.

```
[NORMAL MODE]
  j/k        → move selection down/up
  Enter      → peek selected subagent
  s          → enter SEND MODE
  k (on row) → kill selected subagent
  Ctrl+B     → toggle panel visibility
  Esc        → close panel, return focus to chat

[SEND MODE]
  <any char> → append to message buffer
  Backspace  → delete last char
  Ctrl+J     → send message (if non-empty) OR reject with shake (if empty)
  Esc        → cancel, return to NORMAL MODE
  Ctrl+K     → cancel, return to NORMAL MODE

[PEEK MODE]
  j/k        → scroll output up/down
  Enter      → close peek, return to NORMAL MODE
  Esc        → close peek, return to NORMAL MODE

[CONFIRM MODE]  (kill, retry, dismiss)
  y/Enter    → confirm action
  n/Esc      → cancel action
```

## 7. Component Test Contract

Every UI component → what the Go/React test must verify.

For BackgroundPanel (Go/Bubble Tea):

```go
// Test: panel renders empty state (hidden)
// Test: panel renders single running subagent
// Test: panel renders multiple subagents with mixed states
// Test: panel auto-hides when all subagents dismissed
// Test: panel scrolls when count > viewport
// Test: panel updates live when job state changes
// Test: panel handles unicode in subagent labels
// Test: panel handles CJK in reasoning tail
// Test: panel truncates long labels at width boundary
// Test: panel renders at 40, 60, 80, 120 cols
// Test: panel colors adapt to dark/light/high-contrast themes
// Test: panel animations disabled in reduced-motion mode
// Test: keyboard: j/k navigation, Enter peek, s send, k kill
// Test: keyboard: Esc closes, Ctrl+J sends, Ctrl+K cancels
// Test: send mode: character counter, empty rejection, successful dispatch
// Test: peek mode: tool history, scroll, close
```

## 8. Screenshot Baseline

Every UI-heavy feature → a screenshot taken at implementation time, stored as the reference.

```
.reasonix/screenshots/
├── background-panel/
│   ├── empty.png
│   ├── single-running.png
│   ├── multiple-mixed.png
│   ├── peek-open.png
│   ├── send-mode.png
│   ├── dark-theme.png
│   ├── light-theme.png
│   └── narrow-60col.png
├── repo-picker/
│   ├── welcome.png
│   ├── repo-list.png
│   ├── clone-progress.png
│   └── branch-selector.png
└── ...
```

Screenshots aren't tests. They're the visual reference that answers "does this look right?" without reopening the app.

## Implementation

This doesn't need new code. It needs new files alongside each feature's Gherkin:

```
.reasonix/features/background-monitoring.feature    ← existing
.reasonix/fidelity/background-panel/
├── state-matrix.md       ← every state, what it looks like
├── mockup-80col.txt      ← pixel-level reference at standard width
├── mockup-60col.txt      ← compact breakpoint
├── mockup-120col.txt     ← wide breakpoint
├── colors.md             ← palette with contrast ratios
├── animations.md         ← exact timing values
├── keyboard.md           ← state machine diagram
├── test-contract.md      ← what Go/React tests must verify
└── baseline.png          ← screenshot at ship time
```

**Effort:** ~1 hour per UI-heavy feature to produce the fidelity artifacts. For the ~15 features with significant UI: ~2 days total.

**Value:** Eliminates the "spec says green but which green" ambiguity. Eliminates the "I built what the spec said but it doesn't look right" conversation. Eliminates the "it works at 80 columns but breaks at 60" bug. The Gherkin scenarios test behavior. The fidelity artifacts test appearance. Together, you can't ship a feature that looks wrong or behaves wrong without a test catching it.
