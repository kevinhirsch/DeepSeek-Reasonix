Feature: Observability & Telemetry
  As a developer who wants to improve reasonix
  I want anonymized system-level metrics about subagent performance
  So that improvements are data-driven and users can see their own usage patterns

  Background:
    Given reasonix has opt-in telemetry with zero content collection
    And a local /metrics dashboard is always available

  # ── Local Metrics Dashboard ───────────────────────────────────────

  @telemetry-local
  Scenario: /metrics shows personal usage stats
    When the user types "/metrics"
    Then a dashboard renders with: total tokens, total cost, workflows run, subagents spawned, cache hit rate
    And the time range defaults to "this month"
    And the dashboard is rendered as formatted text with sparklines

  @telemetry-local
  Scenario: /metrics --range <period> shows historical data
    When the user types "/metrics --range week"
    Then the dashboard shows stats for the past 7 days
    And each metric shows the trend: ↑ 12% or ↓ 5% vs previous period

  @telemetry-local
  Scenario: Per-role metrics breakdown
    When the user types "/metrics --by-role"
    Then a table shows per subagent role: spawn count, avg tokens, avg latency, success rate, avg cost per spawn
    And roles are sorted by total cost (highest first)

  @telemetry-local
  Scenario: Per-model cost breakdown
    When the user types "/metrics --by-model"
    Then a table shows per model: requests, tokens, cost, cache hit rate
    And the total cost is shown per model

  # ── Subagent Performance Metrics ───────────────────────────────────

  @telemetry-subagent
  Scenario: Subagent success rate tracked per role
    Given the user has run 100 explorer subagents (98 succeeded, 2 failed)
    When /metrics --by-role renders
    Then explorer shows "98.0% success rate"
    And failed subagents are linked: "2 failures — inspect"

  @telemetry-subagent
  Scenario: Subagent latency percentiles
    Given 1000 explorer subagents with varying durations
    When /metrics --latency renders
    Then p50, p95, and p99 latencies are shown per role
    And the values are in seconds, human-readable

  @telemetry-subagent
  Scenario: Cognitive loop detector firing rate
    Given the loop detector fired on 3 of 100 turns
    When /metrics renders
    Then it shows: "Cognitive loop detector: 3 fires (3.0% of turns) — prevented est. $0.12 waste"

  @telemetry-subagent
  Scenario: Cache hit rate as a time series
    When /metrics --cache renders
    Then a time series shows cache hit rate per day for the past 30 days
    And a trend line shows whether the hit rate is improving or degrading
    And days below 80% are highlighted

  # ── Opt-In Remote Telemetry ────────────────────────────────────────

  @telemetry-opt-in
  Scenario: Telemetry disabled by default
    Given the user has not opted into telemetry
    When reasonix runs
    Then no telemetry data leaves the machine
    And the local /metrics dashboard is still fully functional

  @telemetry-opt-in
  Scenario: Opt-in telemetry sends anonymized counts only
    Given the user has opted into telemetry
    When a workflow completes
    Then the telemetry event includes: workflow strategy, stage count, subagent count, role distribution, total tokens, total cost, cache hit rate
    And the event does NOT include: any prompt text, any tool output, any file paths, any model output

  @telemetry-opt-in
  Scenario: Telemetry privacy guarantee is user-verifiable
    Given the user has opted into telemetry
    When the user wants to verify what is sent
    Then a /telemetry preview command shows the exact JSON payload of the most recent event
    And the user can see every field that would be sent
    And a "Send" / "Don't send" choice is offered

  @telemetry-opt-in
  Scenario: Telemetry opt-out is immediate and permanent
    Given the user has opted into telemetry
    When the user opts out
    Then no further telemetry is sent
    And any queued telemetry is discarded
    And the opt-out is persisted across restarts

  # ── UI ─────────────────────────────────────────────────────────────

  @telemetry-ui
  Scenario: Metrics dashboard renders with progress bars
    When /metrics renders
    Then cache hit rate shows as a percentage bar: [████████░░] 87%
    And the bar color scales: green >80%, yellow 50-80%, red <50%

  @telemetry-ui
  Scenario: Cost sparkline shows daily trend
    When /metrics renders
    Then a 7-day cost sparkline is shown: ▁▂▃▂▅▃▂
    And the current day's cost is labeled: "$0.47 today"

  @telemetry-ui
  Scenario: Metrics export as JSON for external analysis
    When the user types "/metrics --export json"
    Then the full metrics payload is output as JSON
    And the user can pipe it to a file or analysis tool
