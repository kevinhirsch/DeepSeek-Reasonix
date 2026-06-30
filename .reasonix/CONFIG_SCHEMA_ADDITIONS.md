# Config Schema Additions

> Every new TOML section needed. Existing config must not break.

## New Top-Level Sections

```toml
# ── Remote Workers ──────────────────────────────────────────────────

[[remotes]]
name = "build-server"
url = "https://build-server.example.com:9090"
auth_token = "${REASONIX_REMOTE_TOKEN}"   # env var reference, never literal
max_concurrent = 4
prefer_for = ["test", "build"]            # optional: route these subagent kinds
tls_skip_verify = false                   # default: strict TLS
tls_ca_cert = "/path/to/ca.pem"          # optional: custom CA for self-signed certs

# ── Repo Connection ─────────────────────────────────────────────────

[repos]
clone_root = "~/reasonix-projects"        # where repos are cloned locally
default_target = "local"                  # "local" | "<remote-name>"
auto_init_reasonix = true                 # run init if no REASONIX.md found
auto_load_config = true                   # load project reasonix.toml if present

# ── GitHub Integration ──────────────────────────────────────────────

[github]
# Token managed by credential store, not config. This section is for settings only.
default_org = ""                          # filter repo list to this org by default

# ── Workflow Settings ───────────────────────────────────────────────

[workflow]
max_total_tasks = 50                      # hard cap on subagent spawns per workflow
cost_confirm_threshold = 0.50             # prompt user if estimated cost exceeds this
crash_recovery = true                     # persist workflow state for crash recovery
workflow_ttl_days = 7                     # auto-clean completed workflows after this

# ── Scheduling ──────────────────────────────────────────────────────

[schedule]
durable_path = ".reasonix/scheduled_tasks.json"  # persisted jobs file
max_jobs = 20                             # max concurrent scheduled jobs

# ── Notifications ───────────────────────────────────────────────────

[notifications]
enabled = false
turn_done = true
approval_request = true
ask_request = true
workflow_complete = true                  # NEW
provider_outage = true                    # NEW
cost_threshold_exceeded = true            # NEW
push_webhook_url = ""                     # NEW: for mobile push notifications
```

## Modified Existing Sections

```toml
[agent]
# Existing fields preserved. New additions:

# Subagent model routing (already partially exists):
subagent_model = "deepseek-v4-flash"      # default for all subagents
subagent_models = { review = "deepseek-v4-pro", security_review = "deepseek-v4-pro", explore = "deepseek-v4-flash" }
subagent_effort = "high"                  # default effort for all subagents
subagent_efforts = { verify = "max", security_review = "max" }

# Fallback chain:
fallback_enabled = true                   # NEW: allow fallback when primary is down
fallback_cost_threshold = 0.50            # NEW: auto-fallback below this cost
fallback_chain = [                        # NEW: ordered fallback per role
  { role = "explore", providers = ["deepseek-v4-flash", "deepseek-v4-pro", "claude-haiku"] },
  { role = "verify", providers = ["deepseek-v4-pro", "claude-opus-4-8"] }
]

# Quality gates:
quality_gates = true                      # NEW: enable promise/claim/completeness detection

# Observability:
telemetry_enabled = false                 # NEW: opt-in anonymous telemetry
```

```toml
[[providers]]
# Existing fields preserved. New additions:

rate_limit_rps = 0                        # NEW: explicit rate limit (0 = use built-in default)
health_check_interval_seconds = 60        # NEW: how often to health-check
circuit_breaker_threshold = 5             # NEW: consecutive failures before circuit opens
circuit_breaker_cooldown_seconds = 60     # NEW: how long circuit stays open

[sandbox]
engine = "auto"                           # NEW: "auto" | "seatbelt" | "bwrap" | "docker" | "podman"

[accessibility]                           # NEW SECTION
screen_reader_mode = false
reduced_motion = "auto"                   # "auto" | "on" | "off"
audio_cues = false
text_scale = 1.0
focus_style = "default"                   # "default" | "high-contrast" | "thick"
```

## Backward Compatibility Rules

1. All new sections have defaults — existing configs don't need them
2. Unknown fields warn, never error (forward compatibility)
3. Config version bumped from 1 to 2 when any new section is first written
4. Migration creates backup before modifying
5. Provider entries without new fields get sensible defaults
6. `agent.subagent_model` and `agent.subagent_models` preserve existing behavior
