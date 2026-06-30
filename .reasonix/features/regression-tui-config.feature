Feature: Regression — TUI + Config
  As a developer protecting reasonix's existing UI and configuration
  I want no additive feature to break the TUI, config loading, or settings
  So that the product looks and behaves identically before and after new features

  Background:
    Given reasonix TUI is running with a valid config

  # ── TUI: Markdown ─────────────────────────────────────────────────

  @regression-tui-markdown
  Scenario: Headers render at correct levels
    When the agent outputs "# Title", "## Section", "### Subsection"
    Then the TUI renders each at the correct header level
    And headers are visually distinct from body text

  @regression-tui-markdown
  Scenario: Code blocks render in monospace
    When the agent outputs a fenced code block with Go code
    Then the code block background is distinct
    And the code is rendered in monospace
    And syntax highlighting (if enabled) is applied

  @regression-tui-markdown
  Scenario: Lists render with bullets
    When the agent outputs a bullet list with 3 items
    Then each item is indented
    And a bullet character precedes each item

  @regression-tui-markdown
  Scenario: CJK characters render without corruption
    When the agent outputs Chinese text: "这是一个测试"
    Then the characters render correctly
    And no mojibake or encoding errors appear

  # ── TUI: Diff View ────────────────────────────────────────────────

  @regression-tui-diff
  Scenario: Additions shown in green
    Given a diff with added lines
    When the diff view renders
    Then added lines are shown in green (or with + prefix)

  @regression-tui-diff
  Scenario: Deletions shown in red
    Given a diff with removed lines
    When the diff view renders
    Then removed lines are shown in red (or with - prefix)

  @regression-tui-diff
  Scenario: Line numbers shown
    Given a diff with changes at lines 42-45
    When the diff view renders
    Then line numbers are displayed
    And the line numbers correspond to the original file

  # ── TUI: Status Line ──────────────────────────────────────────────

  @regression-tui-statusline
  Scenario: Status line shows active model
    Given the active model is "deepseek-v4-flash"
    When the status line renders
    Then "deepseek-v4-flash" appears in the status line

  @regression-tui-statusline
  Scenario: Status line shows token usage
    Given the current turn consumed 1500 input and 800 output tokens
    When the status line renders
    Then token usage is displayed
    And the format is human-readable (e.g. "2.3K")

  @regression-tui-statusline
  Scenario: Status line shows context gauge
    Given the session has used 45K of a 100K context window
    When the status line renders
    Then a context gauge shows approximately 45% full
    And the gauge is visual (bar or percentage)

  @regression-tui-statusline
  Scenario: Status line shows cost when pricing configured
    Given the provider has pricing configured
    And the session has consumed tokens
    When the status line renders
    Then the session cost is displayed

  # ── TUI: Model Switcher ───────────────────────────────────────────

  @regression-tui-model-switcher
  Scenario: /model lists all configured models
    Given 3 providers are configured with models
    When the user types /model
    Then all available models are listed
    And each entry shows provider name and model name

  @regression-tui-model-switcher
  Scenario: /model switches active model
    Given the user has selected a different model from the list
    When the model switch is confirmed
    Then subsequent turns use the new model
    And the status line updates to show the new model

  # ── TUI: Tool Cards ────────────────────────────────────────────────

  @regression-tui-tool-card
  Scenario: bash tool card shows command
    Given the agent calls bash with "go test ./..."
    When the tool card renders
    Then the command is visible
    And the status spinner animates during execution

  @regression-tui-tool-card
  Scenario: read_file tool card shows file path
    Given the agent calls read_file on "auth/auth.go"
    When the tool card renders
    Then the file path is visible

  @regression-tui-tool-card
  Scenario: Tool card shows error in red
    Given a tool call returned an error
    When the tool card renders
    Then the error text is shown in red
    And the error icon is displayed

  # ── TUI: Transcript ───────────────────────────────────────────────

  @regression-tui-transcript
  Scenario: Transcript preserves message ordering
    Given a conversation with 10 messages alternating user/assistant
    When the transcript renders
    Then messages appear in chronological order
    And user messages are visually distinct from assistant messages

  @regression-tui-transcript
  Scenario: Transcript is scrollable
    Given a conversation with 50 messages exceeding the viewport height
    When the transcript renders
    Then only a portion is visible
    And scrolling reveals earlier and later messages

  # ── Config Loading ────────────────────────────────────────────────

  @regression-config-load
  Scenario: Config loads from project reasonix.toml
    Given a project with ./reasonix.toml containing default_model = "custom-model"
    When config is loaded
    Then DefaultModel is "custom-model"

  @regression-config-load
  Scenario: Config resolution order is correct
    Given user config has default_model = "user-model"
    And project config has default_model = "project-model"
    When config is loaded
    Then DefaultModel is "project-model" (project overrides user)

  @regression-config-load
  Scenario: Flag overrides config
    Given config has default_model = "config-model"
    When the --model "flag-model" flag is passed
    Then the active model is "flag-model"

  @regression-config-load
  Scenario: Missing project config falls back to user config
    Given no ./reasonix.toml exists
    And user config has default_model = "user-model"
    When config is loaded
    Then DefaultModel is "user-model"

  @regression-config-load
  Scenario: Missing all configs uses built-in defaults
    Given no user or project config exists
    When config is loaded with Default()
    Then built-in defaults are used
    And no error is returned

  @regression-config-credentials
  Scenario: API key resolved from env var
    Given provider config has api_key_env = "DEEPSEEK_API_KEY"
    And DEEPSEEK_API_KEY is set to "sk-test-123"
    When the provider is constructed
    Then the API key is "sk-test-123"
    And the key value is never in the config file

  @regression-config-credentials
  Scenario: Missing env var produces empty key
    Given provider config has api_key_env = "MISSING_KEY"
    And MISSING_KEY is not set
    When the provider is constructed
    Then the API key is empty
    And a warning is emitted when RequireKey is false

  # ── Theme ─────────────────────────────────────────────────────────

  @regression-tui-theme
  Scenario: Dark theme renders correctly
    Given theme = "dark"
    When the TUI renders
    Then the background is dark
    And text is light-colored
    And all colors are from the dark palette

  @regression-tui-theme
  Scenario: Theme switch takes effect immediately
    Given the TUI is running with dark theme
    When the user switches to light theme
    Then the TUI re-renders with light theme colors
    And no restart is required

  @regression-tui-theme
  Scenario: Auto theme detects terminal background
    Given theme = "auto"
    When the terminal reports a dark background
    Then the dark theme is used
    When the terminal reports a light background
    Then the light theme is used

  # ── i18n ──────────────────────────────────────────────────────────

  @regression-i18n
  Scenario: English strings resolve correctly
    Given language = "en"
    When any UI string is rendered
    Then the output is in English

  @regression-i18n
  Scenario: Chinese strings resolve correctly
    Given language = "zh"
    When any UI string is rendered
    Then the output is in Chinese

  @regression-i18n
  Scenario: Language auto-detection from LANG
    Given language = "" and LANG = "zh_CN.UTF-8"
    When the language is resolved
    Then "zh" is used
