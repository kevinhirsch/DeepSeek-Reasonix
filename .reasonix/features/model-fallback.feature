Feature: Model Fallback & Provider Resilience
  As a developer who depends on reasonix working
  I want the system to gracefully handle provider outages without silent cost spikes
  So that I'm never blocked by an API outage and never surprised by a bill

  Background:
    Given reasonix has DeepSeek Flash, DeepSeek Pro, and Anthropic Claude configured
    And provider health checks run on boot and periodically

  # ── Provider Health ───────────────────────────────────────────────

  @fallback-health-check
  Scenario: Provider health check on boot
    When reasonix starts
    Then each configured provider is health-checked with a minimal API call
    And the health result (healthy/degraded/down) is surfaced in the status line
    And a notice is emitted for any unhealthy provider

  @fallback-health-check
  Scenario: Provider health check runs periodically
    Given reasonix has been running for 10 minutes
    When the periodic health check fires
    Then each provider is re-checked
    And if a previously-down provider recovers, a notice is emitted: "DeepSeek Flash is back online"
    And the status line updates immediately

  @fallback-health-check
  Scenario: Health check uses minimal tokens
    When a provider is health-checked
    Then the health check request uses a minimal prompt (≤10 tokens)
    And the response is discarded
    And the health check does not count toward the user's token tracking

  # ── Fallback Chain ────────────────────────────────────────────────

  @fallback-chain-explorer
  Scenario: Explorer role falls back through chain
    Given the configured model for explore is DeepSeek Flash
    And DeepSeek Flash is down
    When an explorer subagent is spawned
    Then a fallback warning is emitted: "DeepSeek Flash is down. Trying DeepSeek Pro. This will cost ~3× more."
    And the subagent runs on DeepSeek Pro
    And the cost estimate updates to reflect Pro pricing

  @fallback-chain-explorer
  Scenario: Fallback chain exhausted
    Given deepseek flash is down
    And deepseek pro is down
    And anthropic haiku is available
    When an explorer subagent is spawned
    Then the fallback warning shows the full chain traversal
    And the user is prompted: "All DeepSeek providers are down. Fall back to Anthropic Haiku? This will cost ~7× more. [y/N]"
    And the subagent runs on Haiku only if the user confirms

  @fallback-chain-verify
  Scenario: Verifier role falls back through chain
    Given the configured model for verify is DeepSeek Pro
    And DeepSeek Pro is down
    When a verify subagent is spawned
    Then deepseek flash is NOT attempted (flash is insufficient for verification)
    And the fallback goes directly to the next capable model in the chain
    And the user is warned about the cost difference

  @fallback-chain-no-silent-cross-provider
  Scenario: Cross-provider fallback requires user consent
    Given a DeepSeek provider is down
    And the only available fallback is Anthropic
    When a subagent would be spawned
    Then the fallback does NOT happen silently
    And the user is prompted with explicit cost comparison
    And the subagent waits for user response before proceeding

  @fallback-chain-configured
  Scenario: User-configurable fallback chains
    Given the user configured a fallback chain in reasonix.toml
    When a provider in the chain is down
    Then the fallback follows the user's configured order
    And the user's cost threshold for auto-fallback is respected

  # ── Circuit Breaker ───────────────────────────────────────────────

  @fallback-circuit-breaker
  Scenario: Circuit breaker opens after repeated failures
    Given DeepSeek Flash has returned 5xx for 5 consecutive requests
    When the 6th request would be made
    Then the circuit breaker opens for DeepSeek Flash
    And the provider is marked "degraded" in the status line
    And subsequent requests skip Flash and go to the fallback chain

  @fallback-circuit-breaker
  Scenario: Circuit breaker half-opens for probe
    Given DeepSeek Flash circuit breaker has been open for 60 seconds
    When the probe interval elapses
    Then one request is allowed through to test if Flash is healthy
    And if it succeeds, the circuit closes and Flash is restored
    And if it fails, the circuit remains open and the timer resets

  @fallback-circuit-breaker
  Scenario: Circuit breaker resets on sustained health
    Given the circuit breaker was open for DeepSeek Flash
    And 3 consecutive probe requests succeeded
    When the 3rd success is registered
    Then the circuit closes
    And a notice is emitted: "DeepSeek Flash has recovered"
    And the provider is restored to the active pool

  # ── Cost-Aware Fallback ───────────────────────────────────────────

  @fallback-cost-aware
  Scenario: Cost estimate shown before expensive fallback
    Given a workflow would normally cost $0.05 on DeepSeek
    And the only available fallback is Anthropic at $2.50
    When the workflow is about to start
    Then the cost estimate shows both: "Normal: $0.05 | Fallback: $2.50"
    And the user must explicitly confirm: "Run at $2.50? [y/N]"
    And if the user declines, the workflow is queued until DeepSeek recovers

  @fallback-cost-aware
  Scenario: Auto-fallback within cost threshold
    Given the user set a cost threshold of $0.50 for auto-fallback
    And deepseek flash is down but deepseek pro ($0.15) is available
    When a subagent spawns
    Then the fallback to Pro happens automatically (under threshold)
    And a notice is emitted but no prompt blocks the workflow

  @fallback-cost-aware
  Scenario: Daily cost cap enforced across fallbacks
    Given the user has a daily cost cap of $1.00
    And the user has already spent $0.80 today
    When a fallback to Anthropic would cost $0.50
    Then the fallback is blocked: "This fallback would exceed your daily cost cap ($0.80 + $0.50 > $1.00)"
    And the user can raise the cap or wait for DeepSeek to recover

  # ── UI ─────────────────────────────────────────────────────────────

  @fallback-ui-status
  Scenario: Provider status shown in status line
    Given DeepSeek Flash is healthy and DeepSeek Pro is degraded
    When the status line renders
    Then Flash shows 🟢, Pro shows 🟡
    And hovering shows: "Flash: healthy (12ms) | Pro: degraded (circuit breaker open, 3/5 failures)"

  @fallback-ui-panel
  Scenario: Provider health panel in settings
    When the user opens settings → Providers
    Then each provider shows: current status, latency, recent error rate, circuit breaker state
    And a "Test connection" button per provider
    And a "Fallback chain" configuration per provider

  @fallback-ui-notification
  Scenario: Critical provider down notification
    Given all DeepSeek providers are down
    And the user has notifications enabled
    When the health check detects the outage
    Then a desktop notification fires: "reasonix: DeepSeek is down. Workflows may be delayed or use fallback."
    And the notification includes the current estimated cost impact

  @fallback-ui-workflow-blocked
  Scenario: Workflow blocked by provider outage shows reason
    Given a workflow is queued because all providers are down
    When the background panel renders the workflow
    Then the workflow shows "⏸ Paused — provider outage"
    And a tooltip explains: "DeepSeek Flash and Pro are both down. Will retry in 60s or when you approve Anthropic fallback ($2.50)."
    And a "Use Anthropic ($2.50)" button is available
