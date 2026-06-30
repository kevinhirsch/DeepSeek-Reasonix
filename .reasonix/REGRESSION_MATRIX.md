# Regression Test Matrix — Full Existing Featureset

> Every additive feature must pass regression tests against this matrix.
> A failing regression is a blocking bug. No new feature ships with a known regression.

## Tier 0: Architecture Constants (Must Never Change)

| ID | Invariant | Test Type | Why |
|---|---|---|---|
| R0.1 | Single `CGO_ENABLED=0` static binary | Build | Distribution contract — npm, Homebrew, direct download |
| R0.2 | BurntSushi/toml is the only external dependency | `go list -m all \| wc -l` | Spec commitment |
| R0.3 | Config resolution order: flag > project toml > user toml > defaults | Integration | User-facing contract |
| R0.4 | API keys NEVER stored in config files — `api_key_env` only | Static analysis | Security contract |
| R0.5 | Memory: `runtime.ReadMemStats()` shows zero monotonic heap growth | 50-iteration stress | Reliability contract |
| R0.6 | `go test -race` is clean on all packages | CI gate | Concurrency safety contract |

## Tier 1: CLI Surface (15 commands)

| ID | Command | Regression Test |
|---|---|---|
| R1.1 | `reasonix` (default chat) | TUI launches, renders markdown, accepts input |
| R1.2 | `reasonix run "prompt"` | Single-turn execution, non-zero exit on error |
| R1.3 | `reasonix serve --addr :8080` | HTTP+SSE responds, auth modes work |
| R1.4 | `reasonix complete` | Shell completion generates valid script |
| R1.5 | `reasonix resume` | Loads saved session, continues conversation |
| R1.6 | `reasonix rewind` | Restores workspace to checkpoint |
| R1.7 | `reasonix branch` | Creates named branch, switches context |
| R1.8 | `reasonix bot --kind feishu` | Bot runtime starts, accepts connections |
| R1.9 | `reasonix doctor` | Diagnostic report completes without panic |
| R1.10 | `reasonix upgrade` | Version check, update path works |
| R1.11 | `reasonix config` | Reads/writes config values correctly |
| R1.12 | `reasonix config auto-plan` | Toggles plan mode setting |
| R1.13 | `reasonix config memory-v5` | Toggles memory compiler |
| R1.14 | `reasonix config reasoning-language` | Gets/sets language preference |
| R1.15 | `reasonix --model NAME` | CLI flag overrides config default_model |

## Tier 2: TUI Components (14 components)

| ID | Component | Regression Test |
|---|---|---|
| R2.1 | Markdown renderer | Headers, lists, code blocks, CJK text |
| R2.2 | Diff view | Colored additions/removals, line numbers |
| R2.3 | Model switcher (`/model`) | Lists configured models, switches active |
| R2.4 | Theme engine | dark/light/auto, all theme_styles |
| R2.5 | Status line | Shows model, tokens, cost, context gauge |
| R2.6 | Tool cards | bash, read_file, write_file, edit_file rendered |
| R2.7 | Transcript view | Scrollable, proper message ordering |
| R2.8 | Chat input | Multi-line, paste handling, @ file references |
| R2.9 | Slash command picker | `/` opens menu, fuzzy matches |
| R2.10 | Resume picker | Lists sessions, Enter resumes |
| R2.11 | Skill picker | Lists skills, Enter invokes |
| R2.12 | MCP manager | Lists servers, toggle, add/remove |
| R2.13 | Hook viewer | Lists configured hooks, event types |
| R2.14 | Memory viewer | Lists REASONIX.md entries, editable |

## Tier 3: Agent Core (10 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R3.1 | Agent loop (single model) | User prompt → tool calls → final answer |
| R3.2 | Plan mode (read-only gate) | Toggle bans writer tools, toggle restores |
| R3.3 | Coordinator (two-model) | Planner proposes, executor carries out |
| R3.4 | Compaction | Long session auto-compacts, context stays under window |
| R3.5 | Storm breaker | 6 identical (tool, error) loops → breaker injects |
| R3.6 | Stream recovery | Interrupted SSE → partial text preserved → retry |
| R3.7 | Final-answer readiness gate | Unfinished todo → blocked → nudge |
| R3.8 | Max steps guard | Hit maxSteps → pause, not crash |
| R3.9 | Context window tracking | Usage → context gauge → compaction trigger |
| R3.10 | Reasoning language preference | zh/en/auto setting respected in output |

## Tier 4: Provider Surface (3 providers × N features)

| ID | Provider | Regression Test |
|---|---|---|
| R4.1 | Anthropic: Messages API | Basic message, streaming, tool use, thinking |
| R4.2 | Anthropic: Prompt caching | cache_control placement, cache hit verification |
| R4.3 | Anthropic: Thinking blocks | Signed thinking round-trip, display: summarized |
| R4.4 | Anthropic: Image input | base64 PNG, URL image |
| R4.5 | Anthropic: Stop reasons | end_turn, tool_use, max_tokens, refusal |
| R4.6 | Anthropic: Effort parameter | output_config.effort in request |
| R4.7 | Anthropic: Stop details | refusal → stop_details.category + explanation |
| R4.8 | Anthropic: Custom base_url | Gateway proxying, /v1 suffix stripping |
| R4.9 | OpenAI: DeepSeek | reasoning_content round-trip, effort mapping |
| R4.10 | OpenAI: MiniMax | thinking.type=adaptive, no reasoning_effort |
| R4.11 | OpenAI: Generic compatible | vanilla reasoning_effort scale, chat_url override |
| R4.12 | OpenAI: Vision | vision_detail, image in content array |
| R4.13 | OpenAI: Model fetching | /models probe, auto-detect capabilities |
| R4.14 | OpenAI: Reasoning protocol | auto/deepseek/openai/none detection |
| R4.15 | OpenAI: Stream reconnection | SSE drop → backoff → reconnect |
| R4.16 | All: Token counting | count_tokens returns provider-specific count |
| R4.17 | All: Retry logic | 429/5xx → exponential backoff → max_retries |
| R4.18 | All: Proxy support | HTTP_PROXY/HTTPS_PROXY honored |
| R4.19 | All: API key resolution | env var, credential store, keyring |
| R4.20 | All: Model resolution | provider/model, bare model, provider default |

## Tier 5: Built-in Tools (18 tools)

| ID | Tool | Regression Test |
|---|---|---|
| R5.1 | bash | Command execution, stdout/stderr capture, exit code |
| R5.2 | bash (background) | run_in_background, job ID returned |
| R5.3 | bash_output | Reads streaming output from bg job |
| R5.4 | kill_shell | Kills bg job, status updated |
| R5.5 | wait | Blocks until bg job completes, returns result |
| R5.6 | read_file | Reads file, line range, image/PDF detection |
| R5.7 | write_file | Creates file, overwrites, atomic write |
| R5.8 | edit_file | String replacement, exact match required |
| R5.9 | multi_edit | Multiple edits in one call |
| R5.10 | delete_range | Line range deletion |
| R5.11 | delete_symbol | Symbol-level deletion |
| R5.12 | move_file | Rename/move, cross-directory |
| R5.13 | ls | Directory listing, file details |
| R5.14 | glob | Pattern matching, recursive |
| R5.15 | grep | Regex search, file filtering, context lines |
| R5.16 | code_index | Go symbol outline, search, kind filter |
| R5.17 | web_fetch | URL fetch, SSRF protection, proxy support |
| R5.18 | todo_write | Creates/updates todo items |
| R5.19 | complete_step | Verifies completion against evidence |
| R5.20 | notebook_edit | Jupyter notebook cell edit |

## Tier 6: Sandbox + Permissions (5 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R6.1 | macOS Seatbelt | Command confined, write roots enforced |
| R6.2 | Linux bubblewrap | Command confined, tmpfs overlays |
| R6.3 | Windows unconfined | Command runs, warning emitted |
| R6.4 | Permission: allow/ask/deny | Config rules → decisions match |
| R6.5 | Permission: bash_readonly | Plan-mode safe commands only |
| R6.6 | Guardian: model-as-judge | Review request → assessment event |
| R6.7 | Guardian: circuit breaker | Consecutive denials → interrupt trigger |

## Tier 7: Hooks (11 events)

| ID | Hook Event | Regression Test |
|---|---|---|
| R7.1 | PreToolUse | Fires before tool call, can block (exit 2) |
| R7.2 | PostToolUse | Fires after tool call, output captured |
| R7.3 | PermissionRequest | Fires before approval prompt |
| R7.4 | UserPromptSubmit | Fires before user turn processed |
| R7.5 | Stop | Fires on session end |
| R7.6 | PostLLMCall | Fires after streaming, can translate reasoning |
| R7.7 | SessionStart | Fires when session becomes active |
| R7.8 | SessionEnd | Fires when session closes |
| R7.9 | SubagentStop | Fires when subagent finishes |
| R7.10 | Notification | Fires when agent needs attention |
| R7.11 | PreCompact | Fires before compaction, injects guidance |

## Tier 8: Skills (7 built-ins)

| ID | Skill | Regression Test |
|---|---|---|
| R8.1 | init | Bootstraps AGENTS.md, detects project type |
| R8.2 | explore | Subagent returns file:line references |
| R8.3 | research | Subagent returns code + web citations |
| R8.4 | install-capability | Plans + applies MCP/skill installs |
| R8.5 | review | Produces verdict + per-issue report |
| R8.6 | security-review | Severity-tagged security findings |
| R8.7 | test | Detects runner, runs, fixes, re-runs |

## Tier 9: Memory + Checkpoint (4 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R9.1 | REASONIX.md / AGENTS.md | Loaded into system prompt, cache-stable |
| R9.2 | Memory compiler | Execution traces → contracts, injected |
| R9.3 | Checkpoint: snapshot | Pre-edit snapshot saved, restorable |
| R9.4 | Checkpoint: rewind | Workspace restored to checkpoint state |

## Tier 10: MCP + Plugins (4 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R10.1 | MCP stdio transport | Subprocess starts, tools listed, called |
| R10.2 | MCP HTTP transport | SSE connection, tools listed, called |
| R10.3 | Plugin hot-add | New plugin detected mid-session |
| R10.4 | Plugin codegraph | Semantic tools preferred over code_index |

## Tier 11: Serve + Bot (4 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R11.1 | HTTP server | /v1/chat, /v1/events SSE stream |
| R11.2 | Auth: token | Bearer token validated |
| R11.3 | Auth: password | Hashed password validated |
| R11.4 | Bot: Feishu | Webhook received, agent responds |
| R11.5 | Bot: WeChat | Message received, agent responds |
| R11.6 | Bot: QQ | Message received, agent responds |

## Tier 12: Desktop App (6 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R12.1 | Multi-tab sessions | Tabs independent, settings per-tab |
| R12.2 | Workspace isolation | Project roots don't leak between tabs |
| R12.3 | Settings persistence | Model, theme, language saved/loaded |
| R12.4 | Auto-save | Session saved on close |
| R12.5 | Crash recovery | Pending state reconciled on restart |
| R12.6 | Single instance | Second launch → focuses existing window |

## Tier 13: Cross-Cutting (7 capabilities)

| ID | Capability | Regression Test |
|---|---|---|
| R13.1 | i18n: en | All strings resolve to English |
| R13.2 | i18n: zh | All strings resolve to Chinese |
| R13.3 | i18n: zh-tw | All strings resolve to Traditional Chinese |
| R13.4 | Config: credential store | keyring/file/env var resolution |
| R13.5 | Config: migration | v0.x → v1.0 migration succeeds |
| R13.6 | Config: .env loading | API keys resolved from .env file |
| R13.7 | Output styles | explanatory/learning/concise/custom apply |

---

## Summary

**104 regression tests** across 13 tiers covering the complete existing featureset.

A new feature is regressed if any test in R0-R13 that previously passed now fails. Tier 0 failures are **ship-blocking** (architecture invariant violation). Tier 1-4 failures are **release-blocking** (core user-facing behavior). Tier 5-13 failures are **fix-before-merge** (component-level regression).
