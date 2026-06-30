Feature: Git Repository Connection & Auto-Setup
  As a developer who works across multiple repositories
  I want to connect a git repo, have reasonix auto-set up with the latest code
  So that I can start working immediately without manual clone/setup steps

  Background:
    Given reasonix has git and GitHub integration
    And the user has a GitHub account with repositories

  # ── GitHub OAuth ──────────────────────────────────────────────────

  @repo-github-login
  Scenario: GitHub OAuth login flow
    When the user runs "reasonix github login"
    Then a browser opens to github.com/login/oauth/authorize
    And after authorizing, the callback receives the OAuth code
    And the token is exchanged and stored in the credential store
    And the CLI shows: "✓ Connected to GitHub as <username>"
    And the token has repo scope

  @repo-github-login
  Scenario: GitHub login when already authenticated
    Given the user is already authenticated with GitHub
    When the user runs "reasonix github login"
    Then the CLI shows: "✓ Already connected to GitHub as <username>"
    And no browser opens

  @repo-github-login
  Scenario: GitHub logout
    Given the user is authenticated with GitHub
    When the user runs "reasonix github logout"
    Then the token is removed from the credential store
    And the CLI shows: "✓ Disconnected from GitHub"
    And subsequent "reasonix github repos" prompts to login

  @repo-github-login
  Scenario: GitHub token stored securely
    Given the user completes GitHub OAuth
    When the token is stored
    Then the token is in the OS credential store (keyring/keychain), NOT in reasonix.toml
    And the token is NOT in any .env file
    And the token value is never logged

  # ── Repo Listing ──────────────────────────────────────────────────

  @repo-list
  Scenario: List user's GitHub repositories
    When the user runs "reasonix github repos"
    Then a list of repositories is shown with: name, description, language, stars, last updated
    And repos are sorted by last updated (most recent first)
    And each repo shows its default branch

  @repo-list
  Scenario: Filter repos by organization
    When the user runs "reasonix github repos --org my-company"
    Then only repos belonging to "my-company" are shown
    And the header shows: "Repositories in my-company (42)"

  @repo-list
  Scenario: Search repos by name
    When the user runs "reasonix github repos --search reasonix"
    Then only repos matching "reasonix" in name or description are shown

  @repo-list
  Scenario: List repos when not authenticated
    Given the user is not authenticated with GitHub
    When the user runs "reasonix github repos"
    Then an error is shown: "Not connected to GitHub. Run 'reasonix github login' to connect."
    And the exit code is 1

  @repo-list
  Scenario: Handle GitHub API rate limiting
    Given the unauthenticated GitHub API rate limit is exhausted
    When the user runs "reasonix github repos"
    Then an error is shown: "GitHub API rate limit exceeded. Authenticate with 'reasonix github login' for higher limits."
    And the reset time is shown

  # ── Clone Locally ─────────────────────────────────────────────────

  @repo-clone-local
  Scenario: Clone a GitHub repo locally
    When the user runs "reasonix clone https://github.com/owner/repo"
    Then the repo is cloned to ~/reasonix-projects/owner-repo/
    And a progress indicator shows: "Cloning owner/repo... ✓ 45MB received"
    And after clone, project type is auto-detected
    And REASONIX.md is initialized if not present
    And a new session tab opens with workspace at ~/reasonix-projects/owner-repo/
    And the session starts with: "Connected to owner/repo on branch main. Last commit: abc1234 — 'Fix auth middleware'. Ready."

  @repo-clone-local
  Scenario: Clone to custom directory
    When the user runs "reasonix clone --dir ~/projects/myrepo https://github.com/owner/repo"
    Then the repo is cloned to ~/projects/myrepo/
    And the session workspace is ~/projects/myrepo/

  @repo-clone-local
  Scenario: Clone specific branch
    When the user runs "reasonix clone --branch feature-x https://github.com/owner/repo"
    Then the repo is cloned with branch "feature-x" checked out
    And the session starts on branch "feature-x"

  @repo-clone-local
  Scenario: Reuse existing clone with fetch
    Given ~/reasonix-projects/owner-repo already exists from a previous clone
    When the user runs "reasonix clone https://github.com/owner/repo"
    Then the existing clone is detected
    And "git fetch --all" is run to get the latest
    And the default branch is checked out and pulled
    And a notice shows: "✓ Using existing clone — fetched latest (3 new commits)"
    And the session starts with the updated workspace

  @repo-clone-local
  Scenario: Clone private repo with GitHub token
    Given the user is authenticated with GitHub
    And the repo is private
    When the user runs "reasonix clone https://github.com/owner/private-repo"
    Then the clone uses the GitHub token for authentication
    And the token is passed via git credential helper (not in the URL)
    And the clone succeeds

  @repo-clone-local
  Scenario: Clone generic git URL
    When the user runs "reasonix clone git@gitlab.com:owner/repo.git"
    Then the repo is cloned using SSH
    And the session starts normally

  @repo-clone-local
  Scenario: Clone fails with clear error
    Given the repo URL is invalid or unreachable
    When the user runs "reasonix clone https://github.com/owner/nonexistent"
    Then an error is shown: "Failed to clone: repository not found. Check the URL and your access permissions."
    And no partial clone directory is left behind

  @repo-clone-local
  Scenario: Clone with submodules
    When the user runs "reasonix clone --recurse-submodules https://github.com/owner/repo"
    Then submodules are cloned recursively
    And the progress shows submodule progress

  # ── Clone to Remote Sandbox ───────────────────────────────────────

  @repo-clone-remote
  Scenario: Clone directly to a remote worker
    Given a remote worker "build-server" is configured and online
    When the user runs "reasonix clone --remote build-server https://github.com/owner/repo"
    Then the repo is cloned on the "build-server" worker
    And the clone progress is streamed from the worker
    And after clone, the session is opened with workspace on the remote worker
    And the status line shows: "🌐 build-server · owner/repo · main"

  @repo-clone-remote
  Scenario: Clone to remote with branch
    Given a remote worker "gpu-runner" is online
    When the user runs "reasonix clone --remote gpu-runner --branch develop https://github.com/owner/repo"
    Then the repo is cloned on gpu-runner with branch "develop"

  @repo-clone-remote
  Scenario: Remote clone fails if remote is offline
    Given the remote "build-server" is offline
    When the user runs "reasonix clone --remote build-server https://github.com/owner/repo"
    Then an error is shown: "Remote 'build-server' is offline. Clone locally instead?"
    And a "Clone locally" option is offered

  @repo-clone-remote
  Scenario: Remote clone shows dual progress
    Given the user is cloning to a remote worker
    When the clone runs
    Then the progress shows: "Sending clone command to build-server... ✓ · Cloning on build-server... 34%"
    And both the command delivery and the clone progress are visible

  # ── Auto-Setup After Clone ────────────────────────────────────────

  @repo-autosetup
  Scenario: Auto-detect Go project
    Given a repo with go.mod at the root
    When the clone completes and auto-setup runs
    Then the project type is detected as "Go"
    And the build command is detected as "go build ./..."
    And the test command is detected as "go test ./..."
    And the detected commands are shown: "✓ Detected Go project · build: go build ./... · test: go test ./..."

  @repo-autosetup
  Scenario: Auto-detect TypeScript project
    Given a repo with package.json containing react and typescript
    When auto-setup runs
    Then the project type is detected as "TypeScript (React)"
    And the detected package manager is npm/pnpm/yarn
    And commands are detected from package.json scripts

  @repo-autosetup
  Scenario: Auto-detect Python project
    Given a repo with pyproject.toml
    When auto-setup runs
    Then the project type is detected as "Python"

  @repo-autosetup
  Scenario: Auto-detect Rust project
    Given a repo with Cargo.toml
    When auto-setup runs
    Then the project type is detected as "Rust"

  @repo-autosetup
  Scenario: Auto-detect multi-language project
    Given a repo with both go.mod and package.json
    When auto-setup runs
    Then the primary language is detected (heuristic: most files)
    And a notice shows: "Multi-language project detected. Primary: Go. Also found: TypeScript."
    And the user can switch the primary language

  @repo-autosetup
  Scenario: Init REASONIX.md if absent
    Given the cloned repo has no REASONIX.md or AGENTS.md
    When auto-setup runs
    Then reasonix init runs automatically
    And the generated REASONIX.md is written to the repo root
    And the init output is shown in the transcript

  @repo-autosetup
  Scenario: Skip init if memory file exists
    Given the cloned repo already has AGENTS.md
    When auto-setup runs
    Then reasonix init is NOT run
    And the existing AGENTS.md is loaded into the session prompt

  @repo-autosetup
  Scenario: Load project reasonix.toml if present
    Given the cloned repo has ./reasonix.toml with a custom model
    When auto-setup runs
    Then the project config is loaded
    And the configured model is used for the session
    And a notice shows: "✓ Loaded project config from reasonix.toml"

  @repo-autosetup
  Scenario: Auto-setup reports detected toolchain versions
    When auto-setup completes
    Then a summary is shown:
      And "Go 1.23.4 · Node 22.3.0 · Python 3.12.1"
      And "Dependencies: 142 Go packages · 892 npm packages"
    And the summary is folded into the session context

  # ── Desktop: Repo Picker ──────────────────────────────────────────

  @repo-fe-picker
  Scenario: Repo picker opens from welcome screen
    Given the desktop app is on the welcome screen (no active session)
    When the welcome screen renders
    Then a "Connect a repository" card is shown prominently
    And the card has options: "GitHub", "Git URL", "Local folder"
    And a "Continue without a repo" link is available

  @repo-fe-picker
  Scenario: GitHub repo picker shows repo list
    Given the user clicked "GitHub" and is authenticated
    When the repo picker renders
    Then repos are listed in a scrollable grid
    And each repo card shows: name, description (truncated), language badge, stars, last updated
    And a search bar filters repos by name
    And org filter tabs are shown: "Personal", "my-org", "other-org"

  @repo-fe-picker
  Scenario: Repo picker shows already-cloned repos with different action
    Given ~/reasonix-projects/owner-repo already exists
    When the repo picker lists repos
    Then "owner/repo" shows "Open" instead of "Clone"
    And "Open" opens the existing workspace
    And a "Re-clone" option is in the context menu

  @repo-fe-picker
  Scenario: Repo picker search with debounce
    Given the repo picker is open
    When the user types "rea" and pauses
    Then after 300ms debounce, repos matching "rea" are shown
    And a loading spinner shows during search
    And the results update without full picker re-render

  @repo-fe-picker
  Scenario: Repo picker empty state for no repos
    Given the GitHub account has no repositories
    When the repo picker renders
    Then a message is shown: "No repositories found. Create one at github.com/new."
    And the "Git URL" and "Local folder" options remain available

  @repo-fe-picker
  Scenario: Repo picker for unauthenticated user
    Given the user is not connected to GitHub
    When the user clicks "GitHub" in the repo picker
    Then a "Connect GitHub" button is shown with the GitHub logo
    And clicking it starts the OAuth flow
    And a "Paste a URL instead" link bypasses OAuth

  @repo-fe-picker
  Scenario: Paste GitHub URL directly
    Given the user clicked "Git URL"
    When the user pastes "https://github.com/owner/repo"
    Then the URL is parsed as a GitHub repo
    And the repo name "owner/repo" is extracted
    And a preview card shows: "owner/repo · main · last updated 2 days ago"
    And a "Clone" button is shown with branch selector and target selector (local/remote)

  # ── Desktop: Clone Progress ───────────────────────────────────────

  @repo-fe-clone-progress
  Scenario: Clone progress shows stages
    Given the user confirmed a clone of "owner/repo"
    When the clone starts
    Then a progress panel appears with stages:
      And "🔍 Resolving repository... ✓"
      And "📦 Cloning... ████████░░ 67% (32MB/48MB)"
      And "🔧 Detecting project... (pending)"
      And "📝 Setting up... (pending)"
    And completed stages show ✓, current stage animates, pending stages are dimmed

  @repo-fe-clone-progress
  Scenario: Clone progress shows speed and ETA
    Given the clone is in progress
    When the progress renders
    Then it shows: "Cloning... 32MB/48MB · 8.2 MB/s · ~2s remaining"
    And the speed and ETA update live

  @repo-fe-clone-progress
  Scenario: Clone completes and transitions to session
    Given all clone stages completed
    When the final stage "📝 Setting up... ✓" finishes
    Then the progress panel slides away
    And a new session tab opens with the repo loaded
    And a brief welcome toast appears: "✓ Connected to owner/repo"
    And the composer is focused

  @repo-fe-clone-progress
  Scenario: Clone fails mid-way with actionable error
    Given the clone failed at 45% due to network error
    When the error is detected
    Then the progress panel shows: "✗ Clone failed — network error"
    And the completed stages remain checked
    And a "Retry" button is available
    And a "Clone locally instead" button is available if targeting remote
    And the partial clone is cleaned up

  # ── Desktop: Branch Selector ──────────────────────────────────────

  @repo-fe-branch-selector
  Scenario: Branch selector shows default and recent branches
    Given the repo "owner/repo" has branches: main, develop, feature-x, feature-y
    When the branch selector opens next to the clone button
    Then "main (default)" is pre-selected
    And recent branches are listed first (if known)
    And a search filter narrows the branch list

  @repo-fe-branch-selector
  Scenario: Branch selector for already-cloned repo
    Given "owner/repo" is already cloned locally
    When the user opens the workspace panel for that repo
    Then the current branch is shown
    And a branch switcher dropdown shows all local and remote branches
    And switching branches runs "git checkout" and reloads the workspace context

  # ── Desktop: Workspace Panel Repo Info ────────────────────────────

  @repo-fe-workspace-info
  Scenario: Workspace panel shows repo metadata
    Given a session connected to "owner/repo"
    When the workspace panel renders
    Then the header shows: "owner/repo · main"
    And below: "Last commit: abc1234 — 'Fix auth middleware' (2 hours ago)"
    And the commit hash is clickable (opens GitHub)
    And the branch name is shown with a git-branch icon

  @repo-fe-workspace-info
  Scenario: Workspace panel shows uncommitted changes
    Given the workspace has 3 modified files and 1 untracked file
    When the workspace panel renders
    Then it shows: "3 modified · 1 untracked"
    And the git status is polled every 30 seconds
    And the status updates live

  @repo-fe-workspace-info
  Scenario: Workspace panel shows behind/ahead relative to remote
    Given the local branch is 2 commits behind and 1 commit ahead of origin/main
    When the workspace panel renders
    Then it shows: "↓2 · ↑1 vs origin/main"
    And a "Fetch" button is available

  # ── Desktop: Session Context ──────────────────────────────────────

  @repo-fe-session-context
  Scenario: First turn includes repo context
    Given a session just started from a cloned repo
    When the first agent turn runs
    Then the system prompt includes:
      And "Repository: owner/repo"
      And "Branch: main"
      And "Last commit: abc1234 — 'Fix auth middleware'"
      And "Project type: Go"
      And "Files: 142 Go, 38 Markdown, 12 YAML"
    And the agent's first response references the project context

  # ── Integration: Repo + Remote + Workflow ─────────────────────────

  @repo-integration-remote-workflow
  Scenario: Clone to remote, run workflow on that repo
    Given the user cloned "owner/repo" to remote "build-server"
    When the user runs a security audit workflow on the repo
    Then all subagents run on "build-server" against the cloned repo
    And the background panel shows 🌐 for all subagents
    And the workspace panel shows the remote indicator

  @repo-integration-multi-repo
  Scenario: Multiple repos open in different tabs
    Given tab A has repo "owner/frontend" on main
    And tab B has repo "owner/backend" on develop
    When the user switches between tabs
    Then each tab's workspace is isolated
    And tab A's agent sees only the frontend repo
    And tab B's agent sees only the backend repo
    And no cross-contamination occurs
