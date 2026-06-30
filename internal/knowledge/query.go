// Package knowledge provides a local knowledge base for reasonix.
//
// Query.go implements semantic search and instant-answer retrieval
// (Issue #20 Comp 16). Everything reasonix learns about the codebase is
// embedded and stored. Repeat questions are answered instantly via the
// knowledge base; novel questions trigger explorer subagents that populate
// the KB for future use.
package knowledge

import (
	"sort"
	"strings"
	"time"
)

// QueryResult wraps a knowledge entry with its relevance score.
type QueryResult struct {
	Entry      *Entry
	Score      float64
	ExactMatch bool
}

// SemanticQuery performs a keyword-overlap search with TF-IDF-style scoring
// across the knowledge base, factoring in confidence, recency, and access
// frequency. Returns results sorted by relevance (highest first).
func SemanticQuery(store *Store, query string, max int) []QueryResult {
	if max <= 0 {
		max = 5
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	if queryLower == "" {
		return nil
	}

	queryWords := strings.Fields(queryLower)

	// Get all entries from the store via the exported Search method with a
	// broad keyword match, then re-rank with semantic scoring.
	allEntries := store.Search(query, store.Size())
	if len(allEntries) == 0 {
		return nil
	}

	now := time.Now()
	var results []QueryResult

	for _, e := range allEntries {
		content := strings.ToLower(e.Question + " " + truncateForQuery(e.Answer, 500))
		questionLower := strings.ToLower(e.Question)

		exactMatch := questionLower == queryLower

		matches := 0
		for _, w := range queryWords {
			if strings.Contains(content, w) {
				matches++
			}
		}
		if matches == 0 && !exactMatch {
			continue
		}

		ageDays := now.Sub(e.AccessedAt).Hours() / 24
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

		// Update access metadata.
		e.AccessedAt = now
		e.AccessCount++
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > max {
		results = results[:max]
	}

	return results
}

// InstantAnswer returns a pre-computed answer from the KB if a high-confidence
// exact match exists. Returns an empty string when no cached answer is available,
// signaling that an explorer subagent should investigate the question.
func InstantAnswer(store *Store, query string, minConfidence float64) string {
	results := SemanticQuery(store, query, 1)
	if len(results) == 0 {
		return ""
	}
	r := results[0]
	if r.ExactMatch && r.Entry.Confidence >= minConfidence {
		return r.Entry.Answer
	}
	return ""
}

// HybridSearch combines semantic query results with recency-boosted scoring
// for improved ranking. Used when the caller wants to weigh recent answers
// more heavily even when they aren't the best keyword match.
func HybridSearch(store *Store, query string, max int, recencyWeight float64) []QueryResult {
	results := SemanticQuery(store, query, max*2)
	if len(results) == 0 {
		return nil
	}

	now := time.Now()
	for i := range results {
		entry := results[i].Entry
		ageHours := now.Sub(entry.AccessedAt).Hours()
		if ageHours > 0 {
			decay := 1.0 / (1.0 + ageHours/168.0) // one-week half-life
			results[i].Score = results[i].Score*(1.0-recencyWeight) + decay*recencyWeight
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > max {
		results = results[:max]
	}

	return results
}

// CrossReference finds KB entries that reference the same files as a query,
// providing broader context when a direct match isn't found.
func CrossReference(store *Store, filePath string, max int) []*Entry {
	if max <= 0 {
		max = 3
	}

	allEntries := store.Search(filePath, store.Size())
	if len(allEntries) == 0 {
		return nil
	}

	baseName := baseName(filePath)
	var refs []*Entry
	for _, e := range allEntries {
		for _, f := range e.Files {
			if strings.Contains(f, baseName) {
				refs = append(refs, e)
				break
			}
		}
	}

	if len(refs) > max {
		refs = refs[:max]
	}

	return refs
}

func truncateForQuery(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
