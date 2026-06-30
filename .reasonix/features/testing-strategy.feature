Feature: Testing Strategy
  As a developer contributing to reasonix
  I want deterministic, fast tests that cover provider-specific behavior
  So that every change is confident without spending real money on API calls

  Background:
    Given reasonix has a multi-tier test suite
    And mock providers exist for DeepSeek and Anthropic

  # ── Mock Providers ────────────────────────────────────────────────

  @testing-mock-deepseek
  Scenario: Mock DeepSeek simulates normal behavior
    Given a mock DeepSeek provider is configured in "normal" mode
    When a subagent is spawned
    Then the mock returns a valid response with reasoning_content and content
    And the response includes a tool call if the prompt asks for one
    And usage includes reasoning_tokens

  @testing-mock-deepseek
  Scenario: Mock DeepSeek simulates cognitive loop
    Given a mock DeepSeek provider is configured in "looping" mode
    When a subagent runs for 3 turns
    Then each turn returns identical reasoning_content
    And the cognitive loop detector fires on turn 3
    And the breaker message is injected
    And the mock exits loop mode after the breaker

  @testing-mock-deepseek
  Scenario: Mock DeepSeek simulates rate limiting
    Given a mock DeepSeek is in "rate-limited" mode
    When a request is sent
    Then the mock returns 429 with retry-after: 1
    And the retry logic engages
    And the request succeeds on retry

  @testing-mock-deepseek
  Scenario: Mock DeepSeek simulates 5xx outage
    Given a mock DeepSeek is in "outage" mode
    When 5 consecutive requests are sent
    Then all 5 return 500
    And the circuit breaker opens on the 6th request
    And the fallback chain activates

  @testing-mock-anthropic
  Scenario: Mock Anthropic simulates thinking blocks
    Given a mock Anthropic provider with thinking enabled
    When a request is sent
    Then the response includes thinking blocks with signatures
    And the text follows the thinking blocks
    And the thinking blocks are round-tripped on tool calls

  @testing-mock-anthropic
  Scenario: Mock Anthropic simulates cache hits
    Given a mock Anthropic provider
    When two identical requests are sent
    Then the first response shows cache_creation_input_tokens > 0
    And the second response shows cache_read_input_tokens > 0

  # ── Deterministic Replay ──────────────────────────────────────────

  @testing-replay
  Scenario: Recorded workflow replays deterministically
    Given a real workflow was recorded with all tool calls and responses
    When the replay is executed
    Then the workflow produces structurally identical results
    And the event tree is identical to the recorded event tree
    And the test passes without network access

  @testing-replay
  Scenario: Golden file tests catch behavioral changes
    Given a golden file exists for a standard "review auth.go" workflow
    When the workflow tool is changed
    And the replay is executed against the new code
    Then the output is compared to the golden file
    And if the structure differs, the test fails with a diff
    And the developer must update the golden file intentionally

  # ── Chaos Testing ─────────────────────────────────────────────────

  @testing-chaos
  Scenario: Worker kill mid-execution chaos test
    Given a workflow is running with subagents on a remote worker
    When the worker process is killed mid-execution
    Then the work item is marked failed
    And the workflow detects the failure within the timeout
    And the workflow either re-queues the item or marks it failed
    And no state corruption occurs

  @testing-chaos
  Scenario: Network partition chaos test
    Given a workflow with subagents on a remote worker
    When the network between reasonix and the worker is severed for 10 seconds
    Then the worker's health check fails
    And the work item times out
    And when the network recovers, the worker reconnects
    And the workflow can spawn new subagents on the recovered worker

  @testing-chaos
  Scenario: Concurrent crash chaos test
    Given 5 workflows running with 50 subagents total
    When the reasonix process is killed
    And reasonix is restarted
    Then all 5 workflows are detected as interrupted
    And each can be individually resumed
    And no workflow's persisted state corrupts another's

  # ── CI Pipeline ───────────────────────────────────────────────────

  @testing-ci
  Scenario: Short test suite runs on every commit
    Given a commit is pushed
    When CI runs the short suite
    Then it completes in < 2 minutes
    And it includes: unit tests, mock provider tests, contract tests
    And it does NOT include: real API calls, chaos tests, stress tests

  @testing-ci
  Scenario: Full test suite runs on every PR
    Given a PR is opened
    When CI runs the full suite
    Then it includes: short suite + integration tests + regression matrix (all R-IDs)
    And it does NOT include: live API tests, chaos tests

  @testing-ci
  Scenario: Live API test suite runs nightly
    Given it's 3am UTC
    When CI runs the live suite
    Then it includes: real API calls to DeepSeek and Anthropic
    And it verifies: cache behavior, reasoning round-trip, effort mapping, model resolution
    And failures page the on-call if the API is healthy but the tests fail

  @testing-ci
  Scenario: Stress test suite runs weekly
    Given it's Sunday 3am UTC
    When CI runs the stress suite
    Then it includes: 1000 subagent stress test, 100 workflow stress test, goroutine leak check, memory leak check
    And the results are compared to the previous week's baseline
    And regressions >10% are flagged for investigation

  # ── Coverage Requirements ─────────────────────────────────────────

  @testing-coverage
  Scenario: Gherkin scenario coverage enforced
    Given 650+ Gherkin scenarios exist
    When a PR is opened
    Then every Gherkin scenario tagged with a feature that the PR touches must have a corresponding Go test
    And uncovered scenarios are listed in the PR
    And the PR cannot merge with uncovered scenarios in affected features

  @testing-coverage
  Scenario: Provider behavior coverage enforced
    Given provider packages exist for anthropic and openai
    When a PR touches a provider package
    Then both mock provider tests AND at least one live API test must pass
    And the mock must cover the specific behavior changed
