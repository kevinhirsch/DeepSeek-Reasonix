# 07 — Background Tasks TUI Panel

**Phase:** Monitoring | **Effort:** 2 days | **Priority:** P1

## Problem

No UI for subagent status. User has zero visibility into what's running.

## Design

New file `internal/cli/background_panel.go`. Inline panel (normal buffer — preserves scrollback).

Layout pattern (from Agent Deck, 1,951★):
```
┌─ Background Tasks (3 running · 2 done · 1.2K tokens) ────┐
│  ● explore:auth    [running] 12 calls  last: grep          │
│    "Checking OAuth flow in middleware/auth.go... found 3"  │
│  ◉ verify:sql-1    [running]  5 calls  last: read_file 🌐 │
│  ✓ review:handlers [done]     8 calls  2.3K tokens        │
│  ○ report          [pending]  depends on: verify:*        │
└────────────────────────────────────────────────────────────┘
```

Status icons (dual: state + liveness): ● green running, ◐ yellow waiting, ✓ green done, ✗ red failed, ○ gray pending, 🌐 remote.

Interactions: Enter=peek output, s=send message to subagent, k=collapse. Poll at 500ms.

## Files
- `internal/cli/background_panel.go` (NEW)
- `internal/cli/chat_tui.go` — integrate panel
- `internal/cli/statusline_test.go` — add count to status line

## Acceptance
- [ ] Inline panel preserves scrollback
- [ ] Live tool count, last tool, reasoning tail for running tasks
- [ ] Completed tasks auto-collapse after 3s
- [ ] Panel auto-hides when empty
- [ ] `s` key sends message to selected subagent
- [ ] Remote workers show 🌐
- [ ] Color+icon works on dark and light terminals
