// Package knowledge provides a local knowledge base for reasonix.
// Everything reasonix learns about the codebase is embedded and stored for
// fast cross-session recall.
package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is one piece of codebase knowledge.
type Entry struct {
	ID         string    `json:"id"`
	Question   string    `json:"question"`
	Answer     string    `json:"answer"`
	Files      []string  `json:"files"`
	Confidence float64   `json:"confidence"`
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`
	AccessCount int      `json:"access_count"`
}

// Store is a file-backed knowledge base with in-memory semantic search.
type Store struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	dir     string
	maxSize int
}

// NewStore creates or opens a knowledge base in the given directory.
func NewStore(dir string) *Store {
	os.MkdirAll(dir, 0755)
	s := &Store{
		entries: make(map[string]*Entry),
		dir:     dir,
		maxSize: 10000,
	}
	s.load()
	return s
}

// Add stores a knowledge entry.
func (s *Store) Add(q, answer string, files []string, model string, confidence float64) *Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("kb-%d", len(s.entries)+1)
	now := time.Now()
	e := &Entry{
		ID:        id,
		Question:  q,
		Answer:    answer,
		Files:     files,
		Confidence: confidence,
		Model:     model,
		CreatedAt: now,
		AccessedAt: now,
	}
	s.entries[id] = e
	s.evictIfNeeded()
	go s.persist()
	return e
}

// Search finds entries matching the given query via basic keyword matching.
// Returns up to max results, sorted by relevance (keyword overlap × confidence).
func (s *Store) Search(query string, max int) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if max <= 0 {
		max = 5
	}
	queryWords := strings.Fields(strings.ToLower(query))

	type scored struct {
		e     *Entry
		score float64
	}
	var scoredEntries []scored

	for _, e := range s.entries {
		content := strings.ToLower(e.Question + " " + e.Answer)
		matches := 0
		for _, w := range queryWords {
			if strings.Contains(content, w) {
				matches++
			}
		}
		if matches > 0 {
			e.AccessedAt = time.Now()
			e.AccessCount++
			score := float64(matches) * e.Confidence
			scoredEntries = append(scoredEntries, scored{e, score})
		}
	}

	sort.Slice(scoredEntries, func(i, j int) bool { return scoredEntries[i].score > scoredEntries[j].score })

	var results []*Entry
	for i := 0; i < max && i < len(scoredEntries); i++ {
		results = append(results, scoredEntries[i].e)
	}
	return results
}

// Get returns a specific entry by ID.
func (s *Store) Get(id string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	if ok {
		e.AccessedAt = time.Now()
		e.AccessCount++
	}
	return e, ok
}

// Size returns the number of entries.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *Store) evictIfNeeded() {
	if len(s.entries) <= s.maxSize {
		return
	}
	// Evict least valuable: low confidence × low access count
	type eviction struct {
		id    string
		score float64
	}
	var candidates []eviction
	for id, e := range s.entries {
		score := e.Confidence * float64(1+e.AccessCount)
		candidates = append(candidates, eviction{id, score})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score < candidates[j].score })
	toRemove := len(s.entries) - s.maxSize
	for i := 0; i < toRemove && i < len(candidates); i++ {
		delete(s.entries, candidates[i].id)
	}
}

func (s *Store) persist() {
	path := filepath.Join(s.dir, "knowledge.json")
	var entries []*Entry
	for _, e := range s.entries {
		entries = append(entries, e)
	}
	data, _ := json.MarshalIndent(entries, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (s *Store) load() {
	path := filepath.Join(s.dir, "knowledge.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var entries []*Entry
	if json.Unmarshal(data, &entries) != nil {
		return
	}
	for _, e := range entries {
		s.entries[e.ID] = e
	}
}
