# reasonix Codebase Architecture Audit

**Date:** 2026-06-30
**Source:** Direct codebase analysis of kevinhirsch/DeepSeek-Reasonix v1.0

## Architecture Summary

- **Language:** Go 67.4%, TypeScript 23.7% (legacy + desktop), CSS 6.8%
- **Distribution:** npm, Homebrew, single CGO_ENABLED=0 static binary, 6-platform cross-compile
- **Dependencies:** BurntSushi/toml (only external dep)
- **Config resolution:** flag > ./reasonix.toml > ~/.reasonix/config.toml > built-in defaults

## Key Packages

| Package | Purpose | Status |
|---|---|---|
| `internal/agent/` | Agent loop, session, subagent spawning, compaction | Complete |
| `internal/provider/` | Provider abstraction + OpenAI + Anthropic implementations | Complete |
| `internal/tool/builtin/` | 18 built-in tools | Complete |
| `internal/sandbox/` | Seatbelt (macOS) + bubblewrap (Linux) OS-level confinement | Complete |
| `internal/permission/` | Tiered allow/ask/deny policy | Complete |
| `internal/guardian/` | Model-as-judge safety reviewer subagent | Complete |
| `internal/hook/` | 9 hook event types, shell-command execution | Complete |
| `internal/skill/` | Progressive disclosure skill system, 7 built-in skills | Complete |
| `internal/plugin/` | MCP stdio + HTTP transports | Complete |
| `internal/jobs/` | Background job manager with session-scoped lifecycle | Complete |
| `internal/memory/` | REASONIX.md + AGENTS.md hierarchy | Complete |
| `internal/memorycompiler/` | Execution trace → contract compiler | Complete |
| `internal/checkpoint/` | Git-free snapshot-based edit safety net | Complete |
| `internal/command/` | Slash commands from .reasonix/commands/*.md | Complete |
| `internal/lsp/` | LSP client integration | Complete |
| `internal/serve/` | HTTP+SSE server frontend | Complete |
| `internal/bot/` | Feishu, WeChat, QQ bot integrations | Complete |
| `internal/notify/` | Desktop notifications (macOS/Linux/Windows) | Complete |
| `internal/config/` | TOML configuration loading and merging | Complete |
| `internal/boot/` | Controller assembly from configuration | Complete |

## Subagent Infrastructure (Existing)

- `task` tool — spawn subagent in own session, filtered tool whitelist
- `read_only_task` tool — read-only subagent, bash wrapped to plan-mode safety
- `parallel_tasks` tool — N subagents with DAG-based dependency ordering
- `SubagentStore` — persistent subagent identity, transcript continuation, fork
- `SubagentSpec` — model, effort, tool scope, system prompt per subagent
- `effectiveProfile()` — per-subagent model/effort resolution with fallback chain
- `resolveSubSessionRuntime()` — subagent provider resolution (can route to different model)
- `Steer()` / `steerQueue` — mid-turn message injection (exists, not cross-agent)
- Stream recovery — interrupted streams recovered up to 3 times
- Storm breaker — detects repeating (tool, error) loops, breaks after threshold
- Compaction — cache-aware summary compaction with structured headings
- Final-answer readiness gate — blocks completion on unfinished work

## Built-in Tools

bash, bash_output, kill_shell, wait, read_file, write_file, edit_file, multi_edit, delete_range, delete_symbol, move_file, ls, glob, grep, code_index, web_fetch, todo_write, complete_step, notebook_edit

## Built-in Skills

init, explore, research, install-capability, review, security-review, test
