Feature: Subagent Debugging & Deep Trace
  As a developer who needs to understand subagent decisions
  I want to inspect full subagent transcripts, reasoning, and tool calls
  So that I can debug false refutations and understand why a subagent made a mistake

  Background:
    Given reasonix has subagent transcript persistence
    And --debug mode enables full trace inspection

  # ── Subagent Transcript Inspection ────────────────────────────────

  @debug-inspect-full
  Scenario: Inspect subagent shows full message log
    Given a verifier subagent "verify-sql-injection-1" completed with verdict REFUTED
    When the user opens the subagent inspect view
    Then the full message log is displayed: system prompt, user message, every assistant turn with tool calls, every tool result
    And messages are chronologically ordered
    And tool calls are expandable to show full input/output

  @debug-inspect-full
  Scenario: Inspect shows reasoning blocks
    Given the verifier subagent produced 3 reasoning blocks during its run
    When the user inspects the subagent
    Then each reasoning block is displayed with a toggle: collapsed by default, expandable
    And the reasoning blocks are timestamped relative to the turn

  @debug-inspect-full
  Scenario: Inspect not available for ephemeral subagents
    Given a subagent ran without transcript persistence
    When the user tries to inspect it
    Then a message is shown: "Transcript not available — this subagent ran ephemerally. Re-run with transcript persistence enabled."
    And a "Re-run with persistence" button is shown

  # ── Finding-to-Evidence Trace ──────────────────────────────────────

  @debug-finding-trace
  Scenario: Clicking a finding shows the verifier's evidence chain
    Given a verified finding "nil pointer in auth.go:42" was confirmed by 2 of 3 verifiers
    When the user clicks the finding
    Then the inspect view opens showing the 3 verifier transcripts side-by-side
    And the 2 CONFIRMED verifiers' evidence is highlighted
    And the 1 REFUTED verifier's counter-evidence is highlighted in red
    And the user can see exactly what each verifier read and concluded

  @debug-finding-trace
  Scenario: Finding trace shows file reads per verifier
    Given verifier A read auth.go:40-50 and verifier B read auth.go:35-55
    When the finding trace renders
    Then a diff view shows which lines each verifier saw
    And overlapping coverage and unique coverage are visually distinct
    And the user can see that verifier A missed the nil check at line 44

  # ── Transcript Search ─────────────────────────────────────────────

  @debug-transcript-search
  Scenario: Search across all subagent transcripts
    Given a workflow spawned 15 subagents
    When the user searches for "SQL injection" across all transcripts
    Then matching subagents are listed with hit count per subagent
    And clicking a subagent opens its transcript at the first match

  @debug-transcript-search
  Scenario: Transcript search with regex
    When the user searches with pattern "nil.*pointer|panic.*nil"
    Then all matching lines across all subagent transcripts are returned
    And results are grouped by subagent

  # ── Subagent Diff ─────────────────────────────────────────────────

  @debug-subagent-diff
  Scenario: Diff two verifier transcripts
    Given verifier A confirmed and verifier B refuted the same finding
    When the user diffs their transcripts
    Then the diff shows: which tool calls differed, which evidence each saw, which reasoning paths diverged
    And the exact point of divergence is highlighted

  # ── UI ─────────────────────────────────────────────────────────────

  @debug-ui-peek
  Scenario: Peek panel shows full tool history
    Given the user opens the peek panel on a completed subagent
    When the full view renders
    Then every tool call is listed chronologically with: tool name, duration, success/failure
    And clicking a tool call expands it to show full input and output
    And a "Jump to transcript" button opens the full inspect view

  @debug-ui-peek
  Scenario: Peek panel shows "Why?" button on flagged findings
    Given a finding was flagged as REFUTED
    When the finding renders in the review output
    Then a "Why?" button is shown next to the finding
    And clicking it opens the verifier trace showing the evidence chain
