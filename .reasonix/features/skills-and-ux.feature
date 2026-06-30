Feature: Agent Skills
  As a developer using reasonix
  I want Claude Code-equivalent skills for review, verification, and workflow automation
  So that common tasks are reliable and consistent

  Background:
    Given reasonix has skill infrastructure supporting inline and subagent execution
    And the quality architecture (Issue #16) is active

  # ── code-review skill ─────────────────────────────────────────────

  @gap-skill-code-review
  Scenario: Code review produces structured findings
    Given the code-review skill is invoked on the current branch diff
    When the skill runs
    Then git diff is run to determine the change scope
    And each changed file is reviewed for correctness, security, and tests
    And findings are reported via report_findings with appropriate severity
    And the skill returns a verdict: ship-as-is, minor-nits, or blocking

  @gap-skill-code-review
  Scenario: Code review at low effort finds only high-confidence bugs
    Given the code-review skill is invoked with effort "low"
    When the review completes
    Then findings are limited to correctness bugs and security issues
    And style nits and simplification suggestions are omitted
    And the report_findings level is "low"

  @gap-skill-code-review
  Scenario: Code review at max effort finds uncertain findings
    Given the code-review skill is invoked with effort "max"
    When the review completes
    Then findings include uncertain/plausible items
    And each uncertain finding is marked with confidence: "low"
    And the report_findings level is "max"

  # ── verify skill ──────────────────────────────────────────────────

  @gap-skill-verify
  Scenario: Verify confirms a change works
    Given the verify skill is invoked on the current changes
    When the skill runs
    Then the project's test command is detected and executed
    And the build command is run if tests require compilation
    And the result is reported: "Verified: all tests pass" or "Failed: N tests"

  @gap-skill-verify
  Scenario: Verify detects the correct test command
    Given the project has go.mod
    When the verify skill detects the test command
    Then it runs "go test ./..."
    And does not attempt npm/pytest/cargo

  @gap-skill-verify
  Scenario: Verify fails gracefully on test failure
    Given tests fail with output
    When the verify skill reports
    Then the failure output is included verbatim
    And the report is not truncated or summarized

  # ── simplify skill ────────────────────────────────────────────────

  @gap-skill-simplify
  Scenario: Simplify applies reuse and simplification fixes
    Given the simplify skill is invoked on changed code
    When the skill runs
    Then duplicate patterns are identified and unified
    And dead code is removed
    And over-complex expressions are simplified
    And each simplification is applied via edit_file or multi_edit
    And the skill does NOT hunt for bugs (that's code-review)

  # ── run skill ─────────────────────────────────────────────────────

  @gap-skill-run
  Scenario: Run launches the project's app
    Given the project has a detectable entry point
    When the run skill is invoked
    Then the entry point is identified from language-specific heuristics
    And the app is launched in the background
    And the skill monitors for startup success (e.g. "Listening on :8080")
    And the skill reports the URL and PID

  @gap-skill-run
  Scenario: Run falls back when entry point unclear
    Given the project has no obvious single entry point
    When the run skill is invoked
    Then the skill asks the user which command to run
    And does not guess

  # ── loop skill ────────────────────────────────────────────────────

  @gap-skill-loop
  Scenario: Loop runs a recurring prompt
    Given the loop skill is invoked with prompt "Check for new issues" and interval "5m"
    When the skill runs
    Then a schedule_task is created with cron "*/5 * * * *"
    And the job fires every 5 minutes
    And the prompt is enqueued at each firing

  @gap-skill-loop
  Scenario: Loop self-paces when no interval given
    Given the loop skill is invoked with no interval
    When the skill runs
    Then a schedule_wakeup is used with a 20-30 minute delay
    And the prompt re-queues itself each cycle
    And the loop stops after 7 days

  # ── update-config skill ───────────────────────────────────────────

  @gap-skill-update-config
  Scenario: Update-config modifies settings.json
    Given the update-config skill is invoked with "allow npm commands"
    When the skill runs
    Then the settings.json file is read
    And the appropriate permission rule is added
    And the file is written back
    And the change is validated

  @gap-skill-update-config
  Scenario: Update-config refuses invalid settings
    Given the update-config skill is invoked with an unrecognized request
    When the skill runs
    Then the skill reports: "I don't know how to configure that. Valid settings: ... "

  @gap-skill-update-config-ui
  Scenario: Settings edit shows diff preview before applying
    Given the update-config skill has prepared a change to settings.json
    When the change is ready to apply
    Then a diff preview is rendered showing old and new values
    And an "Apply" button is shown
    And a "Cancel" button is shown
    And the diff highlights the exact lines being added or removed

  @gap-skill-update-config-ui
  Scenario: Settings validation failure renders error with context
    Given the update-config skill attempted a change that would break settings.json
    When the validation fails
    Then an error card renders with: what was attempted, why it failed, and a suggested fix
    And the error card has a "Try suggested fix" button
    And the error card has a "Cancel" button

  # ── review (PR) skill ─────────────────────────────────────────────

  @gap-skill-review-pr
  Scenario: Review fetches and examines a GitHub PR
    Given the review skill is invoked for PR #42 on the current repo
    When the skill runs
    Then the PR metadata is fetched via web_fetch (GitHub API)
    And the PR diff is fetched
    And each changed file is reviewed
    And findings are reported via report_findings

  @gap-skill-review-pr-ui
  Scenario: PR review renders PR metadata header
    Given a PR review is running for PR #42 "Fix auth middleware"
    When the review card renders
    Then the card header shows: "Reviewing PR #42: Fix auth middleware"
    And the PR author is displayed
    And the branch name is displayed
    And the number of changed files is shown

  @gap-skill-review-pr-ui
  Scenario: PR review renders per-file findings
    Given a PR review found 3 issues across 2 files
    When the findings render
    Then findings are grouped by file with file headers
    And each finding shows line number, severity tag, and one-line summary
    And clicking a finding opens the file at that line in the diff view

  @gap-skill-review-pr-ui
  Scenario: PR review shows progress during execution
    Given a PR review is fetching PR data, then diff, then reviewing files
    When the review runs
    Then a progress indicator shows current phase: "Fetching PR #42...", "Analyzing diff...", "Reviewing files (2/5)..."
    And the phase transitions smoothly as each step completes

  # ── fewer-permission-prompts skill ────────────────────────────────

  @gap-skill-permissions
  Scenario: Scan transcripts for read-only calls
    Given the fewer-permission-prompts skill is invoked
    When the skill runs
    Then recent session transcripts are scanned
    And read-only tool calls (read_file, glob, grep, ls) are identified
    And an allowlist entry is proposed
    And the user is asked to confirm before writing

  @gap-skill-permissions-ui
  Scenario: Allowlist proposal renders as a reviewable list
    Given the scan found 8 read-only tool patterns across 3 sessions
    When the proposal renders
    Then each proposed allowlist entry is listed with: tool name, pattern, session count, estimated prompt reduction
    And each entry has an "Approve" and "Skip" button
    And a "Select All" checkbox is at the top
    And approved entries show ✓ after confirmation

  @gap-skill-permissions-ui
  Scenario: Confirm dialog shows exact settings.json changes
    Given the user approved 5 of 8 proposed entries
    When the confirm dialog opens
    Then it shows a diff of settings.json with the new allowlist entries
    And the diff shows exactly which lines will be added
    And a "Write" button applies the changes
    And a "Cancel" button discards them

  # ── keybindings-help skill ────────────────────────────────────────

  @gap-skill-keybindings
  Scenario: Customize keyboard shortcut
    Given the keybindings-help skill is invoked for "rebind ctrl+s to submit"
    When the skill runs
    Then the keybindings.json file is read (or created)
    And the binding is added
    And a conflict check is performed against existing bindings

  @gap-skill-keybindings-ui
  Scenario: Keybinding conflict renders conflict resolution dialog
    Given the user wants to bind Ctrl+S to "submit"
    And Ctrl+S is already bound to "save session"
    When the conflict check runs
    Then a conflict dialog renders showing: the new binding, the existing binding, and the conflict
    And three options are shown: "Replace existing", "Keep existing (cancel)", "Bind to both (may cause issues)"
    And the user's choice is applied

  @gap-skill-keybindings-ui
  Scenario: Keybinding applied shows confirmation toast
    Given the user successfully bound Ctrl+Shift+R to "review"
    When the binding is written
    Then a toast appears: "✓ Bound Ctrl+Shift+R to /review"
    And the toast auto-dismisses after 3 seconds
    And the binding is immediately active (no restart)

  # ── AskUserQuestion Enhancement ────────────────────────────────────

  @gap-ask-multiselect
  Scenario: Ask with multiSelect allows multiple answers
    When the agent calls ask with multiSelect: true and options ["npm", "yarn", "pnpm"]
    Then the UI renders checkboxes, not radio buttons
    And the user can select multiple answers
    And the result includes all selected values

  @gap-ask-preview
  Scenario: Ask with preview renders side-by-side comparison
    When the agent calls ask with options that have preview content (code snippets)
    Then the UI renders a side-by-side layout
    And the preview content is shown in a monospace box
    And the user can scroll the preview independently

  @gap-ask-header
  Scenario: Ask renders header chip above options
    When the agent calls ask with header "Auth method"
    Then the UI shows a chip/badge with "Auth method" above the options
    And the chip is visually distinct from the question text

  @gap-ask-other
  Scenario: Ask always includes "Other" option
    When the agent calls ask with 3 options
    Then the UI renders 4 options: the 3 specified + "Other (custom input)"
    And selecting "Other" opens a free-text input
