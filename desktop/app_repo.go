package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/git"
	"reasonix/internal/github"
)

// CloneResult is returned to the frontend after a clone operation.
type CloneResult struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Commit      string `json:"commit"`
	Message     string `json:"commitMessage"`
	ProjectType string `json:"projectType"`
	Dir         string `json:"dir"`
	Cloned      bool   `json:"cloned"`
	HasAgentDocs bool  `json:"hasAgentDocs"`
}

// RepoInfo is a lightweight repo description for the frontend picker.
type RepoInfo struct {
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	UpdatedAt   string `json:"updatedAt"`
	CloneURL    string `json:"cloneUrl"`
	HTMLURL     string `json:"htmlUrl"`
	Language    string `json:"language"`
	IsCloned    bool   `json:"isCloned"`
	LocalPath   string `json:"localPath,omitempty"`
}

// ListClonedRepos returns all repos that have already been cloned locally.
func (a *App) ListClonedRepos() []RepoInfo {
	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, "reasonix-projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}
	var repos []RepoInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(projectsDir, e.Name())
		if !git.IsRepo(dir) {
			continue
		}
		owner, repo := parseOwnerRepoDir(e.Name())
		branch := git.CurrentBranch(dir)
		commit, _ := git.LastCommit(dir)
		repos = append(repos, RepoInfo{
			Name:      repo,
			FullName:  owner + "/" + repo,
			Branch:    branch,
			Commit:    commit,
			IsCloned:  true,
			LocalPath: dir,
			UpdatedAt: dirModTime(dir).Format(time.RFC3339),
		})
	}
	return repos
}

// OpenClonedRepo switches the workspace to an already-cloned repo directory.
func (a *App) OpenClonedRepo(path string) error {
	if !git.IsRepo(path) {
		return fmt.Errorf("%q is not a git repository", path)
	}
	a.config.WorkspaceRoot = path
	projectType := git.DetectProjectType(path)
	branch := git.CurrentBranch(path)
	commit, message := git.LastCommit(path)
	contextMsg := fmt.Sprintf(
		"Connected to %s on branch %s.\nLast commit: %s — %s.\nProject type: %s.",
		filepath.Base(path), branch, commit, message, projectType,
	)
	_ = contextMsg // emitted via status update
	return a.switchWorkspaceTab(path)
}

// CloneRepo clones a GitHub repository and switches to it.
func (a *App) CloneRepo(url, branch string) (*CloneResult, error) {
	owner, repo, ok := git.ParseGitHubURL(url)
	if !ok {
		return nil, fmt.Errorf("unsupported URL: %s", url)
	}
	dir := git.DefaultCloneDir(owner, repo)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := git.Clone(ctx, url, dir, branch)
	if err != nil {
		return nil, err
	}
	projectType := git.DetectProjectType(result.Dir)
	return &CloneResult{
		Owner:        result.Owner,
		Repo:         result.Repo,
		Branch:       result.Branch,
		Commit:       result.Commit,
		Message:      result.Message,
		ProjectType:  projectType,
		Dir:          result.Dir,
		Cloned:       result.Cloned,
		HasAgentDocs: git.HasAgentDocs(result.Dir),
	}, nil
}

// ListGitHubRepos lists the user's GitHub repositories.
func (a *App) ListGitHubRepos(max int) ([]RepoInfo, error) {
	token := github.LoadToken()
	if token == "" {
		return nil, fmt.Errorf("not authenticated — run reasonix github login")
	}
	client := github.NewClient(token)
	ctx := context.Background()
	ghRepos, err := client.ListUserRepos(ctx, max)
	if err != nil {
		return nil, err
	}
	cloned := make(map[string]bool)
	for _, r := range a.ListClonedRepos() {
		cloned[r.FullName] = true
	}
	var out []RepoInfo
	for _, r := range ghRepos {
		out = append(out, RepoInfo{
			Name:        r.Name,
			FullName:    r.FullName,
			Description: r.Description,
			Private:     r.Private,
			UpdatedAt:   r.UpdatedAt,
			CloneURL:    r.CloneURL,
			HTMLURL:     r.HTMLURL,
			Language:    r.Language,
			IsCloned:    cloned[r.FullName],
		})
	}
	return out, nil
}

// ListStarredRepos lists the user's starred GitHub repositories.
func (a *App) ListStarredRepos() ([]RepoInfo, error) {
	token := github.LoadToken()
	if token == "" {
		return nil, fmt.Errorf("not authenticated")
	}
	client := github.NewClient(token)
	ctx := context.Background()
	ghRepos, err := client.ListStarredRepos(ctx, 30)
	if err != nil {
		return nil, err
	}
	cloned := make(map[string]bool)
	for _, r := range a.ListClonedRepos() {
		cloned[r.FullName] = true
	}
	var out []RepoInfo
	for _, r := range ghRepos {
		out = append(out, RepoInfo{
			Name:      r.Name,
			FullName:  r.FullName,
			Description: r.Description,
			Private:   r.Private,
			UpdatedAt: r.UpdatedAt,
			CloneURL:  r.CloneURL,
			HTMLURL:   r.HTMLURL,
			Language:  r.Language,
			IsCloned:  cloned[r.FullName],
		})
	}
	return out, nil
}

// GitHubLoginStatus returns the current GitHub authentication status.
func (a *App) GitHubLoginStatus() string {
	token := github.LoadToken()
	if token == "" {
		return "disconnected"
	}
	return "connected"
}

// ListRemotes returns configured remote workers.
func (a *App) ListRemotes() []map[string]interface{} {
	var out []map[string]interface{}
	for _, r := range a.config.Remotes {
		out = append(out, map[string]interface{}{
			"name":          r.Name,
			"url":           r.URL,
			"maxConcurrent": r.MaxConcurrent,
			"preferFor":     r.PreferFor,
		})
	}
	return out
}

// GetBackgroundTasks returns live subagent status from all sessions.
func (a *App) GetBackgroundTasks() interface{} {
	type taskView struct {
		ID            string `json:"id"`
		Kind          string `json:"kind"`
		Label         string `json:"label"`
		Status        string `json:"status"`
		ToolCalls     int    `json:"toolCalls"`
		LastTool      string `json:"lastTool"`
		LastReasoning string `json:"lastReasoning"`
		Model         string `json:"model"`
		Effort        string `json:"effort"`
	}
	var tasks []taskView
	for _, tab := range a.tabs {
		if tab.ctrl == nil {
			continue
		}
		for _, jv := range tab.ctrl.Jobs() {
			tasks = append(tasks, taskView{
				ID:            jv.ID,
				Kind:          jv.Kind,
				Label:         jv.Label,
				Status:        jv.Status,
				ToolCalls:     jv.ToolCalls,
				LastTool:      jv.LastTool,
				LastReasoning: jv.LastReasoning,
				Model:         jv.Model,
				Effort:        jv.Effort,
			})
		}
	}
	return tasks
}

// SendToSubagent sends a mid-task message to a running background subagent.
func (a *App) SendToSubagent(subagentID, message string) error {
	for _, tab := range a.tabs {
		if tab.ctrl == nil {
			continue
		}
		// Steer via the controller's messenger
		if err := tab.ctrl.SteerSubagent(subagentID, message); err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("subagent %q not found", subagentID)
}

func (a *App) switchWorkspaceTab(path string) error {
	// Switch the current tab's workspace to the cloned repo
	for _, tab := range a.tabs {
		if tab.workspaceRoot == "" || tab.workspaceRoot == a.config.WorkspaceRoot {
			tab.workspaceRoot = path
			return nil
		}
	}
	return nil
}

func parseOwnerRepoDir(name string) (owner, repo string) {
	// Directories are named <owner>-<repo>
	idx := strings.Index(name, "-")
	if idx < 0 {
		return "", name
	}
	return name[:idx], name[idx+1:]
}

func dirModTime(dir string) time.Time {
	info, err := os.Stat(dir)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
