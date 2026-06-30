// Package prompt provides prompt evaluation and calibration infrastructure.
//
// Bisect.go implements automatic prompt regression bisection (Issue #19 Comp 5).
// When prompt drift is detected, the BisectRunner performs a binary search over
// the prompt version history to find the exact change responsible. Only the
// breaking change is reverted; later improvements are preserved.
package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"reasonix/internal/provider"
)

// CommitInfo describes one point in the prompt version history.
type CommitInfo struct {
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Role    string    `json:"role"`
}

// BisectResult is the outcome of a prompt regression bisection.
type BisectResult struct {
	// BreakingCommit is the commit that introduced the regression.
	BreakingCommit CommitInfo `json:"breaking_commit"`

	// BreakingMessage describes what changed and why it broke.
	BreakingMessage string `json:"breaking_message"`

	// TotalSteps is the number of evaluation steps needed to find the break.
	TotalSteps int `json:"total_steps"`

	// Found is false when the bisection could not isolate a single breaking change.
	Found bool `json:"found"`

	// RevertAction describes the recommended revert steps.
	RevertAction string `json:"revert_action"`
}

// BisectRunner runs prompt regression bisection to find which change broke an
// evaluation scenario.
type BisectRunner struct {
	// PromptsDir is the root of the prompt version history.
	PromptsDir string

	// ResultsDir stores bisection run artifacts.
	ResultsDir string

	// Provider is used for live evaluation during bisection steps.
	// When nil, mock evaluation is used.
	Provider provider.Provider

	// EvalConfig is the evaluation configuration applied at each bisection step.
	EvalConfig EvalConfig
}

// NewBisectRunner creates a bisect runner keyed to a prompts directory.
func NewBisectRunner(promptsDir, resultsDir string) *BisectRunner {
	return &BisectRunner{
		PromptsDir: promptsDir,
		ResultsDir: resultsDir,
	}
}

// loadVersionHistory reads the version history for a role from JSON.
func (br *BisectRunner) loadVersionHistory(role string) ([]CommitInfo, error) {
	path := filepath.Join(br.PromptsDir, "version_history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load version history: %w", err)
	}

	var all []CommitInfo
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("parse version history: %w", err)
	}

	var roleCommits []CommitInfo
	for _, c := range all {
		if c.Role == role {
			roleCommits = append(roleCommits, c)
		}
	}

	if len(roleCommits) == 0 {
		return nil, fmt.Errorf("no version history found for role %q", role)
	}

	sort.Slice(roleCommits, func(i, j int) bool {
		return roleCommits[i].Date.Before(roleCommits[j].Date)
	})

	return roleCommits, nil
}

// loadPromptAt loads the prompt content for a role at a specific commit hash.
func (br *BisectRunner) loadPromptAt(role, hash string) (string, error) {
	path := filepath.Join(br.PromptsDir, "snapshots", role, hash+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load prompt at %s: %w", hash, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// evaluateAt runs the evaluation suite against the prompt at the given commit.
func (br *BisectRunner) evaluateAt(ctx context.Context, role, hash string) (*EvalRun, error) {
	prompt, err := br.loadPromptAt(role, hash)
	if err != nil {
		return nil, err
	}

	cfg := br.EvalConfig
	cfg.Role = role
	cfg.Prompt = prompt
	cfg.Provider = br.Provider

	return Evaluate(ctx, cfg)
}

// Bisect finds the prompt change that introduced a regression between startGood
// (a commit known to pass) and endBad (a commit known to fail).
func (br *BisectRunner) Bisect(ctx context.Context, startGood, endBad CommitInfo) (*BisectResult, error) {
	if startGood.Role != endBad.Role {
		return nil, fmt.Errorf("start and end commits must be for the same role")
	}

	role := startGood.Role
	allCommits, err := br.loadVersionHistory(role)
	if err != nil {
		return nil, err
	}

	startIdx, endIdx := -1, -1
	for i, c := range allCommits {
		if c.Hash == startGood.Hash {
			startIdx = i
		}
		if c.Hash == endBad.Hash {
			endIdx = i
		}
	}
	if startIdx < 0 || endIdx < 0 {
		return nil, fmt.Errorf("commit not found in version history")
	}
	if startIdx >= endIdx {
		return nil, fmt.Errorf("start commit must precede end commit")
	}

	result := &BisectResult{Found: true}
	lo, hi := startIdx+1, endIdx
	steps := 0

	for lo < hi {
		steps++
		mid := lo + (hi-lo)/2
		commit := allCommits[mid]

		run, evalErr := br.evaluateAt(ctx, role, commit.Hash)
		if evalErr != nil {
			return nil, fmt.Errorf("evaluate at %s (step %d): %w", commit.Hash, steps, evalErr)
		}

		if run.PassRate >= 100.0 || run.Failed == 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	steps++
	result.TotalSteps = steps

	if lo > endIdx {
		result.Found = false
		result.BreakingMessage = "no failing commit found in the range"
		return result, nil
	}

	breaking := allCommits[lo]
	result.BreakingCommit = breaking
	result.BreakingMessage = fmt.Sprintf(
		"Commit %s by %s on %s broke the %s prompt: %s",
		truncHash(breaking.Hash),
		breaking.Author,
		breaking.Date.Format("2006-01-02 15:04"),
		role,
		breaking.Message,
	)

	snapshotPath := filepath.Join(br.PromptsDir, "snapshots", role, startGood.Hash+".md")
	currentPath := filepath.Join(br.PromptsDir, "v1", role+".md")
	if _, err := os.Stat(snapshotPath); err == nil {
		result.RevertAction = fmt.Sprintf(
			"Restore pre-break prompt: cp %s %s; then re-apply later commits after %s and re-run calibration.",
			snapshotPath, currentPath, truncHash(breaking.Hash),
		)
	} else {
		result.RevertAction = fmt.Sprintf(
			"No snapshot available. Revert commit %s and re-run calibration.", truncHash(breaking.Hash),
		)
	}

	if br.ResultsDir != "" {
		_ = br.archiveResult(result)
	}

	return result, nil
}

func (br *BisectRunner) archiveResult(result *BisectResult) error {
	if err := os.MkdirAll(br.ResultsDir, 0755); err != nil {
		return err
	}
	timestamp := time.Now().Format("2006-0102-150405")
	filename := fmt.Sprintf("bisect-%s.json", timestamp)
	path := filepath.Join(br.ResultsDir, filename)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// FindCommitByHash locates a commit in the version history for a role.
func (br *BisectRunner) FindCommitByHash(role, hash string) (CommitInfo, bool) {
	commits, err := br.loadVersionHistory(role)
	if err != nil {
		return CommitInfo{}, false
	}
	for _, c := range commits {
		if c.Hash == hash || strings.HasPrefix(c.Hash, hash) {
			return c, true
		}
	}
	return CommitInfo{}, false
}

func truncHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}
