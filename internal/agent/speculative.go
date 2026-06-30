package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// SpeculativeRunner runs the same subagent task N times in parallel and merges
// results by consensus. A finding found by 1/N subagents is "uncertain";
// found by >=threshold/N is "confirmed" (threshold defaults to majority).
type SpeculativeRunner struct {
	taskTool *TaskTool
	runs     int     // default 3
	threshold int    // minimum confirmations needed
}

// NewSpeculativeRunner creates a speculative execution runner.
func NewSpeculativeRunner(taskTool *TaskTool) *SpeculativeRunner {
	return &SpeculativeRunner{taskTool: taskTool, runs: 3, threshold: 2}
}

// WithRuns sets the number of parallel subagent runs.
func (s *SpeculativeRunner) WithRuns(n int) *SpeculativeRunner {
	if n >= 2 {
		s.runs = n
	}
	return s
}

// Run executes the prompt N times and returns consensus output.
func (s *SpeculativeRunner) Run(ctx context.Context, prompt string) (string, error) {
	var wg sync.WaitGroup
	outputs := make([]string, s.runs)
	errs := make([]error, s.runs)

	for i := 0; i < s.runs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			subReg := SubagentToolRegistry(s.taskTool.parentReg, nil)
			sess := NewSession(s.taskTool.sysPrompt)
			outputs[idx], errs[idx] = RunSubAgentWithSession(ctx, s.taskTool.prov, subReg, sess, prompt, Options{
				MaxSteps:  s.taskTool.maxSteps,
				Temperature: s.taskTool.temperature,
				Gate:      s.taskTool.gate,
			}, Discard)
		}(i)
	}
	wg.Wait()

	// Merge by consensus — extract findings from each output
	var allFindings []string
	seen := make(map[string]int) // finding → count across runs
	for i, o := range outputs {
		if errs[i] != nil {
			continue
		}
		for _, f := range extractSpeculativeFindings(o) {
			key := strings.TrimSpace(strings.ToLower(f))
			seen[key]++
			if seen[key] == 1 {
				allFindings = append(allFindings, f)
			}
		}
	}

	// Separate by consensus level
	var confirmed, uncertain []string
	for _, f := range allFindings {
		key := strings.TrimSpace(strings.ToLower(f))
		count := seen[key]
		if count >= s.threshold {
			confirmed = append(confirmed, fmt.Sprintf("CONFIRMED — %d/%d runs: %s", count, s.runs, f))
		} else {
			uncertain = append(uncertain, fmt.Sprintf("UNCERTAIN — %d/%d runs: %s", count, s.runs, f))
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Speculative execution results (%d runs):\n\n", s.runs)
	if len(confirmed) > 0 {
		b.WriteString("## Confirmed\n")
		for _, f := range confirmed {
			b.WriteString("- " + f + "\n")
		}
		b.WriteString("\n")
	}
	if len(uncertain) > 0 {
		b.WriteString("## Uncertain\n")
		for _, f := range uncertain {
			b.WriteString("- " + f + "\n")
		}
	}
	return b.String(), nil
}

// extractSpeculativeFindings extracts bullet-point findings from subagent output.
func extractSpeculativeFindings(output string) []string {
	lines := strings.Split(output, "\n")
	var findings []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			findings = append(findings, strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))
		}
	}
	return findings
}
