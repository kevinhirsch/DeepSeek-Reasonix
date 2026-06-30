Feature: Monitoring, Scheduling & Notifications
  As a developer running long tasks
  I want to monitor commands, schedule recurring work, and get notified
  So that I can walk away and be alerted when something needs attention

  Background:
    Given reasonix has Monitor, schedule_task, schedule_wakeup, and send_notification tools

  # ── Monitor Tool ──────────────────────────────────────────────────

  @gap-monitor-basic
  Scenario: Monitor streams command output as events
    When the agent calls monitor with command "tail -f build.log" and description "Build progress"
    Then the command starts in the background
    And each stdout line is emitted as a monitor event
    And the event carries the description "Build progress"

  @gap-monitor-filter
  Scenario: Monitor filters output with regex
    When the agent calls monitor with command "tail -f app.log" and filter "ERROR|FATAL"
    Then only lines matching the regex are emitted as events
    And non-matching lines are silently discarded

  @gap-monitor-timeout
  Scenario: Monitor kills after timeout
    When the agent calls monitor with timeout_ms: 5000
    And the command runs longer than 5 seconds
    Then the monitor kills the command
    And a final event is emitted: "Monitor 'Build progress' stopped (timeout)"

  @gap-monitor-exit
  Scenario: Monitor stops when command exits naturally
    When the agent calls monitor with command "echo done && sleep 0.1"
    Then the monitor emits each output line
    And the monitor emits a final event: "Monitor stopped (command exited with code 0)"
    And no goroutine leak remains

  @gap-monitor-cancel
  Scenario: Monitor stops on context cancellation
    Given a monitor is running
    When the parent context is cancelled
    Then the command is killed
    And the monitor channel is closed
    And no orphaned subprocess remains

  # ── Schedule Tools ────────────────────────────────────────────────

  @gap-schedule-one-shot
  Scenario: One-shot wakeup fires at the specified time
    When the agent calls schedule_wakeup with delay_seconds: 120 and prompt "Check if the deploy finished"
    Then the wakeup is scheduled
    And after 120 seconds, the prompt "Check if the deploy finished" is enqueued
    And after firing, the wakeup is auto-deleted

  @gap-schedule-recurring
  Scenario: Recurring cron job fires on schedule
    When the agent calls schedule_task with cron "*/5 * * * *" and prompt "Run health check" and recurring: true
    Then the job fires every 5 minutes
    And the prompt "Run health check" is enqueued each time
    And the job auto-expires after 7 days

  @gap-schedule-list
  Scenario: List all scheduled jobs
    Given 3 cron jobs are scheduled
    When the agent calls schedule_task with action "list"
    Then all 3 jobs are returned with their IDs, cron expressions, and prompts

  @gap-schedule-delete
  Scenario: Delete a scheduled job
    Given a cron job with ID "cron_abc123"
    When the agent calls schedule_task with action "delete" and id "cron_abc123"
    Then the job is removed
    And schedule_task action "list" no longer includes "cron_abc123"

  @gap-schedule-list-ui
  Scenario: Scheduled tasks panel renders job list with next-fire times
    Given 3 cron jobs: "health-check" (runs every 5m, next at 14:35), "nightly-report" (runs at 2am, next at 02:00), "expiring-soon" (runs hourly, expires in 2h)
    When the scheduled tasks panel renders
    Then all 3 jobs are listed with: name, cron description, next fire time, status
    And "expiring-soon" shows a warning: "⏳ Expires in 2 hours" in yellow
    And each job row shows the last fire time if it has fired

  @gap-schedule-list-ui
  Scenario: Scheduled tasks panel shows empty state
    Given no cron jobs exist
    When the scheduled tasks panel renders
    Then a message is shown: "No scheduled tasks. Create one with /loop or schedule_task."
    And a "Create Task" button is shown

  @gap-schedule-delete-ui
  Scenario: Deleting a scheduled job shows confirmation
    Given the scheduled tasks panel shows "health-check"
    When the user clicks "Delete" on the health-check job
    Then a confirmation dialog appears: "Delete 'health-check'? This job fires every 5 minutes. This cannot be undone."
    And "Delete" and "Cancel" buttons are shown
    And confirming removes the job from the list
    And the panel shows a brief toast: "✓ health-check deleted"

  @gap-schedule-wakeup-ui
  Scenario: Wakeup countdown shown in status line
    Given a schedule_wakeup is set for 120 seconds from now
    When the status line renders
    Then a countdown appears: "⏰ Next wakeup: 1m 58s"
    And the countdown updates every second
    And at 0, the wakeup fires and the countdown disappears

  @gap-schedule-wakeup-ui
  Scenario: Multiple wakeups stack in status line
    Given 2 schedule_wakeups are active: one at 60s, one at 300s
    When the status line renders
    Then both wakeups are shown: "⏰ Wakeups: 58s, 4m 58s"
    And each countdown updates live
    And fired wakeups are removed

  @gap-schedule-wakeup-ui
  Scenario: Wakeup fires with notification
    Given a schedule_wakeup fires after its countdown
    When the wakeup prompt is enqueued
    Then a brief notification appears: "⏰ Wakeup: 'Check deploy status'"
    And the notification auto-dismisses after 5 seconds
    And the prompt is processed by the agent

  @gap-schedule-off-minute
  Scenario: Recurring cron avoids :00 and :30 minute marks
    When the agent schedules a recurring task with approximate timing "every morning at 9"
    Then the cron expression uses a minute that is NOT 0 or 30 (e.g. 57 8 or 3 9)
    And a note explains that off-minute scheduling avoids thundering-herd

  @gap-schedule-session-only
  Scenario: Scheduled jobs do not survive session restart (default)
    Given a cron job was created with durable: false (default)
    When the session exits and a new session starts
    Then the job no longer exists

  @gap-schedule-durable
  Scenario: Durable jobs survive session restart
    Given a cron job was created with durable: true
    When the session exits and a new session starts
    Then the job is reloaded from .reasonix/scheduled_tasks.json
    And the job continues firing on its schedule

  # ── Push Notifications ────────────────────────────────────────────

  @gap-notify-turn-done
  Scenario: Notification fires when turn completes
    Given notifications are enabled with turn_done: true
    When an agent turn completes
    Then a desktop notification is sent
    And the notification body includes the outcome summary

  @gap-notify-approval
  Scenario: Notification fires when approval is pending
    Given notifications are enabled with approval_request: true
    When a tool call requires approval
    Then a desktop notification is sent: "reasonix: tool approval needed"
    And the notification body names the tool

  @gap-notify-proactive
  Scenario: Agent sends proactive notification via tool
    When the agent calls send_notification with message "Build complete — 3 tests failed" and status "proactive"
    Then a desktop notification is sent with the message
    And if remote push is configured, a mobile notification is also sent

  @gap-notify-disabled
  Scenario: Notifications suppressed when disabled
    Given notifications are disabled (enabled: false)
    When a turn completes
    Then no desktop notification is sent

  @gap-notify-platform-fallback
  Scenario: Notification falls back gracefully on unsupported platform
    Given the platform has no notification system (headless server)
    When a notification would be sent
    Then the notification is silently dropped
    And no error is raised
