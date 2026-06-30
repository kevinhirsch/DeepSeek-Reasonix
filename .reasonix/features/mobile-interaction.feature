Feature: Mobile & Off-Device Interaction
  As a developer who walks away from their desk
  I want to check workflow status and approve actions from my phone
  So that long-running workflows don't stall waiting for my input

  Background:
    Given reasonix has desktop notifications and mobile webhook infrastructure

  # ── Mobile Web View ───────────────────────────────────────────────

  @mobile-web-view
  Scenario: Mobile web view shows active workflows
    Given reasonix serve is running on port 9090
    When the user navigates to https://host:9090/m on a mobile browser
    Then a responsive mobile view renders
    And the view shows: active workflows with progress, pending approvals, running subagent count
    And the view refreshes automatically via SSE

  @mobile-web-view
  Scenario: Mobile web view shows subagent detail
    Given a workflow has 3 finders running
    When the user taps the workflow
    Then a detail view expands showing: each finder's status, tool count, and reasoning tail (last 100 chars)
    And the view is scrollable

  # ── Mobile Approvals ──────────────────────────────────────────────

  @mobile-approval
  Scenario: Approve tool from mobile
    Given a tool approval is pending in the desktop app
    And the user is viewing the mobile web view
    When the mobile view renders
    Then the pending approval is shown prominently: tool name, arguments, risk level
    And "Approve" and "Deny" buttons are large and touch-friendly
    And tapping "Approve" approves the tool immediately

  @mobile-approval
  Scenario: Deny with reason from mobile
    Given a tool approval is pending
    When the user taps "Deny"
    Then an optional reason text field appears
    And the user can provide a reason
    And tapping "Deny" again sends the denial with the reason
    And the agent receives the reason

  # ── Mobile Push Notifications ──────────────────────────────────────

  @mobile-push
  Scenario: Push notification on workflow completion
    Given the user configured a webhook for push notifications
    And a 30-minute workflow completes
    When the workflow finishes
    Then a push notification is sent: "Workflow 'security-audit' complete — 3 findings confirmed"
    And the notification includes: workflow name, duration, finding count, cost
    And tapping the notification opens the mobile web view

  @mobile-push
  Scenario: Push notification on approval request
    Given a tool requires approval
    And the user has been away for >2 minutes
    When the approval has been pending for 30 seconds
    Then a push notification is sent: "reasonix: 'bash rm -rf' needs approval"
    And the notification has "Approve" and "Deny" quick actions

  @mobile-push
  Scenario: Push notification on error
    Given a workflow failed
    When the failure is detected
    Then a push notification is sent: "Workflow 'deploy-check' failed — DeepSeek API rate limited"
    And the notification includes the error summary

  # ── Security ──────────────────────────────────────────────────────

  @mobile-security
  Scenario: Mobile web view requires authentication
    Given reasonix serve has auth enabled
    When the user navigates to /m without authenticating
    Then a login screen is shown
    And the mobile view is not accessible without valid credentials

  @mobile-security
  Scenario: Mobile approval requires re-authentication for sensitive actions
    Given the mobile web view is authenticated
    When the user approves a high-risk tool (bash rm -rf, permission changes, env modifications)
    Then re-authentication is required
    And the approval is only processed after credential verification

  # ── UI ─────────────────────────────────────────────────────────────

  @mobile-ui
  Scenario: Mobile view adapts to screen size
    Given the mobile browser is 375px wide (iPhone)
    When the mobile view renders
    Then the layout is single-column
    And text is readable without zooming
    And buttons are at least 44×44px (minimum touch target)

  @mobile-ui
  Scenario: Mobile view shows battery-friendly mode
    Given the user is on mobile data
    When the mobile view renders
    Then auto-refresh interval is reduced to 5 seconds instead of 1 second
    And a "Battery saver" toggle is available
