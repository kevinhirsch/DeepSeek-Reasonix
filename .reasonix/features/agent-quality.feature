Feature: Agent Quality Behavior
  As a developer using reasonix
  I want the agent to produce thoughtful, careful, minimal code
  So that output quality matches Claude Code without changing models

  Background:
    Given reasonix has the quality system prompt block injected
    And the behavioral gates are wired into the agent loop

  # ── Anti-Overengineering ─────────────────────────────────────────

  @16-anti-overengineering
  Scenario: Agent refuses to refactor surrounding code during a bug fix
    Given the task is "Fix nil pointer dereference in auth.go:42"
    And the agent has identified the bug and a fix
    When the agent produces its changes
    Then only the file with the bug is modified
    And no helper functions are extracted
    And no error handling is added to unrelated functions
    And no imports are reorganized
    And no variable names are renamed outside the changed lines

  @16-anti-overengineering
  Scenario: Agent does not add error handling for impossible scenarios
    Given the task is "Add a new endpoint for user lookup"
    And the agent reads the existing handler pattern
    And existing handlers do not validate that `r *http.Request` is non-nil
    When the agent implements the new endpoint
    Then the agent follows the existing pattern
    And the agent does not add nil checks for framework-guaranteed values
    And the agent does not add validation for "what if the database disappears mid-transaction"

  @16-anti-overengineering
  Scenario: Agent does not design for hypothetical future requirements
    Given the task is "Add a --verbose flag to the CLI"
    And the spec says nothing about log levels
    When the agent implements the flag
    Then the agent adds a boolean flag, not a log-level enum
    And the agent does not add a config file reader "for when we add more flags"
    And the agent does not create a FlagManager abstraction

  @16-anti-overengineering
  Scenario: Agent matches surrounding code style
    Given the project uses `fmt.Errorf("context: %w", err)` for error wrapping
    And the task is to add error handling to a new function
    When the agent adds error returns
    Then every error uses `fmt.Errorf` with context and `%w`
    And no error uses `errors.New` or bare string returns
    And comment style matches surrounding functions (density, format, language)

  # ── Evidence-Grounded Claims ──────────────────────────────────────

  @16-evidence
  Scenario: Agent verifies claims against tool output
    Given the agent claims "all 42 tests pass"
    When the final-answer readiness check runs
    Then the claim is verified against the bash tool output from this turn
    And if no test output exists, the claim is flagged as unverified
    And the agent receives: "You claimed 'all 42 tests pass' but no test output was captured this turn"

  @16-evidence
  Scenario: Agent cannot claim a file was read if it wasn't
    Given the agent claims "handler.go uses context-based cancellation"
    And handler.go was not read this turn
    When the evidence ledger is checked
    Then the claim is flagged: "Claim about handler.go — file not read this turn"

  @16-evidence
  Scenario: Agent reports test failures faithfully
    Given the agent ran tests and 3 failed
    When the agent reports the outcome
    Then the report includes the exact failure output
    And the report does not say "some tests need attention"
    And the report names each failing test with its error

  @16-evidence
  Scenario: Agent reports skipped steps
    Given the task was "Fix auth.go, add tests, update docs"
    And the agent fixed auth.go and added tests but did not update docs
    When the agent reports completion
    Then the report explicitly states: "Skipped: updating docs"
    And the report gives a reason for skipping

  # ── Silence Default ───────────────────────────────────────────────

  @16-silence
  Scenario: Agent writes nothing between routine tool calls
    Given the agent is reading several files in sequence
    When the agent calls read_file on auth.go
    And then read_file on middleware.go
    And then read_file on handlers.go
    Then no text is emitted between the read_file calls
    And no "Now I'll read middleware.go" narration appears

  @16-silence
  Scenario: Agent writes one sentence when changing direction
    Given the agent was planning to use an ORM query
    And grep revealed the ORM doesn't support the needed join
    When the agent changes approach to raw SQL
    Then the agent emits exactly one sentence: "ORM doesn't support this join — switching to raw SQL"
    And no additional explanation is emitted

  @16-silence
  Scenario: Agent writes one sentence when hitting a blocker
    Given the agent attempted to run tests and got "port 8080 already in use"
    When the agent cannot proceed
    Then the agent emits exactly one sentence: "Port 8080 is in use — cannot run tests. Free the port or specify --port."
    And the agent does not emit a paragraph about potential causes

  # ── Promise Detection ─────────────────────────────────────────────

  @16-promise
  Scenario: Agent blocked from ending on a promise
    Given the agent's final answer ends with "I'll add the integration tests next"
    When the promise detector runs
    Then the output is blocked
    And the agent receives: "Your response ends with what reads like a promise about future work: 'I'll add the integration tests next'. Execute that work now with tool calls instead."

  @16-promise
  Scenario: Agent blocked from ending on "Want me to"
    Given the agent's final answer ends with "Want me to also update the README?"
    When the promise detector runs
    Then the output is blocked
    And the agent receives a directive to either do it or state completion

  @16-promise
  Scenario: Agent blocked from ending on a to-do list
    Given the agent's final answer is a bullet list of "Next steps: 1. Add tests 2. Update docs"
    When the promise detector runs
    Then the output is blocked
    And the agent receives: "You listed next steps instead of completing the work. Execute each step now."

  # ── Completeness Check ────────────────────────────────────────────

  @16-completeness
  Scenario: Agent flagged for missing a requested action
    Given the user asked: "Fix the nil pointer in auth.go and add a regression test"
    And the agent fixed the nil pointer but did not add a test
    When the completeness check runs
    Then the agent is flagged: "Requested 'add a regression test' — not found in your response"
    And the agent receives the flag before the turn is accepted

  @16-completeness
  Scenario: Agent passes completeness when all actions done
    Given the user asked: "Rename getCwd to getCurrentWorkingDirectory across the project"
    And the agent ran grep, found 8 call sites, edited all 8, ran tests
    When the completeness check runs
    Then no flags are raised
    And the turn is accepted

  @16-completeness
  Scenario: Agent handles implicit completeness
    Given the user asked: "Why is the login endpoint slow?"
    And the agent profiled, identified the N+1 query, and fixed it
    When the completeness check runs
    Then the check recognizes "why" questions don't require action enumeration
    And the turn is accepted

  # ── Communication Style ────────────────────────────────────────────

  @16-communication
  Scenario: Agent leads with outcome
    Given the agent completed a multi-step refactor
    When the agent reports the result
    Then the first sentence is: "Renamed getCwd to getCurrentWorkingDirectory — 8 call sites updated, tests pass"
    And supporting detail follows after the first sentence

  @16-communication
  Scenario: Agent does not pack identifiers into parenthetical runs
    Given the agent modified auth.go, middleware.go, and handlers.go
    When the agent reports the result
    Then the report does NOT contain: "(auth.go, middleware.go, handlers.go)"
    And each file gets its own clause explaining what changed

  @16-communication
  Scenario: Agent drops details that don't change reader behavior
    Given the agent ran a build that succeeded
    When the agent reports completion
    Then the report says "Build succeeded" not "Ran `go build ./...` which invoked the compiler for 23 packages using 142MB of RAM and completed in 3.2 seconds"

  # ── DeepSeek-Specific Quality ──────────────────────────────────────

  @16-deepseek-quality
  Scenario: DeepSeek does not add helper functions to single-file changes
    Given the provider is DeepSeek
    And the task is a bug fix in one function in one file
    When the agent produces a change
    Then only the file with the bug is modified
    And no new helper functions are extracted from the fixed function

  @16-deepseek-quality
  Scenario: DeepSeek does not re-verify successful tool calls
    Given the provider is DeepSeek
    And the agent ran `go test ./auth/...` which returned "ok"
    When the agent continues working
    Then the agent does not run `go test ./auth/...` again
    And the agent trusts the first result

  @16-deepseek-quality
  Scenario: DeepSeek suppresses between-tool narration
    Given the provider is DeepSeek
    And the agent is executing 5 sequential read_file calls
    When the calls complete
    Then zero narrative messages are emitted between the reads
    And the session history contains tool_calls and tool_results, not interleaved text

  @16-deepseek-quality
  Scenario: Non-DeepSeek providers unaffected by DeepSeek tuning
    Given the provider is Anthropic Claude
    And the DeepSeek-specific quality block is present in the system prompt
    When the agent executes
    Then the DeepSeek-specific rules are not applied (or are harmless to Anthropic behavior)

  # ── Quality Notices & FE Rendering ─────────────────────────────────

  @16-fe-promise-notice
  Scenario: Promise block notice renders in transcript with action button
    Given the promise detector blocked a final answer ending in "I'll add tests next"
    When the notice renders in the transcript
    Then a warning card is shown: "⚠ Response blocked — unfulfilled promise detected"
    And the card shows the extracted promise text
    And a "Continue" button lets the user override and accept the answer anyway
    And a "Let agent execute" button sends the promise back to the agent

  @16-fe-evidence-notice
  Scenario: Unverified claim notice renders with file reference
    Given the claim verifier flagged "Claim about handler.go — file not read this turn"
    When the notice renders in the transcript
    Then a warning card is shown: "⚠ Unverified claim: handler.go was not read this turn"
    And the card shows the claim text that referenced the unread file
    And the file name is clickable (opens in code viewer)

  @16-fe-silence-notice
  Scenario: Silence violation notice renders subtly
    Given the anti-narration check detected "I'll now read the configuration" between tool calls
    When the notice fires
    Then it renders as a dimmed, non-intrusive notice (not a warning)
    And the notice text is: "Narration suppressed — 'I'll now read the configuration...'"
    And the notice does not interrupt the transcript flow

  @16-fe-completeness-notice
  Scenario: Completeness check failure renders with missing actions list
    Given the completeness check detected "add a regression test" was requested but not done
    When the notice renders
    Then a card is shown: "⚠ Incomplete: 'add a regression test' not found in your response"
    And a "Continue anyway" button is shown
    And a "Let agent complete" button sends the missing action back to the agent

  @16-fe-overengineering-notice
  Scenario: Overengineering flag renders with file list
    Given the anti-overengineering gate detected modifications to 3 files beyond the 1 requested
    When the notice renders
    Then a card is shown: "⚠ Scope expanded: 3 files changed beyond auth.go"
    And the 3 extra files are listed: middleware.go, handlers.go, types.go
    And each file is clickable to view the changes
    And a "Justify" button lets the user accept the expanded scope
    And a "Revert extras" button reverts the unrequested changes

  @16-fe-deepseek-quality-notice
  Scenario: DeepSeek quality notices are visually tagged
    Given a DeepSeek-specific quality notice fires
    When the notice renders
    Then it shows a small "[DeepSeek]" tag
    And the tag links to documentation about DeepSeek-specific behavior
