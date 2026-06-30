Feature: Documentation Generation
  As a developer using reasonix
  I want comprehensive, versioned documentation that stays in sync with features
  So that I can learn the tool and reference its capabilities without leaving the terminal

  Background:
    Given reasonix has a docs/ directory and in-app help system

  # ── In-App Help ───────────────────────────────────────────────────

  @docs-in-app
  Scenario: /help shows top-level help with all topics
    When the user types "/help"
    Then a help panel opens listing all documentation topics
    And topics are grouped: Getting Started, Subagents, Workflows, Remote, Configuration, Reference
    And each topic has a one-line description

  @docs-in-app
  Scenario: /help <topic> shows specific help
    When the user types "/help workflows"
    Then the help panel shows the workflows documentation
    And the content includes: overview, strategy descriptions (pipeline/barrier/loop-until-dry), stage configuration, examples
    And code examples are syntax-highlighted

  @docs-in-app
  Scenario: /help <topic> with unknown topic suggests closest match
    When the user types "/help workflow" (singular, not plural)
    Then the help panel shows: "No topic 'workflow'. Did you mean 'workflows'?"
    And the suggested topic is clickable

  @docs-in-app
  Scenario: /help outputs are the same content as docs/
    Given docs/WORKFLOWS.md exists
    When /help workflows renders
    Then the content matches docs/WORKFLOWS.md
    And the rendering is markdown

  # ── Tool & Skill Help ─────────────────────────────────────────────

  @docs-tool-help
  Scenario: Tool descriptions accessible via /help tools/<name>
    When the user types "/help tools/workflow"
    Then the help panel shows: tool name, description, full parameter schema, examples
    And the parameter schema shows types, defaults, and constraints

  @docs-skill-help
  Scenario: Skill help shows system prompt and tool scope
    When the user types "/help skills/explore"
    Then the help panel shows: skill name, description, execution mode (subagent), system prompt body, allowed tools list
    And a note explains: "This is the exact system prompt the explorer subagent receives"

  # ── Worked Examples ───────────────────────────────────────────────

  @docs-examples
  Scenario: /examples lists worked examples
    When the user types "/examples"
    Then a list of worked examples is shown with descriptions
    And examples cover: security audit, code review, dependency update, API migration, test coverage analysis

  @docs-examples
  Scenario: /examples <name> shows full walkthrough
    When the user types "/examples security-audit"
    Then a walkthrough renders showing: the initial prompt, the composed workflow, stage-by-stage output, the final report
    And each stage shows the tool calls made and their results
    And the expected cost is shown at the top: "This example costs ~$0.05 at current DeepSeek rates"

  # ── Documentation Versioning ──────────────────────────────────────

  @docs-version
  Scenario: /help reflects the installed version's features
    Given reasonix v2.1 has a feature not in v2.0
    When the user runs /help on v2.1
    Then the new feature is documented
    And deprecated features show a deprecation notice: "⚠ Deprecated in v2.1 — use X instead"

  @docs-version
  Scenario: Migration guide accessible in-app
    When the user types "/help migrating"
    Then the migration guide renders with sections per version: "v1.0 → v2.0", "v2.0 → v2.1"
    And each section lists: breaking changes, new features, config changes needed

  # ── Config Documentation ──────────────────────────────────────────

  @docs-config
  Scenario: Config reference accessible in-app
    When the user types "/help config"
    Then every config section is listed with its TOML path, description, type, and default value
    And deprecated fields are marked
    And an example reasonix.toml is shown with all sections

  @docs-config
  Scenario: Config doctor validates against documented schema
    When the user runs "reasonix doctor"
    Then every config field is validated against the documented schema
    And unknown fields are flagged: "Unknown config field: [agent].old_field_name"
    And deprecated fields show a migration suggestion

  # ── UI ─────────────────────────────────────────────────────────────

  @docs-ui
  Scenario: Help panel renders as overlay with search
    Given the help panel is open
    When the user types a search query
    Then all documentation sections matching the query are listed
    And results come from: built-in docs, skill bodies, tool descriptions, config reference
    And search is fuzzy and ranks by relevance

  @docs-ui
  Scenario: Help content is scrollable with anchor links
    Given a long help document with sections
    When the document renders
    Then a table of contents is shown in a sidebar
    And clicking a TOC entry scrolls to that section
    And the TOC highlights the current section

  @docs-ui
  Scenario: First-run help prompt
    Given the user has never opened /help
    When the user spawns their first subagent
    Then a one-time notice appears: "💡 Type /help subagents to learn about subagent roles and effort calibration."
    And the notice does not appear again
