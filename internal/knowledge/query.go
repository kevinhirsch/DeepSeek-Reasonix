package knowledge

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// QueryResult wraps a knowledge entry with its relevance score.
type QueryResult struct {
	Entry    *Entry
	Score    float64
	ExactMatch bool
}

// SemanticQuery performs a keyword-overlap search against the knowledge base
// and returns ranked results. Scores favor exact question matches and recent
// high-confidence entries.
func SemanticQuery(store *Store, query string, max int) []QueryResult {
	if max <= 0 {
		max = 5
	}
	queryLower := strings.ToLower(strings.TrimSpace(query))
	queryWords := strings.Fields(queryLower)

	store.mu.RLock()
	defer store.mu.RUnlock()

	var results []QueryResult
	now := time.Now()

	for _, e := range store.entries {
		content := strings.ToLower(e.Question + " " + truncateForQuery(e.Answer, 500))
		questionLower := strings.ToLower(e.Question)

		// Exact question match gets highest boost
		exactMatch := questionLower == queryLower

		// Keyword overlap scoring
		matches := 0
		for _, w := range queryWords {
			if strings.Contains(content, w) {
				matches++
			}
		}
		if matches == 0 && !exactMatch {
			continue
		}

		// Score: keyword overlap × confidence × recency
		ageDays := now.Sub(e.CreatedAt).Hours() / 24
		recencyScore := 1.0
		if ageDays > 0 {
			recencyScore = 1.0 / (1.0 + ageDays/30.0)
		}
		overlapScore := float64(matches) / float64(len(queryWords))
		score := overlapScore * e.Confidence * recencyScore

		if exactMatch {
			score *= 2.0
			if e.AccessCount > 0 {
				score *= 1.5
			}
		}

		results = append(results, QueryResult{
			Entry:      e,
			Score:      score,
			ExactMatch: exactMatch,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > max {
		results = results[:max]
	}
	return results
}

// InstantAnswer returns a pre-computed answer from the KB if a high-confidence
// exact match exists. Returns empty string if no match.
func InstantAnswer(store *Store, query string, minConfidence float64) string {
	results := SemanticQuery(store, query, 1)
	if len(results) == 0 {
		return ""
	}
	r := results[0]
	if r.ExactMatch && r.Entry.Confidence >= minConfidence {
		r.Entry.AccessedAt = time.Now()
		r.Entry.AccessCount++
		return fmt.Sprintf("KB match (%.0f%% confidence, %d previous queries):\n\n%s",
			r.Entry.Confidence*100, r.Entry.AccessCount, r.Entry.Answer)
	}
	return ""
}

func truncateForQuery(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
