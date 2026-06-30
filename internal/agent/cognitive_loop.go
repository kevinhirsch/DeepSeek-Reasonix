package agent

import "strings"

// cognitiveLoopState tracks per-turn reasoning state for DeepSeek cognitive
// loop detection. Unlike stormBreaker which catches (tool, error) loops,
// cognitive loops are text-pattern-based: repeating reasoning, escalating
// uncertainty markers, and mechanical self-explanation without action.
type cognitiveLoopState struct {
	// prevReasoningPrefix is the first 200 chars of the previous turn's
	// reasoning. Empty on the first turn.
	prevReasoningPrefix string

	// uncertaintyCount tracks the number of consecutive turns where
	// uncertainty markers ("tricky"/"confused"/"ambiguous") appeared
	// in the reasoning. Resets to 0 on any turn with tool calls.
	uncertaintyCount int

	// selfExplanationCount tracks consecutive turns where "I need to" or
	// "Let me think" appear in reasoning WITHOUT any tool calls — the
	// model is narrating its thought process instead of acting.
	selfExplanationCount int
}

// uncertaintyMarkers are words that indicate the model is cycling between
// options rather than making a decision.
var uncertaintyMarkers = []string{
	"tricky", "confused", "ambiguous", "i'm not sure",
	"this is tricky", "let me reconsider", "this is confusing",
	"on second thought", "actually", "wait,",
}

// selfExplanationMarkers are phrases that indicate mechanical narration
// without concrete action.
var selfExplanationMarkers = []string{
	"i need to", "let me think", "i should", "let me consider",
	"i'm going to", "i will first", "first i need", "first i should",
}

// detectCognitiveLoop checks the current turn's reasoning for cognitive
// loop patterns. Returns true and the pattern name when a loop is detected.
// Only called when isDeepSeek is true (DeepSeek-specific).
func detectCognitiveLoop(state *cognitiveLoopState, reasoning string, hasToolCalls bool) (bool, string) {
	if state == nil {
		return false, ""
	}

	reasoningLower := strings.ToLower(reasoning)
	prefix := firstN(reasoning, 200)

	// Pattern 1: Uncertainty escalation — uncertainty markers appear and
	// count is rising across turns. Only triggers when uncertainty is
	// persistent (>= 3 consecutive turns).
	uncertaintyInThisTurn := containsAny(reasoningLower, uncertaintyMarkers)
	if uncertaintyInThisTurn && !hasToolCalls {
		state.uncertaintyCount++
	} else {
		state.uncertaintyCount = 0
	}
	if state.uncertaintyCount >= 3 {
		return true, "uncertainty escalation: " + firstN(reasoning, 80) + "..."
	}

	// Pattern 2: Identical reasoning prefix — first 200 chars match
	// the previous turn's reasoning. Indicates the model is re-thinking
	// the same thing without progressing.
	if state.prevReasoningPrefix != "" && prefix != "" &&
		state.prevReasoningPrefix == prefix {
		return true, "identical reasoning prefix (first 200 chars match previous turn)"
	}
	state.prevReasoningPrefix = prefix

	// Pattern 3: Mechanical self-explanation — consecutive turns with
	// "I need to"/"Let me think" markers that produce no tool calls.
	// The model is narrating intent instead of executing.
	if containsAny(reasoningLower, selfExplanationMarkers) && !hasToolCalls {
		state.selfExplanationCount++
	} else {
		state.selfExplanationCount = 0
	}
	if state.selfExplanationCount >= 2 {
		return true, "mechanical self-explanation without action: " + firstN(reasoning, 80) + "..."
	}

	return false, ""
}

// cognitiveLoopBreakerMessage returns the injection message injected when
// a cognitive loop is detected. The message instructs the model to stop
// reasoning and act.
func cognitiveLoopBreakerMessage(pattern string) string {
	return "Stop reasoning and act. Make a concrete decision.\n\n" +
		"[Cognitive loop detected: " + pattern + "]\n" +
		"If you're uncertain, pick the most likely option and proceed. " +
		"You can correct course later if new evidence contradicts your choice. " +
		"Do not re-verify successful tool calls."
}

// resetCognitiveLoopState resets all loop tracking counters. Called when
// the model makes a successful tool call, receives user input, or the
// turn produces meaningful progress.
func resetCognitiveLoopState(state *cognitiveLoopState) {
	if state == nil {
		return
	}
	state.uncertaintyCount = 0
	state.selfExplanationCount = 0
	// prevReasoningPrefix is intentionally not reset — it should track
	// across turns to catch identical-prefix patterns.
}

// containsAny returns true if any of the needles appear in the haystack.
func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

// firstN returns the first n characters of s, or all of s if shorter.
func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
