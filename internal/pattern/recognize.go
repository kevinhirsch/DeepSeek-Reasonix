package pattern

import (
	"fmt"
	"strings"
	"sync"
)

// BugPattern is a learned bug template extracted from past fixes.
type BugPattern struct {
	Name       string
	Pattern    string // the anti-pattern (what to look for)
	Category   string // nil_deref, race_condition, injection, etc.
	Count      int    // how many times this pattern was confirmed
	Confidence float64
}

// PatternRecognizer extracts patterns from bug fixes and flags new code
// matching known bug patterns BEFORE commit.
type PatternRecognizer struct {
	mu       sync.RWMutex
	patterns map[string]*BugPattern
}

// NewPatternRecognizer creates a pattern recognizer.
func NewPatternRecognizer() *PatternRecognizer {
	return &PatternRecognizer{
		patterns: make(map[string]*BugPattern),
	}
}

// LearnFromFix extracts a pattern from a completed bug fix.
func (r *PatternRecognizer) LearnFromFix(bugDescription, fixDescription string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pattern := extractAntiPattern(bugDescription)
	key := strings.ToLower(strings.TrimSpace(pattern))
	if key == "" {
		return
	}
	if existing, ok := r.patterns[key]; ok {
		existing.Count++
		existing.Confidence = min(1.0, existing.Confidence+0.1)
	} else {
		r.patterns[key] = &BugPattern{
			Name:       key,
			Pattern:    pattern,
			Category:   categorize(bugDescription),
			Count:      1,
			Confidence: 0.3,
		}
	}
}

// Scan checks code against known bug patterns. Returns matching patterns.
func (r *PatternRecognizer) Scan(code string) []BugPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []BugPattern
	lower := strings.ToLower(code)
	for _, p := range r.patterns {
		if p.Confidence >= 0.5 && strings.Contains(lower, p.Pattern) {
			matches = append(matches, *p)
		}
	}
	return matches
}

// AllPatterns returns all learned patterns.
func (r *PatternRecognizer) AllPatterns() []BugPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []BugPattern
	for _, p := range r.patterns {
		out = append(out, *p)
	}
	return out
}

func extractAntiPattern(desc string) string {
	lower := strings.ToLower(desc)
	markers := []string{
		"nil pointer", "nil reference", "null pointer",
		"race condition", "data race",
		"not checking", "missing nil check", "missing error check",
		"sql injection", "xss", "path traversal",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return m
		}
	}
	return ""
}

func categorize(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "nil") || strings.Contains(lower, "null"):
		return "nil_deref"
	case strings.Contains(lower, "race") || strings.Contains(lower, "concurrent"):
		return "race_condition"
	case strings.Contains(lower, "injection") || strings.Contains(lower, "xss"):
		return "injection"
	case strings.Contains(lower, "traversal") || strings.Contains(lower, "path"):
		return "path_traversal"
	default:
		return "general"
	}
}
