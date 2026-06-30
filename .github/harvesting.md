# Upstream Harvesting

How we track, evaluate, and cherry-pick upstream PRs from [`esengine/DeepSeek-Reasonix`](https://github.com/esengine/DeepSeek-Reasonix).

## Process

### 1. Discover

```bash
# List open PRs by size (descending) — filter dependabot
gh pr list --repo esengine/DeepSeek-Reasonix --state open --limit 200 \
  --json number,title,author,additions,deletions,labels,headRefName \
  --template '{{range .}}{{if not (contains .title "dependabot")}}{{.number}}|{{.author.login}}|{{.additions}}|{{.deletions}}|{{.headRefName}}|{{.title}}{{"\n"}}{{end}}{{end}}'

# List recently merged (to find merged-after-fork)
gh pr list --repo esengine/DeepSeek-Reasonix --state merged --limit 20 \
  --json number,mergedAt,title
```

### 2. Evaluate

```bash
# View full PR details
gh pr view <N> --repo esengine/DeepSeek-Reasonix \
  --json title,body,additions,deletions,files,headRefName

# Map against our issues — check file overlap
gh pr view <N> --repo esengine/DeepSeek-Reasonix --json files --jq '.files[].path'
```

### 3. Conflict Test

```bash
git fetch upstream refs/pull/<N>/head:pr-upstream/<N>
git merge-tree $(git merge-base HEAD pr-upstream/<N>) HEAD pr-upstream/<N> | grep -c '<<<<<<<'
```

### 4. Harvest (Tier 1 — Clean Cherry-Pick)

```bash
git cherry-pick -x pr-upstream/<N>
git commit --amend -m "$(git log -1 --format=%B)

Co-Authored-By: <github-handle>"
```

### 5. Harvest (Tier 1 — Minor Conflicts)

```bash
git cherry-pick -x pr-upstream/<N>
# Resolve conflicts, usually by taking upstream version for new files
git checkout --theirs <conflicted-file>
git add <conflicted-file>
git cherry-pick --continue --no-edit
```

### 6. Harvest (Tier 2 — Partial)

```bash
# Fetch and extract specific files/directories
git fetch upstream refs/pull/<N>/head
git checkout FETCH_HEAD -- <paths>
git commit -m "scaffolding: extract <component> from upstream #<N>

Co-Authored-By: <github-handle>"
```

### 7. Attribute

Every harvest commit MUST include `Co-Authored-By:` trailers for each upstream contributor. For multi-PR squash commits, list all contributors. Format:

```
harvest: cherry-pick N upstream PRs from esengine/DeepSeek-Reasonix

PR #<N> — <title>
PR #<M> — <title>
...

Co-Authored-By: Name <email>
```

## Harvest Tiers

### Tier 1: Direct Cherry-Pick
Bug fixes, validation, hardening. No design decisions. Always safe.

### Tier 2: Infrastructure Harvest
Take the data layer / backend, rewrite the surface API / tool schema to match our architecture.

### Tier 3: Conceptual
Read, learn design patterns, write fresh. Never copy-paste.

### Not Harvested
- Werewolf games, entertainment features
- Desktop-only polish with no architectural impact
- Heavily conflicted refactors (easier to apply the pattern fresh)
- Dependabot bumps (handled separately)

## Inventory

24 upstream PRs tracked via `pr-upstream/*` refs. Status as of 2026-06-30.

### Harvested (11 PRs — 2 commits)

| Commit | PRs | Scope |
|---|---|---|
| `dd66dc92` | #5461 #5564 #5589 #5643 #5660 | Bug fixes, subagent validation, desktop, Windows paths |
| `0ebba5f8` | #4864 #5055 #5105 #5212 #5362 #5415 | Loop guard, custom effort, bot deadlock, curator |

| # | Source | Title | Lines | Relevant Issue |
|---|---|---|---|---|
| 5461 | bfxh | 21 incremental bugs (1 HIGH, 11 MEDIUM, 7 LOW) | +227/−47 | #4, #3 |
| 5564 | breayhing | Subagent session lineage validation | +160/−6 | #4 |
| 5589 | cyq | Hint escaped Windows paths | +28/−1 | — |
| 5643 | SivanCola | DPI zoom preference authoritative | +369/0 | — |
| 5660 | SauronSkywalker | Permission chip overflow fix | +3/−6 | — |
| 4864 | tonystarknb | Honor custom OpenAI reasoning efforts | +116/−10 | #8 |
| 5055 | eghrhegpe | Balance query with configurable fields | +658/−219 | — |
| 5105 | cecil-su | Bot approval deadlock fix | +106/−2 | — |
| 5212 | dengbushi | Scoped todo panel dismissal | +120/−4 | — |
| 5362 | aznikline | API key env literal value hint | +200/−1 | — |
| 5415 | kuangvszhe | Curator / selfevolve / loopguard | +881/0 | #3 |

### Already Merged in Base (2 PRs — in fork at launch)

| # | Source | Title |
|---|---|---|
| 5452 | SivanCola | Memory and permission hardening regressions |
| 5648 | SivanCola | Memory compiler feedback noise fix |

### Pending — Harvest When Issue Starts

| # | Source | Lines | Title | When | Conflicts |
|---|---|---|---|---|---|
| 5276 | taibai233 | +3,022 | BTW TUI surface reuse | Issue #7 (Background Tasks Panel) | Yes — `chat_tui.go` |
| 5299 | qiaone | +2,063 | Extracted system prompts (14 files) | Issue #18 (Prompt Calibration) | None (add-only) |
| 5405 | drafish | +2,932 | Token tracking (JSONL + CLI + Desktop) | Issue #6 (Enriched jobs.View) | Unknown |
| 4881 | eghrhegpe | +391 | search_sessions tool | Issue #20 (Cross-Session KB) | 2 files |

### Rejected — Not Worth Harvesting

| # | Source | Lines | Title | Reason |
|---|---|---|---|---|
| 3733 | SuMuxi66 | +164 | Toggle for tool call visibility | Desktop-only, not architectural |
| 4516 | yuanyuanlove | +410 | Desktop provider resolution | 1,042 conflicts |
| 4518 | lizhengwu | +150 | Route slog to TUI Notice events | 1,085 conflicts |
| 4841 | wangwangcodecode | +1,244 | Schema versioning + migration refactor | 9 conflicts, files we'll rewrite |
| 4844 | zhangjiayang | +531 | system_control builtin tool | 193 conflicts |
| 4880 | izijiewang | +2,262 | Controller split refactor | 28 conflicts, pattern-applied fresh |
| 5089 | eghrhegpe | +9,583 | 5 edge-case defenses + werewolf game | 200 lines of hardening among 9,500 lines of werewolf |
| 5616 | yekern | +150 | TUI incremental transcript rendering | Conflicts in `chat_tui.go` |

## Rejected Audit Trail

| # | Examined | Decision | Reasoning |
|---|---|---|---|
| 4880 (controller split) | 2026-06-30 | **Rejected** | 28 conflicts over 8 new files whose code will fundamentally change after Phase 2-3 work. The refactoring pattern (split god object by responsibility domain) is documented for fresh application when our architecture stabilizes. |
| 4841 (schema versioning) | 2026-06-30 | **Rejected** | 9 conflicts in migrate.go, provider.go — files our Phase 2-3 subagent work substantially rewrites. Schema versioning concept is correct but the implementation must be re-evaluated after foundation work completes. |
| 4844 (system_control) | 2026-06-30 | **Rejected** | 193 conflicts. The tool design (system_control with allowlisted commands) is informative for our tool-gap work in Phase 6 but should be built fresh against our boot architecture. |
| 4516 (provider config) | 2026-06-30 | **Rejected** | 1,042 conflicts — the upstream has diverged significantly in provider resolution. Our Phase 1+2 work on prompt calibration and subagent roles will touch the same area; re-evaluate after Phase 2. |
| 5089 (edge-case defenses + werewolf) | 2026-06-30 | **Partial** | 9,583 lines, 95% werewolf game scripts. ~200 lines of useful hardening in memory/permission/parallel_tasks. Extract only the hardening, skip the werewolf. |
| 5276 (BTW TUI) | 2026-06-30 | **Deferred** | 3,022 lines of valuable TUI surface reuse with well-tested patterns. But it's a merge commit that conflicts in `chat_tui.go`. Deferred until #7 (Background Tasks Panel) starts — harvest the control/side abstraction then. |
| 5616 (TUI incremental rendering) | 2026-06-30 | **Rejected** | ~150 lines but conflicts in `chat_tui.go` which is already heavily modified by #5276 when that lands. Marginal value vs. rebuild cost. |

## Maintenance

- `pr-upstream/*` refs are local tracking branches. Run `git push origin refs/notes/upstream-harvest:refs/notes/upstream-harvest` after each inventory update.
- Add new evaluated PRs as tracking refs: `git fetch upstream refs/pull/<N>/head:pr-upstream/<N>`
- Delete stale refs for merged PRs we've harvested or rejected: `git branch -D pr-upstream/<N>`
- Update this file when harvesting new PRs or re-evaluating status.
