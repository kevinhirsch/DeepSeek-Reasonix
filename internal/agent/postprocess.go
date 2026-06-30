package agent

import (
	"strings"
	"unicode"
)

// PostProcessor applies three passes to subagent output:
// 1. Correctness — flags unsupported claims
// 2. Conciseness — strips redundant tokens (target ~20% reduction)
// 3. Completeness — checks result against original task
type PostProcessor struct {
	enabled bool
}

// NewPostProcessor creates a post-processor.
func NewPostProcessor() *PostProcessor {
	return &PostProcessor{enabled: true}
}

// Enable or disable post-processing.
func (p *PostProcessor) Enable(v bool) { p.enabled = v }

// Process applies all post-processing passes. Returns the processed output
// and any warnings.
func (p *PostProcessor) Process(output, task string) (string, []string) {
	if !p.enabled || strings.TrimSpace(output) == "" {
		return output, nil
	}
	var warnings []string
	processed := output

	// Pass 1: Correctness — flag unsupported claims
	if w := correctnessPass(processed); len(w) > 0 {
		warnings = append(warnings, w...)
	}

	// Pass 2: Conciseness — strip fluff tokens
	processed = concisenessPass(processed)

	// Pass 3: Completeness — check against original task
	if c := completenessPass(processed, task); c != "" {
		warnings = append(warnings, c)
	}

	return processed, warnings
}

// correctnessPass flags statements that sound unsupported.
func correctnessPass(output string) []string {
	var warnings []string
	claimMarkers := []string{
		"should work", "probably", "most likely", "I think",
		"seems to", "might be", "appears to", "could be a bug",
	}
	for _, marker := range claimMarkers {
		if strings.Contains(strings.ToLower(output), marker) {
			warnings = append(warnings, "unsupported claim detected: \""+marker+"\"")
		}
	}
	return warnings
}

// concisenessPass removes common redundant phrases.
func concisenessPass(output string) string {
	redundancies := []string{
		"I hope this helps",
		"Let me know if you have any questions",
		"Feel free to ask if you need anything",
		"Please let me know if you'd like me to",
		"I'm happy to help",
	}
	result := output
	for _, r := range redundancies {
		result = strings.ReplaceAll(result, r, "")
		result = strings.ReplaceAll(result, strings.ToLower(r), "")
	}
	return strings.TrimSpace(result)
}

// completenessPass checks whether the output addresses the task.
func completenessPass(output, task string) string {
	if task == "" {
		return ""
	}
	taskWords := extractKeywords(task)
	outputLower := strings.ToLower(output)
	missing := 0
	for _, w := range taskWords {
		if !strings.Contains(outputLower, w) {
			missing++
		}
	}
	if missing > len(taskWords)/2 {
		return "completeness warning: output may not fully address the task"
	}
	return ""
}

// extractKeywords returns the most salient words from a task description.
func extractKeywords(task string) []string {
	words := strings.Fields(strings.ToLower(task))
	var keywords []string
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"be": true, "to": true, "of": true, "in": true, "for": true,
		"on": true, "with": true, "and": true, "or": true, "it": true,
	}
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		if len(w) > 2 && !stopwords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}
