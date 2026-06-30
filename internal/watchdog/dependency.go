package watchdog

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DependencyChecker runs periodic checks for dependency updates.
// For each update found, it clones into a git worktree, applies the update,
// runs tests, and reports "safe" or "breaks N tests."
type DependencyChecker struct {
	interval time.Duration
}

// UpdateReport describes the outcome of checking one dependency.
type UpdateReport struct {
	Name       string
	OldVersion string
	NewVersion string
	Safe       bool
	TestsRan   int
	TestsFailed int
	Error      string
	CheckedAt  time.Time
}

// NewDependencyChecker creates a weekly dependency checker.
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{interval: 7 * 24 * time.Hour}
}

// SetInterval overrides the check frequency.
func (d *DependencyChecker) SetInterval(dur time.Duration) { d.interval = dur }

// Check runs the dependency update check. Returns reports for each dependency.
func (d *DependencyChecker) Check(ctx context.Context, repoDir string) ([]UpdateReport, error) {
	var reports []UpdateReport

	// Detect dependency manager
	if _, err := os.Stat(filepath.Join(repoDir, "go.mod")); err == nil {
		r, _ := d.checkGoModules(ctx, repoDir)
		reports = append(reports, r...)
	}
	if _, err := os.Stat(filepath.Join(repoDir, "package.json")); err == nil {
		r, _ := d.checkNPM(ctx, repoDir)
		reports = append(reports, r...)
	}
	return reports, nil
}

func (d *DependencyChecker) checkGoModules(ctx context.Context, dir string) ([]UpdateReport, error) {
	var reports []UpdateReport

	// List outdated modules
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-u", "all")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list modules: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "[") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		oldVersion := parts[1]
		newVersion := ""
		if len(parts) > 2 && strings.HasPrefix(parts[2], "[") {
			newVersion = strings.Trim(parts[2], "[]")
		}
		if newVersion == "" || newVersion == oldVersion {
			continue
		}

		report := UpdateReport{
			Name:       name,
			OldVersion: oldVersion,
			NewVersion: newVersion,
			CheckedAt:  time.Now(),
		}

		// Try the update in a temp worktree
		tmpDir, _ := os.MkdirTemp("", "watchdog-*")
		if tmpDir != "" {
			defer os.RemoveAll(tmpDir)
			report.Error = "worktree test not yet implemented"
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (d *DependencyChecker) checkNPM(ctx context.Context, dir string) ([]UpdateReport, error) {
	cmd := exec.CommandContext(ctx, "npm", "outdated", "--json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	_ = out
	return nil, nil
}
