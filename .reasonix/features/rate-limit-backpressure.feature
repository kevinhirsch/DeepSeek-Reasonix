Feature: Rate Limit Backpressure
  As a developer running parallel workflows
  I want the system to respect provider rate limits and avoid thundering herds
  So that workflows complete efficiently without wasting tokens on 429 retries

  Background:
    Given reasonix has provider rate limit awareness
    And DeepSeek Flash has a rate limit of 10 requests/second

  # ── Rate Limit Awareness ──────────────────────────────────────────

  @backpressure-limit-aware
  Scenario: Provider tracks remaining quota from response headers
    Given a provider response includes rate limit headers
    When the response is processed
    Then the remaining quota is extracted and stored
    And the quota is updated after every request
    And the quota is shared across all subagents using that provider

  @backpressure-limit-aware
  Scenario: Quota exhausted blocks new requests
    Given the DeepSeek Flash remaining quota is 0
    When a new request would be sent
    Then the request is queued, not sent
    And the queue position is displayed
    And the request waits for the quota window to reset

  @backpressure-limit-aware
  Scenario: Quota resets at window boundary
    Given the rate limit window is 60 seconds
    And the quota was exhausted at second 45
    When the window resets at second 60
    Then the quota is refreshed
    And queued requests are released in order

  # ── Workflow Spawn Limiter ────────────────────────────────────────

  @backpressure-workflow-batch
  Scenario: Workflow batches subagent spawns to rate limit
    Given a workflow stage would spawn 27 verifier subagents
    And the provider rate limit is 10 requests/second
    When the stage begins
    Then at most 10 subagents are spawned in the first second
    And the remaining 17 are queued
    And they release at 10/second until all are spawned

  @backpressure-workflow-batch
  Scenario: Batching respects provider-specific limits
    Given a workflow uses Flash for finders (10 req/s) and Pro for verifiers (5 req/s)
    When both stages would spawn simultaneously (pipeline mode)
    Then finders spawn at up to 10/s
    And verifiers spawn at up to 5/s
    And the two limits are independent

  @backpressure-workflow-batch
  Scenario: Spawn limiter shows queue state in UI
    Given 27 verifiers are queued, 10 have spawned
    When the background panel renders
    Then the workflow stage shows: "verify: 10/27 spawned, 17 queued (releasing at 5/s)"
    And the count updates live as subagents spawn

  # ── Retry with Jitter ─────────────────────────────────────────────

  @backpressure-jitter
  Scenario: Retry backoff includes random jitter
    Given a request got a 429 with retry-after: 1
    When the retry fires
    Then the actual delay is 1s + random(0, 500ms)
    And 10 simultaneous 429s would NOT all retry at the exact same millisecond

  @backpressure-jitter
  Scenario: Exponential backoff with jitter on repeated 429s
    Given a request has been retried 3 times
    When the 4th retry delay is calculated
    Then the delay is base * 2^3 + jitter
    And the delay increases with each attempt

  # ── Static Rate Limit Configuration ───────────────────────────────

  @backpressure-config
  Scenario: Provider rate limit configurable when headers unavailable
    Given the provider does not return rate limit headers
    When the user configures rate_limit_rps in the provider entry
    Then the spawn limiter uses the configured value
    And the limiter is active even without header support

  @backpressure-config
  Scenario: Default rate limits for known providers
    Given a DeepSeek provider with no explicit rate_limit_rps
    When the provider is constructed
    Then a built-in default rate limit is applied
    And the default is conservative (below the actual limit)

  # ── UI ─────────────────────────────────────────────────────────────

  @backpressure-ui
  Scenario: Status line shows rate limit status
    Given the DeepSeek Flash quota is 3/10 remaining with 45s until reset
    When the status line renders
    Then it shows: "Flash: 3/10 req/s (reset in 45s)"
    And the indicator is yellow when quota < 30%

  @backpressure-ui
  Scenario: Rate limit warning when approaching exhaustion
    Given DeepSeek Flash has 1 request remaining
    When a subagent is about to spawn
    Then a warning is emitted: "Rate limit nearly exhausted (1/10). Subagent queued."
    And the subagent waits rather than failing with 429
