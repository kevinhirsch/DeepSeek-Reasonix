package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"reasonix/internal/git"
	"reasonix/internal/github"
)

func cloneCommand(args []string) int {
	fs := flag.NewFlagSet("clone", flag.ContinueOnError)
	branch := fs.String("branch", "", "Branch to check out")
	remoteName := fs.String("remote", "", "Remote worker name for clone target")
	dir := fs.String("dir", "", "Local directory (default: ~/reasonix-projects/<owner>-<repo>/)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	url := fs.Arg(0)
	if url == "" {
		fmt.Fprintln(os.Stderr, "Usage: reasonix clone <url> [--branch <name>] [--remote <name>]")
		fmt.Fprintln(os.Stderr, "  Clone a git repository and start a reasonix session in it.")
		return 2
	}

	// Remote target (future)
	if *remoteName != "" {
		fmt.Fprintf(os.Stderr, "Remote clone to %q is not yet supported.\n", *remoteName)
		return 1
	}

	owner, repo, ok := git.ParseGitHubURL(url)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unsupported URL: %s. Use https://github.com/owner/repo\n", url)
		return 1
	}

	cloneDir := *dir
	if cloneDir == "" {
		cloneDir = git.DefaultCloneDir(owner, repo)
	}

	fmt.Printf("Cloning %s/%s...\n", owner, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := git.Clone(ctx, url, cloneDir, *branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Clone failed: %v\n", err)
		return 1
	}

	// Detect project type
	projectType := git.DetectProjectType(result.Dir)

	// Auto-setup: check for agent docs
	if !git.HasAgentDocs(result.Dir) {
		fmt.Println("No AGENTS.md or REASONIX.md found. Running reasonix init...")
		initCmd := exec.Command("reasonix", "init")
		initCmd.Dir = result.Dir
		initCmd.Stdout = os.Stdout
		initCmd.Stderr = os.Stderr
		initCmd.Run() // non-fatal
	}

	// Build context message
	msg := git.SessionContextMessage(result, projectType)
	fmt.Println()
	fmt.Println(msg)

	if result.Cloned {
		fmt.Printf("Repository cloned to %s\n", result.Dir)
	} else {
		fmt.Printf("Existing repository at %s — fetched latest changes\n", result.Dir)
	}
	fmt.Printf("\nStarting reasonix session in %s...\n", result.Dir)

	// Change to the repo directory and start a session
	origDir, _ := os.Getwd()
	os.Chdir(result.Dir)
	defer os.Chdir(origDir)

	return runInteractiveSession(nil)
}

func githubCommand(args []string) int {
	if len(args) == 0 {
		githubUsage()
		return 2
	}
	switch args[0] {
	case "login":
		return githubLoginCommand(args[1:])
	case "repos":
		return githubReposCommand(args[1:])
	case "clone":
		return cloneCommand(args[1:])
	default:
		githubUsage()
		return 2
	}
}

func githubLoginCommand(_ []string) int {
	fmt.Println("Opening GitHub device authorization...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resp, err := github.OAuthLogin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OAuth initialization failed: %v\n", err)
		return 1
	}

	fmt.Printf("\nOpen this URL in your browser:\n  %s\n", resp.VerificationURI)
	fmt.Printf("Enter this code: %s\n\n", resp.UserCode)
	fmt.Println("Waiting for authorization...")

	token, err := github.PollForToken(ctx, resp.DeviceCode, resp.Interval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authorization failed: %v\n", err)
		return 1
	}

	if err := github.SaveToken(token); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save token: %v\n", err)
		return 1
	}

	fmt.Println("✓ GitHub connected. Token stored.")
	return 0
}

func githubReposCommand(args []string) int {
	fs := flag.NewFlagSet("github repos", flag.ContinueOnError)
	max := fs.Int("limit", 30, "Max repos to list")
	starred := fs.Bool("starred", false, "List starred repos")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	token := github.LoadToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run: reasonix github login")
		return 1
	}

	client := github.NewClient(token)
	ctx := context.Background()

	var repos []github.RepoInfo
	var err error
	if *starred {
		repos, err = client.ListStarredRepos(ctx, *max)
	} else {
		repos, err = client.ListUserRepos(ctx, *max)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list repos: %v\n", err)
		return 1
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return 0
	}

	for _, r := range repos {
		visibility := ""
		if r.Private {
			visibility = " 🔒"
		}
		lang := ""
		if r.Language != "" {
			lang = fmt.Sprintf(" [%s]", r.Language)
		}
		fmt.Printf("%-40s %s%s%s\n", r.FullName, lang, visibility, truncateDesc(r.Description, 60))
	}
	return 0
}

func githubUsage() {
	fmt.Println("Usage: reasonix github <subcommand>")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  login        Authorize reasonix with your GitHub account")
	fmt.Println("  repos        List your GitHub repositories")
	fmt.Println("  clone <url>  Clone a GitHub repo and start a session")
}

func truncateDesc(s string, n int) string {
	if s == "" {
		return ""
	}
	if len(s) <= n {
		return " — " + s
	}
	return " — " + s[:n-3] + "..."
}
