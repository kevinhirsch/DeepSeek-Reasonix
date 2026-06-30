// Package pattern provides bug pattern extraction and recognition.
//
// Recognize.go implements bug pattern recognition (Issue #20 Comp 17). The
// PatternRecognizer extracts patterns from every bug fix and flags new code
// that matches known bug patterns BEFORE commit. The pattern library grows
// organically as more fixes are analyzed.
package pattern

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// BugPattern is a learned bug template extracted from past fixes.
type BugPattern struct {
	// Name is the unique identifier for this pattern.
	Name string `json:"name"`

	// Pattern is the anti-pattern text that indicates this bug.
	Pattern string `json:"pattern"`

	// Category classifies the bug type (nil_deref, race_condition, injection, etc.).
	Category string `json:"category"`

	// Count is how many times this pattern was confirmed in real fixes.
	Count int `json:"count"`

	// Confidence is the estimated reliability of this pattern (0.0-1.0).
	Confidence float64 `json:"confidence"`

	// FirstSeen records when this pattern was first identified.
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen records the most recent confirmation of this pattern.
	LastSeen time.Time `json:"last_seen"`

	// Severity estimates the typical impact (critical, high, medium, low).
	Severity string `json:"severity"`

	// FixTemplate is a suggested fix for code matching this pattern.
	FixTemplate string `json:"fix_template,omitempty"`
}

// ScanResult is one match found by the recognizer.
type ScanResult struct {
	Pattern    BugPattern `json:"pattern"`
	File       string     `json:"file"`
	Line       int        `json:"line"`
	Context    string     `json:"context"`
	Suggestion string     `json:"suggestion"`
}

// PatternRecognizer extracts patterns from bug fixes and flags new code
// matching known bug patterns BEFORE commit.
type PatternRecognizer struct {
	mu        sync.RWMutex
	patterns  map[string]*BugPattern
	storePath string
}

// NewPatternRecognizer creates a pattern recognizer. If storePath is non-empty,
// patterns are persisted to disk and loaded on creation.
func NewPatternRecognizer(storePath string) *PatternRecognizer {
	pr := &PatternRecognizer{
		patterns:  make(map[string]*BugPattern),
		storePath: storePath,
	}
	pr.load()
	return pr
}

// LearnFromFix extracts a pattern from a completed bug fix. The bugDescription
// and fixDescription are human-readable summaries of what broke and how it was
// fixed. The recognizer extracts the anti-pattern (what to watch for) and the
// fix template (how to resolve it).
func (r *PatternRecognizer) LearnFromFix(bugDescription, fixDescription string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pattern := extractAntiPattern(bugDescription)
	key := strings.ToLower(strings.TrimSpace(pattern))
	if key == "" {
		return
	}

	category := categorizeBug(bugDescription)
	severity := estimateSeverity(bugDescription)
	fixTemplate := extractFixTemplate(fixDescription)
	now := time.Now()

	if existing, ok := r.patterns[key]; ok {
		existing.Count++
		existing.LastSeen = now
		existing.Confidence = clampFloat(existing.Confidence+0.05, 0, 1.0)
		if fixTemplate != "" && existing.FixTemplate == "" {
			existing.FixTemplate = fixTemplate
		}
	} else {
		r.patterns[key] = &BugPattern{
			Name:        key,
			Pattern:     pattern,
			Category:    category,
			Count:       1,
			Confidence:  0.3,
			FirstSeen:   now,
			LastSeen:    now,
			Severity:    severity,
			FixTemplate: fixTemplate,
		}
	}

	r.persist()
}

// Scan checks code content against all known bug patterns. Returns matching
// patterns with file and line context where available.
func (r *PatternRecognizer) Scan(code string, file string) []ScanResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ScanResult
	lower := strings.ToLower(code)

	for _, p := range r.patterns {
		if p.Confidence < 0.4 {
			continue // skip low-confidence patterns
		}

		if !strings.Contains(lower, p.Pattern) {
			continue
		}

		// Find the line number and context for the match.
		line, context := findMatchContext(code, p.Pattern)

		results = append(results, ScanResult{
			Pattern:    *p,
			File:       file,
			Line:       line,
			Context:    context,
			Suggestion: buildSuggestion(p),
		})
	}

	return results
}

// ScanFile reads a file and scans it against known bug patterns.
func (r *PatternRecognizer) ScanFile(filePath string) ([]ScanResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return r.Scan(string(data), filePathBase(filePath)), nil
}

// ScanDiff scans changed lines in a diff against known patterns. Only new or
// modified lines are checked, avoiding false positives on existing code.
func (r *PatternRecognizer) ScanDiff(diff string) []ScanResult {
	// Focus on added lines (prefixed with +) in unified diffs.
	var addedLines []string
	lines := strings.Split(diff, "\n")

	var currentFile string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+++ b/") {
			currentFile = filepath.Base(strings.TrimPrefix(trimmed, "+++ b/"))
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedLines = append(addedLines, line[1:])
		}
	}

	if len(addedLines) == 0 {
		return nil
	}

	return r.Scan(strings.Join(addedLines, "\n"), currentFile)
}

// AllPatterns returns all learned patterns sorted by confidence descending.
func (r *PatternRecognizer) AllPatterns() []BugPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []BugPattern
	for _, p := range r.patterns {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Confidence > out[j].Confidence
	})
	return out
}

// HighConfidencePatterns returns patterns with confidence above the threshold.
func (r *PatternRecognizer) HighConfidencePatterns(threshold float64) []BugPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []BugPattern
	for _, p := range r.patterns {
		if p.Confidence >= threshold {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count
	})
	return out
}

// Stats returns recognition statistics.
func (r *PatternRecognizer) Stats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.patterns)
	highConf := 0
	catCounts := make(map[string]int)
	for _, p := range r.patterns {
		catCounts[p.Category]++
		if p.Confidence >= 0.7 {
			highConf++
		}
	}

	stats := map[string]int{
		"total":          total,
		"high_confidence": highConf,
	}
	for cat, count := range catCounts {
		stats["cat_"+cat] = count
	}
	return stats
}

// --- internal helpers ---

func extractAntiPattern(desc string) string {
	lower := strings.ToLower(desc)
	markers := []string{
		"nil pointer dereference",
		"nil map assignment",
		"slice bounds out of range",
		"index out of range",
		"race condition",
		"data race",
		"goroutine leak",
		"deadlock",
		"missing nil check",
		"missing error check",
		"sql injection",
		"xss",
		"cross-site scripting",
		"path traversal",
		"command injection",
		"integer overflow",
		"use after free",
		"double close",
		"send on closed channel",
		"unchecked type assertion",
		"divide by zero",
		"infinite loop",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return m
		}
	}
	return ""
}

func categorizeBug(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "nil") || strings.Contains(lower, "null"):
		return "nil_deref"
	case strings.Contains(lower, "race") || strings.Contains(lower, "concurrent") ||
		strings.Contains(lower, "goroutine") || strings.Contains(lower, "deadlock"):
		return "concurrency"
	case strings.Contains(lower, "injection") || strings.Contains(lower, "xss") ||
		strings.Contains(lower, "traversal"):
		return "injection"
	case strings.Contains(lower, "overflow") || strings.Contains(lower, "bounds") ||
		strings.Contains(lower, "divide"):
		return "arithmetic"
	case strings.Contains(lower, "channel") || strings.Contains(lower, "close"):
		return "channel"
	default:
		return "general"
	}
}

func estimateSeverity(desc string) string {
	lower := strings.ToLower(desc)
	if strings.Contains(lower, "crash") || strings.Contains(lower, "panic") ||
		strings.Contains(lower, "security") || strings.Contains(lower, "data loss") {
		return "critical"
	}
	if strings.Contains(lower, "deadlock") || strings.Contains(lower, "leak") ||
		strings.Contains(lower, "race") {
		return "high"
	}
	if strings.Contains(lower, "incorrect") || strings.Contains(lower, "wrong") {
		return "medium"
	}
	return "low"
}

func extractFixTemplate(fixDesc string) string {
	lower := strings.ToLower(fixDesc)

	templates := map[string]string{
		"nil pointer dereference": "Add nil check before dereference: if ptr != nil { ... }",
		"missing nil check":       "Add nil guard: if v == nil { return err }",
		"missing error check":     "Check error return: if err != nil { return err }",
		"slice bounds out of range": "Validate index: if idx < len(slice) { ... }",
		"race condition":          "Add mutex or use sync/atomic for shared state access",
		"goroutine leak":          "Ensure goroutine exits via context cancellation or done channel",
		"send on closed channel":  "Use select with context or check channel state before send",
		"unchecked type assertion": "Use comma-ok pattern: v, ok := x.(T); if !ok { ... }",
	}

	for key, tmpl := range templates {
		if strings.Contains(lower, key) {
			return tmpl
		}
	}
	return fixDesc
}

func findMatchContext(code, pattern string) (int, string) {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), pattern) {
			context := strings.TrimSpace(line)
			if len(context) > 120 {
				context = context[:120] + "..."
			}
			return i + 1, context
		}
	}
	return 0, ""
}

func buildSuggestion(p *BugPattern) string {
	if p.FixTemplate != "" {
		return p.FixTemplate
	}
	switch p.Category {
	case "nil_deref":
		return "Add nil check before accessing the value."
	case "concurrency":
		return "Add synchronization (mutex, channel, or atomic) around the shared state."
	case "injection":
		return "Sanitize or parameterize inputs. Use prepared statements or escape output."
	case "arithmetic":
		return "Add bounds checking or overflow detection before the operation."
	case "channel":
		return "Check channel state before operation or use select with default."
	default:
		return "Review this pattern — it matches a known bug template."
	}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func filePathBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// --- persistence ---

func (r *PatternRecognizer) persist() {
	if r.storePath == "" {
		return
	}
	dir := filepath.Dir(r.storePath)
	os.MkdirAll(dir, 0755)

	var patterns []*BugPattern
	for _, p := range r.patterns {
		patterns = append(patterns, p)
	}
	data, _ := json.MarshalIndent(patterns, "", "  ")
	os.WriteFile(r.storePath, data, 0644)
}

func (r *PatternRecognizer) load() {
	if r.storePath == "" {
		return
	}
	data, err := os.ReadFile(r.storePath)
	if err != nil {
		return
	}
	var patterns []*BugPattern
	if json.Unmarshal(data, &patterns) != nil {
		return
	}
	for _, p := range patterns {
		r.patterns[p.Name] = p
	}
}
