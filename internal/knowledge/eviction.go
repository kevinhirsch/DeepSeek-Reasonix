package knowledge

import (
	"math"
	"sort"
	"strings"
	"time"
)

// EvictionPolicy defines how entries are scored for eviction.
type EvictionPolicy struct {
	MaxEntries       int     `json:"max_entries"`
	RecencyWeight    float64 `json:"recency_weight"`
	AccessWeight     float64 `json:"access_weight"`
	ConfidenceWeight float64 `json:"confidence_weight"`
	UniquenessWeight float64 `json:"uniqueness_weight"`
	DedupThreshold   float64 `json:"dedup_threshold"` // keyword overlap for merging
}

// DefaultEvictionPolicy returns sensible defaults.
func DefaultEvictionPolicy() EvictionPolicy {
	return EvictionPolicy{
		MaxEntries:       10000,
		RecencyWeight:    0.3,
		AccessWeight:     0.2,
		ConfidenceWeight: 0.35,
		UniquenessWeight: 0.15,
		DedupThreshold:   0.8,
	}
}

// EvictionScore computes a value score for an entry. Lower scores are
// evicted first. Higher scores mean the entry is more valuable.
func (p EvictionPolicy) EvictionScore(e *Entry) float64 {
	now := time.Now()
	ageDays := now.Sub(e.CreatedAt).Hours() / 24
	recencyScore := math.Exp(-ageDays / 30) * p.RecencyWeight
	accessScore := math.Log2(2+float64(e.AccessCount)) * p.AccessWeight
	confidenceScore := e.Confidence * p.ConfidenceWeight
	uniquenessScore := 1.0 * p.UniquenessWeight // simplified

	return recencyScore + accessScore + confidenceScore + uniquenessScore
}

// Evict removes the lowest-scoring entries until within MaxEntries.
func (p EvictionPolicy) Evict(entries map[string]*Entry) []string {
	if len(entries) <= p.MaxEntries {
		return nil
	}

	type scored struct {
		id    string
		score float64
	}
	var list []scored
	for id, e := range entries {
		list = append(list, scored{id, p.EvictionScore(e)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score < list[j].score })

	toRemove := len(entries) - p.MaxEntries
	var removed []string
	for i := 0; i < toRemove && i < len(list); i++ {
		removed = append(removed, list[i].id)
		delete(entries, list[i].id)
	}
	return removed
}

// Dedup merges or evicts entries with >threshold keyword overlap.
func (p EvictionPolicy) Dedup(entries map[string]*Entry) int {
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}

	merged := 0
	for i := 0; i < len(ids); i++ {
		a, ok := entries[ids[i]]
		if !ok {
			continue
		}
		for j := i + 1; j < len(ids); j++ {
			b, ok2 := entries[ids[j]]
			if !ok2 {
				continue
			}
			overlap := keywordOverlap(a.Answer, b.Answer)
			if overlap >= p.DedupThreshold {
				// Merge or evict the lower-confidence entry
				if a.Confidence >= b.Confidence {
					a.AccessCount += b.AccessCount
					delete(entries, ids[j])
				} else {
					delete(entries, ids[i])
					merged++
					break
				}
				merged++
			}
		}
	}
	return merged
}

func keywordOverlap(a, b string) float64 {
	aw := strings.Fields(strings.ToLower(a))
	bw := strings.Fields(strings.ToLower(b))
	if len(aw) == 0 || len(bw) == 0 {
		return 0
	}
	matches := 0
	for _, w := range aw {
		if strings.Contains(strings.ToLower(b), w) {
			matches++
		}
	}
	return float64(matches) / float64(len(aw))
}
