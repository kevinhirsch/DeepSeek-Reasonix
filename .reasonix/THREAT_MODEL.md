# Threat Model

> What an attacker can do, and how reasonix defends against it.
> Scope: the remote worker, work queue, cross-machine communication channels.

## Assets

| Asset | Value | Location |
|---|---|---|
| User API keys (DeepSeek, Anthropic) | High — enables billing fraud, data access | Local credential store; worker env vars |
| Work item payloads (prompts, file paths) | Medium — reveals code structure, business logic | In transit; at rest in work queue |
| Subagent output (code, findings, reports) | Medium — reveals code, bugs, decisions | In transit; at rest in session transcripts |
| GitHub OAuth token | High — repository access | Local credential store |
| Worker identity (private key) | High — enables impersonation | Remote worker filesystem |
| Session transcripts | Medium — reveals conversation history | Local disk |

## Threat Actors

| Actor | Capability | Target |
|---|---|---|
| Network eavesdropper | Passive intercept of work queue traffic | Work item payloads, API keys in transit |
| Compromised build server | Full access to worker filesystem, env vars | API keys, worker identity, work item contents |
| Malicious reasonix plugin | Access to tool calls within sandbox | Workspace files, sandbox escape |
| Rogue worker | Impersonates a legitimate worker | Work queue, API key usage |

## Trust Boundaries

```
┌─ Trusted ─────────────────────┐     ┌─ Semi-Trusted ──┐     ┌─ Untrusted ────────────┐
│                                │     │                  │     │                         │
│  Local reasonix process        │────▶│  reasonix serve  │────▶│  Remote worker host     │
│  (user's machine)              │     │  (user-controlled │     │  (cloud/build server)   │
│                                │     │   server)        │     │                         │
│  Credential store ✓           │     │  TLS terminator ✓ │     │  Docker containers ✓   │
│  Sandbox (seatbelt/bwrap) ✓  │     │  Auth gate ✓      │     │  Worker binary          │
└────────────────────────────────┘     └──────────────────┘     └─────────────────────────┘
```

## Threats and Mitigations

### T1: API Key Exfiltration from Remote Worker

**Threat:** Compromised build server reads worker env vars containing API keys.

**Mitigation:**
- API keys on worker are scoped to specific models with spending limits
- Keys never written to worker filesystem — injected as Docker env vars, tmpfs only
- Worker has no API to retrieve its own configured keys
- Keys are rotated on worker token rotation
- Spending limits catch exfiltration within one billing cycle

### T2: Worker Impersonation

**Threat:** Attacker registers as "build-server" and claims work items intended for the real worker.

**Mitigation:**
- Each worker has a unique authentication token (not shared)
- First-connection identity challenge: worker must prove possession of its private key
- Duplicate connection detection: second "build-server" is rejected, existing connection survives
- Token rotation invalidates old tokens immediately

### T3: Work Item Interception (MITM)

**Threat:** Network attacker intercepts work item payloads in transit, reading prompts and file paths.

**Mitigation:**
- TLS on all work queue endpoints (HTTPS required, HTTP rejected)
- Certificate pinning: worker validates server certificate fingerprint
- Work item payloads encrypted with worker's public key (only intended worker can decrypt)
- Decryption happens inside the Docker container, output to tmpfs

### T4: Replay Attack

**Threat:** Attacker captures a valid work-claim request and replays it to steal work or corrupt state.

**Mitigation:**
- Nonce on every work-claim request — replay detected by duplicate nonce
- Sequence numbers on event stream submissions — duplicate sequence numbers silently dropped
- Timestamp validation: requests older than 30s rejected

### T5: Sandbox Escape from Malicious Plugin

**Threat:** A malicious MCP plugin or skill attempts to read/write outside the workspace.

**Mitigation:**
- Sandbox enforced at OS level (Seatbelt/bubblewrap), not Go level
- WriteRoots: only workspace + temp + toolchain caches
- ForbidReadRoots: configurable blocklist
- Docker sandbox for remote: full container isolation with restricted network
- Plugin tools run as subprocesses, not in-process

### T6: Prompt Injection via Malicious Subagent Output

**Threat:** A compromised subagent returns output designed to manipulate the parent agent.

**Mitigation:**
- Subagent output is treated as data, not instructions
- Parent agent's system prompt instructs it to verify subagent claims against evidence
- Adversarial verification: subagent claims checked by independent verifier subagents
- Subagent output is isolated in the event stream (nested under the tool call)

### T7: Credential Leakage in Session Transcripts

**Threat:** API keys or tokens appear in tool output and are persisted in session transcripts.

**Mitigation:**
- API keys never in config files
- Keys masked in logs: `sk-...abc`
- `maxToolOutputBytes` (32KB) limits transcript exposure
- Session export with `--redact-paths` strips absolute paths
- Credential store (keyring/keychain) for tokens, never in .env or .toml

## Security Review Cadence

- Threat model reviewed quarterly
- New features that touch trust boundaries require threat model update
- Remote worker security reviewed on every major version bump
- Penetration testing: annual third-party engagement recommended
