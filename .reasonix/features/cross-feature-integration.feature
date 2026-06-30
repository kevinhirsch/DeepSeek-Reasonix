Feature: Cross-Feature Integration
  As a developer relying on the full subagent architecture
  I want cross-feature interactions to be well-defined and tested
  So that composing features doesn't produce emergent bugs

  # ── Workflow + Messenger integration ─────────────────────────────

  @integration-workflow-messenger
  Scenario: Parent steers a specific subagent within a running workflow
    Given a workflow is running with 3 finders and 9 verifiers
    And finder "finder-1" has returned a finding about auth.go:42
    When the parent sends "Also check the OAuth callback at auth.go:108" to subagent "finder-1"
    Then only "finder-1" receives the steer message
    And the other 2 finders continue unaffected
    And the verifiers spawned from "finder-1" results receive the additional context

  @integration-workflow-messenger
  Scenario: Messenger survives workflow stage transitions
    Given a workflow stage "find" has completed and all its subagents are unregistered
    And stage "verify" has spawned 9 verifiers registered with the messenger
    When the parent sends "Be stricter about false positives" to all verify subagents
    Then all 9 verifiers receive the steer
    And no finder subagents (already unregistered) receive the steer

  # ── Workflow + Remote integration ────────────────────────────────

  @integration-workflow-remote
  Scenario: Workflow fans out to local and remote workers
    Given 3 finders in a workflow stage
    And finder-1 targets local, finder-2 targets "build-server", finder-3 targets local
    When the workflow stage executes
    Then finder-1 and finder-3 run in local sandboxes
    And finder-2 is queued on "build-server"
    And the background panel shows 🌐 for finder-2 and ● for finder-1 and finder-3
    And all three results are collected before the next stage begins (barrier mode)

  @integration-workflow-remote
  Scenario: Remote worker offline mid-workflow
    Given a workflow stage targets "build-server" for all finders
    And "build-server" goes offline after finder-1's work item is queued but before finder-2's
    When finder-1 completes on the remote worker
    And finder-2's work item cannot be delivered
    Then finder-2 fails with "remote 'build-server' is not reachable"
    And finder-3 (if any) does not attempt to queue on the offline remote
    And the workflow records finder-2 as failed but continues with finder-1's result

  # ── Messenger + Remote integration ────────────────────────────────

  @integration-messenger-remote
  Scenario: Steering a remote subagent
    Given a subagent is running on remote "build-server"
    And the subagent is registered in the local messenger
    When the parent sends "Skip the integration test, it's flaky" via send_to_subagent
    Then the message is relayed to the remote worker
    And the remote worker injects it into the subagent's steer queue
    And the subagent sees it on its next tool-call round

  # ── Background Panel + Workflow integration ───────────────────────

  @integration-panel-workflow
  Scenario: Panel shows workflow hierarchy
    Given a workflow "security-audit" is running with stages: find(3), verify(9), report(1)
    And 3 finders are running, 2 verifiers are running (findings flowing through pipeline)
    When the background panel renders
    Then the panel groups subagents by stage: "find (3 running)", "verify (2 running, 7 pending)"
    And the report stage shows "pending — depends on: verify:*"

  @integration-panel-workflow
  Scenario: Panel updates live as pipeline progresses
    Given a pipeline workflow with stage "find" running 3 subagents
    When finder-1 completes
    And its findings flow into the verify stage spawning 3 verifiers
    Then the panel shows "find (1 running, 1 done)" and "verify (3 running)"
    And the just-completed finder shows ✓ with its finding count

  # ── Role Prompts + Workflow integration ───────────────────────────

  @integration-prompts-workflow
  Scenario: Workflow stage role maps to correct subagent prompt
    Given a workflow stage specifies role: "verify"
    When the stage spawns subagents
    Then each subagent receives the DefaultVerifierPrompt
    And the prompt is injected via user message if provider is DeepSeek
    And the DeepSeekCommonNotes are appended

  @integration-prompts-workflow
  Scenario: Workflow stage with unknown role falls back to DefaultTaskSystemPrompt
    Given a workflow stage specifies role: "custom_unknown_role"
    When the stage spawns subagents
    Then each subagent receives DefaultTaskSystemPrompt
    And a warning notice is emitted: "unknown subagent role 'custom_unknown_role', using default prompt"

  # ── Effort Calibration + Workflow integration ─────────────────────

  @integration-effort-workflow
  Scenario: Workflow stage effort overrides calibration default
    Given the effort calibration table says explore should use "high"
    But a workflow stage specifies effort: "max" for an explore role
    When the stage spawns subagents
    Then each subagent runs at effort "max"
    And the stage-level override wins over the calibration default

  @integration-effort-workflow
  Scenario: Workflow stage inherits calibration default when effort omitted
    Given the effort calibration table says verify should use "max"
    And a workflow stage specifies role: "verify" with no effort field
    When the stage spawns subagents
    Then each subagent runs at effort "max"

  # ── Cognitive Loop Detector + Workflow integration ────────────────

  @integration-loop-workflow
  Scenario: Cognitive loop in one workflow subagent doesn't affect siblings
    Given a workflow has 3 finders running concurrently
    And finder-2 enters a cognitive loop
    When the loop detector fires on finder-2
    Then finder-1 and finder-3 continue unaffected
    And finder-2 receives the breaker message
    And if finder-2 breaks the loop, the workflow continues normally
    And if finder-2 fails, it's marked failed but the workflow continues with finder-1 and finder-3 results

  # ── Temperature Override + Remote integration ─────────────────────

  @integration-temp-remote
  Scenario: Remote worker applies DeepSeek temperature override
    Given a subagent targets remote "build-server"
    And the remote worker is configured with a DeepSeek provider
    And the subagent spec has temperature: 0.0
    When the remote worker constructs the subagent
    Then the effective temperature is 0.6
    And the worker emits an info notice about the override

  # ── Nested Workflow integration ───────────────────────────────────

  @integration-nested-workflow
  Scenario: Workflow subagent spawns its own subagent via task tool
    Given a workflow stage "report" spawns an executor subagent
    And the executor subagent decides to spawn its own research subagent
    When the nested subagent runs
    Then it receives the subagent boundary (no recursive task tool access)
    And its events are nested under the executor's tool call
    And the parent workflow's event stream shows: workflow → report → executor → research
    And the nested subagent cannot spawn further subagents (one level of delegation)

  @integration-nested-workflow
  Scenario: Nested subagent tool scope excludes meta tools
    Given an executor subagent has access to the task tool
    When it spawns a research subagent
    Then the research subagent's tool registry excludes: task, parallel_tasks, read_only_task, workflow
    And the bash tool is wrapped to foreground-only
    And the subagent cannot spawn further agents
