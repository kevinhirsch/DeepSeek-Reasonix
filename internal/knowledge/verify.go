package knowledge

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// modelVersionDiffThreshold is the minimum number of major-version bumps that
// trigger a re-verification. For example, if an entry was produced by model
// version "v3" and the current model is "v5", the entry needs re-verification.
const modelVersionDiffThreshold = 1

// VerificationResult reports the outcome of verifying (or re-verifying) a KB
// entry after a cross-validation pass.
type VerificationResult struct {
	// Verified is true when a second-pass check confirmed the entry.
	Verified bool
	// NeedReverify is true when the entry came from a model version that is
	// significantly different from the current version and should be re-verified.
	NeedReverify bool
	// Confidence is the adjusted confidence after verification.
	Confidence float64
	// Reason explains what action, if any, was taken.
	Reason string
}

// KBVerifier manages knowledge-base entry verification: before an entry is
// stored it cross-validates key claims with a second verification pass; at
// access time it flags stale entries (tagged with an obsolete model version);
// and entries from significantly different model versions are re-verified.
//
// The verifier does NOT call an LLM itself — that is the caller's
// responsibility. It provides the decision: whether verification is needed,
// whether the entry is stale, and what the adjusted confidence should be.
type KBVerifier struct {
	mu    sync.RWMutex
	model string // current model id (e.g. "deepseek-chat")
	// version is the current producing model version tag, independent of the
	// model id: it captures a deployment-granularity stamp (e.g. "2025-12-v2",
	// "v4.1") so two model ids may share a version lineage.
	version string
	// versionHistory records when each model version was first observed.
	versionHistory map[string]time.Time
}

// NewKBVerifier creates a verifier keyed to the given model and version.
func NewKBVerifier(model, version string) *KBVerifier {
	return &KBVerifier{
		model:          model,
		version:        version,
		versionHistory: map[string]time.Time{version: time.Now()},
	}
}

// SetModelVersion updates the current model and version. If the version is new,
// it is recorded in the history.
func (v *KBVerifier) SetModelVersion(model, version string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.model = model
	v.version = version
	if _, ok := v.versionHistory[version]; !ok {
		v.versionHistory[version] = time.Now()
	}
}

// CurrentModel returns the verifier's active model id.
func (v *KBVerifier) CurrentModel() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.model
}

// CurrentVersion returns the verifier's active model version.
func (v *KBVerifier) CurrentVersion() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.version
}

// VerifyBeforeStore checks whether a candidate entry should be stored. It
// performs a lightweight cross-validation of the entry's key claims.
func (v *KBVerifier) VerifyBeforeStore(entry *Entry) VerificationResult {
	answerLen := utf8.RuneCountInString(entry.Answer)
	questionLen := utf8.RuneCountInString(entry.Question)

	// Detect degenerate entries: answer is nearly the same as the question.
	if questionLen > 0 && answerLen > 0 {
		overlap := jaccardKeywordOverlap(entry.Question, entry.Answer)
		if overlap > 0.85 {
			return VerificationResult{
				Verified:     false,
				NeedReverify: true,
				Confidence:   0.1,
				Reason:       "answer content nearly identical to question - likely degenerate",
			}
		}
	}

	// High confidence but too short: likely overconfident.
	if entry.Confidence > 0.9 && answerLen < 80 {
		return VerificationResult{
			Verified:     false,
			NeedReverify: true,
			Confidence:   entry.Confidence * 0.5,
			Reason:       fmt.Sprintf("high confidence (%.2f) with very short answer (%d chars) - request second pass", entry.Confidence, answerLen),
		}
	}

	// Cross-validate file references: every listed file should appear in the
	// answer context. Missing references may indicate hallucinated file links.
	answerLower := strings.ToLower(entry.Answer)
	missingFiles := 0
	for _, f := range entry.Files {
		base := baseName(f)
		if !strings.Contains(answerLower, strings.ToLower(base)) {
			missingFiles++
		}
	}
	if missingFiles > 0 && missingFiles == len(entry.Files) {
		penalty := float64(missingFiles) * 0.15
		adj := entry.Confidence - penalty
		if adj < 0.1 {
			adj = 0.1
		}
		return VerificationResult{
			Verified:     false,
			NeedReverify: true,
			Confidence:   adj,
			Reason:       fmt.Sprintf("%d file references not found in answer text - possible hallucination", missingFiles),
		}
	}

	// Passed all checks.
	return VerificationResult{
		Verified:   true,
		Confidence: entry.Confidence,
		Reason:     "pre-store cross-validation passed",
	}
}

// CheckStale examines an entry at access time. It returns true and a reason when
// the entry was produced by a model version that is no longer current.
func (v *KBVerifier) CheckStale(entry *Entry) (stale bool, reason string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if entry.Model == "" {
		return false, ""
	}
	if entry.Model == v.model {
		return false, ""
	}
	return true, fmt.Sprintf("entry produced by model %q (current: %q)", entry.Model, v.model)
}

// NeedsReverify reports whether an entry should be re-verified because its
// producing model version is significantly different from the current version.
func (v *KBVerifier) NeedsReverify(entry *Entry) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if entry.Model == "" {
		return false
	}

	entryMajor := extractMajorVersion(entry.Model)
	currentMajor := extractMajorVersion(v.version)
	if entryMajor == 0 || currentMajor == 0 {
		return false
	}

	diff := currentMajor - entryMajor
	if diff < 0 {
		diff = -diff
	}
	return diff > modelVersionDiffThreshold
}

// TagEntry stamps the entry with the current model id (as the producing model
// tag). The caller should call this before persisting a new or re-verified entry.
func (v *KBVerifier) TagEntry(entry *Entry) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	entry.Model = v.model
}

// ReverifyReason returns a human-readable reason string suitable for including in
// a re-verification prompt when NeedsReverify returned true.
func (v *KBVerifier) ReverifyReason(entry *Entry) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return fmt.Sprintf(
		"entry produced by model version %q - current version is %q; re-verify for correctness",
		entry.Model, v.version,
	)
}

// extractMajorVersion returns the numeric major component from a version string
// like "v4.1" (returns 4), "2025-12-v2" (returns 2025), or "" (returns 0).
func extractMajorVersion(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	start := 0
	for start < len(s) && !unicode.IsDigit(rune(s[start])) {
		start++
	}
	if start >= len(s) {
		return 0
	}
	var n int
	for i := start; i < len(s) && unicode.IsDigit(rune(s[i])); i++ {
		n = n*10 + int(rune(s[i])-'0')
	}
	return n
}

// jaccardKeywordOverlap returns the Jaccard similarity (0..1) of the keyword sets
// of two strings. Keywords are lowercased, split on whitespace, and filtered to
// tokens of at least 3 characters.
func jaccardKeywordOverlap(a, b string) float64 {
	ka := keywordSet(a)
	kb := keywordSet(b)
	if len(ka) == 0 && len(kb) == 0 {
		return 1.0
	}
	intersection := 0
	for k := range ka {
		if kb[k] {
			intersection++
		}
	}
	union := len(ka) + len(kb) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// keywordSet returns a set of normalized keywords (>=3 chars) from s.
func keywordSet(s string) map[string]bool {
	words := strings.Fields(strings.ToLower(s))
	set := make(map[string]bool, len(words))
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if utf8.RuneCountInString(w) >= 3 {
			set[w] = true
		}
	}
	return set
}

// baseName returns the last element of a file path.
func baseName(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// --- semantic dedup helpers shared with eviction.go ---

// entryKeywordOverlap returns the Jaccard similarity of two entries' combined question
// and answer text. A value > 0.8 is considered a semantic duplicate.
func entryKeywordOverlap(a, b *Entry) float64 {
	textA := strings.ToLower(a.Question + " " + a.Answer)
	textB := strings.ToLower(b.Question + " " + b.Answer)
	return jaccardKeywordOverlap(textA, textB)
}

// mergeEntries combines two entries into one by keeping the higher-confidence
// entry's core and folding in any unique files from the lower.
func mergeEntries(keep, drop *Entry) {
	seen := make(map[string]bool)
	for _, f := range keep.Files {
		seen[f] = true
	}
	for _, f := range drop.Files {
		if !seen[f] {
			keep.Files = append(keep.Files, f)
			seen[f] = true
		}
	}
	if drop.Confidence > keep.Confidence {
		keep.Confidence = drop.Confidence
	}
	keep.Confidence *= 0.95
	keep.AccessCount = (keep.AccessCount + drop.AccessCount) / 2
	if drop.AccessedAt.After(keep.AccessedAt) {
		keep.AccessedAt = drop.AccessedAt
	}
	if !strings.Contains(keep.Answer, "[merged from ") {
		keep.Answer += fmt.Sprintf("\n\n[merged from duplicate entry %s]", drop.ID)
	}
}

// sortEntriesByScore sorts entries in-place by eviction priority (lowest score
// first = evicted first).
func sortEntriesByScore(entries []*Entry, now time.Time) {
	type scored struct {
		e     *Entry
		score float64
	}
	scoredEntries := make([]scored, len(entries))
	for i, e := range entries {
		scoredEntries[i] = scored{e: e, score: entryValueScore(e, now)}
	}
	sort.Slice(scoredEntries, func(i, j int) bool { return scoredEntries[i].score < scoredEntries[j].score })
	for i, s := range scoredEntries {
		entries[i] = s.e
	}
}

// entryValueScore computes how valuable an entry is: higher = keep longer.
func entryValueScore(e *Entry, now time.Time) float64 {
	if e == nil {
		return 0
	}

	ageHours := now.Sub(e.AccessedAt).Hours()
	recency := 1.0
	if ageHours > 0 {
		recency = expDecay(ageHours, 168.0)
	}

	freq := 0.3
	if e.AccessCount > 1 {
		freq = logScale(float64(e.AccessCount), 100.0)
	}
	if freq > 1.0 {
		freq = 1.0
	}

	conf := e.Confidence
	if conf < 0 {
		conf = 0
	}
	if conf > 1.0 {
		conf = 1.0
	}

	ansLen := float64(utf8.RuneCountInString(e.Answer))
	uniq := (ansLen/2000.0 + float64(len(e.Files))/5.0) / 2.0
	if uniq > 1.0 {
		uniq = 1.0
	}
	if uniq < 0.05 {
		uniq = 0.05
	}

	return recency * freq * conf * uniq
}

// expDecay returns an approximate exponential-decay factor for scoring.
func expDecay(t, halfLife float64) float64 {
	if halfLife <= 0 {
		return 1.0
	}
	lambda := 0.6931471805599453 / halfLife
	r := 1.0 - lambda*t
	if r < 0 {
		return 0
	}
	return r
}

// logScale returns an approximate log10 scaling factor, clamped to [0, 1].
func logScale(x, max float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= max {
		return 1.0
	}
	lnX := 0
	for tmp := int(x + 1); tmp > 1; tmp /= 10 {
		lnX++
	}
	lnMax := 0
	for tmp := int(max + 1); tmp > 1; tmp /= 10 {
		lnMax++
	}
	if lnMax == 0 {
		return 0
	}
	return float64(lnX) / float64(lnMax)
}
