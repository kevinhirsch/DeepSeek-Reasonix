package agent

import (
	"strings"
)

// SemanticDiff compares N outputs from speculative execution and identifies
// agreements, disagreements, and contradictions between them.
type SemanticDiff struct {
	threshold float64 // overlap threshold for "agreement"
}

// NewSemanticDiff creates a semantic diff comparator.
func NewSemanticDiff() *SemanticDiff {
	return &SemanticDiff{threshold: 0.5}
}

// DiffResult categorizes findings across speculative runs.
type DiffResult struct {
	Agreements     []string `json:"agreements"`
	Disagreements  []string `json:"disagreements"`
	Contradictions []string `json:"contradictions"`
}

// Compare analyzes N outputs and returns categorized findings.
func (s *SemanticDiff) Compare(outputs []string) *DiffResult {
	if len(outputs) < 2 {
		return &DiffResult{}
	}
	result := &DiffResult{}
	findings := make([][]string, len(outputs))
	for i, out := range outputs {
		findings[i] = extractSpeculativeFindings(out)
	}

	// Build a map of finding → which runs found it
	occurrences := make(map[string][]int)
	for i, fset := range findings {
		for _, f := range fset {
			key := normalizeFinding(f)
			occurrences[key] = append(occurrences[key], i)
		}
	}

	total := len(outputs)
	for key, runs := range occurrences {
		count := len(runs)
		switch {
		case count == total:
			result.Agreements = append(result.Agreements, key)
		case count >= total/2:
			result.Disagreements = append(result.Disagreements, key)
		default:
			// Check for contradictions: do any outputs claim the opposite?
			if hasOpposite(outputs, key) {
				result.Contradictions = append(result.Contradictions, key)
			} else {
				result.Disagreements = append(result.Disagreements, key)
			}
		}
	}
	return result
}

func normalizeFinding(f string) string {
	f = strings.TrimSpace(f)
	f = strings.TrimPrefix(f, "- ")
	f = strings.TrimPrefix(f, "* ")
	return strings.ToLower(f)
}

func hasOpposite(outputs []string, key string) bool {
	lower := strings.ToLower(key)
	// Simple heuristic: look for negation patterns
	negations := []string{"not", "no ", "doesn't", "does not", "isn't", "is not"}
	for _, out := range outputs {
		outLower := strings.ToLower(out)
		if containsAnyNegation(outLower, lower, negations) {
			return true
		}
	}
	return false
}

func containsAnyNegation(output, key string, negations []string) bool {
	for _, neg := range negations {
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, key) && strings.Contains(line, neg) {
				return true
			}
		}
	}
	return false
}
