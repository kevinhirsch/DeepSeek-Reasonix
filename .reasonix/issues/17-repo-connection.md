# 17 — Git Repository Connection & Auto-Setup

**Phase:** Foundation | **Effort:** 3 days | **Priority:** P0

## Problem

Users must manually clone repos, cd into them, and start reasonix. Claude Code's repo selection UX — pick a repo, it auto-clones, sets up, and starts a session — doesn't exist in reasonix. Additionally, repos can't be cloned directly to remote sandboxes for isolated work.

## Design

### Repo Sources
- **GitHub OAuth:** Connect GitHub account → list user's repos, org repos, starred repos. Browse and search.
- **GitHub URL:** Paste `https://github.com/owner/repo` — detect it's GitHub, offer to clone/fetch.
- **Generic Git URL:** Any `git@` or `https://` git remote URL.
- **Local path:** Already-cloned repo on disk. Detect it's a git repo.

### Clone Targets
- **Local:** Clone to `~/reasonix-projects/<owner>-<repo>/` or user-configured directory. Reuse existing clone if already present (fetch latest instead).
- **Remote:** Clone to a configured remote worker. The worker receives the clone command, sets up the workspace, and the session runs against the remote sandbox.

### Auto-Setup
After clone/fetch:
1. Detect project type (go.mod, package.json, pyproject.toml, Cargo.toml, etc.)
2. Run `reasonix init` if no AGENTS.md/REASONIX.md exists
3. Load project's reasonix.toml if present
4. Open a new tab with the repo as workspace root
5. Session starts with a context message: "Connected to <owner>/<repo> on branch <branch>. Last commit: <hash> — <message>."

### Branch Selection
- Default: repo's default branch (main/master)
- User can select a different branch before cloning
- After clone, branch can be switched from the workspace panel

### GitHub Integration
- OAuth flow: `reasonix github login` → browser → authorize → token stored in credential store
- `reasonix github repos` lists accessible repos
- `reasonix github clone owner/repo` clones and starts a session
- Desktop: repo picker with search, filter by org, sort by updated

## Files
- `internal/git/` (NEW) — git operations: clone, fetch, branch list, commit info
- `internal/github/` (NEW) — GitHub API client: OAuth, repo list, search
- `internal/cli/repo.go` (NEW) — CLI: `reasonix clone`, `reasonix github`
- `desktop/frontend/src/components/RepoPicker.tsx` (NEW) — desktop repo picker
- `desktop/frontend/src/components/RepoSetupProgress.tsx` (NEW) — clone progress
- `desktop/frontend/src/lib/github.ts` (NEW) — frontend GitHub API helpers
- `desktop/app_repo.go` (NEW) — backend handlers for repo operations
- `internal/boot/boot.go` — modify: accept repo URL as workspace root, auto-setup on boot
- `internal/sandbox/` — modify: accept git clone as pre-execution step for remote workers
- `internal/config/config.go` — add `[repos]` config section

## Acceptance
- [ ] GitHub OAuth: `reasonix github login` completes in browser, token stored
- [ ] `reasonix github repos` lists user's repos with descriptions and last-updated
- [ ] `reasonix clone https://github.com/owner/repo` clones, detects project, starts session
- [ ] `reasonix clone --branch feature-x https://github.com/owner/repo` clones specific branch
- [ ] `reasonix clone --remote build-server https://github.com/owner/repo` clones to remote worker
- [ ] Reusing existing clone fetches latest instead of re-cloning
- [ ] Auto-setup detects project type correctly across Go, TS, Python, Rust
- [ ] Session starts with context message showing repo, branch, and last commit
- [ ] Desktop: repo picker with search, org filter, sort-by-updated
- [ ] Desktop: clone progress bar with stage indicators
- [ ] Desktop: repo list shows already-cloned repos with "Open" vs "Clone" action
- [ ] Token stored in credential store, never in config files
- [ ] Private repos work (credential passed through)
