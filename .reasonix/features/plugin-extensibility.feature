Feature: Plugin API & Extensibility
  As a developer extending reasonix
  I want to register custom subagent roles, workflow stages, sandbox engines, and providers
  So that reasonix stays config-driven and plugin-driven as the Spec requires

  Background:
    Given reasonix has plugin infrastructure via MCP and init() self-registration

  # ── Subagent Role Registration ────────────────────────────────────

  @plugin-role-register
  Scenario: Plugin registers a custom subagent role
    Given a plugin defines a "custom-auditor" role with a system prompt and tool scope
    When the plugin registers via reasonix's role registry
    Then "custom-auditor" appears in the workflow tool's role enum
    And subagents spawned with role "custom-auditor" receive the plugin's system prompt
    And the tool scope matches the plugin's declaration

  @plugin-role-register
  Scenario: Custom role appears in role picker UI
    Given a custom role "custom-auditor" is registered
    When the user configures a workflow stage
    Then "custom-auditor" appears in the role dropdown alongside built-in roles
    And the role shows the plugin's description

  @plugin-role-register
  Scenario: Custom role tool scope enforcement
    Given a custom role declares allowed_tools: ["read_file", "grep", "bash"]
    When a subagent with that role calls write_file
    Then the tool is not in the subagent's registry
    And the model cannot call it

  # ── Workflow Stage Registration ───────────────────────────────────

  @plugin-stage-register
  Scenario: Plugin registers a custom workflow stage handler
    Given a plugin defines a "consensus" stage type that requires 4/5 agreement
    When the plugin registers via the stage registry
    Then "consensus" appears as a valid mode in workflow stage config
    And the stage handler receives items and returns results in the standard format
    And the handler has access to the TaskTool for spawning subagents

  # ── Sandbox Engine Registration ───────────────────────────────────

  @plugin-sandbox-register
  Scenario: Plugin registers a custom sandbox engine
    Given a plugin provides a Podman-based sandbox engine
    When the plugin registers via sandbox.Register("podman", factory)
    Then [sandbox].engine = "podman" activates the plugin's engine
    And the engine receives sandbox.Spec and returns an exec.Cmd
    And the engine is used for all bash tool calls

  # ── Provider Registration ─────────────────────────────────────────

  @plugin-provider-register
  Scenario: Plugin registers a custom provider kind
    Given a plugin provides a "local-llm" provider via provider.Register("local-llm", factory)
    When a config entry has kind = "local-llm"
    Then the provider is constructed via the plugin's factory
    And the provider appears in the model switcher
    And the provider is used for subagent spawning

  @plugin-provider-register
  Scenario: Provider plugin validates config at construction
    Given a "local-llm" provider requires base_url and model_path
    And the config is missing model_path
    When the provider is constructed
    Then the factory returns an error: "local-llm: model_path is required"
    And the error surfaces at boot time, not at request time

  # ── Capability Discovery ──────────────────────────────────────────

  @plugin-capability
  Scenario: Capabilities panel shows loaded plugins
    When the user opens the capabilities panel
    Then all loaded plugins are listed with: name, version, kind, and registered capabilities
    And MCP servers show their tool count and connection status
    And custom providers show their model list

  @plugin-capability
  Scenario: Plugin load failure does not crash reasonix
    Given a plugin fails to load (invalid binary, missing dependency)
    When reasonix boots
    Then a warning is emitted: "Plugin 'custom-auditor' failed to load: exec format error"
    And reasonix continues with built-in roles only
    And the failed plugin is listed in the capabilities panel as "error"

  # ── Contract Stability ────────────────────────────────────────────

  @plugin-contract
  Scenario: Plugin interface is versioned
    Given a plugin was compiled against the v1 role registration interface
    When the interface changes to v2 in a backward-incompatible way
    Then v1 plugins are detected at load time
    And a warning is emitted: "Plugin 'custom-auditor' targets API v1 but reasonix requires v2"
    And the plugin is not loaded

  @plugin-contract
  Scenario: Plugin API changelog is generated
    Given the plugin API surface has changed between versions
    When the changelog is generated
    Then it lists: added interfaces, deprecated interfaces, removed interfaces
    And migration notes are included for each change
