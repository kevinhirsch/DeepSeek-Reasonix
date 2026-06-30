Feature: Regression — Desktop Application
  As a developer protecting reasonix's desktop app
  I want no additive feature to break the desktop UI, session management, settings, or window behavior
  So that desktop users see no difference before and after new features

  Background:
    Given the reasonix desktop app is running (Go backend + React frontend)

  # ── Session Management (desktop/sessions.go) ──────────────────────

  @regression-desktop-session-create
  Scenario: New session creates a tab with default settings
    When the user creates a new session
    Then a tab appears with the session title
    And the tab shows the active model name
    And the chat input is focused
    And the session is persisted to disk

  @regression-desktop-session-create
  Scenario: New session inherits workspace from project root
    Given the desktop app was opened in /home/user/projects/myapp
    When a new session is created
    Then the session's workspace root is /home/user/projects/myapp
    And the session loads reasonix.toml from that directory

  @regression-desktop-session-list
  Scenario: Session list shows all tabs ordered by last activity
    Given 3 sessions exist: A (active 1 min ago), B (active 5 min ago), C (active now)
    When the session list renders
    Then sessions are ordered: C, A, B
    And the active session is highlighted

  @regression-desktop-session-rename
  Scenario: Renaming a session updates the tab and persisted data
    When the user renames session "Untitled" to "Auth Module Work"
    Then the tab label updates to "Auth Module Work"
    And the session metadata on disk reflects the new title

  @regression-desktop-session-delete
  Scenario: Deleting a session removes it from list and disk
    Given a session "Old Work" exists on disk
    When the user deletes it
    Then the tab is removed from the session list
    And the session directory is cleaned up
    And a confirmation undo toast appears: "Session deleted. Undo?"

  @regression-desktop-session-archive
  Scenario: Archiving a session hides it from active list
    Given a session "Completed Task" exists
    When the user archives it
    Then the session no longer appears in the active session list
    And the session is moved to the archive view
    And the session data is preserved on disk

  # ── Multi-Tab Isolation (desktop/tabs.go) ─────────────────────────

  @regression-desktop-tab-isolation
  Scenario: Each tab has its own session and model
    Given tab A is using model "deepseek-v4-flash"
    And tab B is using model "deepseek-v4-pro"
    When the user switches between tabs
    Then tab A's model remains "deepseek-v4-flash"
    And tab B's model remains "deepseek-v4-pro"
    And changing models in one tab does not affect the other

  @regression-desktop-tab-isolation
  Scenario: Each tab has its own conversation context
    Given tab A has a conversation about authentication
    And tab B has a conversation about database migrations
    When the user asks tab A "What were we discussing?"
    Then tab A's agent references the authentication conversation
    And does NOT reference the database migration conversation from tab B

  @regression-desktop-tab-isolation
  Scenario: Closing a tab does not affect other tabs
    Given 3 tabs are open
    When the user closes tab 2
    Then tabs 1 and 3 remain open with their conversations intact
    And tab 1's agent is unaffected

  @regression-desktop-tab-order
  Scenario: Tab order persists across app restart
    Given tabs are ordered: A, B, C
    When the app is closed and reopened
    Then tabs appear in order: A, B, C
    And the previously active tab is focused

  @regression-desktop-tab-runtime-status
  Scenario: Tab shows agent runtime status
    Given tab A's agent is running a turn
    When the tab renders
    Then the tab shows a running indicator (spinner or ●)
    When the turn completes
    Then the tab indicator shows idle

  # ── Workspace Isolation (desktop/workspace.go) ────────────────────

  @regression-desktop-workspace
  Scenario: Each tab can have a different workspace root
    Given tab A's workspace root is /home/user/project-a
    And tab B's workspace root is /home/user/project-b
    When tab A's agent runs `ls`
    Then it lists files from /home/user/project-a
    And when tab B's agent runs `ls`
    Then it lists files from /home/user/project-b

  @regression-desktop-workspace
  Scenario: Workspace changes are detected live
    Given a tab is monitoring the workspace for file changes
    When a file is created, modified, or deleted outside the agent
    Then the workspace change is detected
    And the agent's context reflects the change on the next turn

  # ── Settings Persistence (desktop/settings_app.go) ────────────────

  @regression-desktop-settings
  Scenario: Settings persist across app restart
    Given the user changes theme to "light" and default model to "pro"
    When the app is closed and reopened
    Then the theme is "light"
    And the default model is "pro"

  @regression-desktop-settings
  Scenario: Settings are scoped per workspace when appropriate
    Given the user set model to "flash" in project-a
    And the user set model to "pro" in project-b
    When opening project-a, the model is "flash"
    When opening project-b, the model is "pro"

  @regression-desktop-settings
  Scenario: Provider access list restricts visible models
    Given provider_access = ["deepseek"]
    When the model selector renders
    Then only DeepSeek models are listed
    And Anthropic models are hidden

  @regression-desktop-settings
  Scenario: Status bar items are configurable
    Given status_bar_items = ["model", "tokens", "cost"]
    When the status bar renders
    Then model, tokens, and cost are shown in that order
    And other available items are hidden

  @regression-desktop-settings
  Scenario: Layout style changes take effect immediately
    Given the current layout is "classic"
    When the user changes to "workbench"
    Then the layout updates without restart
    And panels reposition to the workbench configuration

  @regression-desktop-settings
  Scenario: Display mode compact reduces UI density
    Given display_mode changes from "standard" to "compact"
    When the UI re-renders
    Then padding and spacing are reduced
    And font sizes may decrease
    And all functionality remains accessible

  @regression-desktop-settings
  Scenario: Close behavior "background" hides to tray
    Given close_behavior = "background"
    When the user clicks the window close button
    Then the window hides
    And the app continues running in the system tray
    And sessions continue processing

  @regression-desktop-settings
  Scenario: Close behavior "quit" exits the app
    Given close_behavior = "quit"
    When the user clicks the window close button
    Then the app saves all sessions
    And the app exits

  @regression-desktop-settings
  Scenario: Default tool approval mode applied to new sessions
    Given default_tool_approval_mode = "yolo"
    When a new session is created
    Then tool approval mode is "yolo"
    And tools execute without prompting

  # ── Auto-Save (desktop/app_autosave_test.go) ──────────────────────

  @regression-desktop-autosave
  Scenario: Session auto-saves on close
    Given a session with 20 messages
    When the tab is closed
    Then the session JSONL is written to disk
    And all 20 messages are preserved

  @regression-desktop-autosave
  Scenario: Session auto-saves periodically during use
    Given a session is active for 5 minutes
    And messages have been added
    Then the session is auto-saved at least once
    And the on-disk transcript matches the in-memory state

  # ── Crash Recovery (desktop/crash_pending.go) ─────────────────────

  @regression-desktop-crash
  Scenario: Pending state reconciled on restart after crash
    Given the app crashed while a session had unpersisted messages
    When the app restarts
    Then the pending cleanup reconciler runs
    And the session is restored to the last known good state
    And a notice is shown: "Recovered 1 session from previous crash"

  @regression-desktop-crash
  Scenario: No false crash detection on clean shutdown
    Given the app exited cleanly
    When the app restarts
    Then no crash recovery notice is shown
    And sessions load from their normal persisted state

  # ── Single Instance (desktop/single_instance.go) ──────────────────

  @regression-desktop-single-instance
  Scenario: Second launch focuses existing window
    Given the app is already running
    When the user launches the app again
    Then a new window is NOT created
    And the existing window is focused and brought to the foreground

  @regression-desktop-single-instance
  Scenario: Second launch with file opens file in existing instance
    Given the app is already running
    When the user launches `reasonix path/to/file.txt`
    Then the existing instance opens the file
    And no second instance is created

  # ── System Tray (desktop/tray.go) ─────────────────────────────────

  @regression-desktop-tray
  Scenario: Tray icon shown when app is running
    When the desktop app starts
    Then a tray icon appears
    And the tray menu includes "Show", "New Session", "Quit"

  @regression-desktop-tray
  Scenario: Tray "Show" restores hidden window
    Given the window is hidden to tray
    When the user clicks "Show" in the tray menu
    Then the window is restored and focused

  @regression-desktop-tray
  Scenario: Tray icon reflects agent activity
    Given an agent is actively working
    When the tray icon renders
    Then the icon shows an activity indicator (different from idle)

  # ── Updater (desktop/updater.go) ──────────────────────────────────

  @regression-desktop-updater
  Scenario: Update check runs on startup when enabled
    Given check_updates = true
    When the app starts
    Then a version check is performed
    And if an update is available, a notification is shown

  @regression-desktop-updater
  Scenario: Update check skipped when disabled
    Given check_updates = false
    When the app starts
    Then no version check is performed

  @regression-desktop-updater
  Scenario: Update downloads and prompts for restart
    Given an update is available
    When the user clicks "Update"
    Then the update is downloaded
    And a restart prompt appears: "Update ready. Restart now?"

  # ── Chat Input (desktop/frontend/Composer.tsx) ────────────────────

  @regression-desktop-composer
  Scenario: Chat input accepts multi-line text
    When the user types multiple lines in the composer
    Then all lines are preserved
    And Shift+Enter inserts a newline
    And Enter submits the message

  @regression-desktop-composer
  Scenario: @ file reference opens file picker
    When the user types "@" in the composer
    Then a file picker menu appears
    And typing filters the file list
    And selecting a file inserts the reference

  @regression-desktop-composer
  Scenario: / slash command opens command picker
    When the user types "/" in the composer
    Then a command picker appears listing available slash commands
    And built-in skills are included
    And custom .reasonix/commands are included

  @regression-desktop-composer
  Scenario: Paste handler strips formatting
    When the user pastes rich text
    Then only plain text is inserted
    And no HTML or styling is preserved

  @regression-desktop-composer
  Scenario: Composer shows character/token count at threshold
    When the composer exceeds a configured length
    Then a character or token count indicator appears
    And the indicator updates live as the user types

  # ── Desktop Notifications ─────────────────────────────────────────

  @regression-desktop-notifications
  Scenario: Turn completion triggers OS notification
    Given notifications are enabled
    When an agent turn completes
    Then an OS notification is shown
    And the notification body summarizes the turn outcome

  @regression-desktop-notifications
  Scenario: Approval request triggers OS notification
    Given notifications are enabled
    When a tool requires approval
    Then an OS notification is shown: "reasonix: tool approval needed"
    And clicking the notification focuses the app

  @regression-desktop-notifications
  Scenario: Notifications suppressed when app is focused
    Given notifications are enabled
    And the app window is focused and visible
    When a turn completes
    Then no OS notification is shown (user is already watching)

  # ── Desktop Keyboard Shortcuts ────────────────────────────────────

  @regression-desktop-keyboard
  Scenario: Ctrl+N creates new session
    When the user presses Ctrl+N
    Then a new session tab is created
    And the composer is focused in the new tab

  @regression-desktop-keyboard
  Scenario: Ctrl+W closes current tab
    Given multiple tabs are open
    When the user presses Ctrl+W
    Then the current tab is closed
    And focus moves to the adjacent tab

  @regression-desktop-keyboard
  Scenario: Ctrl+Tab cycles through tabs
    Given 3 tabs are open
    When the user presses Ctrl+Tab
    Then focus moves to the next tab
    And pressing repeatedly cycles through all tabs

  @regression-desktop-keyboard
  Scenario: Ctrl+Shift+Tab cycles backwards through tabs
    Given 3 tabs are open and focus is on tab 3
    When the user presses Ctrl+Shift+Tab
    Then focus moves to tab 2
