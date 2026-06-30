Feature: Internationalization
  As a Chinese-speaking developer using reasonix
  I want every new feature to be available in my language from day one
  So that my primary language experience is never degraded by new features

  Background:
    Given reasonix has i18n infrastructure with en, zh, and zh-tw locales
    And every user-facing string uses an i18n key

  # ── New Feature Strings ───────────────────────────────────────────

  @i18n-new-features
  Scenario: All new tool descriptions have zh translations
    Given the workflow, send_to_subagent, web_search, monitor, report_findings, schedule_task, and enter_worktree tools exist
    When the language is set to zh
    Then every tool description renders in Chinese
    And no tool description falls back to English

  @i18n-new-features
  Scenario: All new skill bodies have zh translations
    Given the deep-research, code-review, verify, simplify, run, loop, update-config, review(PR), permissions, and keybindings skills exist
    When the language is set to zh
    Then every skill body renders in Chinese
    And the DeepSeek-specific quality notes are translated
    And the effort calibration table renders in Chinese

  @i18n-new-features
  Scenario: All new error messages have zh translations
    Given any new tool or feature produces an error
    When the language is set to zh
    Then the error message renders in Chinese
    And the error structure (what happened, why, what to do) is preserved

  @i18n-new-features
  Scenario: All new UI labels have zh translations
    Given the background tasks panel, workflow visualizations, remotes panel, and metrics dashboard exist
    When the language is set to zh
    Then all panel headers, button labels, status text, placeholder text, and tooltips render in Chinese

  # ── Translation Completeness ──────────────────────────────────────

  @i18n-completeness
  Scenario: Translation coverage check runs in CI
    Given a new user-facing string was added without a zh translation
    When the i18n coverage check runs
    Then the CI fails: "Missing zh translation for key 'workflow.stage.progress'"
    And the failure message names the exact key and file where it was introduced

  @i18n-completeness
  Scenario: Missing translation falls back gracefully with an indicator
    Given a string is missing from zh but present in en
    When the string renders in zh mode
    Then the English fallback is used
    And a non-intrusive marker indicates the fallback: "[en]"
    And the marker links to the translation contribution guide

  # ── CJK Rendering ─────────────────────────────────────────────────

  @i18n-cjk
  Scenario: CJK characters in subagent reasoning display correctly
    Given a subagent's reasoning contains Chinese text
    When the reasoning renders in the background panel
    Then the characters are displayed correctly
    And no garbled characters appear
    And character truncation respects CJK character boundaries

  @i18n-cjk
  Scenario: CJK text in tool output does not break layout
    Given a subagent's tool output contains a mix of Chinese and English
    When the output renders
    Then the layout does not overflow
    And text wraps correctly at CJK word boundaries
    And monospace alignment is maintained

  @i18n-cjk
  Scenario: CJK input in composer works with IME
    Given the user's system IME is active for Chinese input
    When the user composes text in the desktop composer
    Then the composition state is rendered correctly
    And the final committed text is inserted without artifacts

  # ── Locale Switch ─────────────────────────────────────────────────

  @i18n-switch
  Scenario: Language switch updates all visible strings immediately
    Given the TUI is running in English
    When the user switches to Chinese
    Then all visible UI strings update to Chinese without restart
    And the switch takes effect within one render frame

  @i18n-switch
  Scenario: Locale persists across sessions
    Given the user set language to zh
    When reasonix restarts
    Then the UI renders in Chinese
    And the language setting is loaded from config

  @i18n-switch
  Scenario: Language auto-detection from environment
    Given language is set to "" (auto)
    And LANG=zh_CN.UTF-8
    When reasonix starts
    Then the UI renders in Chinese
    And the detected language is logged
