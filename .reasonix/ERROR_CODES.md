# Error Code Taxonomy

> Every new tool, provider, and subsystem must use structured error codes.
> Format: `ERR_<DOMAIN>_<CODE>` — all caps, underscores.

## Error Code Domains

| Domain | Prefix | Scope |
|---|---|---|
| Provider | `ERR_PROV_` | Provider construction, API calls, rate limits |
| Tool | `ERR_TOOL_` | Tool validation, execution, sandbox |
| Workflow | `ERR_WF_` | Workflow creation, stage execution, crash recovery |
| Subagent | `ERR_SUB_` | Subagent spawn, messenger, transcript |
| Remote | `ERR_REMOTE_` | Worker connection, auth, work queue |
| Repo | `ERR_REPO_` | Clone, fetch, GitHub API |
| Config | `ERR_CFG_` | Config loading, validation, migration |
| Schedule | `ERR_SCHED_` | Cron creation, firing, persistence |
| Export | `ERR_EXPORT_` | Session export, share, import |
| Sandbox | `ERR_SBOX_` | OS confinement, Docker, path validation |
| Quality | `ERR_QUAL_` | Promise detection, claim verification, completeness |

## Error Codes

### Provider (ERR_PROV_)

```
ERR_PROV_AUTH_FAILED          API key invalid or missing
ERR_PROV_RATE_LIMITED         Rate limit exceeded, retry after N seconds
ERR_PROV_OVERLOADED           Provider returning 529, retry with backoff
ERR_PROV_TIMEOUT              Request exceeded timeout
ERR_PROV_CONNECTION           Network error, cannot reach provider
ERR_PROV_INVALID_RESPONSE     Provider returned malformed response
ERR_PROV_CIRCUIT_OPEN         Circuit breaker open, provider temporarily disabled
ERR_PROV_FALLBACK_EXHAUSTED   All providers in fallback chain exhausted
ERR_PROV_MODEL_NOT_FOUND      Requested model not available on this provider
ERR_PROV_CONTEXT_EXHAUSTED    Context window exceeded
ERR_PROV_REFUSAL              Model refused the request (safety)
```

### Tool (ERR_TOOL_)

```
ERR_TOOL_INVALID_ARGS          Tool arguments failed validation
ERR_TOOL_MISSING_REQUIRED      Required parameter not provided
ERR_TOOL_EXECUTION_FAILED      Tool execution returned an error
ERR_TOOL_TIMEOUT               Tool execution exceeded time limit
ERR_TOOL_SANDBOX_BLOCKED       Sandbox prevented the operation
ERR_TOOL_PERMISSION_DENIED     Permission policy denied the tool
ERR_TOOL_NOT_FOUND             Tool not in registry
ERR_TOOL_RATE_LIMITED          Tool-level rate limit (e.g., web_search)
ERR_TOOL_SSRF_BLOCKED          web_fetch blocked internal IP
ERR_TOOL_PATH_ESCAPE           Path outside allowed roots
ERR_TOOL_OUTPUT_TOO_LARGE      Tool output exceeds maxToolOutputBytes
```

### Workflow (ERR_WF_)

```
ERR_WF_INVALID_STRATEGY        Unknown workflow strategy
ERR_WF_INVALID_STAGE           Stage configuration invalid
ERR_WF_CIRCULAR_DEPENDENCY     Stage dependency cycle detected
ERR_WF_MAX_TASKS_EXCEEDED      Would exceed max_total_tasks
ERR_WF_STAGE_FAILED            A stage failed (all subagents errored)
ERR_WF_CANCELLED               Workflow was cancelled by user
ERR_WF_CRASH_RECOVERY_FAILED   Could not recover workflow state after crash
ERR_WF_MANIFEST_CORRUPT        Workflow manifest file is corrupted
ERR_WF_RESUME_STALE            Workflow too old to resume (abandoned)
```

### Subagent (ERR_SUB_)

```
ERR_SUB_SPAWN_FAILED           Subagent could not be created
ERR_SUB_PROMPT_EMPTY           Prompt is required
ERR_SUB_TOOLS_EMPTY            No tools available for subagent
ERR_SUB_MESSENGER_NOT_FOUND    Target subagent not in messenger registry
ERR_SUB_TRANSCRIPT_LOAD        Could not load subagent transcript
ERR_SUB_TRANSCRIPT_SAVE        Could not save subagent transcript
ERR_SUB_CONTINUE_NO_SESSION    Continuation requires a persisted session
ERR_SUB_FORK_CONFLICT          continue_from and fork_from are mutually exclusive
```

### Remote (ERR_REMOTE_)

```
ERR_REMOTE_OFFLINE             Remote worker not reachable
ERR_REMOTE_AUTH_FAILED         Worker authentication failed
ERR_REMOTE_NOT_CONFIGURED      Remote name not in config
ERR_REMOTE_CAPACITY_FULL       All worker slots busy
ERR_REMOTE_TIMEOUT             Work item timed out on remote
ERR_REMOTE_CONTAINER_FAILED    Docker container creation failed
ERR_REMOTE_NONCE_REUSED        Replay attack detected
ERR_REMOTE_CERT_MISMATCH       TLS certificate fingerprint mismatch
ERR_REMOTE_IMPERSONATION       Duplicate worker identity detected
```

### Repo (ERR_REPO_)

```
ERR_REPO_CLONE_FAILED          Git clone failed
ERR_REPO_FETCH_FAILED          Git fetch failed
ERR_REPO_INVALID_URL           URL is not a valid git remote
ERR_REPO_NOT_FOUND             Repository does not exist
ERR_REPO_AUTH_REQUIRED         Authentication required (private repo)
ERR_REPO_GITHUB_RATE_LIMITED   GitHub API rate limit exceeded
ERR_REPO_GITHUB_NOT_AUTH       Not authenticated with GitHub
ERR_REPO_BRANCH_NOT_FOUND      Requested branch does not exist
```

### Config (ERR_CFG_)

```
ERR_CFG_PARSE                  TOML parse error
ERR_CFG_VALIDATION             Config validation failed
ERR_CFG_MIGRATION              Config migration failed
ERR_CFG_DEPRECATED             Deprecated field in use
ERR_CFG_REQUIRED_MISSING       Required field not set
ERR_CFG_TYPE_MISMATCH          Field has wrong type
ERR_CFG_UNKNOWN_SECTION        Unknown config section (forward compat)
```

### Schedule (ERR_SCHED_)

```
ERR_SCHED_INVALID_CRON         Cron expression is invalid
ERR_SCHED_MAX_JOBS             Maximum job count reached
ERR_SCHED_NOT_FOUND            Job ID not found
ERR_SCHED_DURABLE_WRITE        Could not persist durable job
ERR_SCHED_EXPIRED              Job has expired
```

### Export (ERR_EXPORT_)

```
ERR_EXPORT_SESSION_EMPTY       Nothing to export
ERR_EXPORT_WRITE_FAILED        Could not write export file
ERR_EXPORT_FORMAT_UNSUPPORTED  Unknown export format
ERR_EXPORT_SHARE_TOO_LARGE     Share package exceeds size limit
```

### Sandbox (ERR_SBOX_)

```
ERR_SBOX_UNAVAILABLE           Sandbox not available on this platform
ERR_SBOX_ENGINE_NOT_FOUND      Requested sandbox engine not registered
ERR_SBOX_CONTAINER_FAILED      Docker/Podman container creation failed
ERR_SBOX_WRITE_DENIED          Write outside allowed roots
ERR_SBOX_READ_DENIED           Read from forbidden root
ERR_SBOX_NETWORK_DENIED        Network access blocked by sandbox
```

### Quality (ERR_QUAL_)

```
ERR_QUAL_PROMISE_DETECTED      Final answer ends in a promise
ERR_QUAL_CLAIM_UNVERIFIED      Claim references file not read this turn
ERR_QUAL_COMPLETENESS_FAILED   Requested action not in response
ERR_QUAL_OVERENGINEERING       Changes exceed requested scope
ERR_QUAL_LOOP_DETECTED         Cognitive loop detected (DeepSeek)
ERR_QUAL_STORM_DETECTED        Storm breaker fired (repeated failures)
```

## Error Message Format

Every user-facing error must follow this structure:

```
<CODE>: <what happened>
Cause: <why it happened>
Action: <what to do next>
```

Example:
```
ERR_PROV_RATE_LIMITED: DeepSeek API rate limit exceeded
Cause: 10 requests/second limit reached for model deepseek-v4-flash
Action: Retrying in 30s. To increase throughput, add a second provider or reduce workflow concurrency.
```
