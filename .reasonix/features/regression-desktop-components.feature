Feature: Regression — Desktop Component-Level
  As a developer protecting reasonix's desktop React frontend
  I want no additive feature to break any existing React component, library, or store
  So that desktop users see no visual or behavioral regression

  Background:
    Given the reasonix desktop app is running with the React frontend loaded

  # ── Composer (components/Composer.tsx) ────────────────────────────

  @regression-fe-composer-send
  Scenario: Composer sends message on Enter
    When the user types "Fix the auth bug" and presses Enter
    Then the message is sent to the agent
    And the composer clears
    And the sent message appears in the transcript

  @regression-fe-composer-send
  Scenario: Composer does not send on Shift+Enter
    When the user types "line1" then Shift+Enter then "line2" then Enter
    Then the message sent contains "line1\nline2"
    And Shift+Enter inserted a newline, did not send

  @regression-fe-composer-send
  Scenario: Composer does not send empty message
    When the user presses Enter on an empty composer
    Then no message is sent
    And the composer remains focused

  @regression-fe-composer-send
  Scenario: Composer does not send whitespace-only message
    When the user types "   " and presses Enter
    Then no message is sent
    And the whitespace is cleared

  @regression-fe-composer-history
  Scenario: Up arrow in empty composer recalls last sent message
    Given the user sent "previous message"
    When the user presses Up in an empty composer
    Then "previous message" is loaded
    And the cursor is at the end of the text

  @regression-fe-composer-history
  Scenario: Down arrow clears recalled message
    Given the user pressed Up and recalled "previous message"
    When the user presses Down
    Then the composer clears to empty

  @regression-fe-composer-draft
  Scenario: Composer draft persists across tab switches
    Given the user typed "incomplete message" in tab A
    When the user switches to tab B and back to tab A
    Then "incomplete message" is still in the composer

  @regression-fe-composer-goal
  Scenario: Goal toggle enables goal mode
    When the user toggles goal mode in the composer
    Then the composer shows a goal input instead of chat input
    And the placeholder changes to goal-specific text

  @regression-fe-composer-image
  Scenario: Composer supports image attachment
    When the user attaches an image to the composer
    Then a thumbnail preview appears in the composer area
    And the image is sent with the message

  @regression-fe-composer-keyboard
  Scenario: Composer keyboard shortcuts are active
    When the composer is focused
    Then Ctrl+Enter or Cmd+Enter sends the message
    And Escape blurs the composer
    And standard text editing shortcuts work (Ctrl+A, Ctrl+Z, etc.)

  @regression-fe-composer-profile
  Scenario: Composer shows active model profile
    Given the active model is "deepseek-v4-pro"
    When the composer renders
    Then the model name is shown near the composer
    And the effort level is visible

  # ── Transcript (components/Transcript.tsx) ────────────────────────

  @regression-fe-transcript-render
  Scenario: Transcript renders user and assistant messages distinctly
    Given a conversation with user and assistant messages
    When the transcript renders
    Then user messages are right-aligned or visually distinct
    And assistant messages are left-aligned or visually distinct
    And message roles are clear at a glance

  @regression-fe-transcript-scroll
  Scenario: Transcript auto-scrolls to bottom on new message
    Given the transcript is scrolled to the bottom
    When a new assistant message streams in
    Then the transcript scrolls to stay at the bottom
    And the newest content is always visible

  @regression-fe-transcript-scroll
  Scenario: Transcript does not auto-scroll when user scrolled up
    Given the user scrolled up to read older messages
    When a new assistant message streams in
    Then the transcript does NOT auto-scroll
    And the user's scroll position is preserved
    And a "↓ New messages" indicator appears

  @regression-fe-transcript-grouping
  Scenario: Sequential tool calls are grouped
    Given the agent makes 3 tool calls in sequence
    When the transcript renders
    Then the 3 tool calls are visually grouped
    And the group shows a summary: "3 tool calls"

  @regression-fe-transcript-tool-status
  Scenario: Tool cards show live status
    Given a bash tool call is running
    When the tool card renders
    Then a spinner or progress indicator is shown
    When the tool completes
    Then the indicator changes to ✓ (success) or ✗ (error)
    And the duration is shown

  @regression-fe-transcript-reasoning
  Scenario: Reasoning display shows thinking content when expanded
    Given the agent produced thinking content
    When the reasoning display is collapsed
    Then a collapsed indicator is shown: "Thinking..."
    When the user clicks to expand
    Then the thinking content is displayed
    And it is visually distinct from the final answer

  @regression-fe-transcript-diff
  Scenario: Inline diff shows code changes
    Given the agent edited a file
    When the diff renders in the transcript
    Then added lines are shown in green
    And removed lines are shown in red
    And line numbers are preserved

  # ── Message (components/Message.tsx) ──────────────────────────────

  @regression-fe-message-copy
  Scenario: Message text is selectable and copyable
    When the user selects text in an assistant message
    Then the selection is captured
    And Ctrl+C copies the selected text
    And no extra formatting is included in the copy

  @regression-fe-message-copy
  Scenario: Copy button copies full message
    Given a message has a copy button
    When the user clicks the copy button
    Then the full message text is copied
    And a "Copied" confirmation appears briefly

  @regression-fe-message-edit-replay
  Scenario: Editing a user message replays the conversation
    Given a user message sent 3 turns ago
    When the user edits that message and confirms
    Then the conversation is truncated at that message
    And the agent reprocesses from the edited message

  # ── Markdown (components/Markdown.tsx, MarkdownRenderer.tsx) ──────

  @regression-fe-markdown-code
  Scenario: Code blocks render with syntax highlighting
    Given a message contains a fenced Go code block
    When the markdown renders
    Then syntax highlighting is applied to the Go code
    And the code block has a distinct background

  @regression-fe-markdown-code
  Scenario: Code block copy button copies raw code
    Given a code block with syntax highlighting
    When the user clicks the copy button on the code block
    Then the raw code text is copied (no highlighting markup)
    And a "Copied" tooltip appears

  @regression-fe-markdown-mermaid
  Scenario: Mermaid diagrams render as SVG
    Given a message contains a mermaid code block
    When the markdown renders
    Then the mermaid diagram is rendered as an SVG
    And the diagram is visible and correctly laid out

  @regression-fe-markdown-math
  Scenario: LaTeX math renders correctly
    Given a message contains inline math $E = mc^2$ and display math $$\int_0^1$$
    When the markdown renders
    Then both expressions are rendered as formatted math
    And inline math stays inline, display math is block

  @regression-fe-markdown-math
  Scenario: Young diagrams render correctly
    Given a message contains a Young diagram notation
    When the markdown renders
    Then the Young diagram is rendered visually

  @regression-fe-markdown-table
  Scenario: Tables render with proper alignment
    Given a message contains a markdown table
    When the markdown renders
    Then columns are aligned
    And headers are visually distinct

  # ── Code Viewer (components/CodeViewer.tsx) ───────────────────────

  @regression-fe-code-viewer
  Scenario: Code viewer opens file with syntax highlighting
    When the user opens a .go file in the code viewer
    Then Go syntax highlighting is applied
    And line numbers are shown
    And the file is scrollable

  @regression-fe-code-viewer
  Scenario: Code viewer renders diff mode
    When the code viewer opens in diff mode with + and - lines
    Then additions are green, deletions are red
    And line numbers from both old and new files are shown

  # ── Diff View (components/DiffView.tsx, InlineDiff.tsx) ───────────

  @regression-fe-diff-view
  Scenario: Diff view shows hunks with context
    Given a diff with changes in 3 separate hunks
    When the diff view renders
    Then each hunk is visually separated
    And context lines between hunks are shown
    And each hunk shows the line range

  @regression-fe-diff-view
  Scenario: Inline diff collapses unchanged large sections
    Given a diff where 50 unchanged lines separate two hunks
    When the inline diff renders
    Then the 50 unchanged lines are collapsed
    And a "Show 50 unchanged lines" expander is shown

  # ── Model Switcher (components/ModelSwitcher.tsx) ─────────────────

  @regression-fe-model-switcher
  Scenario: Model switcher lists all configured providers
    Given 3 providers are configured
    When the model switcher opens
    Then all 3 providers are listed
    And each provider's models are listed underneath

  @regression-fe-model-switcher
  Scenario: Switching model refreshes the session
    Given the user selects a different model
    When the switch is confirmed
    Then the active model changes
    And the status bar updates immediately
    And subsequent messages use the new model

  @regression-fe-model-switcher
  Scenario: Model switcher shows effort level per model
    When the model switcher renders
    Then each model shows its configured effort level
    And the effort can be changed from the switcher

  # ── Effort Switcher (components/EffortSwitcher.tsx) ───────────────

  @regression-fe-effort-switcher
  Scenario: Effort switcher shows available levels
    Given the active model supports "high" and "max"
    When the effort switcher renders
    Then only "high" and "max" are shown
    And the current effort is highlighted

  @regression-fe-effort-switcher
  Scenario: Changing effort takes effect on next turn
    When the user changes effort from "high" to "max"
    Then the effort is updated
    And the next agent turn uses "max"

  # ── Tab Bar (components/TabBar.tsx) ───────────────────────────────

  @regression-fe-tab-bar
  Scenario: Tab bar shows all open sessions
    Given 3 sessions are open
    When the tab bar renders
    Then all 3 tabs are visible
    And the active tab is highlighted

  @regression-fe-tab-bar
  Scenario: Clicking a tab switches to that session
    Given tabs A and B are open, A is active
    When the user clicks tab B
    Then tab B becomes active
    And the transcript for session B is shown

  @regression-fe-tab-bar
  Scenario: Closing a tab via middle-click or X button
    Given 3 tabs are open
    When the user clicks the X on tab 2
    Then tab 2 closes
    And its session is persisted
    And tab 1 or 3 becomes active

  @regression-fe-tab-bar
  Scenario: Tab shows session runtime status
    Given session A's agent is running
    When the tab renders
    Then tab A shows a running indicator
    And idle tabs show no indicator

  # ── Settings Panel (components/SettingsPanel.tsx) ─────────────────

  @regression-fe-settings
  Scenario: Settings panel shows all config sections
    When the settings panel opens
    Then sections are listed: General, Appearance, Models, Tools, Permissions, Notifications, Hooks, Skills, MCP, Network, About

  @regression-fe-settings
  Scenario: Changing a setting writes to config
    When the user changes a setting
    Then the config file is updated
    And the change takes effect without restart

  @regression-fe-settings
  Scenario: Settings panel shows current values
    When the settings panel renders
    Then every setting shows its current value
    And no setting shows a stale or default value

  @regression-fe-settings
  Scenario: Settings refresh after external change
    Given the user edited reasonix.toml externally
    When the settings panel is still open
    Then the changed settings update to reflect the file

  # ── Command Palette (components/CommandPalette.tsx) ───────────────

  @regression-fe-command-palette
  Scenario: Command palette opens with Ctrl+K
    When the user presses Ctrl+K (or Cmd+K)
    Then the command palette opens
    And available commands are listed

  @regression-fe-command-palette
  Scenario: Command palette filters by fuzzy match
    When the user types "model" in the command palette
    Then commands matching "model" are shown
    And non-matching commands are hidden

  @regression-fe-command-palette
  Scenario: Selecting a command executes it
    Given the command palette is open
    When the user selects "Switch Model"
    Then the model switcher opens
    And the command palette closes

  @regression-fe-command-palette
  Scenario: Command palette closes on Escape
    Given the command palette is open
    When the user presses Escape
    Then the command palette closes

  # ── Approval Modal (components/ApprovalModal.tsx) ─────────────────

  @regression-fe-approval
  Scenario: Approval modal appears for tool requiring permission
    Given tool approval mode is "ask"
    When the agent calls bash with "rm -rf"
    Then an approval modal appears
    And the modal shows: tool name, arguments, risk description

  @regression-fe-approval
  Scenario: Approving a tool executes it
    Given the approval modal is open
    When the user clicks "Approve"
    Then the tool executes
    And the modal closes
    And the tool result appears in the transcript

  @regression-fe-approval
  Scenario: Denying a tool blocks it
    Given the approval modal is open
    When the user clicks "Deny"
    Then the tool is blocked
    And the agent receives a denial message
    And the agent adapts its approach

  @regression-fe-approval
  Scenario: Approval modal shows file references
    Given the tool arguments reference a file path
    When the approval modal renders
    Then the file path is clickable
    And clicking opens the file in the code viewer

  # ── Ask Card (components/AskCard.tsx) ─────────────────────────────

  @regression-fe-ask-card
  Scenario: Ask card renders multiple-choice question
    When the agent calls the ask tool with a question and options
    Then an ask card appears
    And the question text is displayed
    And the options are rendered as clickable buttons

  @regression-fe-ask-card
  Scenario: Ask card with multiSelect renders checkboxes
    When the agent calls ask with multiSelect: true
    Then checkboxes are rendered instead of radio buttons
    And multiple options can be selected

  @regression-fe-ask-card
  Scenario: Selecting an answer returns it to the agent
    When the user selects an option
    Then the answer is sent back
    And the ask card closes
    And the agent continues with the answer

  # ── Process Card (components/ProcessCard.tsx) ─────────────────────

  @regression-fe-process-card
  Scenario: Process card shows running job status
    Given a background job is running
    When the process card renders
    Then the job label and kind are shown
    And a status indicator animates

  @regression-fe-process-card
  Scenario: Process card updates on job completion
    Given a running background job completes
    When the process card updates
    Then the status shows ✓ complete
    And the result or output is shown

  # ── Todo Panel (components/TodoPanel.tsx) ─────────────────────────

  @regression-fe-todo-panel
  Scenario: Todo panel shows task list
    Given the agent created a todo list with 3 items
    When the todo panel renders
    Then all 3 items are listed with their statuses
    And completed items are marked ✓

  @regression-fe-todo-panel
  Scenario: Todo panel updates live as tasks complete
    Given a todo item is pending
    When the agent calls complete_step for that item
    Then the todo panel updates to show it as completed

  # ── Context Panel (components/ContextPanel.tsx) ────────────────────

  @regression-fe-context-panel
  Scenario: Context panel shows token breakdown
    When the context panel opens
    Then input tokens, output tokens, and cache tokens are shown
    And a visual bar shows context window usage

  @regression-fe-context-panel
  Scenario: Context panel shows cost breakdown
    When the context panel renders
    Then session cost, turn cost, and daily cost are shown
    And hover reveals per-provider breakdown

  # ── Workspace Panel (components/WorkspacePanel.tsx) ───────────────

  @regression-fe-workspace-panel
  Scenario: Workspace panel shows file tree
    When the workspace panel renders
    Then the project file tree is displayed
    And directories are expandable/collapsible

  @regression-fe-workspace-panel
  Scenario: Workspace panel reflects file changes
    Given the agent created a new file
    When the workspace panel refreshes
    Then the new file appears in the tree

  @regression-fe-workspace-panel
  Scenario: Clicking a file opens it in the code viewer
    Given the workspace panel is open
    When the user clicks a .go file
    Then the file opens in the code viewer
    And syntax highlighting is applied

  # ── Project Tree (components/ProjectTree.tsx) ─────────────────────

  @regression-fe-project-tree
  Scenario: Project tree renders hierarchical file structure
    Given a project with nested directories
    When the project tree renders
    Then the hierarchy is correct
    And directory nesting is visually indented

  @regression-fe-project-tree
  Scenario: Project tree search filters files
    When the user types in the project tree search
    Then only matching files and directories are shown
    And non-matching items are hidden

  # ── Status Bar (components/StatusBar.tsx) ─────────────────────────

  @regression-fe-status-bar
  Scenario: Status bar shows configured items in order
    Given status_bar_items = ["model", "tokens", "cost", "context"]
    When the status bar renders
    Then model, tokens, cost, and context are shown in that order
    And each item is visually separated

  @regression-fe-status-bar
  Scenario: Status bar items update live
    Given the agent completes a turn consuming 500 tokens
    When the status bar updates
    Then the token count increments by 500
    And the cost updates accordingly

  @regression-fe-status-bar
  Scenario: Status bar workspace indicator
    Given the current workspace is /home/user/project
    When the status bar renders
    Then the workspace path or project name is shown

  # ── History Panel (components/HistoryPanel.tsx) ────────────────────

  @regression-fe-history-panel
  Scenario: History panel lists past sessions
    When the history panel opens
    Then past sessions are listed in reverse chronological order
    And each entry shows: title, model, date, message count

  @regression-fe-history-panel
  Scenario: Clicking a history entry opens that session
    Given a session "Auth work" exists in history
    When the user clicks it
    Then the session opens in a new or existing tab
    And the full conversation is loaded

  # ── Memory Panel (components/MemoryPanel.tsx) ─────────────────────

  @regression-fe-memory-panel
  Scenario: Memory panel shows REASONIX.md contents
    When the memory panel opens
    Then the contents of the project's memory file are displayed
    And sections are rendered as markdown

  @regression-fe-memory-panel
  Scenario: Memory compiler traces are visible
    Given memory compiler is enabled
    When the memory panel renders
    Then compiled execution traces are shown
    And memory citations are visible

  # ── Error Boundary (components/ErrorBoundary.tsx) ─────────────────

  @regression-fe-error-boundary
  Scenario: Error boundary catches render errors
    Given a component throws during render
    When the error boundary catches it
    Then an error state is displayed
    And the rest of the app continues to function
    And a "Reload" button is shown

  @regression-fe-error-boundary
  Scenario: Error boundary reports crash
    Given the error boundary catches an error
    When crash reporting is enabled
    Then the error is reported via the crash bridge

  # ── Resizable Drawer (components/ResizableDrawer.tsx) ─────────────

  @regression-fe-resizable-drawer
  Scenario: Drawer resizes via drag handle
    Given a resizable drawer is open
    When the user drags the resize handle
    Then the drawer width or height changes
    And the minimum/maximum constraints are enforced

  # ── Slash Menu (components/SlashMenu.tsx) ─────────────────────────

  @regression-fe-slash-menu
  Scenario: Slash menu opens from composer
    When the user types "/" in the composer
    Then the slash menu appears
    And available slash commands are listed with descriptions

  @regression-fe-slash-menu
  Scenario: Slash menu filters by typed text
    When the user types "/expl"
    Then commands matching "expl" are shown
    And the first match is highlighted

  @regression-fe-slash-menu
  Scenario: Selecting a slash command inserts it and shows args
    When the user selects a slash command with arguments
    Then the command is inserted with argument placeholders
    And the composer focuses the first argument

  # ── File Reference Menu (components/FileReferenceMenu.tsx) ────────

  @regression-fe-file-menu
  Scenario: @ opens file reference menu
    When the user types "@" in the composer
    Then the file reference menu opens
    And workspace files and directories are listed

  @regression-fe-file-menu
  Scenario: @ filters by filename as user types
    When the user types "@auth"
    Then only files matching "auth" appear
    And the menu narrows as more characters are typed

  # ── Tooltip (components/Tooltip.tsx) ──────────────────────────────

  @regression-fe-tooltip
  Scenario: Tooltip appears on hover
    When the user hovers over an element with a tooltip
    Then the tooltip appears after a brief delay
    And the tooltip content matches the element's description

  @regression-fe-tooltip
  Scenario: Tooltip dismisses on mouse leave
    Given a tooltip is visible
    When the user moves the mouse away
    Then the tooltip disappears

  # ── Toast (lib/toast.tsx) ─────────────────────────────────────────

  @regression-fe-toast
  Scenario: Toast notification appears briefly
    When a toast is triggered
    Then the toast appears in a corner
    And the toast auto-dismisses after a configurable duration

  @regression-fe-toast
  Scenario: Multiple toasts stack
    When 3 toasts are triggered in sequence
    Then all 3 are visible
    And they stack vertically without overlapping

  # ── Bridge (lib/bridge.ts) ────────────────────────────────────────

  @regression-fe-bridge
  Scenario: Bridge calls Go backend methods
    When the frontend calls a bridge method
    Then the Go backend receives the call
    And the result is returned to the frontend

  @regression-fe-bridge
  Scenario: Bridge handles backend errors gracefully
    When a bridge call fails
    Then an error is returned to the frontend
    And the frontend shows an error state, not a blank screen

  # ── Theme (lib/theme.ts) ──────────────────────────────────────────

  @regression-fe-theme
  Scenario: Dark theme applies dark color scheme
    Given theme is "dark"
    When the app renders
    Then the background is dark
    And text is light
    And all components use the dark palette

  @regression-fe-theme
  Scenario: Light theme applies light color scheme
    Given theme is "light"
    When the app renders
    Then the background is light
    And text is dark

  @regression-fe-theme
  Scenario: Auto theme detects system preference
    Given theme is "auto"
    And the system prefers dark mode
    When the app renders
    Then the dark theme is applied

  @regression-fe-theme
  Scenario: Theme CSS variables are consistent
    Given any theme is active
    When any component renders
    Then colors come from CSS variables
    And no hardcoded color values bypass the theme

  # ── i18n (lib/i18n.tsx) ──────────────────────────────────────────

  @regression-fe-i18n
  Scenario: UI strings render in configured language
    Given language is "zh"
    When any UI string renders
    Then the string is in Chinese
    And no English fallback is visible

  @regression-fe-i18n
  Scenario: Locale change takes effect without reload
    When the user changes language from "en" to "zh"
    Then all visible UI strings update to Chinese
    And no page reload is required

  @regression-fe-i18n
  Scenario: Missing translation falls back gracefully
    Given a string is missing from the zh locale
    When that string would render
    Then the English fallback is used
    And no blank or error is shown

  # ── Keyboard Shortcuts (lib/keyboardShortcuts.ts) ─────────────────

  @regression-fe-shortcuts
  Scenario: All registered keyboard shortcuts function
    Given the shortcuts registry is loaded
    When any registered shortcut is pressed
    Then the corresponding action fires
    And no shortcut is silently broken

  @regression-fe-shortcuts
  Scenario: Shortcuts cheatsheet shows all bindings
    When the user opens the shortcuts cheatsheet
    Then all registered shortcuts are listed
    And each shows the key combination and action description

  # ── Startup ───────────────────────────────────────────────────────

  @regression-fe-startup
  Scenario: Startup splash shows while loading
    When the app is loading
    Then a splash screen is shown
    And the splash dismisses when the app is ready

  @regression-fe-startup
  Scenario: Settings contract is validated on startup
    When the app starts
    Then the settings contract is validated
    And invalid settings are flagged with a warning

  @regression-fe-startup
  Scenario: Open sessions restore on startup
    Given 2 sessions were open when the app closed
    When the app restarts
    Then both sessions are restored
    And the previously active session is focused

  # ── Update Banner (components/UpdateBanner.tsx) ───────────────────

  @regression-fe-update-banner
  Scenario: Update banner shows when update available
    Given an update is available
    When the app renders
    Then a banner appears: "Update available — v2.0.0"

  @regression-fe-update-banner
  Scenario: Update banner dismisses
    Given the update banner is visible
    When the user dismisses it
    Then the banner hides
    And does not reappear for this version

  # ── Render Optimization ───────────────────────────────────────────

  @regression-fe-render-perf
  Scenario: Transcript does not re-render on every keystroke
    Given the transcript has 100 messages
    When the user types in the composer
    Then the transcript does NOT re-render (no DOM mutations in transcript area)
    And only the composer updates

  @regression-fe-render-perf
  Scenario: Tool cards do not re-render unrelated siblings
    Given 3 tool cards are visible from different tools
    When tool 2's status changes
    Then only tool 2's card re-renders
    And tool 1 and tool 3 are not re-rendered
