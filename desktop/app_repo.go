package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"reasonix/internal/git"
	"reasonix/internal/github"
)

// ── Repo picker types (mirrored in the frontend bridge) ──

// RepoPickerItem is one repository returned to the frontend's repo picker.
type RepoPickerItem struct {
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	UpdatedAt   string `json:"updatedAt"`
	CloneURL    string `json:"cloneUrl"`
	HTMLURL     string `json:"htmlUrl"`
	Language    string `json:"language"`
}

// CloneResultView is the frontend-facing clone outcome.
type CloneResultView struct {
	Dir     string `json:"dir"`
	Cloned  bool   `json:"cloned"`
	Branch  string `json:"branch"`
	Commit  string `json:"commit"`
	Message string `json:"message"`
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Error   string `json:"error,omitempty"`
}

// ClonedRepoView is one already-cloned repo directory.
type ClonedRepoView struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	FullName string `json:"fullName"`
	ClonedAt string `json:"clonedAt"`
}

// RepoCloneProgress is emitted on the "repo:clone-progress" event channel
// during CloneRepo so the frontend can animate a stage-by-stage progress UI.
type RepoCloneProgress struct {
	Stage    string  `json:"stage"`    // "cloning" | "detecting" | "init" | "ready" | "error"
	Progress float64 `json:"progress"` // 0.0 – 1.0
	Message  string  `json:"message"`
	RepoURL  string  `json:"repoUrl"`
	Dir      string  `json:"dir"`
}

const repoCloneProgressChannel = "repo:clone-progress"

// ── Bound methods ──

// OpenRepoPicker returns a list of GitHub repos for the frontend's repo picker.
// filter is "mine" (user repos), "starred" (starred repos), or a free-text
// search query. Returns an error when no GitHub token is configured.
func (a *App) OpenRepoPicker(filter string) ([]RepoPickerItem, error) {
	token := github.LoadToken()
	if token == "" {
		return nil, fmt.Errorf("no github token: connect your GitHub account first")
	}

	client := github.NewClient(token)
	ctx := a.bootContext()

	var repos []github.RepoInfo
	var err error

	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "starred":
		repos, err = client.ListStarredRepos(ctx, 50)
	case "mine":
		repos, err = client.ListUserRepos(ctx, 50)
	default:
		q := strings.TrimSpace(filter)
		if q == "" {
			repos, err = client.ListUserRepos(ctx, 50)
		} else {
			repos, err = client.SearchRepos(ctx, q, 30)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}

	out := make([]RepoPickerItem, 0, len(repos))
	for _, r := range repos {
		desc := strings.TrimSpace(r.Description)
		out = append(out, RepoPickerItem{
			Name:        r.Name,
			FullName:    r.FullName,
			Description: desc,
			Private:     r.Private,
			UpdatedAt:   r.UpdatedAt,
			CloneURL:    r.CloneURL,
			HTMLURL:     r.HTMLURL,
			Language:    r.Language,
		})
	}
	return out, nil
}

// CloneRepo clones a git repository URL to dir, streaming progress events on
// the "repo:clone-progress" channel. If dir already exists and is a git repo,
// it fetches the latest instead of re-cloning. Returns the clone result.
func (a *App) CloneRepo(url, branch, dir string) (*CloneResultView, error) {
	url = strings.TrimSpace(url)
	branch = strings.TrimSpace(branch)
	dir = normalizeCloneDir(strings.TrimSpace(dir), url)

	a.emitRepoProgress("cloning", 0.15, "", url, dir)

	result, err := git.Clone(context.Background(), url, dir, branch)
	if err != nil {
		a.emitRepoProgress("error", 0, err.Error(), url, dir)
		return &CloneResultView{Error: err.Error()}, err
	}

	a.emitRepoProgress("detecting", 0.55, "", url, dir)
	projectType := git.DetectProjectType(dir)

	a.emitRepoProgress("init", 0.75, projectType, url, dir)

	// Run any project-specific init (the kernel boot will do the full init
	// when the workspace is opened; here we just note what was detected).
	_ = projectType

	a.emitRepoProgress("ready", 1.0, "", url, dir)

	// Persist the clone as a known workspace entry so it appears in the
	// project list and the ListClonedRepos inventory.
	saveWorkspace(dir)

	return &CloneResultView{
		Dir:     result.Dir,
		Cloned:  result.Cloned,
		Branch:  result.Branch,
		Commit:  result.Commit,
		Message: result.Message,
		Owner:   result.Owner,
		Repo:    result.Repo,
	}, nil
}

// ListClonedRepos returns every cloned repository discovered under the
// standard reasonix-projects directory.
func (a *App) ListClonedRepos() ([]ClonedRepoView, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, "reasonix-projects")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []ClonedRepoView{}, nil
		}
		return nil, err
	}

	var out []ClonedRepoView
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if !git.IsRepo(dir) {
			continue
		}
		info, err := entry.Info()
		modTime := ""
		if err == nil {
			modTime = info.ModTime().UTC().Format(time.RFC3339)
		}

		branch := git.CurrentBranch(dir)
		fullName := entry.Name() // default: dir name
		// Try to derive owner/repo from the git remote.
		if o, r, ok := git.ParseGitHubURL(remoteURL(dir)); ok {
			fullName = o + "/" + r
		}
		name := entry.Name()
		if branch != "unknown" {
			name = name + " (" + branch + ")"
		}

		out = append(out, ClonedRepoView{
			Path:     dir,
			Name:     name,
			FullName: fullName,
			ClonedAt: modTime,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ClonedAt > out[j].ClonedAt
	})
	return out, nil
}

// OpenClonedRepo opens a previously cloned repo directory and switches the
// active workspace to it. It returns the normalized workspace root path.
func (a *App) OpenClonedRepo(path string) (string, error) {
	return a.SwitchWorkspace(path)
}

// ── helpers ──

func (a *App) emitRepoProgress(stage string, progress float64, message, url, dir string) {
	if a.ctx == nil {
		return
	}
	payload, err := json.Marshal(RepoCloneProgress{
		Stage:    stage,
		Progress: progress,
		Message:  message,
		RepoURL:  url,
		Dir:      dir,
	})
	if err != nil {
		return
	}
	runtime.EventsEmit(a.ctx, repoCloneProgressChannel, string(payload))
}

func normalizeCloneDir(dir, url string) string {
	if dir != "" {
		if abs, err := filepath.Abs(dir); err == nil {
			return abs
		}
		return dir
	}
	owner, repo, ok := git.ParseGitHubURL(url)
	if !ok {
		// Fall back to extracting the last path segment.
		repo = filepath.Base(strings.TrimSuffix(url, ".git"))
		owner = ""
	}
	if owner != "" {
		return git.DefaultCloneDir(owner, repo)
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return repo
	}
	return filepath.Join(home, "reasonix-projects", repo)
}

func remoteURL(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
