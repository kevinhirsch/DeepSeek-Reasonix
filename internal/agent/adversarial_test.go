package agent

import (
	"context"
	"fmt"
	"strings"
)

// AdversarialTester implements Comp 19: after feature implementation, spawns
// a tester subagent (Pro, max) whose goal is to BREAK the code. Writes tests
// exploiting edge cases. Executor fixes. Loop until tester finds nothing.
type AdversarialTester struct {
	taskTool *TaskTool
	maxRounds int
}

// NewAdversarialTester creates an adversarial tester.
func NewAdversarialTester(taskTool *TaskTool) *AdversarialTester {
	return &AdversarialTester{taskTool: taskTool, maxRounds: 5}
}

// TestResult reports one round of adversarial testing.
type TestResult struct {
	Round       int
	TesterReport string
	FoundBreak   bool
	FixApplied   string
}

// Run executes the adversarial test-fix loop until no breaks are found.
func (t *AdversarialTester) Run(ctx context.Context, featureDescription, codeDiff string) ([]TestResult, error) {
	var results []TestResult

	for round := 0; round < t.maxRounds; round++ {
		// Phase 1: Tester tries to break it
		testerPrompt := fmt.Sprintf(
			"Your job is to BREAK this code. Find edge cases, error conditions, "+
			"and unexpected inputs that would cause the following feature to fail.\n\n"+
			"## Feature\n%s\n\n## Code\n%s\n\n"+
			"Write specific test cases that would break this code. Be adversarial.",
			featureDescription, codeDiff)

		subReg := SubagentToolRegistry(t.taskTool.parentReg, nil)
		sess := NewSession(DefaultVerifierPrompt)
		testerReport, err := RunSubAgentWithSession(ctx, t.taskTool.prov, subReg, sess,
			testerPrompt, Options{
				MaxSteps:   10,
				Temperature: t.taskTool.temperature,
				Gate:       t.taskTool.gate,
				IsDeepSeek: t.taskTool.isDeepSeek,
			}, event.Discard)

		result := TestResult{Round: round + 1}
		if err != nil {
			result.TesterReport = fmt.Sprintf("ERROR: %v", err)
			results = append(results, result)
			continue
		}
		result.TesterReport = testerReport

		// Check if tester found any breaks
		if isNoBreaksFound(testerReport) {
			results = append(results, result)
			break
		}
		result.FoundBreak = true
		results = append(results, result)
	}
	return results, nil
}

func isNoBreaksFound(report string) bool {
	lower := strings.ToLower(report)
	noBreakMarkers := []string{
		"no issues found", "no breaks", "could not break",
		"no edge cases", "all tests pass", "appears correct",
	}
	for _, m := range noBreakMarkers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}
