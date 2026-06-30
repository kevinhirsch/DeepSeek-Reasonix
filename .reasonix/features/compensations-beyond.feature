Feature: Beyond Structural — Continuous Background Intelligence
  As a developer using reasonix on DeepSeek
  I want always-on intelligence that compounds over time and pre-computes answers before I ask
  So that reasonix gets smarter the longer I use it, at a cost Anthropic cannot economically match

  Background:
    Given reasonix has compensations 11-18 enabled
    And DeepSeek Flash costs $0.14/M input

  # ── Comp 11: Ambient Code Guardian ───────────────────────────────

  @comp-ambient-guardian
  Scenario: Guardian detects nil dereference on save
    Given the Ambient Guardian is active
    When the user saves a file adding `config.SomeField` without a nil check
    And config could be nil on first call
    Then within 500ms of save, a notice appears: "⚠ Guardian: possible nil dereference in auth.go:42 — config is nil on first call"
    And the notice includes the file path and line number
    And the notice has a "Show me" button that opens the file at that line

  @comp-ambient-guardian
  Scenario: Guardian is silent on safe changes
    Given the Ambient Guardian is active
    When the user saves a file with a comment-only change
    Then no notice appears (false positive rate < 5%)

  @comp-ambient-guardian
  Scenario: Guardian detects security issue
    Given the Ambient Guardian is active
    When the user saves a file adding raw SQL concatenation with user input
    Then a notice appears: "⚠ Guardian: possible SQL injection in db.go:108 — user input concatenated into query"

  @comp-ambient-guardian
  Scenario: Guardian respects do-not-disturb mode
    Given the user has enabled do-not-disturb
    When the Guardian detects an issue
    Then the notice is queued, not shown immediately
    And it appears when do-not-disturb is disabled

  @comp-ambient-guardian
  Scenario: Guardian cost appears in status line
    Given the Guardian has fired 47 times today at total cost $0.014
    When the status line renders
    Then it shows: "🛡️ Guardian: 47 checks · $0.014 today"
    And the indicator is dimmed (ambient, not attention-grabbing)

  # ── Comp 12: Idle-Time Pre-Computation ───────────────────────────

  @comp-idle-prefetch
  Scenario: Idle exploration pre-computes answers
    Given reasonix has been idle for 2 minutes
    When the idle timer fires
    Then an explorer subagent spawns to understand the most-referenced unexplored module
    And the result is stored in the local knowledge base
    And the status line shows: "💤 Pre-computing: understanding auth module..."
    And on the next keystroke, pre-computation pauses immediately

  @comp-idle-prefetch
  Scenario: Pre-computed answer serves instantly
    Given the knowledge base has a pre-computed answer for "How does auth middleware work?"
    When the user asks "How does auth middleware work?"
    Then the answer is retrieved from the knowledge base in <100ms
    And no new subagent is spawned
    And the response notes: "⚡ Instant answer — pre-computed 5 minutes ago during idle time"

  @comp-idle-prefetch
  Scenario: Pre-computed answer is stale
    Given the knowledge base has a pre-computed answer from 3 days ago
    And auth middleware was modified yesterday (per git log)
    When the user asks about auth middleware
    Then the stale answer is shown with a warning: "⚠ This analysis is 3 days old. The code has changed since then."
    And a "Refresh analysis" button spawns a new explorer

  @comp-idle-prefetch
  Scenario: Idle budget is enforced
    Given the idle budget is 10 subagents per session, max $0.01
    When the 10th subagent completes
    Then pre-computation stops
    And the status line shows: "💤 Idle pre-computation: budget reached (10/10 subagents · $0.01)"
    And no more subagents spawn until the next user interaction

  # ── Comp 13: Dependency Watchdog ──────────────────────────────────

  @comp-dependency-watchdog
  Scenario: Watchdog checks weekly for updates
    Given a project with 15 dependencies
    When the weekly watchdog cron fires
    Then all 15 dependencies are checked for newer versions
    And for each outdated dependency: a worktree is created, the update is applied, tests run
    And the report shows: "3 updates available. 2 safe to merge. 1 breaks tests (react 19.1 → 19.2: 3 test failures)"

  @comp-dependency-watchdog
  Scenario: Watchdog report renders as a card
    Given the watchdog completed its weekly check
    When the report renders
    Then a card shows: "📦 Dependency Watchdog · July 1, 2026"
    And below: "✓ react 19.1 → 19.2: all 42 tests pass. Safe to merge."
    And "✓ typescript 5.7 → 5.8: all 42 tests pass. Safe to merge."
    And "✗ next 14.2 → 15.0: 3 tests fail. See test output."
    And each entry has an "Apply update" button

  @comp-dependency-watchdog
  Scenario: Watchdog silent when everything is current
    Given all dependencies are at latest versions
    When the watchdog runs
    Then a brief notice appears: "📦 All 15 dependencies up to date"
    And no card is shown (not interesting enough)

  # ── Comp 14: Self-Play Code Generation ────────────────────────────

  @comp-self-play
  Scenario: Three implementations, judge picks best
    Given the user asked to implement a rate limiter
    When self-play code generation is active
    Then 3 executor subagents spawn with approach hints: simplicity, clarity, performance
    And all 3 implement the rate limiter independently
    When a judge subagent compares the 3 implementations
    Then the best is selected with rationale: "Clarity implementation chosen — most readable, adequate performance"
    And ideas from runners-up are grafted: "Adopted Simplicity's token bucket algorithm. Adopted Performance's lock-free counter."
    And the synthesized implementation is presented

  @comp-self-play
  Scenario: Self-play shows comparison card
    Given 3 implementations were generated and judged
    When the result renders
    Then a comparison card shows each implementation with metrics:
      "Simplicity: 42 lines, 3 files, passes all tests ✓"
      "Clarity: 58 lines, 4 files, passes all tests ✓ ★ WINNER"
      "Performance: 89 lines, 5 files, passes 39/42 tests ✗"
    And the winner is highlighted
    And "View diff" buttons compare any two implementations

  @comp-self-play
  Scenario: Self-play can be disabled for simple tasks
    Given the task is "Add a comment to calculateTotal"
    When self-play evaluates the task
    Then it determines the task is too simple for ensemble generation
    And only one executor runs (standard behavior)
    And a note shows: "Self-play skipped — task too simple for ensemble benefit"

  # ── Comp 15: Ensemble Code Review ─────────────────────────────────

  @comp-ensemble-review
  Scenario: Three reviewers with different philosophies
    Given the user is about to commit changes
    When ensemble code review runs
    Then 3 reviewers spawn: security-focused, correctness-focused, performance-focused
    And a synthesizer merges their findings with deduplication
    And the report shows: "Ensemble review: 5 findings (1 security · 3 correctness · 1 performance)"
    And findings are color-coded by reviewer type

  @comp-ensemble-review
  Scenario: Ensemble review finds what single reviewer misses
    Given a change has a subtle performance regression (N+1 query)
    And a standard reviewer missed it
    When the performance-focused reviewer examines the change
    Then it catches the N+1 query
    And the ensemble report includes it

  # ── Comp 16: Cross-Session Knowledge Base ─────────────────────────

  @comp-knowledge-base
  Scenario: Knowledge base serves instant answer for repeated question
    Given the user asked "How does authentication work?" 2 weeks ago and got a detailed analysis
    When the user asks the same question today
    Then the knowledge base returns the prior analysis in <100ms
    And a synthesis subagent checks: "Is this analysis still current? Code changed: no. → still accurate."
    And the answer is served instantly without new exploration

  @comp-knowledge-base
  Scenario: Knowledge base grows across sessions
    Given the user has used reasonix for 30 days across 6 projects
    When the knowledge base stats are queried
    Then it shows: "1,247 entries · 3.2MB · 87% hit rate on repeat queries · $0.47 saved in avoided re-exploration"

  @comp-knowledge-base
  Scenario: Knowledge base search is semantic
    Given the knowledge base has entries about "JWT token validation"
    When the user asks "How do we verify tokens?"
    Then the semantic search matches "JWT token validation" entries
    And the answer is retrieved despite different wording

  @comp-knowledge-base
  Scenario: Knowledge base privacy — local only
    Given the knowledge base contains analysis of proprietary code
    When the user verifies privacy
    Then all data is on the local filesystem at .reasonix/knowledge/
    And no knowledge base data is included in telemetry
    And no knowledge base data is in session exports unless explicitly included

  # ── Comp 17: Bug Pattern Recognition ──────────────────────────────

  @comp-bug-pattern
  Scenario: Pattern recognized from past bug
    Given a bug was fixed 3 weeks ago: "nil pointer in handler when request body is empty"
    And the pattern was extracted and stored
    When the user writes new code with the same pattern (accessing request body without nil check)
    Then a notice appears: "⚠ Pattern match: this code resembles a bug fixed 3 weeks ago in handlers.go:108. Fix then: add `if req.Body == nil` guard."
    And the notice links to the original bug fix commit

  @comp-bug-pattern
  Scenario: Pattern library grows over time
    Given the user has fixed 47 bugs over 3 months
    When the pattern library stats are queried
    Then it shows: "47 patterns learned · 12 proactive catches · $0.34 total cost · est. 8 hours of debugging prevented"

  @comp-bug-pattern
  Scenario: False positive rate tracked
    Given the bug pattern detector has fired 20 times
    And 15 were genuine matches, 5 were false positives
    When the accuracy stats are queried
    Then it shows: "75% precision (15/20) · false positive rate within acceptable range"
    And if precision drops below 50%, the detector auto-pauses and notifies the user

  # ── Comp 18: Impact Analysis Pre-Computation ──────────────────────

  @comp-impact-analysis
  Scenario: Signature change triggers impact analysis
    Given the user changes a function signature from `func Auth(token string)` to `func Auth(ctx context.Context, token string)`
    When the file is saved
    Then an impact analysis subagent spawns: "Find every call site. Report which will break."
    And within 2 seconds, the result is available: "12 call sites found. 12 will break. 8 are trivial fixes (add ctx). 4 need deeper changes."
    And when the user asks "What breaks?" the answer is instant

  @comp-impact-analysis
  Scenario: Impact analysis result renders as call-site table
    Given impact analysis found 12 affected call sites
    When the result renders
    Then a table shows: File | Line | Current Call | Status
    And "Will break" entries are red, "Safe" entries are green
    And each entry is clickable to jump to the call site
    And a "Fix all" button spawns an executor to fix all 8 trivial call sites

  # ── Compensations Dashboard ───────────────────────────────────────

  @comp-dashboard
  Scenario: Compensations dashboard shows all active compensations
    When the user opens Settings → Compensations
    Then a dashboard shows all 18 compensations with: name, status (active/paused/disabled), daily cost, monthly cost, value metric
    And compensations are grouped by class: Structural Quality, Continuous Background, Ensemble Intelligence, Compounding Learning, Pre-Emptive Intelligence
    And each can be toggled on/off individually
    And cost is shown per compensation and as a total

  @comp-dashboard
  Scenario: Compensation value metric justifies cost
    Given the Ambient Guardian costs $0.03/day
    And it has caught 3 bugs this month that would have taken ~2 hours to debug
    When the dashboard renders
    Then the Guardian row shows: "$0.03/day · caught 3 bugs this month · ~2h debugging saved"
    And the value metric is: "ROI: ~50× (saved $300 in developer time for $6 in tokens)"

  @comp-dashboard
  Scenario: Compensation suggests disabling if value is negative
    Given a compensation costs $0.05/day and has produced zero value in 30 days
    When the dashboard renders
    Then the compensation shows a suggestion: "💡 Low value — consider disabling"
    And the user can dismiss the suggestion or act on it

  @comp-status-bar
  Scenario: Status bar shows aggregate compensation status
    Given 12 compensations are active
    When the status line renders
    Then it shows: "⚙️ 12 compensations · $0.08 today · est. $4.20 saved"
    And the indicator is dimmed (ambient, not attention-grabbing)
    And clicking opens the compensations dashboard
