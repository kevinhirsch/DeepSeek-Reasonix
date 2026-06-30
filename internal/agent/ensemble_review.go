package agent

import (
	"context"
	"strings"
	"sync"

	"reasonix/internal/event"
)

// EnsembleReviewer spawns 3 reviewers with different philosophies (security,
// correctness, performance) and merges findings. Catches what single reviewers miss.
type EnsembleReviewer struct {
	taskTool *TaskTool
}

// NewEnsembleReviewer creates an ensemble code reviewer.
func NewEnsembleReviewer(taskTool *TaskTool) *EnsembleReviewer {
	return &EnsembleReviewer{taskTool: taskTool}
}

// ReviewLens describes one review perspective.
type ReviewLens struct {
	Name        string
	SystemPrompt string
}

var reviewLenses = []ReviewLens{
	{
		Name: "security",
		SystemPrompt: `You are a security reviewer. Focus exclusively on security issues:
- SQL injection, XSS, path traversal, command injection
- Missing authentication, authorization bypass
- Secret leakage, unsafe defaults, CWE vulnerabilities
Report every finding with file:line references.`,
	},
	{
		Name: "correctness",
		SystemPrompt: `You are a correctness reviewer. Focus exclusively on correctness:
- Null/nil dereference, out-of-bounds access, race conditions
- Logic errors, edge case handling, error propagation
- Type mismatches, incorrect API usage, broken contracts
Report every finding with file:line references.`,
	},
	{
		Name: "performance",
		SystemPrompt: `You are a performance reviewer. Focus exclusively on performance:
- Unnecessary allocations, goroutine leaks, lock contention
- N+1 queries, excessive copying, missing caching opportunities
- Hot-path inefficiencies, memory pressure, CPU waste
Report every finding with file:line references.`,
	},
}

// EnsembleResult contains findings from all review lenses, merged.
type EnsembleResult struct {
	Findings []Finding
	ByLens   map[string][]Finding
}

// Finding is one review observation.
type Finding struct {
	Lens   string
	File   string
	Line   string
	Summary string
}

// Review spawns 3 concurrent reviewers and merges their findings.
func (r *EnsembleReviewer) Review(ctx context.Context, diff string) (*EnsembleResult, error) {
	var wg sync.WaitGroup
	outputs := make([]string, len(reviewLenses))
	errs := make([]error, len(reviewLenses))

	for i, lens := range reviewLenses {
		wg.Add(1)
		go func(idx int, l ReviewLens) {
			defer wg.Done()
			subReg := SubagentToolRegistry(r.taskTool.parentReg, nil)
			sess := NewSession(l.SystemPrompt)
			outputs[idx], errs[idx] = RunSubAgentWithSession(ctx, r.taskTool.prov, subReg, sess,
				"Review these code changes:\n\n"+diff, Options{
					MaxSteps:    15,
					Temperature: r.taskTool.temperature,
					Gate:        r.taskTool.gate,
				}, event.Discard)
		}(i, lens)
	}
	wg.Wait()

	result := &EnsembleResult{ByLens: make(map[string][]Finding)}
	for i, lens := range reviewLenses {
		if errs[i] != nil {
			continue
		}
		findings := extractFindingsFromOutput(outputs[i], lens.Name)
		result.ByLens[lens.Name] = findings
		result.Findings = append(result.Findings, findings...)
	}

	// Deduplicate findings across lenses
	result.Findings = deduplicateFindings(result.Findings)
	return result, nil
}

func extractFindingsFromOutput(output, lens string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			findings = append(findings, Finding{
				Lens:    lens,
				Summary: strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "),
			})
		}
	}
	return findings
}

func deduplicateFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	var out []Finding
	for _, f := range findings {
		key := strings.ToLower(strings.TrimSpace(f.Summary))[:40]
		if !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}
	return out
}
