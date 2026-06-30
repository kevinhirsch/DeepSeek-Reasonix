// Package watchdog provides dependency monitoring and update safety checking.
//
// Dependency.go implements the dependency watchdog (Issue #20 Comp 13): a
// weekly cron-like checker that scans every dependency for updates. For each
// update found, it clones into a git worktree, applies the update, runs the
// test suite, and reports "safe" or "breaks N tests" for each candidate.
package watchdog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// UpdateStatus classifies the outcome of applying a dependency update.
type UpdateStatus string

const (
	StatusSafe    UpdateStatus = "safe"
	StatusBreaks  UpdateStatus = "breaks"
	StatusSkipped UpdateStatus = "skipped"
	StatusError   UpdateStatus = "error"
)

// UpdateReport records the result of checking one dependency for updates.
type UpdateReport struct {
	Dependency      string     `json:"dependency"`
	CurrentVersion  string     `json:"current_version"`
	LatestVersion   string     `json:"latest_version"`
	UpdateAvailable bool       `json:"update_available"`
	Status          UpdateStatus `json:"status"`
	TestsBroken     int        `json:"tests_broken,omitempty"`
	BrokenTests     []string   `json:"broken_tests,omitempty"`
	Error           string     `json:"error,omitempty"`
	Duration        time.Duration `json:"duration"`
	CheckedAt       time.Time  `json:"checked_at"`
	DiffStat        string     `json:"diff_stat,omitempty"`
}

// CheckSummary aggregates all update reports.
type CheckSummary struct {
	Total      int            `json:"total"`
	Safe       int            `json:"safe"`
	Breaks     int            `json:"breaks"`
	Skipped    int            `json:"skipped"`
	Errors     int            `json:"errors"`
	Reports    []UpdateReport `json:"reports"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	Duration   time.Duration  `json:"duration"`
}

// DependencyChecker is a periodic dependency-update safety checker.
type DependencyChecker struct {
	RepoRoot       string
	WorktreeDir    string
	TestTimeout    time.Duration
	MaxConcurrent  int
	ExcludePrefixes []string
	ReportDir      string

	mu sync.Mutex
}

// NewDependencyChecker creates a checker for the given repository.
func NewDependencyChecker(repoRoot string) *DependencyChecker {
	return &DependencyChecker{
		RepoRoot:      repoRoot,
		TestTimeout:   10 * time.Minute,
		MaxConcurrent: 4,
	}
}

// CheckAll scans all direct dependencies for updates, applies each in an
// isolated git worktree, runs the test suite, and reports the outcome.
func (dc *DependencyChecker) CheckAll(ctx context.Context) (*CheckSummary, error) {
	summary := &CheckSummary{
		StartedAt: time.Now(),
	}

	deps, err := dc.listOutdatedDeps(ctx)
	if err != nil {
		return nil, fmt.Errorf("list outdated dependencies: %w", err)
	}
	if len(deps) == 0 {
		summary.FinishedAt = time.Now()
		summary.Duration = summary.FinishedAt.Sub(summary.StartedAt)
		return summary, nil
	}
	summary.Total = len(deps)

	sem := make(chan struct{}, dc.MaxConcurrent)
	if dc.MaxConcurrent <= 0 {
		sem = make(chan struct{}, 4)
	}

	var wg sync.WaitGroup
	results := make([]UpdateReport, len(deps))

	for i, dep := range deps {
		select {
		case <-ctx.Done():
			wg.Wait()
			return summary, ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(idx int, d depInfo) {
			defer wg.Done()
			defer func() { <-sem }()
			report := dc.checkOneDep(ctx, d)
			dc.mu.Lock()
			results[idx] = report
			dc.mu.Unlock()
		}(i, dep)
	}
	wg.Wait()

	for _, r := range results {
		summary.Reports = append(summary.Reports, r)
		switch r.Status {
		case StatusSafe:
			summary.Safe++
		case StatusBreaks:
			summary.Breaks++
		case StatusSkipped:
			summary.Skipped++
		case StatusError:
			summary.Errors++
		}
	}

	summary.FinishedAt = time.Now()
	summary.Duration = summary.FinishedAt.Sub(summary.StartedAt)

	if dc.ReportDir != "" {
		_ = dc.archiveSummary(summary)
	}

	return summary, nil
}

type depInfo struct {
	Module  string
	Current string
	Latest  string
}

func (dc *DependencyChecker) listOutdatedDeps(ctx context.Context) ([]depInfo, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-u", "-json", "all")
	cmd.Dir = dc.RepoRoot

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, string(bytes.TrimSpace(out)))
	}

	var deps []depInfo
	decoder := json.NewDecoder(bytes.NewReader(out))

	for {
		var mod struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Update   *struct {
				Version string `json:"Version"`
			} `json:"Update,omitempty"`
			Indirect bool `json:"Indirect"`
			Main     bool `json:"Main"`
		}
		if err := decoder.Decode(&mod); err != nil {
			break
		}

		if mod.Main || mod.Indirect {
			continue
		}
		if dc.isExcluded(mod.Path) {
			continue
		}
		if mod.Update == nil {
			continue
		}

		deps = append(deps, depInfo{
			Module:  mod.Path,
			Current: mod.Version,
			Latest:  mod.Update.Version,
		})
	}

	sort.Slice(deps, func(i, j int) bool { return deps[i].Module < deps[j].Module })
	return deps, nil
}

func (dc *DependencyChecker) checkOneDep(ctx context.Context, d depInfo) UpdateReport {
	start := time.Now()
	report := UpdateReport{
		Dependency:      d.Module,
		CurrentVersion:  d.Current,
		LatestVersion:   d.Latest,
		UpdateAvailable: true,
		CheckedAt:       start,
	}

	// Create a git worktree for isolated testing.
	worktreePath, err := dc.createWorktree(ctx, d.Module)
	if err != nil {
		report.Status = StatusError
		report.Error = fmt.Sprintf("create worktree: %v", err)
		report.Duration = time.Since(start)
		return report
	}
	defer dc.removeWorktree(worktreePath)

	// Apply the update.
	updateCmd := exec.CommandContext(ctx, "go", "get", fmt.Sprintf("%s@%s", d.Module, d.Latest))
	updateCmd.Dir = worktreePath
	updateOut, err := updateCmd.CombinedOutput()
	if err != nil {
		report.Status = StatusError
		report.Error = fmt.Sprintf("go get: %v: %s", err, string(updateOut))
		report.Duration = time.Since(start)
		return report
	}

	// Tidy.
	tidyCmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidyCmd.Dir = worktreePath
	_ = tidyCmd.Run()

	// Diff stat.
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--stat")
	diffCmd.Dir = worktreePath
	diffOut, _ := diffCmd.Output()
	report.DiffStat = strings.TrimSpace(string(diffOut))

	// Run tests.
	tCtx := ctx
	if dc.TestTimeout > 0 {
		var cancel context.CancelFunc
		tCtx, cancel = context.WithTimeout(ctx, dc.TestTimeout)
		defer cancel()
	}

	testCmd := exec.CommandContext(tCtx, "go", "test", "./...")
	testCmd.Dir = worktreePath
	testOutBytes, testErr := testCmd.CombinedOutput()
	testOut := string(testOutBytes)

	if testErr == nil {
		report.Status = StatusSafe
		report.Duration = time.Since(start)
		return report
	}

	report.TestsBroken, report.BrokenTests = parseTestFailures(testOut)
	if report.TestsBroken > 0 {
		report.Status = StatusBreaks
	} else {
		report.Status = StatusError
		report.Error = fmt.Sprintf("test error: %v: %s", testErr, testOut)
	}

	report.Duration = time.Since(start)
	return report
}

func (dc *DependencyChecker) createWorktree(ctx context.Context, depModule string) (string, error) {
	if dc.WorktreeDir == "" {
		dc.WorktreeDir = filepath.Join(os.TempDir(), "reasonix-depcheck")
	}
	if err := os.MkdirAll(dc.WorktreeDir, 0755); err != nil {
		return "", err
	}

	safeName := strings.NewReplacer("/", "-", "\\", "-", ":", "-", ".", "-").Replace(depModule)
	safeName = strings.Trim(safeName, "-")
	branchName := fmt.Sprintf("depcheck/%s-%d", safeName, time.Now().UnixNano())
	worktreePath := filepath.Join(dc.WorktreeDir, branchName)

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", worktreePath)
	cmd.Dir = dc.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add: %w: %s", err, string(out))
	}

	return worktreePath, nil
}

func (dc *DependencyChecker) removeWorktree(path string) {
	_ = os.RemoveAll(path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "worktree", "prune")
	cmd.Dir = dc.RepoRoot
	_ = cmd.Run()
}

func (dc *DependencyChecker) isExcluded(modulePath string) bool {
	for _, prefix := range dc.ExcludePrefixes {
		if strings.HasPrefix(modulePath, prefix) {
			return true
		}
	}
	return false
}

func parseTestFailures(output string) (int, []string) {
	lines := strings.Split(output, "\n")
	var failures []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "--- FAIL:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				if idx := strings.Index(name, "("); idx >= 0 {
					name = strings.TrimSpace(name[:idx])
				}
				failures = append(failures, name)
			}
		}
	}

	if len(failures) > 20 {
		failures = failures[:20]
	}

	return len(failures), failures
}

func (dc *DependencyChecker) archiveSummary(summary *CheckSummary) error {
	if err := os.MkdirAll(dc.ReportDir, 0755); err != nil {
		return err
	}
	timestamp := summary.StartedAt.Format("2006-0102-150405")
	path := filepath.Join(dc.ReportDir, fmt.Sprintf("depcheck-%s.json", timestamp))
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Schedule describes a periodic check schedule.
type Schedule struct {
	Interval time.Duration
	LastRun  time.Time
	Enabled  bool
}

// DefaultSchedule returns the default weekly schedule.
func DefaultSchedule() Schedule {
	return Schedule{
		Interval: 7 * 24 * time.Hour,
		Enabled:  true,
	}
}

// IsDue returns true if a check is due according to the schedule.
func (s Schedule) IsDue(now time.Time) bool {
	if !s.Enabled {
		return false
	}
	if s.LastRun.IsZero() {
		return true
	}
	return now.After(s.LastRun.Add(s.Interval))
}
