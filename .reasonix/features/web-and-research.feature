Feature: Web Search & Research Tools
  As a developer needing current information
  I want web search and deep research capabilities
  So that the agent can find answers beyond its training data

  Background:
    Given reasonix has web_search and web_fetch tools registered
    And the deep-research skill is available

  # ── web_search tool ───────────────────────────────────────────────

  @gap-websearch-basic
  Scenario: Simple web search returns results
    When the agent calls web_search with query "golang context cancellation best practices 2026"
    Then the result contains at least one result block
    And each result has a title, URL, and snippet
    And the results are relevant to the query

  @gap-websearch-domains
  Scenario: Domain filtering restricts results
    When the agent calls web_search with query "error handling" and allowed_domains: ["golang.org", "pkg.go.dev"]
    Then all returned URLs are from golang.org or pkg.go.dev
    And no results from other domains appear

  @gap-websearch-max-results
  Scenario: Max results cap is respected
    When the agent calls web_search with query "test" and max_results: 3
    Then at most 3 results are returned

  @gap-websearch-empty
  Scenario: Empty search returns error
    When the agent calls web_search with query ""
    Then the tool returns error: "query is required"

  @gap-websearch-network-error
  Scenario: Network failure returns error
    Given the search backend is unreachable
    When the agent calls web_search
    Then the tool returns error: "search failed: network error"
    And the error includes the underlying cause

  # ── deep-research skill ───────────────────────────────────────────

  @gap-deep-research
  Scenario: Deep research decomposes question into angles
    Given the deep-research skill is invoked with question "What are the best Go libraries for building LLM agents?"
    When the skill runs
    Then the parent agent decomposes the question into at least 3 search angles
    And a workflow is composed with one finder per angle
    And each finder uses web_search + web_fetch

  @gap-deep-research
  Scenario: Deep research extracts claims from sources
    Given web_fetch has returned 5 source pages
    When the claim extraction phase runs
    Then each source produces at least one falsifiable claim
    And each claim includes: the claim text, the source URL, and a confidence level

  @gap-deep-research
  Scenario: Deep research adversarially verifies claims
    Given 10 claims have been extracted from sources
    When the verification phase runs
    Then each claim is verified by 3 independent skeptic subagents
    And claims with ≥2 refutations are dropped
    And surviving claims are marked CONFIRMED

  @gap-deep-research
  Scenario: Deep research synthesizes cited report
    Given 7 claims survived verification
    When the synthesis phase runs
    Then a single subagent composes a report
    And every claim in the report cites its source URL
    And the report has sections: Summary, Findings, Sources, Confidence

  @gap-deep-research
  Scenario: Deep research fails gracefully on no results
    Given web_search returned zero results for the query
    When the deep-research skill runs
    Then the skill reports "No results found for any search angle"
    And the report includes the search queries that were attempted

  @gap-deep-research
  Scenario: Deep research respects max_total_tasks
    Given max_total_tasks is set to 20
    And the deep research would spawn 30 subagents
    When the workflow runs
    Then at most 20 subagents are spawned
    And the report notes what was skipped

  # ── FE: Search & Research Rendering ───────────────────────────────

  @gap-fe-search-results
  Scenario: Web search results render as clickable cards
    Given web_search returned 5 results for "golang context cancellation"
    When the results render in the transcript
    Then each result is a card with: title (bold, clickable URL), URL (dimmed), snippet (2-3 lines)
    And clicking a title opens the URL in the default browser
    And the cards are separated by subtle dividers

  @gap-fe-search-empty
  Scenario: Empty search renders "no results" state
    Given web_search returned zero results
    When the result renders
    Then a card shows: "No results found for '<query>'"
    And the card suggests: "Try different search terms or check your spelling"
    And a "Search web manually" link opens the query in the default browser

  @gap-fe-deep-research-progress
  Scenario: Deep research progress renders phase-by-phase
    Given a deep-research is running with 5 phases
    When the progress renders in the transcript
    Then it shows: "🔍 Researching: Phase 1/5 — Decomposing question..."
    And the phase description updates as phases progress
    And each completed phase shows ✓ with a duration

  @gap-fe-deep-research-report
  Scenario: Deep research final report renders as structured document
    Given a deep-research completed with 7 confirmed claims
    When the report renders
    Then it has sections with headers: Summary, Findings, Sources, Confidence
    And each finding shows: claim text, source citation (clickable URL), confidence level (colored tag)
    And a "Copy as Markdown" button is available
    And a "Copy as PR comment" button pre-formats the findings

  @gap-fe-deep-research-sources
  Scenario: Source list renders with verification status per source
    Given 5 sources were fetched and 3 claims verified
    When the sources section renders
    Then each source shows: URL, fetch status (✓ fetched, ✗ failed), claims extracted, claims verified
    And clicking a source expands its extracted claims
