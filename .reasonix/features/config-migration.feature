Feature: Configuration Migration & Compatibility
  As a user upgrading reasonix
  I want my config to keep working across versions without manual migration
  So that upgrades are safe and backward compatibility is guaranteed

  Background:
    Given reasonix has config_version tracking and automated migration

  # ── Version Bump Migration ────────────────────────────────────────

  @config-migration-version-bump
  Scenario: Config version bump triggers migration
    Given a user has config_version = 1 in their reasonix.toml
    And reasonix v2.0 requires config_version = 2
    When reasonix v2.0 loads the config
    Then the migration from v1 to v2 runs automatically
    And the config version is updated to 2
    And a notice is emitted: "Config migrated from v1 to v2. Backup saved to reasonix.toml.v1.bak"

  @config-migration-version-bump
  Scenario: Migration creates backup before modifying
    Given a migration is about to run
    When the migration starts
    Then the original config file is copied to reasonix.toml.v<N>.bak
    And the backup has the same permissions as the original
    And the backup is created before any modification

  @config-migration-version-bump
  Scenario: Migration failure rolls back
    Given a migration encounters an unrecoverable error
    When the migration fails
    Then the original config is NOT modified
    And the backup is preserved
    And an error is shown: "Config migration failed. Original config unchanged. See reasonix.toml.v1.bak."
    And reasonix continues with the original config and a compatibility warning

  # ── New Config Sections ────────────────────────────────────────────

  @config-migration-new-sections
  Scenario: New config sections get sensible defaults
    Given reasonix v2.0 adds [agent.subagent_models] section
    And the user's v1 config has no such section
    When the config is loaded
    Then [agent.subagent_models] is populated with built-in defaults
    And the defaults match the documented recommendations
    And the user is NOT required to add the section manually

  @config-migration-new-sections
  Scenario: New required fields get prompted defaults
    Given reasonix v2.0 adds a required field to [[remotes]]
    And the user's v1 config has [[remotes]] entries without that field
    When the config is loaded
    Then the field is populated with a safe default
    And a notice is emitted: "Added default 'timeout: 30m' to remote 'build-server'. Edit reasonix.toml to change."

  # ── Deprecated Fields ──────────────────────────────────────────────

  @config-migration-deprecated
  Scenario: Deprecated field shows warning
    Given reasonix v2.0 deprecates [agent].old_effort_field in favor of [agent].effort
    And the user's config still has old_effort_field
    When the config is loaded
    Then a warning is emitted: "[agent].old_effort_field is deprecated. Use [agent].effort instead."
    And the deprecated field's value is used if the new field is not set
    And the deprecated value is migrated to the new field

  @config-migration-deprecated
  Scenario: Removed field shows error with migration suggestion
    Given reasonix v2.0 removed a field that existed in v1
    And the user's config has the removed field
    When the config is loaded
    Then an error is emitted: "[agent].removed_field is no longer supported. Use [agent].new_field instead."
    And the error message includes the exact replacement

  # ── Forward Compatibility ─────────────────────────────────────────

  @config-migration-forward
  Scenario: Unknown fields warn but don't error
    Given the user has a field [future_section].experimental = true
    And reasonix doesn't recognize that section
    When the config is loaded
    Then a warning is emitted: "Unknown config section [future_section] — may be from a newer version"
    And the field is preserved in the config (not deleted)
    And reasonix continues to load

  # ── Config Validation ─────────────────────────────────────────────

  @config-migration-validate
  Scenario: reasonix doctor validates entire config
    When the user runs "reasonix doctor"
    Then every config section is validated
    And errors are reported with file path, line number, and fix suggestion
    And warnings are reported for deprecated fields
    And a summary shows: "0 errors, 2 warnings"

  @config-migration-validate
  Scenario: reasonix --dry-run validates config without starting agent
    When the user runs "reasonix --dry-run"
    Then the config is loaded and validated
    And provider connections are tested (API key validity)
    And a report is printed: "Config valid. 3 providers configured. Health: DeepSeek Flash ✓, DeepSeek Pro ✓, Anthropic ✓."
    And no agent loop starts

  # ── UI ─────────────────────────────────────────────────────────────

  @config-migration-ui
  Scenario: Migration notice renders with details
    Given a config migration ran on startup
    When the notice renders
    Then it shows: what was migrated, what the new values are, and a "View diff" link
    And the diff shows old config vs new config side by side

  @config-migration-ui
  Scenario: Doctor results render as structured report
    When "reasonix doctor" runs
    Then the output has sections: Configuration, Providers, Sandbox, Permissions, Skills, Plugins
    And each section has a status: ✓ OK, ⚠ Warnings (N), ✗ Errors (N)
    And errors and warnings are clickable to show detail
