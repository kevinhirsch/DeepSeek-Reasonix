// Package agent provides the agent execution engine.
//
// Selfplay.go implements self-play code generation (Issue #20 Comp 14). For
// feature implementation requests, the SelfPlayRunner spawns three executor
// subagents, each given a different approach hint (simplicity, clarity,
// performance). A fourth judge subagent evaluates all three implementations,
// picks the best one, and grafts useful ideas from the runners-up into the
// chosen implementation.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// ApproachHint guides an executor subagent toward a specific implementation
// strategy.
type ApproachHint string

const (
	HintSimplicity  ApproachHint = "simplicity"
	HintClarity     ApproachHint = "clarity"
	HintPerformance ApproachHint = "performance"
)

// selfPlayExecResult holds one executor's output.
type selfPlayExecResult struct {
	Hint   ApproachHint
	Output string
	Err    error
}

// SelfPlayResult is the final output of a self-play code generation run.
type SelfPlayResult struct {
	SelectedHint   ApproachHint            `json:"selected_hint"`
	SelectedCode   string                  `json:"selected_code"`
	JudgeRationale string                  `json:"judge_rationale"`
	GraftedIdeas   []string                `json:"grafted_ideas,omitempty"`
	AllOutputs     map[ApproachHint]string `json:"all_outputs,omitempty"`
}

// SelfPlayConfig configures a self-play run.
type SelfPlayConfig struct {
	FeatureDescription string
	ContextFiles       []string
	Model              string
	Effort             string
	MaxSteps           int
}

// SelfPlayRunner spawns multiple executor subagents with different approach
// hints, then uses a judge subagent to select the best result.
type SelfPlayRunner struct {
	prov    provider.Provider
	pricing *provider.Pricing
	parentReg *tool.Registry
}

// NewSelfPlayRunner creates a self-play runner.
func NewSelfPlayRunner(prov provider.Provider, pricing *provider.Pricing, parentReg *tool.Registry) *SelfPlayRunner {
	return &SelfPlayRunner{
		prov:      prov,
		pricing:   pricing,
		parentReg: parentReg,
	}
}

// Run executes the self-play pipeline: 3 executors + 1 judge.
func (sr *SelfPlayRunner) Run(ctx context.Context, cfg SelfPlayConfig) (*SelfPlayResult, error) {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 15
	}

	hints := []ApproachHint{HintSimplicity, HintClarity, HintPerformance}

	var wg sync.WaitGroup
	results := make([]selfPlayExecResult, len(hints))
	sink := event.Discard

	for i, hint := range hints {
		wg.Add(1)
		go func(idx int, h ApproachHint) {
			defer wg.Done()
			prompt := sr.buildExecutorPrompt(cfg.FeatureDescription, h, cfg.ContextFiles)
			output, err := sr.runSubagent(ctx, prompt, cfg.MaxSteps, sink)
			results[idx] = selfPlayExecResult{
				Hint:   h,
				Output: strings.TrimSpace(output),
				Err:    err,
			}
		}(i, hint)
	}
	wg.Wait()

	successful := make(map[ApproachHint]string)
	for _, r := range results {
		if r.Err == nil && r.Output != "" {
			successful[r.Hint] = r.Output
		}
	}
	if len(successful) == 0 {
		return nil, fmt.Errorf("all executors failed")
	}

	bestResult := &SelfPlayResult{
		AllOutputs: successful,
	}

	if len(successful) == 1 {
		for hint, output := range successful {
			bestResult.SelectedHint = hint
			bestResult.SelectedCode = output
			bestResult.JudgeRationale = fmt.Sprintf("Only the %s executor succeeded.", hint)
		}
		return bestResult, nil
	}

	// Phase 2: Judge evaluates all successful outputs.
	judgePrompt := sr.buildJudgePrompt(cfg.FeatureDescription, successful)
	judgeOutput, err := sr.runSubagent(ctx, judgePrompt, cfg.MaxSteps, sink)
	if err != nil {
		for hint, output := range successful {
			bestResult.SelectedHint = hint
			bestResult.SelectedCode = output
			bestResult.JudgeRationale = fmt.Sprintf("Judge failed (%v); fell back to %s.", err, hint)
		}
		return bestResult, nil
	}

	selectedHint, rationale, grafted := parseSelfPlayJudgeOutput(judgeOutput, successful)
	bestResult.SelectedHint = selectedHint
	bestResult.SelectedCode = successful[selectedHint]
	bestResult.JudgeRationale = rationale
	bestResult.GraftedIdeas = grafted

	return bestResult, nil
}

func (sr *SelfPlayRunner) buildExecutorPrompt(feature string, hint ApproachHint, contextFiles []string) string {
	var b strings.Builder

	b.WriteString("Implement the following feature. Follow the approach directive precisely.\n\n")
	b.WriteString("## Feature\n\n")
	b.WriteString(feature)
	b.WriteString("\n\n## Approach: ")
	b.WriteString(string(hint))
	b.WriteString("\n\n")

	switch hint {
	case HintSimplicity:
		b.WriteString("Use the SIMPLEST possible solution. Minimal dependencies, straightforward logic, fewest lines that correctly solve the problem. Avoid over-engineering. Prefer standard library.\n")
	case HintClarity:
		b.WriteString("Create the MOST READABLE solution. Self-documenting code, clear variable names, well-structured functions. The code should be immediately understandable. Favor explicit over clever.\n")
	case HintPerformance:
		b.WriteString("Create the FASTEST possible solution. Efficient data structures, minimal allocations, optimal algorithms. Profile-aware. Prefer performance over brevity.\n")
	}

	if len(contextFiles) > 0 {
		b.WriteString("\n## Context Files\n\n")
		for _, f := range contextFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		b.WriteString("\nRead these files before implementing.\n")
	}

	b.WriteString("\nOutput ONLY the implementation code. No markdown fences or commentary.\n")
	return b.String()
}

func (sr *SelfPlayRunner) buildJudgePrompt(feature string, candidates map[ApproachHint]string) string {
	var b strings.Builder

	b.WriteString("Evaluate three implementations of the same feature. Select the BEST and identify useful ideas from others to graft in.\n\n")
	b.WriteString("## Feature\n\n")
	b.WriteString(feature)
	b.WriteString("\n\n")

	for _, hint := range []ApproachHint{HintSimplicity, HintClarity, HintPerformance} {
		code, ok := candidates[hint]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n```\n%s\n```\n\n", hint, code)
	}

	b.WriteString("## Judgment Format\n\n")
	b.WriteString("SELECTED: <simplicity|clarity|performance>\n")
	b.WriteString("RATIONALE: <2-3 sentences>\n")
	b.WriteString("GRAFTED:\n")
	b.WriteString("- <idea or \"none\">\n")
	return b.String()
}

func (sr *SelfPlayRunner) runSubagent(ctx context.Context, prompt string, maxSteps int, sink event.Sink) (string, error) {
	subReg := SubagentToolRegistry(sr.parentReg, nil)
	sess := NewSession(DefaultExecutorPrompt)

	return RunSubAgentWithSession(ctx, sr.prov, subReg, sess, prompt, Options{
		MaxSteps:    maxSteps,
		Temperature: 0.4,
		Pricing:     sr.pricing,
	}, sink)
}

func parseSelfPlayJudgeOutput(output string, candidates map[ApproachHint]string) (ApproachHint, string, []string) {
	lines := strings.Split(output, "\n")

	var selected ApproachHint
	var rationale string
	var grafted []string

	currentSection := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "SELECTED:"):
			choice := strings.TrimSpace(strings.TrimPrefix(trimmed, "SELECTED:"))
			switch ApproachHint(strings.ToLower(choice)) {
			case HintSimplicity:
				selected = HintSimplicity
			case HintClarity:
				selected = HintClarity
			case HintPerformance:
				selected = HintPerformance
			}

		case strings.HasPrefix(trimmed, "RATIONALE:"):
			rationale = strings.TrimSpace(strings.TrimPrefix(trimmed, "RATIONALE:"))
			currentSection = "rationale"

		case strings.HasPrefix(trimmed, "GRAFTED:"):
			currentSection = "grafted"

		case strings.HasPrefix(trimmed, "-") && currentSection == "grafted":
			idea := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if idea != "" && strings.ToLower(idea) != "none" {
				grafted = append(grafted, idea)
			}

		case currentSection == "rationale" && trimmed != "":
			rationale += " " + trimmed
		}
	}

	if selected == "" {
		for _, h := range []ApproachHint{HintSimplicity, HintClarity, HintPerformance} {
			if _, ok := candidates[h]; ok {
				selected = h
				break
			}
		}
	}
	if rationale == "" {
		rationale = fmt.Sprintf("Selected %s by automated evaluation.", selected)
	}

	return selected, rationale, grafted
}
