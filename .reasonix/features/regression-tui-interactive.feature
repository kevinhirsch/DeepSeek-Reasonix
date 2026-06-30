Feature: Regression — TUI Interactive Components
  As a developer protecting reasonix's TUI
  I want no additive feature to break chat interaction, pickers, managers, or viewers
  So that TUI users see no difference in interactive behavior before and after new features

  Background:
    Given reasonix TUI is running in a terminal

  # ── Chat Input (internal/cli/chat_tui.go) ─────────────────────────

  @regression-tui-chat-input
  Scenario: Multi-line input with Shift+Enter
    When the user types "line1" then Shift+Enter then "line2" then Enter
    Then the message sent contains "line1\nline2"
    And the composer clears after send

  @regression-tui-chat-input
  Scenario: Paste preserves multi-line content
    When the user pastes "line1\nline2\nline3"
    Then all 3 lines appear in the composer
    And no line is truncated

  @regression-tui-chat-input
  Scenario: Paste handler strips ANSI escape sequences
    When the user pastes text with ANSI color codes
    Then the ANSI codes are stripped
    And only the plain text is inserted

  @regression-tui-chat-input
  Scenario: Chat input shows placeholder text
    When the composer is empty
    Then a placeholder is shown: "Type a message... (Ctrl+N new session, / for commands)"
    And the placeholder disappears when the user starts typing

  @regression-tui-chat-input
  Scenario: Chat input handles CJK IME composition
    When the user composes Chinese characters via IME
    Then the composition state is rendered correctly
    And the final composed text is inserted
    And no artifacts from intermediate composition states remain

  @regression-tui-chat-input
  Scenario: Chat input width adapts to terminal resize
    Given the terminal is 120 columns wide
    When the terminal is resized to 80 columns
    Then the chat input width adapts to 80 columns
    And existing text wraps correctly

  @regression-tui-chat-input
  Scenario: Up arrow recalls previous message
    Given the user sent "previous message" in this session
    When the user presses Up arrow in an empty composer
    Then "previous message" is loaded into the composer
    And the cursor is at the end

  # ── @ File References (internal/cli/chat_attachment_test.go) ──────

  @regression-tui-at-reference
  Scenario: @ opens file picker
    When the user types "@" in the composer
    Then a file picker overlay appears
    And the picker lists files and directories in the current workspace

  @regression-tui-at-reference
  Scenario: @ filters as user types
    When the user types "@auth"
    Then only files matching "auth" are shown
    And the filter is case-insensitive

  @regression-tui-at-reference
  Scenario: @ with no matches shows empty state
    When the user types "@xyznonexistent"
    Then the picker shows "No files match 'xyznonexistent'"

  @regression-tui-at-reference
  Scenario: Selecting a file inserts the reference
    Given the @ picker is open with matching files
    When the user selects a file
    Then the file path is inserted at the cursor position
    And the picker closes

  @regression-tui-at-reference
  Scenario: @ dismissal with Escape
    Given the @ picker is open
    When the user presses Escape
    Then the picker closes
    And the "@" character remains in the composer (user may want to type a literal @)

  # ── Slash Command Picker (internal/cli/select.go) ─────────────────

  @regression-tui-slash
  Scenario: / opens command picker
    When the user types "/" in the composer
    Then a command picker overlay appears
    And built-in commands and skills are listed
    And custom .reasonix/commands are listed

  @regression-tui-slash
  Scenario: / filters by fuzzy match
    When the user types "/expl"
    Then "explore" appears in the results
    And non-matching commands are hidden

  @regression-tui-slash
  Scenario: Selecting a slash command inserts it
    Given the slash picker is open
    When the user selects "explore"
    Then "/explore " is inserted into the composer
    And the cursor is positioned after the space for arguments

  @regression-tui-slash
  Scenario: /dismissal with Escape
    Given the slash picker is open
    When the user presses Escape
    Then the picker closes
    And the "/" character remains

  # ── Model Switcher (internal/cli/model.go) ────────────────────────

  @regression-tui-model
  Scenario: /model lists configured models
    When the user types "/model"
    Then a list of configured models appears
    And each entry shows: provider icon, model name, effort level, context window

  @regression-tui-model
  Scenario: /model switches on selection
    Given the /model list is open
    When the user selects a different model
    Then the active model changes
    And the status line updates immediately
    And a notice confirms: "Switched to deepseek-v4-pro"

  @regression-tui-model
  Scenario: /model shows current model highlighted
    Given the current model is "deepseek-v4-flash"
    When /model opens
    Then "deepseek-v4-flash" is highlighted or marked as current

  # ── Session Resume Picker (internal/cli/resume.go) ────────────────

  @regression-tui-resume
  Scenario: --resume lists previous sessions
    When the user runs "reasonix --resume"
    Then a list of previous sessions appears
    And each entry shows: title, model, date, message count

  @regression-tui-resume
  Scenario: Resuming a session loads full history
    Given a session "Auth work" with 30 messages
    When the user selects and resumes it
    Then all 30 messages are loaded into the conversation
    And the agent context includes the full history

  @regression-tui-resume
  Scenario: Resuming a session restores model and effort
    Given a session was using "deepseek-v4-pro" with effort "max"
    When the user resumes the session
    Then the active model is "deepseek-v4-pro"
    And the effort is "max"

  # ── MCP Manager (internal/cli/mcp_manager.go) ─────────────────────

  @regression-tui-mcp
  Scenario: MCP manager lists configured servers
    When the user opens the MCP manager
    Then all configured MCP servers are listed
    And each entry shows: name, transport type, status (connected/disconnected)

  @regression-tui-mcp
  Scenario: MCP manager shows tool count per server
    Given a connected MCP server exposes 5 tools
    When the manager renders
    Then the server entry shows "5 tools"

  @regression-tui-mcp
  Scenario: Adding an MCP server updates the list
    When the user adds a new MCP server via the manager
    Then the server appears in the list
    And its tools are available to the agent

  @regression-tui-mcp
  Scenario: Toggling an MCP server enables/disables its tools
    Given an MCP server is enabled with 3 tools
    When the user disables it
    Then the 3 tools are removed from the agent's tool registry
    When the user re-enables it
    Then the 3 tools are restored

  # ── Skill Picker (internal/cli/skill_picker.go) ───────────────────

  @regression-tui-skill
  Scenario: Skill picker lists available skills
    When the user invokes the skill picker
    Then all available skills are listed
    And each entry shows: skill name, description, origin (builtin/project/custom)

  @regression-tui-skill
  Scenario: Selecting a skill invokes it
    Given the skill picker is open
    When the user selects "review"
    Then the review skill is invoked on the current changes
    And the skill's output appears in the conversation

  @regression-tui-skill
  Scenario: Skill picker filters by name
    When the user types in the skill picker
    Then skills matching the filter are shown
    And non-matching skills are hidden

  # ── Hook Viewer (internal/cli/hooks_view.go) ──────────────────────

  @regression-tui-hooks
  Scenario: /hooks lists configured hooks
    When the user types "/hooks"
    Then all configured hooks are listed
    And each entry shows: event type, command, scope (project/global)

  @regression-tui-hooks
  Scenario: /hooks shows hook status
    Given a hook is configured but the command is not executable
    When /hooks renders
    Then the hook shows a warning indicator
    And the issue is described

  # ── Memory Viewer (internal/cli/memory_view.go) ───────────────────

  @regression-tui-memory
  Scenario: Memory viewer shows REASONIX.md contents
    When the user opens the memory viewer
    Then the contents of REASONIX.md (or AGENTS.md) are displayed
    And sections are rendered with headers

  @regression-tui-memory
  Scenario: Memory viewer allows editing
    Given the memory viewer is open
    When the user edits a section
    Then the changes are saved to the file
    And the agent's next turn reflects the updated memory

  # ── Review Mode (internal/cli/review.go) ──────────────────────────

  @regression-tui-review
  Scenario: /review invokes code review on current changes
    When the user types "/review"
    Then the code review skill is invoked
    And findings are rendered as a typed list
    And each finding is navigable (j/k)

  @regression-tui-review
  Scenario: Review findings show severity grouping
    Given a review found 2 blocking, 3 should-fix, and 1 nit
    When the findings render
    Then blocking issues are shown first with a red indicator
    And should-fix are shown next with yellow
    And nits are shown last or collapsed

  # ── Rewind (internal/cli/rewind.go) ───────────────────────────────

  @regression-tui-rewind
  Scenario: /rewind lists checkpoints
    When the user types "/rewind"
    Then a list of checkpoints is shown
    And each entry shows: turn number, prompt, timestamp

  @regression-tui-rewind
  Scenario: Selecting a checkpoint rewinds the workspace
    Given a checkpoint from turn 3 is selected
    When the user confirms rewind
    Then files modified after turn 3 are restored
    And the conversation is truncated at turn 3

  # ── Diff View (internal/cli/diffview.go) ──────────────────────────

  @regression-tui-diff-view
  Scenario: Diff view shows file changes before apply
    Given the agent is about to write to "auth.go"
    When the diff preview renders
    Then additions are shown in green
    And deletions are shown in red
    And unchanged context lines are shown in default color

  @regression-tui-diff-view
  Scenario: Diff view is scrollable for large changes
    Given the diff spans 100 lines
    When the diff view renders
    Then only a viewport of lines is shown
    And scrolling reveals the rest

  # ── Help Viewer (internal/cli/help_view.go) ───────────────────────

  @regression-tui-help
  Scenario: /help shows available commands
    When the user types "/help"
    Then a help panel appears
    And it lists all available slash commands and keyboard shortcuts

  # ── Clear Confirmation ────────────────────────────────────────────

  @regression-tui-clear
  Scenario: /clear prompts for confirmation
    When the user types "/clear"
    Then a confirmation prompt appears: "Clear the current conversation?"
    And selecting "Yes" clears the conversation

  @regression-tui-clear
  Scenario: /clear does not delete persisted session
    Given a conversation with messages
    When the user clears the conversation
    Then the in-memory conversation is cleared
    But the persisted session file on disk is not deleted
