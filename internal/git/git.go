// Package git provides git repository operations for reasonix: clone,
// fetch, branch listing, commit info, and project-type detection.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Clone clones a repository URL to the given directory. If the directory
// already exists and is a git repo, it fetches the latest instead.
// branch is the branch to check out ("" for default).
func Clone(ctx context.Context, url, dir, branch string) (*CloneResult, error) {
	if isGitRepo(dir) {
		return fetchExisting(ctx, dir, branch)
	}
	return cloneFresh(ctx, url, dir, branch)
}

// CloneResult reports what happened during a clone operation.
type CloneResult struct {
	Dir     string
	Cloned  bool // true if this was a fresh clone, false if existing was fetched
	Branch  string
	Commit  string
	Message string
	Owner   string
	Repo    string
}

func cloneFresh(ctx context.Context, url, dir, branch string) (*CloneResult, error) {
	args := []string{"clone", "--depth=1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dir)

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}

	return buildResult(ctx, dir, true, url)
}

func fetchExisting(ctx context.Context, dir, branch string) (*CloneResult, error) {
	cmd := exec.CommandContext(ctx, "git", "fetch", "--all", "--prune")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git fetch failed: %w", err)
	}
	if branch != "" {
		checkout := exec.CommandContext(ctx, "git", "checkout", branch)
		checkout.Dir = dir
		checkout.Run() // non-fatal: branch may not exist
	}
	return buildResult(ctx, dir, false, "")
}

func buildResult(ctx context.Context, dir string, cloned bool, url string) (*CloneResult, error) {
	branch := currentBranch(dir)
	commit := lastCommitHash(dir)
	message := lastCommitMessage(dir)
	owner, repo := parseOwnerRepo(url, dir)
	return &CloneResult{
		Dir:     dir,
		Cloned:  cloned,
		Branch:  branch,
		Commit:  commit,
		Message: message,
		Owner:   owner,
		Repo:    repo,
	}, nil
}

// IsRepo reports whether the given directory is a git repository.
func IsRepo(dir string) bool {
	return isGitRepo(dir)
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// CurrentBranch returns the checked-out branch name.
func CurrentBranch(dir string) string {
	return currentBranch(dir)
}

func currentBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// LastCommit returns the most recent commit hash and message.
func LastCommit(dir string) (hash, message string) {
	return lastCommitHash(dir), lastCommitMessage(dir)
}

func lastCommitHash(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%H")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))[:12]
}

func lastCommitMessage(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ListBranches returns all branches in the repository.
func ListBranches(dir string) ([]string, error) {
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "remotes/origin/")
		if line == "" || strings.HasPrefix(line, "HEAD ->") {
			continue
		}
		branches = append(branches, line)
	}
	return unique(branches), nil
}

// DefaultBranch returns the default branch name for a repo directory.
func DefaultBranch(dir string) string {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	ref := strings.TrimSpace(string(out))
	return strings.TrimPrefix(ref, "refs/remotes/origin/")
}

// DetectProjectType scans a directory for known project marker files.
func DetectProjectType(dir string) string {
	markers := map[string]string{
		"go.mod":          "Go",
		"package.json":    "Node.js/TypeScript",
		"pyproject.toml":  "Python",
		"setup.py":        "Python",
		"Cargo.toml":      "Rust",
		"Makefile":        "C/C++ (Make)",
		"CMakeLists.txt":  "C/C++ (CMake)",
		"Gemfile":         "Ruby",
		"mix.exs":         "Elixir",
		"pom.xml":         "Java (Maven)",
		"build.gradle":    "Java (Gradle)",
		"rebar.config":    "Erlang",
		"tsconfig.json":   "TypeScript",
	}
	for file, lang := range markers {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return lang
		}
	}
	return "unknown"
}

// HasAgentDocs reports whether the repo has AGENTS.md or REASONIX.md.
func HasAgentDocs(dir string) bool {
	for _, name := range []string{"AGENTS.md", "REASONIX.md", "CLAUDE.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// SessionContextMessage builds the session-start context string.
func SessionContextMessage(r *CloneResult, projectType string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Connected to %s/%s on branch %s.\n", r.Owner, r.Repo, r.Branch)
	fmt.Fprintf(&b, "Last commit: %s — %s.\n", r.Commit, r.Message)
	if projectType != "unknown" {
		fmt.Fprintf(&b, "Project type: %s.\n", projectType)
	}
	return b.String()
}

// DefaultCloneDir returns the standard clone directory path.
func DefaultCloneDir(owner, repo string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return fmt.Sprintf("%s-%s", owner, repo)
	}
	return filepath.Join(home, "reasonix-projects", fmt.Sprintf("%s-%s", owner, repo))
}

// ParseGitHubURL extracts owner and repo from a GitHub URL.
func ParseGitHubURL(url string) (owner, repo string, ok bool) {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "git@github.com:")
	parts := strings.SplitN(url, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseOwnerRepo(cloneURL, dir string) (string, string) {
	if owner, repo, ok := ParseGitHubURL(cloneURL); ok {
		return owner, repo
	}
	// Try to derive from remote
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err == nil {
		if o, r, ok := ParseGitHubURL(strings.TrimSpace(string(out))); ok {
			return o, r
		}
	}
	// Fall back to directory name
	name := filepath.Base(dir)
	return "", name
}

func unique(items []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func init() {
	// Verify git is available
	if _, err := exec.LookPath("git"); err != nil {
		// Package-level functions will return errors for callers to handle
	}
}
