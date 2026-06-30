# Migration Path: v1.0 → v2.0

> For existing reasonix users upgrading from v1.0. Every change must have a migration.

## User-Visible Migration Steps

1. **Install v2.0:** `npm i -g reasonix@2` or `brew upgrade reasonix`
2. **First run:** Config migration runs automatically. Backup created.
3. **Review changes:** A notice summarizes what changed in the config.
4. **Done.** No manual steps required.

## What Changes Automatically

| v1.0 Config | v2.0 Config | Migration |
|---|---|---|
| `config_version = 1` | `config_version = 2` | Automatic |
| `[agent]` (existing) | `[agent]` + `quality_gates = true` etc. | New fields get defaults |
| `[[providers]]` (existing) | Same structure + `rate_limit_rps`, `health_check_*` | New fields get defaults |
| No `[workflow]` | `[workflow]` with defaults | Section added |
| No `[notifications]` sub-fields | `workflow_complete`, `provider_outage`, `push_webhook_url` added | New fields get defaults |
| No `[schedule]` | `[schedule]` with defaults | Section added |
| No `[repos]` | `[repos]` with defaults | Section added |
| No `[remotes]` | No automatic entry | User adds manually |
| No `[accessibility]` | `[accessibility]` with defaults | Section added |
| `[agent].subagent_model` (existing) | Same field preserved | No change |
| `[agent].subagent_models` (existing) | Same field preserved | No change |

## What Does NOT Change

- Provider entries: all `kind`, `base_url`, `model`, `api_key_env`, `prices` preserved verbatim
- Existing built-in tools: all preserved, all work identically
- Existing skills: all preserved, all work identically
- Sandbox config: preserved
- Permission rules: preserved
- Hook definitions: preserved
- MCP server configs: preserved
- Desktop settings: preserved
- Session transcripts: preserved, loadable in v2.0

## New Features Available Post-Migration

- Subagent role-specific prompts (automatic — no config needed)
- DeepSeek temperature override (automatic for subagents)
- Quality behavioral gates (automatic when `quality_gates = true`)
- Cognitive loop detector (automatic for DeepSeek)
- Workflow tool (available after first subagent spawn)
- Background tasks panel (visible when background tasks exist)

## Features Requiring User Configuration

- Remote workers: add `[[remotes]]` section
- GitHub integration: run `reasonix github login`
- Custom subagent models: configure `subagent_models`
- Fallback chains: configure `fallback_chain`
- Model fallback: configure `fallback_enabled`, `fallback_chain`
- Telemetry: opt in via `telemetry_enabled = true`
- Mobile push: configure `push_webhook_url`

## Rollback

- Backup file `reasonix.toml.v1.bak` created before migration
- To rollback: copy backup over reasonix.toml, reinstall v1.0
- Sessions created in v2.0 may not load in v1.0 (new transcript fields)

## Breaking Changes (None for Most Users)

- `temperature: 0.0` for DeepSeek subagents is overridden to 0.6 (info notice emitted)
- Reasoning language preference now applies to subagents
- Unknown config fields now warn (previously silently ignored)
