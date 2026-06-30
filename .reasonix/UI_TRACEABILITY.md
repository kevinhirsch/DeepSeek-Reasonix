# New Feature → UI Scenario Traceability Matrix

> Every new feature/capability must have at least one UI-level Gherkin scenario.
> "UI" means: a panel, card, menu, modal, status indicator, keyboard shortcut,
> or visual state change the user can see.

## Yes — Has UI Coverage

| Issue/Gap | UI Feature Files | Key UI Scenarios |
|---|---|---|
| **01 Role Prompts** | `frontend-delight.feature` | Subagent label shows role icon per type |
| **02 DeepSeek Temp** | `agent-quality.feature` | Info notice emitted; notice renders in transcript |
| **03 Cognitive Loop** | `agent-quality.feature` | Warning notice emitted; warning icon in status |
| **04 Subagent Messenger** | `frontend-delight.feature` | send-message input bar, placeholder, Ctrl+J/Ctrl+K, cancel, empty rejection |
| **05 Workflow Tool** | `dynamic-workflows.feature`, `frontend-delight.feature` | Stage progress bar, pipeline flow, barrier "Waiting for" message, loop round counter, voting ✓✓✗, cancel summary |
| **06 Enriched Job View** | `background-monitoring.feature` | Tool count, last tool, reasoning tail, token count, result preview |
| **07 Background Panel** | `background-monitoring.feature`, `frontend-delight.feature` | Panel render, collapse animation, auto-hide, status line badge pulse, peek, send message, kill, scroll, many-tasks, long label truncation, unicode rendering |
| **08 Effort Calibration** | `background-monitoring.feature` | Model/effort shown in panel, cost color scaling |
| **09 Remote Bootstrap** | `remote-environments.feature` | Bootstrap script spinners, resource summary, API key validation |
| **10 Worker Package** | `remote-environments.feature` | Container lifecycle, event streaming |
| **11 Worker Binary** | `remote-environments.feature` | Binary download, startup |
| **12 Work Queue** | `remote-environments.feature` | Poll, claim, expire |
| **13 Docker Sandbox** | `remote-environments.feature` | Container creation, workspace mount, cleanup |
| **14 Remotes Config** | `remote-environments.feature` | Settings table: Name, Status, Load, OS, Uptime; status dot colors; offline "Last seen"; remote test command |
| **15 Remote Target** | `remote-environments.feature`, `frontend-delight.feature` | 🌐 indicator in panel; tooltip showing remote name + capabilities |
| **16 Quality Architecture** | `agent-quality.feature` | Notice on overengineering flag, silence enforcement, promise block |
| **web_search** | `web-and-research.feature` | Search results rendering |
| **deep-research** | `web-and-research.feature` | Research progress, phase transitions |
| **Monitor** | `monitoring-scheduling.feature` | Command output streaming, filter, timeout |
| **ReportFindings** | `structured-task-system.feature` | Findings list rendering, clickable file:line, severity grouping, outcome tracking |
| **Cron/Schedule** | `monitoring-scheduling.feature` | Schedule list, create, delete, one-shot vs recurring |
| **PushNotification** | `monitoring-scheduling.feature` | Desktop notification content, proactive vs reactive |
| **Task System** | `structured-task-system.feature` | Task list, status transitions, dependency unblock |
| **EnterWorktree** | `structured-task-system.feature` | Worktree create/exit, dirty detection |
| **code-review skill** | `skills-and-ux.feature` | Findings output, verdict rendering |
| **verify skill** | `skills-and-ux.feature` | Test output rendering, failure display |
| **simplify skill** | `skills-and-ux.feature` | Edit application rendering |
| **run skill** | `skills-and-ux.feature` | URL + PID display |
| **loop skill** | `skills-and-ux.feature` | Schedule management |
| **AskUserQuestion** | `skills-and-ux.feature` | multiSelect checkboxes, preview side-by-side, header chip, Other option |

## No — Missing UI Coverage

> **All gaps closed 2026-06-30.** Every new feature now has at least one UI-level Gherkin scenario.

| Issue/Gap | Where Added | Scenarios Added |
|---|---|---|
| **review (PR) skill** | `skills-and-ux.feature` | PR metadata header, per-file findings, progress indicator (3) |
| **update-config skill** | `skills-and-ux.feature` | Diff preview, validation error card (2) |
| **fewer-permission-prompts skill** | `skills-and-ux.feature` | Allowlist proposal list, confirm diff dialog (2) |
| **keybindings-help skill** | `skills-and-ux.feature` | Conflict resolution dialog, confirmation toast (2) |
| **TaskOutput** | `structured-task-system.feature` | Expandable result card, error with traceback toggle, live count badge (3) |
| **ScheduleWakeup** | `monitoring-scheduling.feature` | Countdown in status line, multiple wakeups, fire notification (3) |
| **CronList/CronDelete** | `monitoring-scheduling.feature` | Job list with next-fire times, empty state, delete confirmation (3) |

**18 UI scenarios added. Every new feature now has ≥1 UI scenario.**
