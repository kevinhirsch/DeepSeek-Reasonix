# Persistence Design

> On-disk format, directory layout, and lifecycle for all new state.

## Directory Layout

```
.reasonix/
├── config.toml                    # existing
├── scheduled_tasks.json           # NEW: durable cron jobs
├── sessions/                      # existing
│   └── <session-id>/
│       ├── transcript.jsonl       # existing
│       └── subagents/             # existing
│           └── <sa-ref>/
│               ├── transcript.jsonl   # existing
│               └── meta.json          # existing
├── workflows/                     # NEW
│   └── <wf-id>/
│       ├── manifest.json          # workflow lifecycle state
│       ├── stage-1.json           # completed stage results
│       ├── stage-2.json
│       └── ...
├── exports/                       # NEW: /export output
│   └── <session>-<date>.md
├── share/                         # NEW: /share output
│   └── <session-id>.reasonix      # gzipped tarball
├── archive/                       # NEW: archived sessions
│   └── <session-id>/
├── worktrees/                     # NEW: EnterWorktree clones
│   └── <name>/
├── projects/                      # or ~/reasonix-projects/ (configurable)
│   └── <owner>-<repo>/            # cloned repos
└── telemetry/                     # NEW: local telemetry data
    └── metrics.jsonl              # append-only, one line per day
```

## Workflow Manifest Format

```json
{
  "id": "wf_01ABC...",
  "created_at": "2026-06-30T14:22:00Z",
  "updated_at": "2026-06-30T14:22:45Z",
  "status": "running",
  "strategy": "pipeline",
  "stages": [
    {"name": "find", "role": "review", "status": "completed", "subagent_count": 3},
    {"name": "verify", "role": "verify", "status": "running", "subagent_count": 9},
    {"name": "report", "role": "execute", "status": "pending", "subagent_count": 1}
  ],
  "items": ["auth.go", "middleware.go"],
  "cost_estimate": 0.05,
  "cost_actual": 0.03,
  "total_subagents": 13,
  "completed_subagents": 3,
  "parent_session": "sesn_01XYZ..."
}
```

## Stage Result Format

```json
{
  "stage": "find",
  "completed_at": "2026-06-30T14:22:10Z",
  "subagent_results": [
    {
      "ref": "sa_abc123",
      "item": "auth.go",
      "status": "completed",
      "findings": [
        {"file": "auth.go", "line": 42, "summary": "nil pointer", "severity": "high"}
      ],
      "tokens": {"input": 15000, "output": 800, "cache_hit": 12000},
      "cost": 0.002
    }
  ]
}
```

## Cron Job Persistence

```json
{
  "id": "cron_abc123",
  "created_at": "2026-06-30T14:22:00Z",
  "cron": "*/5 * * * *",
  "prompt": "Check health status",
  "recurring": true,
  "durable": true,
  "last_fired_at": null,
  "next_fire_at": "2026-06-30T14:25:00Z",
  "expires_at": "2026-07-07T14:22:00Z"
}
```

## Telemetry Format (Local)

```jsonl
{"date":"2026-06-30","tokens_in":45000,"tokens_out":8200,"cost":0.047,"workflows":2,"subagents":33,"cache_hit_rate":0.87,"loop_detector_fires":0}
{"date":"2026-07-01","tokens_in":32000,"tokens_out":6100,"cost":0.031,"workflows":1,"subagents":15,"cache_hit_rate":0.91,"loop_detector_fires":1}
```

## Atomic Write Guarantee

All persistence uses write-to-temp-then-rename:
```go
func atomicWrite(path string, data []byte) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmp, path) // atomic on same filesystem
}
```

## Cleanup

- Workflow directories: deleted after `workflow_ttl_days` (default 7)
- Archived sessions: never auto-deleted (user-managed)
- Worktrees: deleted on ExitWorktree(action="remove") or on controller close for stale ones
- Telemetry: oldest entries dropped when file exceeds 1MB (rolling window)
- Cron jobs: non-durable jobs deleted on session exit; durable jobs expire after 7 days
