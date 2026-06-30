Feature: Subagent Communication
  As a parent agent orchestrating subagents
  I want to send mid-task guidance to running subagents
  So that I can redirect work, provide new context, or ask follow-ups without restarting

  Background:
    Given reasonix has SubagentMessenger wired into task and parallel_tasks tools
    And a background subagent is running with ref "sa_abc123"

  @04-messenger
  Scenario: Parent sends guidance to running subagent by subagent ref
    When the parent agent calls send_to_subagent with subagent_id "sa_abc123" and message "Also check error handling in the caller"
    Then the subagent receives the message as a steer on its next tool-call round
    And the parent receives confirmation: "Message delivered to subagent 'sa_abc123'"
    And the message is wrapped with MidTurnSteerPrefix before injection

  @04-messenger
  Scenario: Parent sends guidance to running subagent by job ID
    Given the subagent has job ID "job_xyz789"
    When the parent agent calls send_to_subagent with subagent_id "job_xyz789" and message "Skip the quota check"
    Then the messenger resolves the job ID to the subagent ref
    And the message is delivered to the correct subagent

  @04-messenger
  Scenario: Messaging a completed subagent returns error
    Given the subagent "sa_abc123" has completed
    When the parent agent calls send_to_subagent with subagent_id "sa_abc123" and message "Add one more check"
    Then the tool returns an error: "subagent 'sa_abc123' not found or not running"

  @04-messenger
  Scenario: Concurrent subagent registration is thread-safe
    Given 10 parallel_tasks are spawning subagents simultaneously
    When each subagent registers with the messenger on start
    Then no data race occurs
    And all 10 subagents are reachable by their refs

  @04-messenger
  Scenario: Subagent unregistered on completion
    Given a background subagent "sa_abc123" is running
    When the subagent completes
    Then the subagent is removed from the messenger's registry
    And subsequent send_to_subagent calls for "sa_abc123" return an error

  # ── FE: Message Rendering ─────────────────────────────────────────

  @04-fe-send-input
  Scenario: Send message input bar renders with subagent name
    Given the background panel is visible and "auth-review" is selected
    When the user presses 's'
    Then an input bar appears at the panel bottom
    And the placeholder reads: "Message to auth-review..."
    And the bar has a character counter: "0/500"
    And the bar border pulses briefly to draw attention

  @04-fe-send-confirmation
  Scenario: Sent message shows confirmation with content preview
    Given the user sent "Also check the OAuth callback" to subagent "auth-review"
    When the confirmation renders
    Then a brief toast appears: "✓ Sent to auth-review: 'Also check the OAuth callback'"
    And the toast auto-dismisses after 3 seconds
    And the subagent's reasoning tail updates to show it received the steer

  @04-fe-send-rejection
  Scenario: Rejected send shows red flash on input bar
    Given the send message input is empty
    When the user presses Ctrl+J
    Then the input bar border flashes red for 200ms
    And a subtle shake animation plays
    And no message is dispatched

  @04-fe-steer-received
  Scenario: Subagent steer arrival shown in panel
    Given a message was sent to subagent "auth-review"
    And the subagent received the steer on its next tool-call round
    When the background panel renders
    Then the subagent's row shows a small "📨" indicator briefly (3 seconds)
    And the tooltip on hover shows: "Steer received: 'Also check...'"
    And the indicator disappears after the subagent acts on the message
