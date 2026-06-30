Feature: Session Portability
  As a developer who produces valuable output with reasonix
  I want to export, share, and archive sessions and workflow results
  So that findings are useful outside the tool and sessions can move between machines

  Background:
    Given reasonix has session persistence and structured output (report_findings)

  # ── Session Export ────────────────────────────────────────────────

  @portability-export-markdown
  Scenario: /export generates Markdown report of current session
    When the user types "/export"
    Then a Markdown file is generated at .reasonix/exports/<session>-<date>.md
    And the report includes: session title, date, model used, token usage, cost
    And all user messages and assistant responses are included
    And tool calls are collapsed by default with an expand toggle
    And report_findings output is rendered as a formatted table

  @portability-export-markdown
  Scenario: /export --format json produces machine-readable output
    When the user types "/export --format json"
    Then a JSON file is generated with the full session transcript
    And the JSON includes structured metadata: session ID, timestamps, model, effort, token counts
    And the format is documented and stable across versions

  @portability-export-markdown
  Scenario: /export --findings-only exports just the structured findings
    When the user types "/export --findings-only"
    Then only report_findings output is exported
    And findings are rendered as a table: Severity | File | Line | Summary | Verdict
    And the format is PR-ready (can be pasted directly into a GitHub PR comment)

  @portability-export-markdown
  Scenario: /export respects user privacy settings
    Given the session contains file paths under /home/user/secret-project
    When the user types "/export --redact-paths"
    Then file paths are replaced with relative paths from the workspace root
    And no absolute paths containing the user's home directory are included

  # ── Workflow Export ───────────────────────────────────────────────

  @portability-workflow-export
  Scenario: Workflow results export as structured report
    Given a security review workflow completed with 5 findings
    When the user exports the workflow
    Then the report includes: workflow summary (strategy, stages, items), finding-by-finding detail, voting breakdown per finding
    And the exporter notes: "5 findings total: 3 CONFIRMED, 1 PLAUSIBLE, 1 REFUTED"

  @portability-workflow-export
  Scenario: Workflow export includes cost breakdown
    Given a workflow cost $0.047 across 33 subagents
    When the user exports the workflow
    Then the report includes: total cost, cost per stage, cost per subagent role, cache savings

  # ── Session Sharing ───────────────────────────────────────────────

  @portability-sharing
  Scenario: /share generates shareable session package
    When the user types "/share"
    Then a .reasonix/share/<session-id>.reasonix file is created
    And the file is a gzipped tarball containing: session transcript, config snapshot (no API keys), metadata
    And the share file is self-contained (can be imported on another machine)
    And the share file size is reported

  @portability-sharing
  Scenario: Import shared session
    Given a share file from another developer
    When the user runs "reasonix import session.reasonix"
    Then the session is loaded into the session list
    And the session opens in the current tab
    And a notice shows: "Imported session from <original-machine> — <original-date>. Model: <model>. Tokens: <count>."

  @portability-sharing
  Scenario: Shared session is read-only by default
    Given an imported session
    When the user tries to continue the conversation
    Then the session is read-only unless the user explicitly chooses "Continue"
    And "Continue" forks the session into a new writeable session

  # ── Archive ────────────────────────────────────────────────────────

  @portability-archive
  Scenario: Sessions auto-archived after configurable TTL
    Given session TTL is 90 days
    And a session was last active 91 days ago
    When the controller performs periodic maintenance
    Then the session is moved to .reasonix/archive/
    And the session remains importable but is removed from the active session list

  @portability-archive
  Scenario: Archived session can be restored
    Given a session was archived
    When the user selects it from the archive view and clicks "Restore"
    Then the session returns to the active session list
    And the conversation is intact

  # ── UI ─────────────────────────────────────────────────────────────

  @portability-ui
  Scenario: Export progress shown for large sessions
    Given a session with 500 messages
    When the user exports
    Then a progress bar shows: "Exporting... 350/500 messages"
    And the export completes without blocking the UI

  @portability-ui
  Scenario: Export location opened in file manager
    Given an export completed
    When the export completes
    Then a notification shows: "Exported to .reasonix/exports/auth-review-2026-06-30.md"
    And a "Open folder" button opens the file manager at the export location

  @portability-ui
  Scenario: Shared session metadata visible before import
    Given a share file exists
    When the user selects it for import
    Then a preview shows: original machine, date, session title, model, message count, token usage
    And an "Import" button is shown
    And a "Cancel" button dismisses
