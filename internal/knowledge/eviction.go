package knowledge

import (
	"math"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Eviction scoring: score = recency_factor × access_freq × confidence × uniqueness
// ---------------------------------------------------------------------------

// EvictionScore computes a retention score for a knowledge entry.
// Higher scores mean the entry is more valuable and should be kept.
// Lower scores mean the entry is a candidate for eviction.
//
// The score is the product of four factors:
//   - recency:    how recently the entry was accessed (0..1, exponential decay)
//   - accessFreq: how often the entry is accessed (1 + log of access count)
//   - confidence: the entry's stated confidence value
//   - uniqueness: how unique the entry's keywords are vs. the corpus (0..1)
//
// Entries with scores below the eviction threshold are candidates for removal.
type EvictionScore struct {
	EntryID     string
	Recency     float64 // 0..1, higher = more recent
	AccessFreq  float64 // 1 + log(accessCount+1)
	Confidence  float64 // entry.Confidence
	Uniqueness  float64 // 0..1, higher = more unique
	Total       float64 // product of all four factors
}

// EvictionConfig tunes the eviction policy.
type EvictionConfig struct {
	// MaxEntries is the hard cap on knowledge store size.
	MaxEntries int

	// SoftCap triggers eviction at this count (below MaxEntries, as headroom).
	// Defaults to MaxEntries * 0.85.
	SoftCap int

	// MinScore is the eviction floor; entries scoring below this are
	// always candidates regardless of size. 0 disables.
	MinScore float64

	// RecencyHalfLife controls the exponential decay of access recency.
	// An entry accessed this long ago gets a recency factor of 0.5.
	// Default 7 days.
	RecencyHalfLife time.Duration

	// DedupThreshold is the keyword overlap ratio above which two entries
	// are considered duplicates (0..1). Default 0.8 (80%).
	DedupThreshold float64

	// UniquenessWeightOptional boosts or suppresses uniqueness in scoring.
	// 1.0 = default weight. Values above 1 penalise duplication more.
	UniquenessWeight float64
}

func (c *EvictionConfig) defaults(maxEntries int) {
	if c.MaxEntries <= 0 {
		c.MaxEntries = maxEntries
	}
	if c.MaxEntries <= 0 {
		c.MaxEntries = 10000
	}
	if c.SoftCap <= 0 {
		c.SoftCap = int(float64(c.MaxEntries) * 0.85)
	}
	if c.RecencyHalfLife <= 0 {
		c.RecencyHalfLife = 7 * 24 * time.Hour
	}
	if c.DedupThreshold <= 0 {
		c.DedupThreshold = 0.8
	}
	if c.UniquenessWeight <= 0 {
		c.UniquenessWeight = 1.0
	}
}

// ---------------------------------------------------------------------------
// Eviction scorer
// ---------------------------------------------------------------------------

// EvictionScorer ranks entries for eviction using the four-factor formula.
type EvictionScorer struct {
	cfg       EvictionConfig
	now       time.Time
	corpusVec map[string]keywordVector // entry ID → keyword vector (computed lazily)
}

// keywordVector maps normalized keyword → term frequency.
type keywordVector map[string]float64

// NewEvictionScorer creates a scorer with the given config.
func (c EvictionConfig) NewEvictionScorer() *EvictionScorer {
	c.defaults(0)
	return &EvictionScorer{
		cfg:       c,
		now:       time.Now(),
		corpusVec: make(map[string]keywordVector),
	}
}

// Score computes the eviction score for a single entry against the full
// set of entries (needed for uniqueness computation).
func (es *EvictionScorer) Score(e *Entry, allEntries []*Entry) EvictionScore {
	s := EvictionScore{
		EntryID:    e.ID,
		Confidence: e.Confidence,
	}

	// Recency factor: exponential decay based on time since last access.
	age := es.now.Sub(e.AccessedAt)
	s.Recency = math.Exp(-math.Ln2 * age.Hours() / es.cfg.RecencyHalfLife.Hours())
	if s.Recency < 0.01 {
		s.Recency = 0.01 // floor: entries never decay to zero
	}

	// Access frequency: log-scaled so high-frequency entries pull upward
	// but don't dominate the score.
	s.AccessFreq = 1.0 + math.Log2(float64(e.AccessCount)+1)

	// Uniqueness: measured against all other entries in the corpus.
	s.Uniqueness = es.computeUniqueness(e, allEntries)

	s.Total = s.Recency * s.AccessFreq * s.Confidence * s.Uniqueness
	return s
}

// computeUniqueness measures how distinct an entry's keyword vector is
// from the rest of the corpus. Returns 0..1 where 1 = entirely unique,
// 0 = completely duplicated.
func (es *EvictionScorer) computeUniqueness(e *Entry, allEntries []*Entry) float64 {
	if len(allEntries) <= 1 {
		return 1.0 // sole entry is always unique
	}
	thisVec := es.getVector(e)
	var maxOverlap float64
	count := 0
	for _, other := range allEntries {
		if other.ID == e.ID {
			continue
		}
		otherVec := es.getVector(other)
		overlap := keywordOverlap(thisVec, otherVec)
		if overlap > maxOverlap {
			maxOverlap = overlap
		}
		count++
	}
	if count == 0 {
		return 1.0
	}
	// Uniqueness = 1 - (weighted max overlap). The weight lets the config
	// penalise duplication more aggressively.
	uniqueness := 1.0 - maxOverlap*es.cfg.UniquenessWeight
	if uniqueness < 0 {
		uniqueness = 0
	}
	return uniqueness
}

func (es *EvictionScorer) getVector(e *Entry) keywordVector {
	if v, ok := es.corpusVec[e.ID]; ok {
		return v
	}
	v := buildKeywordVector(e.Question + " " + e.Answer)
	es.corpusVec[e.ID] = v
	return v
}

// ---------------------------------------------------------------------------
// Keyword vector operations
// ---------------------------------------------------------------------------

// buildKeywordVector tokenizes text into a bag-of-normalized-words with
// term frequencies. Stop words are stripped.
func buildKeywordVector(text string) keywordVector {
	vec := make(keywordVector)
	words := strings.Fields(strings.ToLower(text))
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}<>-*/\\|`~@#$%^&+=_")
		if w == "" || isStopWord(w) {
			continue
		}
		vec[w]++
	}
	// Normalize to unit vector.
	var sumSq float64
	for _, v := range vec {
		sumSq += v * v
	}
	if sumSq > 0 {
		norm := math.Sqrt(sumSq)
		for k, v := range vec {
			vec[k] = v / norm
		}
	}
	return vec
}

// keywordOverlap computes the cosine similarity between two keyword vectors.
// Returns 0..1 where higher = more similar.
func keywordOverlap(a, b keywordVector) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	// Use the smaller vector as the iteration base.
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	// Both vectors are already unit-normalized by buildKeywordVector.
	return dot
}

// isStopWord filters common English words that carry little semantic value.
func isStopWord(w string) bool {
	switch w {
	case "a", "an", "the", "and", "or", "but", "in", "on", "at", "to",
		"for", "of", "with", "by", "from", "is", "are", "was", "were",
		"be", "been", "being", "have", "has", "had", "do", "does", "did",
		"will", "would", "could", "should", "may", "might", "can", "shall",
		"it", "its", "this", "that", "these", "those", "not", "no", "nor",
		"so", "if", "then", "else", "when", "where", "which", "who", "whom",
		"what", "how", "all", "each", "every", "both", "few", "more",
		"most", "other", "some", "such", "only", "own", "same", "into",
		"up", "out", "about", "over", "under", "after", "before", "between",
		"just", "very", "too", "also", "now", "here", "there":
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Size-limited eviction with four-factor scoring
// ---------------------------------------------------------------------------

// EvictRanked removes the lowest-scoring entries until the store is within
// the soft cap. Deduplication of entries with >80% keyword overlap is
// performed first: among duplicates, the one with the highest combined
// confidence × access_freq is kept.
//
// Returns the number of entries evicted and the number of duplicates merged.
func (s *Store) EvictRanked(cfg EvictionConfig) (evicted int, deduped int) {
	cfg.defaults(s.maxSize)

	s.mu.Lock()
	entries := make([]*Entry, 0, len(s.entries))
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	s.mu.Unlock()

	// Phase 1: Deduplicate entries with >80% keyword overlap.
	entries, deduped = deduplicateCorpus(entries, cfg.DedupThreshold)

	// Phase 2: Score and rank.
	scorer := cfg.NewEvictionScorer()
	type ranked struct {
		e     *Entry
		score EvictionScore
	}
	var rankedEntries []ranked
	for _, e := range entries {
		score := scorer.Score(e, entries)
		rankedEntries = append(rankedEntries, ranked{e: e, score: score})
	}

	// Sort ascending by total score (lowest first = eviction candidates).
	sort.Slice(rankedEntries, func(i, j int) bool {
		return rankedEntries[i].score.Total < rankedEntries[j].score.Total
	})

	// Determine how many to evict.
	need := len(rankedEntries) - cfg.SoftCap
	if need <= 0 {
		need = 0
	}
	// Always evict entries below MinScore regardless of size.
	for _, r := range rankedEntries {
		if r.score.Total < cfg.MinScore && cfg.MinScore > 0 {
			need++
		}
	}
	if need <= 0 {
		return 0, deduped
	}
	if need > len(rankedEntries) {
		need = len(rankedEntries)
	}

	// Remove entries.
	s.mu.Lock()
	for i := 0; i < need && i < len(rankedEntries); i++ {
		delete(s.entries, rankedEntries[i].e.ID)
		evicted++
	}
	s.mu.Unlock()

	// Persist the reduced store.
	s.persist()
	return evicted, deduped
}

// ---------------------------------------------------------------------------
// Deduplication
// ---------------------------------------------------------------------------

// deduplicateCorpus finds groups of entries whose keyword overlap exceeds
// the threshold and merges each group into its keeper (the entry with the
// highest confidence × access_freq). Returns the deduplicated slice and
// the count of entries removed.
func deduplicateCorpus(entries []*Entry, threshold float64) ([]*Entry, int) {
	if len(entries) <= 1 {
		return entries, 0
	}

	// Pre-build vectors.
	vecs := make(map[string]keywordVector, len(entries))
	for _, e := range entries {
		vecs[e.ID] = buildKeywordVector(e.Question + " " + e.Answer)
	}

	// Union-find groups of entries above the overlap threshold.
	type group struct {
		keeper  *Entry
		members []*Entry
	}
	parent := make(map[string]string)
	var find func(string) string
	find = func(id string) string {
		if parent[id] == "" || parent[id] == id {
			return id
		}
		root := find(parent[id])
		parent[id] = root // path compression
		return root
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			overlap := keywordOverlap(vecs[entries[i].ID], vecs[entries[j].ID])
			if overlap >= threshold {
				union(entries[i].ID, entries[j].ID)
			}
		}
	}

	// Group by root.
	groups := make(map[string]*group)
	for _, e := range entries {
		root := find(e.ID)
		g, ok := groups[root]
		if !ok {
			g = &group{}
			groups[root] = g
		}
		g.members = append(g.members, e)
		// Keeper = highest confidence × access_freq.
		if g.keeper == nil ||
			(e.Confidence*float64(1+e.AccessCount) > g.keeper.Confidence*float64(1+g.keeper.AccessCount)) {
			g.keeper = e
		}
	}

	// Build the deduplicated list.
	var out []*Entry
	removed := 0
	for _, g := range groups {
		if len(g.members) == 1 {
			out = append(out, g.members[0])
			continue
		}
		// Merge knowledge: combine file lists, average confidence, keep
		// the highest-quality answer.
		g.keeper.Files = mergeFileLists(g.members)
		var confSum float64
		for _, m := range g.members {
			confSum += m.Confidence
		}
		g.keeper.Confidence = confSum / float64(len(g.members))
		out = append(out, g.keeper)
		removed += len(g.members) - 1
	}

	return out, removed
}

// mergeFileLists combines file references from deduplicated entries,
// removing duplicates while preserving order.
func mergeFileLists(entries []*Entry) []string {
	seen := make(map[string]bool)
	var out []string
	for _, e := range entries {
		for _, f := range e.Files {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

// EvictionDiagnostics is a snapshot of the eviction scoring for
// observability and debugging.
type EvictionDiagnostics struct {
	StoreSize        int     `json:"store_size"`
	SoftCap          int     `json:"soft_cap"`
	MaxEntries       int     `json:"max_entries"`
	MinObservedScore float64 `json:"min_observed_score"`
	MaxObservedScore float64 `json:"max_observed_score"`
	MeanScore        float64 `json:"mean_score"`
	DedupGroups      int     `json:"dedup_groups"`
	EvictionCandidates int   `json:"eviction_candidates"`
}

// DiagnoseEviction returns an eviction diagnostic snapshot.
func (s *Store) DiagnoseEviction(cfg EvictionConfig) EvictionDiagnostics {
	cfg.defaults(s.maxSize)

	s.mu.RLock()
	entries := make([]*Entry, 0, len(s.entries))
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	s.mu.RUnlock()

	scorer := cfg.NewEvictionScorer()
	d := EvictionDiagnostics{
		StoreSize:  len(entries),
		SoftCap:    cfg.SoftCap,
		MaxEntries: cfg.MaxEntries,
	}

	if len(entries) == 0 {
		return d
	}

	var minScore, maxScore, sumScore float64
	minScore = math.MaxFloat64
	candidates := 0
	for _, e := range entries {
		score := scorer.Score(e, entries)
		sumScore += score.Total
		if score.Total < minScore {
			minScore = score.Total
		}
		if score.Total > maxScore {
			maxScore = score.Total
		}
		if score.Total < cfg.MinScore || len(entries) > cfg.SoftCap {
			candidates++
		}
	}

	d.MinObservedScore = minScore
	d.MaxObservedScore = maxScore
	d.MeanScore = sumScore / float64(len(entries))
	d.EvictionCandidates = candidates

	// Count dedup groups.
	vecs := make(map[string]keywordVector, len(entries))
	for _, e := range entries {
		vecs[e.ID] = buildKeywordVector(e.Question + " " + e.Answer)
	}
	seen := make(map[string]bool)
	groups := 0
	for i := 0; i < len(entries); i++ {
		if seen[entries[i].ID] {
			continue
		}
		hasDup := false
		for j := i + 1; j < len(entries); j++ {
			if seen[entries[j].ID] {
				continue
			}
			if keywordOverlap(vecs[entries[i].ID], vecs[entries[j].ID]) >= cfg.DedupThreshold {
				seen[entries[j].ID] = true
				hasDup = true
			}
		}
		if hasDup {
			groups++
		}
	}
	d.DedupGroups = groups

	return d
}
