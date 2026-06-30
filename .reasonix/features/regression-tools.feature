Feature: Regression — Built-in Tools
  As a developer protecting reasonix's existing tool behavior
  I want no additive feature to break any of the 18 built-in tools
  So that every tool works identically before and after new features

  Background:
    Given reasonix has all 18 built-in tools registered

  # ── bash ──────────────────────────────────────────────────────────

  @regression-tool-bash
  Scenario: bash executes command and returns output
    When the bash tool executes "echo hello world"
    Then the output contains "hello world"
    And the exit code is 0

  @regression-tool-bash
  Scenario: bash captures stderr
    When the bash tool executes "echo error >&2"
    Then the output contains "error"
    And stderr is merged into the output

  @regression-tool-bash
  Scenario: bash returns non-zero exit code
    When the bash tool executes "exit 1"
    Then the output contains the exit code
    And the tool reports the error

  @regression-tool-bash
  Scenario: bash run_in_background returns job ID
    When the bash tool executes "sleep 30" with run_in_background: true
    Then a job ID is returned
    And the command continues running across turns

  @regression-tool-bash
  Scenario: bash_output reads streaming output
    Given a background bash job is running "for i in 1 2 3; do echo $i; sleep 0.1; done"
    When bash_output is called with the job ID
    Then the output contains the lines emitted so far

  @regression-tool-bash
  Scenario: kill_shell terminates background job
    Given a background bash job is running "sleep 300"
    When kill_shell is called with the job ID
    Then the job status changes to killed
    And the job is removed from the active job list

  @regression-tool-bash
  Scenario: wait blocks until background job completes
    Given a background bash job is running "sleep 0.5 && echo done"
    When wait is called with the job ID
    Then wait blocks until the job completes
    And the result contains "done"

  # ── File Tools ────────────────────────────────────────────────────

  @regression-tool-read-file
  Scenario: read_file reads whole file
    Given a file "test.txt" exists with content "line1\nline2\nline3"
    When read_file is called on "test.txt"
    Then the output contains all 3 lines
    And the file path is in the output

  @regression-tool-read-file
  Scenario: read_file with line range returns subset
    Given a file "test.txt" with 20 lines
    When read_file is called with line range 5-10
    Then the output contains lines 5 through 10
    And lines outside 5-10 are NOT in the output

  @regression-tool-write-file
  Scenario: write_file creates file
    When write_file is called with path "new.txt" and content "hello"
    Then the file "new.txt" exists on disk
    And the file contains "hello"

  @regression-tool-write-file
  Scenario: write_file overwrites existing file
    Given a file "existing.txt" exists with content "old"
    When write_file is called with path "existing.txt" and content "new"
    Then the file contains "new"
    And the old content is gone

  @regression-tool-edit-file
  Scenario: edit_file replaces exact string match
    Given a file with content "func old() { return 1 }"
    When edit_file is called with old_string "old()" and new_string "new()"
    Then the file contains "func new() { return 1 }"
    And no other changes were made

  @regression-tool-edit-file
  Scenario: edit_file fails on non-unique match
    Given a file with content "x = 1\nx = 1"
    When edit_file is called with old_string "x = 1"
    Then the tool returns an error: "old_string is not unique"

  @regression-tool-multi-edit
  Scenario: multi_edit applies multiple edits atomically
    Given a file with 3 distinct strings to change
    When multi_edit is called with 3 old_string/new_string pairs
    Then all 3 changes are applied
    And the file is written once

  @regression-tool-delete-range
  Scenario: delete_range removes lines
    Given a file with 10 lines
    When delete_range is called with range 3-5
    Then lines 3, 4, 5 are removed
    And lines 1, 2, 6, 7, 8, 9, 10 remain

  @regression-tool-move-file
  Scenario: move_file renames file
    Given a file "old_name.txt" exists
    When move_file is called from "old_name.txt" to "new_name.txt"
    Then "old_name.txt" no longer exists
    And "new_name.txt" exists with the original content

  # ── Search Tools ──────────────────────────────────────────────────

  @regression-tool-grep
  Scenario: grep finds pattern matches
    Given a directory with Go files containing "TODO"
    When grep is called with pattern "TODO"
    Then matching files and lines are returned
    And each result includes file path and line number

  @regression-tool-grep
  Scenario: grep with context lines returns surrounding lines
    When grep is called with pattern "func" and context_lines: 2
    Then each match shows 2 lines before and 2 lines after

  @regression-tool-glob
  Scenario: glob matches file patterns
    Given a directory with .go and .md files
    When glob is called with pattern "*.go"
    Then only .go files are returned
    And .md files are not in the results

  @regression-tool-ls
  Scenario: ls lists directory contents
    When ls is called on the project root
    Then files and directories are listed
    And each entry has a name and type

  # ── web_fetch ─────────────────────────────────────────────────────

  @regression-tool-web-fetch
  Scenario: web_fetch returns page content
    When web_fetch is called on a valid URL
    Then the response contains the page text
    And the content is rendered as markdown or plain text

  @regression-tool-web-fetch
  Scenario: web_fetch blocks SSRF to internal addresses
    When web_fetch is called on "http://127.0.0.1:8080"
    Then the request is blocked
    And the error message indicates the address is blocked

  @regression-tool-web-fetch
  Scenario: web_fetch blocks SSRF to special-use IPv4
    When web_fetch is called on "http://100.100.100.200"
    Then the request is blocked
    And the error message references the blocked IP

  # ── todo_write + complete_step ────────────────────────────────────

  @regression-tool-todo
  Scenario: todo_write creates task list
    When todo_write is called with items: [{id:"1", content:"Fix auth bug", status:"pending"}, {id:"2", content:"Add tests", status:"pending"}]
    Then both items are stored in the agent's todoState
    And the task list is emitted as an event

  @regression-tool-todo
  Scenario: complete_step verifies completion against evidence
    Given todo item "1" is pending
    And the agent has run tests showing they pass
    When complete_step is called for item "1"
    Then item "1" is marked completed
    And the agent must cite evidence from this turn

  @regression-tool-todo
  Scenario: complete_step rejects unsubstantiated completion
    Given todo item "1" is pending
    And the agent has NOT run tests
    When complete_step is called for item "1" claiming "tests pass"
    Then the completion is rejected
    And the readiness check flags the missing evidence
