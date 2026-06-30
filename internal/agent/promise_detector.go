package agent

import "strings"

// futurePromisePatterns match phrases that indicate the model is promising
// future work instead of executing it now. When detected at the end of a
// final answer, the readiness gate injects an action directive.
var futurePromisePatterns = []string{
	"I'll ",
	"I will ",
	"let me ",
	"next I'll",
	"next I will",
	"then we can",
	"after that",
	"after that I'll",
	"once that's done",
	"once that's done I'll",
	"we should also",
	"want me to",
	"shall I",
	"should I also",
}

// detectUnfulfilledPromise checks whether the final answer ends with what
// reads like a promise about future work the model hasn't done yet.
// Returns the matching context (up to 140 chars around the match) and true
// if a promise was detected.
func detectUnfulfilledPromise(finalText string) (string, bool) {
	lower := strings.ToLower(finalText)
	for _, pattern := range futurePromisePatterns {
		idx := strings.LastIndex(lower, pattern)
		if idx < 0 {
			continue
		}
		// Extract context around the match — prefer the tail of the text
		// since promises usually appear at the end of a final answer.
		start := idx - 20
		if start < 0 {
			start = 0
		}
		end := idx + 120
		if end > len(finalText) {
			end = len(finalText)
		}
		return strings.TrimSpace(finalText[start:end]), true
	}
	return "", false
}

// promiseRetryMessage builds the action directive injected when a promise is
// detected in a final answer.
func promiseRetryMessage(context string) string {
	return "Your response ends with what reads like a promise about future work: '" +
		context + "'. Execute that work now with tool calls instead of stating your intention."
}

// extractFileLineRefs extracts file:line references from text.
// Returns a deduplicated list of (file, line) pairs found in the text.
func extractFileLineRefs(text string) []fileLineRef {
	// Simple heuristic: find patterns like "file.go:123" or "path/to/file.go:42"
	// This is intentionally simple — regex would add overhead.
	var refs []fileLineRef
	seen := make(map[string]bool)

	words := strings.Fields(text)
	for _, w := range words {
		w = strings.Trim(w, ",;:.()[]{}'\"`")
		// Look for <filename>.<ext>:<number>
		colonIdx := strings.LastIndex(w, ":")
		if colonIdx < 1 {
			continue
		}
		// Must end with digits after the colon
		linePart := w[colonIdx+1:]
		if !isDigits(linePart) {
			continue
		}
		filePart := w[:colonIdx]
		// Must look like a filename (contains a dot for extension)
		if !strings.Contains(filePart, ".") {
			continue
		}
		key := filePart + ":" + linePart
		if !seen[key] {
			seen[key] = true
			refs = append(refs, fileLineRef{file: filePart, line: linePart})
		}
	}
	return refs
}

type fileLineRef struct {
	file string
	line string
}

func (f fileLineRef) String() string {
	return f.file + ":" + f.line
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// extractActionVerbs extracts action-oriented verbs from a request.
// Returns a list of verb roots present in the text.
func extractActionVerbs(text string) []string {
	lower := strings.ToLower(text)
	verbs := []string{"fix", "add", "test", "document", "refactor", "implement",
		"remove", "update", "create", "change", "modify", "rewrite", "debug",
		"optimize", "search", "find", "review", "verify", "check", "build",
		"deploy", "configure", "install", "migrate", "replace"}
	var found []string
	for _, v := range verbs {
		if strings.Contains(lower, v) {
			found = append(found, v)
		}
	}
	return found
}
