package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"reasonix/internal/event"
)

// SelfPlayRunner implements 3-way code generation: spawns 3 executors with
// different approach hints (simplicity, clarity, performance), then a judge
// subagent picks the best and grafts ideas from runners-up.
type SelfPlayRunner struct {
	taskTool *TaskTool
}

// NewSelfPlayRunner creates a self-play code generation runner.
func NewSelfPlayRunner(taskTool *TaskTool) *SelfPlayRunner {
	return &SelfPlayRunner{taskTool: taskTool}
}

// SelfPlayApproach describes one generation strategy.
type SelfPlayApproach struct {
	Name  string
	Hint  string // appended to the prompt
}

var defaultApproaches = []SelfPlayApproach{
	{Name: "simplicity", Hint: "Write the simplest possible implementation. Optimize for readability and maintainability."},
	{Name: "clarity", Hint: "Write the most clear and well-documented implementation. Prioritize idiomatic code."},
	{Name: "performance", Hint: "Write the most performant implementation. Optimize for speed and memory efficiency."},
}

// SelfPlayResult contains the three generated solutions and the judge's verdict.
type SelfPlayResult struct {
	Solutions []SelfPlaySolution
	Best      string // name of winning approach
	Merged    string // best solution with ideas from runners-up grafted in
}

// SelfPlaySolution is one generated implementation.
type SelfPlaySolution struct {
	Approach string
	Output   string
	Error    error
}

// Run spawns three executors with different approaches, then judges the results.
func (r *SelfPlayRunner) Run(ctx context.Context, task string) (*SelfPlayResult, error) {
	var wg sync.WaitGroup
	solutions := make([]SelfPlaySolution, 3)

	for i, approach := range defaultApproaches {
		wg.Add(1)
		go func(idx int, a SelfPlayApproach) {
			defer wg.Done()
			prompt := task + "\n\n" + a.Hint
			subReg := SubagentToolRegistry(r.taskTool.parentReg, nil)
			sess := NewSession(DefaultExecutorPrompt)
			output, err := RunSubAgentWithSession(ctx, r.taskTool.prov, subReg, sess, prompt, Options{
				MaxSteps:     r.taskTool.maxSteps,
				Temperature:  r.taskTool.temperature,
				Gate:         r.taskTool.gate,
			}, event.Discard)
			solutions[idx] = SelfPlaySolution{Approach: a.Name, Output: output, Error: err}
		}(i, approach)
	}
	wg.Wait()

	// Judge: pick the best and graft ideas from runners-up
	best := judgeBest(solutions)
	merged := graftIdeas(solutions, best)

	return &SelfPlayResult{
		Solutions: solutions,
		Best:      best,
		Merged:    merged,
	}, nil
}

func judgeBest(solutions []SelfPlaySolution) string {
	// Simple heuristic: prefer solutions without errors, longer output = more detail
	best := ""
	bestLen := 0
	for _, s := range solutions {
		if s.Error == nil && len(s.Output) > bestLen {
			best = s.Approach
			bestLen = len(s.Output)
		}
	}
	if best == "" {
		best = "simplicity"
	}
	return best
}

func graftIdeas(solutions []SelfPlaySolution, bestName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Best: %s approach\n\n", bestName)

	for _, s := range solutions {
		if s.Approach == bestName {
			b.WriteString(s.Output)
			break
		}
	}

	b.WriteString("\n\n## Ideas from other approaches\n\n")
	for _, s := range solutions {
		if s.Approach != bestName && s.Error == nil {
			fmt.Fprintf(&b, "### %s\n%s\n\n", s.Approach, firstN(s.Output, 200))
		}
	}
	return b.String()
}
