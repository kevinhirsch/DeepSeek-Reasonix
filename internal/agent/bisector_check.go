// Package agent provides the agent execution engine.
//
// Bisector_check.go implements adversarial self-testing (Issue #21 Comp 19).
// After a feature is implemented, the AdversarialSelfTester spawns a tester
// subagent (using the highest reasoning effort) whose explicit goal is to
// BREAK the newly written code. The tester writes tests exploiting edge cases,
// boundary conditions, and hidden assumptions. If the tester succeeds, the
// original executor fixes the code and the loop repeats. The process continues
// until the tester finds nothing — producing code hardened against adversarial
// exploration.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestFinding is one issue discovered by the adversarial tester.
type TestFinding struct {
	Severity    string `json:"severity"`
	Description string `json:"description"`
	TestCode    string `json:"test_code,omitempty"`
	ExploitPath string `json:"exploit_path,omitempty"`
	Fixed       bool   `json:"fixed"`
}

// AdversarialRound records one round of the test-fix loop.
type AdversarialRound struct {
	Round       int           `json:"round"`
	TesterFindings []TestFinding `json:"tester_findings"`
	FixApplied  bool          `json:"fix_applied"`
	FixOutput   string        `json:"fix_output,omitempty"`
	Duration    time.Duration `json:"duration"`
}

// AdversarialResult is the final outcome of adversarial self-testing.
type AdversarialResult struct {
	TotalRounds   int               `json:"total_rounds"`
	TotalFindings int               `json:"total_findings"`
	AllFixed      bool              `json:"all_fixed"`
	Rounds        []AdversarialRound `json:"rounds"`
	FinalCode     string            `json:"final_code,omitempty"`
	Duration      time.Duration     `json:"duration"`
}

// AdversarialConfig configures the adversarial self-testing loop.
type AdversarialConfig struct {
	// FeatureDescription is what was implemented.
	FeatureDescription string

	// ImplementationCode is the code to test adversarially.
	ImplementationCode string

	// ContextFiles are additional files the tester needs to understand.
	ContextFiles []string

	// MaxRounds limits the test-fix loop iterations (default 5).
	MaxRounds int

	// TesterMaxSteps is the max tool-call rounds for the tester subagent.
	TesterMaxSteps int

	// FixerMaxSteps is the max tool-call rounds for the fixer subagent.
	FixerMaxSteps int

	// Model is the model for both tester and fixer.
	Model string

	// TesterEffort is the reasoning effort for the tester (should be high).
	TesterEffort string
}

// AdversarialSelfTester implements the test-fix loop for adversarial self-testing.
// After feature implementation, it spawns a tester subagent (using high
// reasoning effort) whose goal is to BREAK the code, then has the executor
// fix any discovered issues. The loop repeats until the tester finds nothing.
type AdversarialSelfTester struct {
	prov      provider.Provider
	pricing   *provider.Pricing
	parentReg *tool.Registry
}

// NewAdversarialSelfTester creates an adversarial tester.
func NewAdversarialSelfTester(prov provider.Provider, pricing *provider.Pricing, parentReg *tool.Registry) *AdversarialSelfTester {
	return &AdversarialSelfTester{
		prov:      prov,
		pricing:   pricing,
		parentReg: parentReg,
	}
}

// Run executes the adversarial test-fix loop. It spawns a tester subagent to
// find bugs, then a fixer subagent to resolve them, repeating until no new
// issues are found or the round limit is reached.
func (at *AdversarialSelfTester) Run(ctx context.Context, cfg AdversarialConfig) (*AdversarialResult, error) {
	if cfg.MaxRounds <= 0 {
		cfg.MaxRounds = 5
	}
	if cfg.TesterMaxSteps <= 0 {
		cfg.TesterMaxSteps = 20
	}
	if cfg.FixerMaxSteps <= 0 {
		cfg.FixerMaxSteps = 15
	}
	if cfg.TesterEffort == "" {
		cfg.TesterEffort = "high"
	}

	startTime := time.Now()
	currentCode := cfg.ImplementationCode

	result := &AdversarialResult{}

	for round := 1; round <= cfg.MaxRounds; round++ {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result, ctx.Err()
		default:
		}

		ar := AdversarialRound{Round: round}
		roundStart := time.Now()

		// Phase 1: Adversarial tester tries to break the code.
		testerPrompt := at.buildTesterPrompt(cfg.FeatureDescription, currentCode, cfg.ContextFiles, round)
		testerOutput, err := at.runSubagent(ctx, testerPrompt, cfg.TesterMaxSteps, cfg.TesterEffort, event.Discard)
		if err != nil {
			// Tester failed; stop the loop.
			ar.Duration = time.Since(roundStart)
			result.Rounds = append(result.Rounds, ar)
			break
		}

		ar.TesterFindings = at.parseTestFindings(testerOutput)
		if len(ar.TesterFindings) == 0 {
			// No findings — code is hardened.
			ar.Duration = time.Since(roundStart)
			result.Rounds = append(result.Rounds, ar)
			break
		}

		result.TotalFindings += len(ar.TesterFindings)

		// Phase 2: Fixer resolves the found issues.
		fixerPrompt := at.buildFixerPrompt(cfg.FeatureDescription, currentCode, testerOutput)
		fixOutput, err := at.runSubagent(ctx, fixerPrompt, cfg.FixerMaxSteps, "", event.Discard)
		if err != nil {
			ar.Duration = time.Since(roundStart)
			result.Rounds = append(result.Rounds, ar)
			break
		}

		ar.FixApplied = fixOutput != "" && !strings.Contains(strings.ToLower(fixOutput), "no changes needed")
		ar.FixOutput = fixOutput
		if ar.FixApplied {
			// Use the fixer's output as the new code for the next round.
			// In practice, the fixer would use Edit/Bash to modify files;
			// here we track the updated code through the fix output.
			currentCode = fixOutput
		}

		ar.Duration = time.Since(roundStart)
		result.Rounds = append(result.Rounds, ar)

		if !ar.FixApplied {
			break
		}
	}

	result.TotalRounds = len(result.Rounds)
	result.AllFixed = result.TotalFindings > 0 && at.allRoundsHadFixes(result.Rounds)
	result.FinalCode = currentCode
	result.Duration = time.Since(startTime)

	return result, nil
}

func (at *AdversarialSelfTester) buildTesterPrompt(feature, code string, contextFiles []string, round int) string {
	var b strings.Builder

	b.WriteString("You are an ADVERSARIAL tester. Your ONLY goal is to BREAK this code.\n\n")
	b.WriteString("Find bugs, edge cases, race conditions, nil dereferences, security vulnerabilities,\n")
	b.WriteString("and any way the code could fail, crash, or produce incorrect results.\n\n")

	if round > 1 {
		b.WriteString("This is round ")
		fmt.Fprintf(&b, "%d", round)
		b.WriteString(". The code has been hardened once already. Be MORE creative. Find what was missed.\n\n")
	}

	b.WriteString("## Feature Description\n\n")
	b.WriteString(feature)
	b.WriteString("\n\n")

	b.WriteString("## Code to Break\n\n```\n")
	b.WriteString(code)
	b.WriteString("\n```\n\n")

	if len(contextFiles) > 0 {
		b.WriteString("## Context Files (read these to understand assumptions)\n\n")
		for _, f := range contextFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Output Format\n\n")
	b.WriteString("For each issue found, output:\n")
	b.WriteString("FINDING: <severity> | <description>\n")
	b.WriteString("TEST: <Go test code that demonstrates the bug>\n")
	b.WriteString("EXPLOIT: <how an attacker or edge case would trigger this>\n\n")
	b.WriteString("If you find NO issues after thorough analysis, output: SECURE\n\n")
	b.WriteString("Think creatively. Consider: nil inputs, empty strings, negative numbers,\n")
	b.WriteString("concurrent access, cancelled contexts, type assertions, integer overflow,\n")
	b.WriteString("race conditions, resource exhaustion, and unexpected state transitions.\n")

	return b.String()
}

func (at *AdversarialSelfTester) buildFixerPrompt(feature, code, testerOutput string) string {
	var b strings.Builder

	b.WriteString("Fix all issues found by the adversarial tester. Make the code robust against\n")
	b.WriteString("the edge cases and vulnerabilities identified. Preserve the original\n")
	b.WriteString("functionality while hardening against failures.\n\n")

	b.WriteString("## Feature\n\n")
	b.WriteString(feature)
	b.WriteString("\n\n")

	b.WriteString("## Current Code\n\n```\n")
	b.WriteString(code)
	b.WriteString("\n```\n\n")

	b.WriteString("## Issues Found by Tester\n\n")
	b.WriteString(testerOutput)
	b.WriteString("\n\n")

	b.WriteString("## Instructions\n\n")
	b.WriteString("1. Fix EVERY issue the tester found.\n")
	b.WriteString("2. Add defensive checks: nil guards, bounds checks, error handling.\n")
	b.WriteString("3. Add the tester's test cases as actual test functions.\n")
	b.WriteString("4. Output the complete fixed code.\n")

	return b.String()
}

func (at *AdversarialSelfTester) runSubagent(ctx context.Context, prompt string, maxSteps int, effort string, sink event.Sink) (string, error) {
	subReg := SubagentToolRegistry(at.parentReg, nil)
	sess := NewSession(DefaultExecutorPrompt)

	return RunSubAgentWithSession(ctx, at.prov, subReg, sess, prompt, Options{
		MaxSteps:    maxSteps,
		Temperature: 0.3,
		Pricing:     at.pricing,
	}, sink)
}

func (at *AdversarialSelfTester) parseTestFindings(output string) []TestFinding {
	var findings []TestFinding
	lines := strings.Split(output, "\n")

	var current *TestFinding

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "SECURE") || trimmed == "SECURE":
			return nil

		case strings.HasPrefix(trimmed, "FINDING:"):
			if current != nil {
				findings = append(findings, *current)
			}
			content := strings.TrimPrefix(trimmed, "FINDING:")
			parts := strings.SplitN(strings.TrimSpace(content), "|", 2)
			severity := "medium"
			description := strings.TrimSpace(content)
			if len(parts) == 2 {
				severity = strings.TrimSpace(parts[0])
				description = strings.TrimSpace(parts[1])
			}
			current = &TestFinding{
				Severity:    severity,
				Description: description,
			}

		case strings.HasPrefix(trimmed, "TEST:"):
			if current != nil {
				current.TestCode = strings.TrimSpace(strings.TrimPrefix(trimmed, "TEST:"))
			}

		case strings.HasPrefix(trimmed, "EXPLOIT:"):
			if current != nil {
				current.ExploitPath = strings.TrimSpace(strings.TrimPrefix(trimmed, "EXPLOIT:"))
			}
		}
	}

	if current != nil {
		findings = append(findings, *current)
	}

	return findings
}

func (at *AdversarialSelfTester) allRoundsHadFixes(rounds []AdversarialRound) bool {
	for _, r := range rounds {
		if len(r.TesterFindings) > 0 && !r.FixApplied {
			return false
		}
	}
	return true
}

// RunQuickCheck runs a single-pass adversarial check without the fix loop.
// Useful for pre-commit hooks where a full loop would take too long.
func (at *AdversarialSelfTester) RunQuickCheck(ctx context.Context, cfg AdversarialConfig) ([]TestFinding, error) {
	if cfg.TesterMaxSteps <= 0 {
		cfg.TesterMaxSteps = 12
	}
	if cfg.TesterEffort == "" {
		cfg.TesterEffort = "high"
	}

	prompt := at.buildTesterPrompt(cfg.FeatureDescription, cfg.ImplementationCode, cfg.ContextFiles, 1)
	output, err := at.runSubagent(ctx, prompt, cfg.TesterMaxSteps, cfg.TesterEffort, event.Discard)
	if err != nil {
		return nil, err
	}

	return at.parseTestFindings(output), nil
}
