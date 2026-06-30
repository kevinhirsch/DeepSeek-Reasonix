# CLI Flag Additions

> Every new feature that needs CLI flags. Existing flags must not break.

## New Top-Level Commands

```
reasonix clone <url>                Clone a git repo and start a session
  --dir <path>                      Clone to custom directory (default: ~/reasonix-projects/<owner>-<repo>)
  --branch <name>                   Clone specific branch (default: repo default)
  --remote <name>                   Clone to remote worker instead of locally
  --recurse-submodules              Clone submodules recursively
  --no-init                         Skip auto-init of REASONIX.md
  --no-config                       Skip loading project reasonix.toml

reasonix github                     GitHub integration
  github login                      Authenticate with GitHub OAuth
  github logout                     Remove GitHub authentication
  github status                     Show authentication status
  github repos                      List accessible repositories
    --org <name>                    Filter by organization
    --search <query>                Search by name/description
    --sort <field>                  Sort by: updated, stars, name (default: updated)
    --limit <n>                     Max results (default: 50)

reasonix remote                     Remote worker management
  remote bootstrap --name <name>    Generate bootstrap script
    --os <auto|ubuntu|debian|fedora> Target OS (default: auto-detect)
  remote test <name>                Test connection to a remote worker
  remote list                       List configured remotes with status
  remote token rotate <name>        Rotate authentication token for a worker

reasonix workflow                   Workflow management
  workflow resume <id>              Resume an interrupted workflow
  workflow list                     List active and interrupted workflows
  workflow discard <id>             Discard an interrupted workflow
  workflow export <id>              Export workflow results

reasonix export                     Session export
  export                            Export current session as Markdown
    --format <md|json>              Output format (default: md)
    --findings-only                 Export only report_findings output
    --redact-paths                  Replace absolute paths with relative
  share                             Create shareable session package
  import <file>                     Import a shared session

reasonix schedule                   Schedule management
  schedule list                     List all scheduled tasks
  schedule delete <id>              Delete a scheduled task
  schedule run <id>                 Trigger a scheduled task immediately

reasonix metrics                    Observability
  metrics                           Show personal usage dashboard
    --range <day|week|month|year>   Time range (default: month)
    --by-role                       Breakdown by subagent role
    --by-model                      Breakdown by model
    --export <json>                 Export metrics as JSON

reasonix doctor                     Enhanced with new checks
  doctor                            Existing: diagnostic report
    --dry-run                       Validate config without starting agent
    --fix                           Attempt to fix common issues
```

## Modified Existing Commands

```
reasonix                           Default chat
  --repo <url>                     NEW: connect to repo and start session
  --branch <name>                  NEW: repo branch
  --remote <name>                  NEW: run against remote workspace

reasonix run                       Single-turn execution
  --target <local|remote-name>     NEW: where to run
```

## Flag Precedence (unchanged)

```
CLI flag > project reasonix.toml > user config.toml > built-in defaults
```

## New Environment Variables

| Variable | Purpose |
|---|---|
| `REASONIX_REMOTE_TOKEN` | Auth token for remote workers |
| `REASONIX_GITHUB_TOKEN` | GitHub PAT (fallback if OAuth not used) |
| `REASONIX_CLONE_ROOT` | Override default clone directory |
| `REASONIX_TELEMETRY_ENABLED` | Override telemetry opt-in (0/1) |
