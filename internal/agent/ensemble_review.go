// Package agent provides the agent execution engine.
//
// Ensemble_review.go implements ensemble code review (Issue #20 Comp 15). On
// pre-commit or on demand, the EnsembleReviewer spawns three reviewer
// subagents — each focused on a different dimension (security, correctness,
// performance). A fourth synthesizer subagent merges their findings into a
// single unified review, catching issues that any single reviewer would miss.
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

// ReviewDimension names one focus area for an ensemble reviewer.
type ReviewDimension string

const (
	DimSecurity    ReviewDimension = "security"
	DimCorrectness ReviewDimension = "correctness"
	DimPerformance ReviewDimension = "performance"
)

var allDims = []ReviewDimension{DimSecurity, DimCorrectness, DimPerformance}

// singleRevResult captures one reviewer's output.
type singleRevResult struct {
	Dimension ReviewDimension
	Findings  string
	Err       error
}

// EnsembleFinding is one issue found during ensemble review.
type EnsembleFinding struct {
	Dimension      ReviewDimension `json:"dimension"`
	Severity       string          `json:"severity"`
	File           string          `json:"file"`
	Line           int             `json:"line,omitempty"`
	Description    string          `json:"description"`
	Recommendation string          `json:"recommendation"`
}

// EnsembleResult is the unified output of an ensemble code review.
type EnsembleResult struct {
	Findings         []EnsembleFinding            `json:"findings"`
	SecurityCount    int                          `json:"security_count"`
	CorrectnessCount int                          `json:"correctness_count"`
	PerformanceCount int                          `json:"performance_count"`
	Summary          string                       `json:"summary"`
	RawReviews       map[ReviewDimension]string   `json:"raw_reviews,omitempty"`
}

// EnsembleReviewConfig configures an ensemble review run.
type EnsembleReviewConfig struct {
	Diff             string
	Model            string
	Effort           string
	MaxSteps         int
	ExtraInstructions string
}

// EnsembleReviewer runs ensemble code review by spawning three specialist
// reviewer subagents and a synthesizer that merges their findings.
type EnsembleReviewer struct {
	prov      provider.Provider
	pricing   *provider.Pricing
	parentReg *tool.Registry
}

// NewEnsembleReviewer creates an ensemble reviewer.
func NewEnsembleReviewer(prov provider.Provider, pricing *provider.Pricing, parentReg *tool.Registry) *EnsembleReviewer {
	return &EnsembleReviewer{
		prov:      prov,
		pricing:   pricing,
		parentReg: parentReg,
	}
}

// Review runs the full ensemble review pipeline.
func (er *EnsembleReviewer) Review(ctx context.Context, cfg EnsembleReviewConfig) (*EnsembleResult, error) {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}

	sink := event.Discard

	// Phase 1: Three specialist reviewers in parallel.
	var wg sync.WaitGroup
	revResults := make([]singleRevResult, len(allDims))

	for i, dim := range allDims {
		wg.Add(1)
		go func(idx int, d ReviewDimension) {
			defer wg.Done()
			prompt := er.buildReviewerPrompt(cfg.Diff, d, cfg.ExtraInstructions)
			output, err := er.runSubagent(ctx, prompt, cfg.MaxSteps, sink)
			revResults[idx] = singleRevResult{
				Dimension: d,
				Findings:  strings.TrimSpace(output),
				Err:       err,
			}
		}(i, dim)
	}
	wg.Wait()

	rawReviews := make(map[ReviewDimension]string)
	for _, r := range revResults {
		if r.Err == nil && r.Findings != "" {
			rawReviews[r.Dimension] = r.Findings
		}
	}
	if len(rawReviews) == 0 {
		return nil, fmt.Errorf("all reviewers failed")
	}

	// Phase 2: Synthesizer merges findings.
	synthPrompt := er.buildSynthesizerPrompt(cfg.Diff, rawReviews)
	synthOutput, err := er.runSubagent(ctx, synthPrompt, cfg.MaxSteps, sink)
	if err != nil {
		return er.buildFallbackResult(rawReviews), nil
	}

	return er.buildResult(rawReviews, synthOutput), nil
}

func (er *EnsembleReviewer) buildReviewerPrompt(diff string, dim ReviewDimension, extra string) string {
	var b strings.Builder

	b.WriteString("You are a specialist code reviewer focused EXCLUSIVELY on ")
	switch dim {
	case DimSecurity:
		b.WriteString("SECURITY. Find vulnerabilities, injection risks, unsafe deserialization, auth bypasses, exposed secrets, path traversal, data leakage. Only report security issues.\n")
	case DimCorrectness:
		b.WriteString("CORRECTNESS. Find logic bugs, off-by-one errors, nil dereferences, race conditions, missing error handling, edge cases, contract violations. Only report correctness issues.\n")
	case DimPerformance:
		b.WriteString("PERFORMANCE. Find algorithmic inefficiency, excessive allocations, unnecessary blocking, missing caching, lock contention, redundant work. Only report performance issues.\n")
	}

	if extra != "" {
		b.WriteString("\nExtra instructions: ")
		b.WriteString(extra)
		b.WriteString("\n")
	}

	b.WriteString("\n## Code to Review\n\n```diff\n")
	b.WriteString(truncForReview(diff, 15000))
	b.WriteString("\n```\n\n")

	b.WriteString("## Output\n\nFor each finding, output one line:\n")
	b.WriteString("FINDING: <severity> | <file>:<line> | <description> | <fix recommendation>\n")
	b.WriteString("Severity: critical, high, medium, or low.\n")
	b.WriteString("If NO issues found, output: NO_FINDINGS\n")
	return b.String()
}

func (er *EnsembleReviewer) buildSynthesizerPrompt(diff string, rawReviews map[ReviewDimension]string) string {
	var b strings.Builder

	b.WriteString("Merge code review findings from three specialist reviewers into a unified report.\n\n")
	b.WriteString("1. Remove duplicate findings (same issue from multiple reviewers).\n")
	b.WriteString("2. Remove false positives.\n")
	b.WriteString("3. Rank by severity (critical > high > medium > low).\n")
	b.WriteString("4. Produce a clear, actionable summary.\n\n")

	for _, dim := range allDims {
		findings, ok := rawReviews[dim]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "## %s Reviewer\n\n%s\n\n", strings.ToUpper(string(dim)), findings)
	}

	b.WriteString("## Diff Context\n\n```diff\n")
	b.WriteString(truncForReview(diff, 5000))
	b.WriteString("\n```\n\n")

	b.WriteString("## Output Format\n\n")
	b.WriteString("SUMMARY: <1-3 sentence overview>\n")
	b.WriteString("FINDING: <dimension> | <severity> | <file>:<line> | <description> | <recommendation>\n")
	b.WriteString("If no actionable findings, output: CLEAN — no issues found.\n")
	return b.String()
}

func (er *EnsembleReviewer) runSubagent(ctx context.Context, prompt string, maxSteps int, sink event.Sink) (string, error) {
	subReg := SubagentToolRegistry(er.parentReg, nil)
	sess := NewSession("")

	return RunSubAgentWithSession(ctx, er.prov, subReg, sess, prompt, Options{
		MaxSteps:    maxSteps,
		Temperature: 0.2,
		Pricing:     er.pricing,
	}, sink)
}

func (er *EnsembleReviewer) buildResult(rawReviews map[ReviewDimension]string, synthOutput string) *EnsembleResult {
	result := &EnsembleResult{
		RawReviews: rawReviews,
	}

	lines := strings.Split(synthOutput, "\n")
	var secCount, corCount, perfCount int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "SUMMARY:"):
			result.Summary = strings.TrimSpace(strings.TrimPrefix(trimmed, "SUMMARY:"))

		case strings.HasPrefix(trimmed, "FINDING:"):
			f := parseEnsembleFinding(strings.TrimPrefix(trimmed, "FINDING:"))
			if f != nil {
				result.Findings = append(result.Findings, *f)
				switch f.Dimension {
				case DimSecurity:
					secCount++
				case DimCorrectness:
					corCount++
				case DimPerformance:
					perfCount++
				}
			}

		case strings.HasPrefix(trimmed, "CLEAN"):
			if result.Summary == "" {
				result.Summary = "No issues found by ensemble review."
			}
		}
	}

	result.SecurityCount = secCount
	result.CorrectnessCount = corCount
	result.PerformanceCount = perfCount

	if result.Summary == "" {
		result.Summary = fmt.Sprintf("Found %d issues: %d security, %d correctness, %d performance.",
			len(result.Findings), secCount, corCount, perfCount)
	}

	return result
}

func (er *EnsembleReviewer) buildFallbackResult(rawReviews map[ReviewDimension]string) *EnsembleResult {
	totalFindings := 0
	for _, output := range rawReviews {
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "FINDING:") {
				totalFindings++
			}
		}
	}
	return &EnsembleResult{
		RawReviews: rawReviews,
		Summary:    fmt.Sprintf("Ensemble review completed with %d findings (synthesizer unavailable). See raw reviews for details.", totalFindings),
	}
}

func parseEnsembleFinding(line string) *EnsembleFinding {
	parts := strings.SplitN(strings.TrimSpace(line), "|", 5)
	if len(parts) < 4 {
		return nil
	}

	dimension := ReviewDimension(strings.TrimSpace(parts[0]))
	if dimension != DimSecurity && dimension != DimCorrectness && dimension != DimPerformance {
		return nil
	}

	severity := strings.TrimSpace(parts[1])
	fileLine := strings.TrimSpace(parts[2])

	var file string
	var lineNum int
	if idx := strings.LastIndex(fileLine, ":"); idx >= 0 {
		file = strings.TrimSpace(fileLine[:idx])
		fmt.Sscanf(strings.TrimSpace(fileLine[idx+1:]), "%d", &lineNum)
	} else {
		file = fileLine
	}

	description := strings.TrimSpace(parts[3])
	recommendation := ""
	if len(parts) >= 5 {
		recommendation = strings.TrimSpace(parts[4])
	}

	return &EnsembleFinding{
		Dimension:      dimension,
		Severity:       severity,
		File:           file,
		Line:           lineNum,
		Description:    description,
		Recommendation: recommendation,
	}
}

func truncForReview(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
