Feature: Accessibility
  As a developer who relies on assistive technology
  I want reasonix to be fully usable with screen readers, keyboard-only navigation, and configurable display
  So that my development environment is inclusive

  Background:
    Given reasonix has accessibility infrastructure
    And screen_reader_mode can be enabled in config

  # ── Desktop Screen Reader ─────────────────────────────────────────

  @a11y-desktop-aria
  Scenario: Every interactive element has an ARIA label
    Given screen reader mode is enabled
    When the desktop app renders
    Then every button has an aria-label describing its action
    And every input has an aria-label describing its purpose
    And every tab has an aria-label with the session name and status
    And every tool card has an aria-label with the tool name and result summary

  @a11y-desktop-focus
  Scenario: Focus order is logical and visible
    Given keyboard navigation is active
    When the user tabs through the interface
    Then focus moves: composer → send button → tab bar → transcript → status bar → composer
    And a visible focus ring is always present on the focused element
    And focus never disappears or jumps unexpectedly

  @a11y-desktop-announcements
  Scenario: Dynamic content changes are announced
    Given screen reader mode is enabled
    When a new assistant message arrives
    Then an aria-live region announces: "New message received"
    When a tool completes
    Then an aria-live region announces: "Tool 'bash' completed successfully"
    When a subagent finishes
    Then an aria-live region announces: "Subagent 'auth-review' completed"

  # ── Keyboard-Only Navigation ──────────────────────────────────────

  @a11y-keyboard-full
  Scenario: Every action has a keyboard equivalent
    Given the user navigates with keyboard only
    When the user needs to perform any action
    Then a keyboard shortcut exists
    And the shortcut is documented in the shortcuts cheatsheet (/shortcuts)
    And no action requires a mouse

  @a11y-keyboard-full
  Scenario: Modal traps focus correctly
    Given the approval modal is open
    When the user presses Tab at the last focusable element
    Then focus wraps to the first focusable element in the modal
    And focus does NOT escape to elements behind the modal
    And pressing Escape closes the modal and restores focus

  @a11y-keyboard-full
  Scenario: Skip link for rapid navigation
    Given the desktop app has rendered
    When the user presses Tab on the first element
    Then a "Skip to transcript" link appears
    And activating it moves focus directly to the transcript
    And a "Skip to composer" link moves focus to the composer

  # ── Text Size & Display ───────────────────────────────────────────

  @a11y-text-size
  Scenario: Text scale configurable
    Given the user sets text_scale = 1.5 in config
    When the UI renders
    Then all text is scaled by 1.5×
    And layout adapts to accommodate larger text
    And no text is clipped or overlapping

  @a11y-text-size
  Scenario: Minimum contrast ratio enforced
    Given any theme is active
    When text renders
    Then the contrast ratio between text and background is ≥ 4.5:1 for normal text
    And ≥ 3:1 for large text
    And the high-contrast theme meets ≥ 7:1

  # ── Reduced Motion ────────────────────────────────────────────────

  @a11y-reduced-motion
  Scenario: Animations disabled in reduced motion mode
    Given reduced_motion is enabled
    When any animation would play
    Then the transition is instant
    And no easing, sliding, or fading occurs
    And spinners are replaced with static text indicators

  @a11y-reduced-motion
  Scenario: Auto-detection of system preference
    Given reduced_motion is set to "auto"
    And the OS has "Reduce motion" enabled
    When the UI renders
    Then animations are disabled

  # ── Audio Cues ────────────────────────────────────────────────────

  @a11y-audio
  Scenario: Audio cues for status changes
    Given audio_cues are enabled
    When a turn completes
    Then a distinct completion sound plays
    When an approval is requested
    Then a distinct alert sound plays
    When a workflow fails
    Then a distinct error sound plays
    And the sounds are configurable or disableable

  # ── UI ─────────────────────────────────────────────────────────────

  @a11y-ui
  Scenario: Accessibility settings panel
    When the user opens Settings → Accessibility
    Then options are shown: Screen reader mode, Text scale, Reduced motion, Audio cues, Focus style, High contrast
    And each option has a description

  @a11y-ui
  Scenario: Accessibility mode persisted across sessions
    Given the user enabled screen_reader_mode
    When reasonix restarts
    Then screen_reader_mode is still enabled
    And all ARIA labels and announcements are active from the first frame
