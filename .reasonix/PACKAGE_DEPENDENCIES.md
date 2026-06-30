# Package Dependency Graph

> New packages, modified packages, and their import relationships.
> Acyclic: higher layers import lower layers, never reverse.

## New Packages

```
internal/
├── subagent/              NEW: subagent prompt catalog + role resolution
│   └── prompts.go         RolePrompt(role, isDeepSeek) → string
├── messenger/             NEW: cross-agent message bus
│   └── messenger.go       SubagentMessenger: Register, Unregister, Send
├── workflow/              NEW: workflow orchestration
│   ├── workflow.go        WorkflowTool: pipeline/barrier/loop-until-dry
│   ├── stage.go           Stage execution, fan-out, voting
│   └── persist.go         Manifest + stage result persistence
├── worker/                NEW: remote worker daemon
│   ├── worker.go          Main loop: poll → claim → execute → stream
│   ├── config.go          Worker config loading
│   ├── poller.go          Long-poll GET /v1/work/pending
│   ├── executor.go        Docker container execution
│   └── streamer.go        Event relay to work queue
├── git/                   NEW: git operations
│   ├── clone.go           Clone, fetch, branch list, commit info
│   └── worktree.go        Worktree create/remove
├── github/                NEW: GitHub API client
│   ├── oauth.go           OAuth flow
│   ├── repos.go           Repo list, search
│   └── client.go          HTTP client with token auth
├── sandbox/
│   └── docker.go          NEW: Docker sandbox engine
├── schedule/              NEW: cron scheduler
│   └── cron.go            CronCreate, CronDelete, CronList, fire loop
├── monitor/               NEW: background command monitor
│   └── monitor.go         Event stream from long-running command
├── export/                NEW: session export
│   ├── markdown.go        Markdown export
│   └── share.go           Share package create/import
├── telemetry/             NEW: local telemetry
│   └── metrics.go         Accumulate, query, export
└── config/
    └── remote.go          NEW: RemoteEntry + validation (modify config.go)
```

## Modified Packages

```
internal/
├── agent/
│   ├── agent.go           ADD: quality gates (promise/claim/completeness)
│   ├── agent.go           ADD: deepseekCognitiveLoopDetected()
│   ├── task.go            MODIFY: role prompt injection, temperature override, remote target
│   ├── parallel_tasks.go  MODIFY: SubagentMessenger registration, remote target
│   └── session.go         MODIFY: workflow context
├── boot/
│   └── boot.go            ADD: quality block to sysPrompt, DeepSeek tuning, effort table
│   └── boot.go            ADD: workflow tool registration
│   └── boot.go            ADD: messenger wiring
│   └── boot.go            ADD: repo auto-setup on boot
├── tool/builtin/
│   ├── websearch.go       NEW: web_search tool
│   ├── monitor.go         NEW: monitor tool
│   ├── report_findings.go NEW: report_findings tool
│   ├── schedule.go        NEW: schedule_task + schedule_wakeup tools
│   ├── notify.go          NEW: send_notification tool
│   ├── worktree.go        NEW: enter_worktree + exit_worktree tools
│   └── workspace.go       MODIFY: add web_search, monitor, report_findings
├── jobs/
│   └── jobs.go            MODIFY: enriched Job.View, live state setters
├── event/
│   └── event.go           MODIFY: new event kinds (SteerReceived, WorkflowStage, WorkflowComplete)
├── config/
│   └── config.go          MODIFY: new config sections
├── cli/
│   ├── background_panel.go NEW: background tasks TUI panel
│   ├── repo.go            NEW: repo clone + github commands
│   ├── remote.go          NEW: remote bootstrap + test + list
│   ├── workflow.go        NEW: workflow resume + list + discard
│   ├── export.go          NEW: export + share + import
│   ├── schedule.go        NEW: schedule list + delete + run
│   └── metrics.go         NEW: /metrics dashboard
├── serve/
│   └── workqueue.go       NEW: work queue endpoints
└── sandbox/
    └── docker.go          NEW: Docker engine
```

## Import Direction (Acyclic)

```
cli → {workflow, messenger, subagent, git, github, worker, export, telemetry}
serve → {worker}
worker → {sandbox, git}
workflow → {subagent, messenger, agent}
messenger → {agent}
agent → {subagent, event}
subagent → {config}
git → {} (stdlib only)
github → {} (stdlib + net/http)
```

## What Stays Untouched

- `internal/provider/` — no changes to provider interface
- `internal/skill/` — no changes to skill infrastructure
- `internal/permission/` — no changes to permission policy
- `internal/guardian/` — no changes to guardian
- `internal/hook/` — no changes to hooks
- `internal/checkpoint/` — no changes to checkpoints
- `internal/memory/` — no changes to memory
- `internal/lsp/` — no changes to LSP
- `internal/plugin/` — no changes to plugin transport
- `internal/bot/` — no changes to bot integrations
- `desktop/` (Go backend + TS frontend) — additive only
