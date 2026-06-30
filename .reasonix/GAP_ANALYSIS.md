# Complete Tool & Skill Gap Analysis: Claude Code vs reasonix

> Every Claude Code tool and skill, mapped against reasonix. Gaps designed for implementation.

## Tools

### Core File & Code Tools — ✅ Parity or Better

| Claude Code | reasonix | Status |
|---|---|---|
| Read | read_file | ✅ |
| Write | write_file | ✅ |
| Edit | edit_file, multi_edit, delete_range, delete_symbol | ✅ **reasonix ahead** (multi_edit, delete_range, delete_symbol) |
| Glob | glob | ✅ |
| Grep | grep | ✅ |
| Bash | bash (+ background, kill, wait) | ✅ **reasonix ahead** (bg jobs) |
| NotebookEdit | notebook_edit | ✅ |
| PowerShell | bash (handles PS via ShellKind) | ✅ |

### Agent & Orchestration Tools

| Claude Code | reasonix | Status |
|---|---|---|
| Agent (fork/explore/plan/general-purpose) | task, read_only_task, parallel_tasks | ⚠️ Partial — needs role prompts (Issue #1), workflow (Issue #5) |
| Workflow (pipeline/barrier DSL) | — | ❌ **Gap** — Issue #5 designs this |
| SendMessage | — | ❌ **Gap** — Issue #4 designs this |
| EnterWorktree/ExitWorktree | — | ❌ **Gap** — not yet designed |
| TaskCreate/TaskGet/TaskList/TaskUpdate/TaskOutput/TaskStop | todo_write, complete_step, wait, bash_output, kill_shell | ⚠️ Partial — todo_write tracks, complete_step verifies. Missing: structured task DAG with dependencies, blocking/waiting, output retrieval |
| ScheduleWakeup | — | ❌ **Gap** — not designed |
| CronCreate/CronDelete | — | ❌ **Gap** — not designed |

### Research & Knowledge Tools

| Claude Code | reasonix | Status |
|---|---|---|
| WebSearch (server-side) | — | ❌ **Gap** |
| WebFetch | web_fetch (client-side) | ✅ |
| deep-research skill (multi-phase) | research skill (single subagent) | ❌ **Gap** — reasonix research is one agent, not a workflow |

### Monitoring & Notifications

| Claude Code | reasonix | Status |
|---|---|---|
| Monitor (stdout event stream) | — | ❌ **Gap** |
| PushNotification | notify (TurnDone, ApprovalRequest, AskRequest) | ⚠️ Partial — missing proactive push, missing custom message routing to phone |
| ReportFindings (typed findings list) | — | ❌ **Gap** |

### UX & Interaction

| Claude Code | reasonix | Status |
|---|---|---|
| AskUserQuestion (multiSelect, preview) | ask | ⚠️ Partial — ask is simpler, no multiSelect, no preview |
| Skill (invoke built-in skills) | slash_command, run_skill, read_only_skill | ✅ |
| DesignSync | — | ❌ Not needed (Claude-specific) |
| EnterPlanMode/ExitPlanMode | plan_mode toggle (Ctrl+Shift+P) | ✅ Different UX, same function |

### Background & Scheduling

| Claude Code | reasonix | Status |
|---|---|---|
| ScheduleWakeup (self-paced loop) | — | ❌ **Gap** |
| CronCreate/CronDelete | — | ❌ **Gap** |
| /loop (recurring prompt) | — | ❌ **Gap** |

---

## Skills

### Skills Claude Code Has That reasonix Is Missing

| Claude Code Skill | What It Does | Gap Priority |
|---|---|---|
| **deep-research** | Multi-phase: fan-out web searches → fetch sources → adversarial verify → synthesize cited report | **P0** |
| **code-review** | Review diff for bugs + simplifications at configurable effort. Uses ReportFindings for structured output. | **P0** |
| **verify** | Verify a change actually works by running the app and observing behavior | **P1** |
| **simplify** | Review changed code for reuse/simplification/efficiency cleanups, then apply fixes | **P1** |
| **loop** | Run a prompt on a recurring interval with self-paced wakeup | **P1** |
| **update-config** | Configure settings.json, hooks, permissions, env vars | **P1** |
| **keybindings-help** | Customize keyboard shortcuts | **P2** |
| **run** | Launch and drive the app to confirm a change works | **P1** |
| **review** | Review a GitHub PR (distinct from code-review which is for working diff) | **P1** |
| **fewer-permission-prompts** | Scan transcripts for read-only calls, add allowlist | **P2** |
| **claude-api** | Reference for Claude API (not needed for DeepSeek) | N/A |
| **design-sync** | Design system sync (Claude-specific) | N/A |

### Skills reasonix Has That Claude Code Is Missing

| reasonix Skill | What It Does |
|---|---|
| **init** | Bootstrap AGENTS.md from codebase analysis |
| **install-capability** | Install MCP servers and skills from URLs, packages, folders |
| **test** | Run tests, diagnose failures, apply fixes, re-run until green |
| **explore** | Wide-net read-only codebase investigation |
| **research** | Combined web_fetch + code reading |

---

## Gap Implementation Designs

### Gap 1: web_search Tool — P0

**Design:** Add `internal/tool/builtin/websearch.go`. DeepSeek doesn't have server-side web search, so this is a client-side tool. Uses a configurable search backend (first implementation: DuckDuckGo HTML scraping, fallback: user's configured search API).

```json
{
  "name": "web_search",
  "description": "Search the web. Returns result blocks with titles, URLs, and snippets.",
  "parameters": {
    "query": {"type": "string", "description": "Search query"},
    "max_results": {"type": "integer", "default": 10},
    "allowed_domains": {"type": "array", "items": {"type": "string"}}
  }
}
```

**Files:** `internal/tool/builtin/websearch.go` (NEW, ~200 lines)
**Effort:** 1 day

### Gap 2: deep-research Skill — P0

**Design:** This is NOT a single subagent. It's a skill that orchestrates the `workflow` tool (Issue #5) with a specific multi-phase pattern:

```
Phase 1: Decompose question into N search angles
Phase 2: N parallel web_search + web_fetch subagents (one per angle)
Phase 3: Extract falsifiable claims from fetched sources
Phase 4: Adversarial verify (3 skeptics per claim, 2/3 refutes to kill)
Phase 5: Synthesize cited report
```

This is implemented as `.reasonix/skills/deep-research/SKILL.md` (inline skill) that directs the parent agent to compose a workflow call. Can also be exposed as a `/deep-research` slash command.

**Files:** `.reasonix/skills/deep-research/SKILL.md` (NEW)
**Effort:** 0.5 day (skill is declarative, workflow tool does the heavy lifting)

### Gap 3: Monitor Tool — P0

**Design:** Add `internal/tool/builtin/monitor.go`. A tool that starts a background command and streams each stdout line as an event. The existing `bash` tool's `run_in_background` + `bash_output` already provides the infrastructure — Monitor wraps it with event streaming.

```json
{
  "name": "monitor",
  "description": "Start a background monitor that streams events from a long-running command. Each stdout line becomes an event notification.",
  "parameters": {
    "command": {"type": "string", "description": "Shell command to run"},
    "description": {"type": "string", "description": "Human-readable description shown in notifications"},
    "timeout_ms": {"type": "integer", "description": "Kill after this many ms"},
    "filter": {"type": "string", "description": "Only emit lines matching this regex"}
  }
}
```

**Files:** `internal/tool/builtin/monitor.go` (NEW, ~150 lines)
**Effort:** 1 day

### Gap 4: ReportFindings Tool — P0

**Design:** Add `internal/tool/builtin/report_findings.go`. Used by the code-review skill to output structured review findings that the UI can render as a typed list.

```json
{
  "name": "report_findings",
  "description": "Report code-review findings as a typed list. Used at the end of a review to surface structured results.",
  "parameters": {
    "level": {"type": "string", "enum": ["low", "medium", "high", "xhigh", "max"]},
    "findings": {
      "type": "array",
      "items": {
        "file": {"type": "string"},
        "line": {"type": "integer"},
        "summary": {"type": "string"},
        "failure_scenario": {"type": "string"}
      }
    }
  }
}
```

**Files:** `internal/tool/builtin/report_findings.go` (NEW, ~80 lines)
**Effort:** 0.5 day

### Gap 5: Cron + ScheduleWakeup — P1

**Design:** Add `internal/agent/cron.go` with a session-level cron scheduler. Jobs fire only while the REPL is idle (not mid-query). Uses standard 5-field cron in local timezone. Recurring jobs auto-expire after 7 days.

Two tools:
- `schedule_task` — create/delete/list cron jobs. `cron` expression, `prompt` to fire, `recurring` flag.
- `schedule_wakeup` — one-shot wakeup after N seconds (for self-paced /loop).

The infrastructure exists: `jobs.Manager` already handles background execution with session-scoped contexts.

**Files:** `internal/agent/cron.go` (NEW, ~200 lines), `internal/tool/builtin/schedule.go` (NEW, ~100 lines)
**Effort:** 2 days

### Gap 6: PushNotification (proactive) — P1

**Design:** Enhance `internal/notify/` to support proactive push. Currently only reacts to events (TurnDone, ApprovalRequest, AskRequest). Add:
- `send_notification` tool for the model to push arbitrary messages
- Remote push via webhook (for phone notifications when Remote Control is connected)
- Notification priority levels (info, warning, critical)

**Files:** `internal/notify/sender_*.go` (modify), `internal/notify/push.go` (NEW), `internal/tool/builtin/notify.go` (NEW)
**Effort:** 1.5 days

### Gap 7: Structured Task System — P1

**Design:** Enhance `todo_write` and `complete_step` with dependency tracking and task output retrieval. Claude Code's task system is essentially a structured todo list with:
- Blocking/waiting (tasks can block on other tasks)
- Output retrieval (get a completed task's result)
- Status tracking (pending → in_progress → completed)

The existing `todo_write` already tracks tasks. Add:
- `task_get` — retrieve a task by ID with full description and context
- `task_output` — retrieve a completed task's output
- `task_block` / `task_unblock` — dependency management between tasks
- Enhanced `complete_step` to verify against todo items

**Files:** `internal/tool/builtin/todo.go` (modify, ~150 lines added)
**Effort:** 1.5 days

### Gap 8: EnterWorktree — P1

**Design:** Add `internal/worktree/` package. Uses `git worktree add` to create isolated working copies for parallel subagents. The worktree is auto-removed when the subagent completes (unless kept).

```json
{
  "name": "enter_worktree",
  "description": "Create an isolated git worktree for a subagent. The subagent gets its own working copy to avoid conflicts.",
  "parameters": {
    "name": {"type": "string", "description": "Short name for the worktree"},
    "branch": {"type": "string", "description": "Branch to create (defaults to auto-generated)"}
  }
}
```

The sandbox already supports WriteRoots — the worktree becomes the subagent's workspace root, sandbox-enforced.

**Files:** `internal/worktree/worktree.go` (NEW, ~120 lines), `internal/agent/task.go` (modify to accept worktree option), `internal/tool/builtin/worktree.go` (NEW)
**Effort:** 1.5 days

### Gap 9: Skills (code-review, verify, simplify, run, loop) — P0-P1

**Design:** All of these are skill files, not code changes. Each is a `.reasonix/skills/<name>/SKILL.md` that uses existing tools. The skill infrastructure already supports inline and subagent execution.

| Skill | Execution | Tools Used | Effort |
|---|---|---|---|
| **code-review** | Inline → subagent for deep scan | bash (git diff), read_file, grep, report_findings | 0.5 day |
| **verify** | Inline | bash (run tests), read_file | 0.25 day |
| **simplify** | Inline | edit_file, multi_edit (apply simplifications) | 0.25 day |
| **run** | Inline | bash (launch app), wait, bash_output | 0.25 day |
| **loop** | Inline | schedule_task (cron) or schedule_wakeup | 0.25 day |
| **update-config** | Inline | edit_file (edit settings.json) | 0.25 day |
| **review (PR)** | Subagent | web_fetch (GitHub API), bash (git), read_file, report_findings | 0.5 day |
| **fewer-permission-prompts** | Inline | read_file, edit_file | 0.25 day |
| **keybindings-help** | Inline | edit_file (edit keybindings.json) | 0.25 day |

**Files:** `.reasonix/skills/{code-review,verify,simplify,run,loop,update-config,review,permissions,keybindings}/SKILL.md` (NEW, 9 files)
**Effort:** 2.5 days total for all 9 skills

### Gap 10: AskUserQuestion Enhancement — P2

**Design:** Enhance the existing `ask` tool to support multiSelect and preview rendering. The current `ask` tool accepts questions but has a flat structure.

```json
{
  "name": "ask",
  "parameters": {
    "questions": {
      "type": "array",
      "items": {
        "question": {"type": "string"},
        "header": {"type": "string"},
        "options": {
          "type": "array", 
          "items": {
            "label": {"type": "string"},
            "description": {"type": "string"},
            "preview": {"type": "string"}
          }
        },
        "multiSelect": {"type": "boolean"}
      }
    }
  }
}
```

**Files:** `internal/agent/ask.go` (modify schema), `internal/cli/` (render preview columns)
**Effort:** 1 day

---

## Consolidated Gap Summary

| # | Gap | Type | Effort | Priority |
|---|---|---|---|---|
| 1 | web_search tool | Code | 1 day | P0 |
| 2 | deep-research skill | Skill | 0.5 day | P0 |
| 3 | Monitor tool | Code | 1 day | P0 |
| 4 | ReportFindings tool | Code | 0.5 day | P0 |
| 5 | code-review skill | Skill | 0.5 day | P0 |
| 6 | verify skill | Skill | 0.25 day | P1 |
| 7 | simplify skill | Skill | 0.25 day | P1 |
| 8 | run skill | Skill | 0.25 day | P1 |
| 9 | review (PR) skill | Skill | 0.5 day | P1 |
| 10 | Cron + ScheduleWakeup | Code | 2 days | P1 |
| 11 | PushNotification (proactive) | Code | 1.5 days | P1 |
| 12 | Structured Task System | Code | 1.5 days | P1 |
| 13 | EnterWorktree | Code | 1.5 days | P1 |
| 14 | loop skill | Skill | 0.25 day | P1 |
| 15 | update-config skill | Skill | 0.25 day | P1 |
| 16 | AskUserQuestion enhancement | Code | 1 day | P2 |
| 17 | fewer-permission-prompts skill | Skill | 0.25 day | P2 |
| 18 | keybindings-help skill | Skill | 0.25 day | P2 |

**Total new effort: 13 days (on top of the 19.5 days from the 15-issue backlog).**

**Grand total with subagent architecture: ~32.5 days to bridge every gap.**

> Note: DesignSync and claude-api are Claude-specific and not applicable to reasonix.
