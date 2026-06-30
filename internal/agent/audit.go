// Package agent — audit.go: change audit trail for subagent file edits.
//
// Every file change made by a subagent is recorded with its pre- and post-edit
// content, producing a searchable timeline. The trail is append-only, thread-safe,
// and supports querying by time range and by file path.
package agent

import (
	"sort"
	"sync"
	"time"
)

// AuditEntry is one recorded file change attributed to a subagent. It captures
// the file path, the content before and after the edit, and the timestamp so the
// trail can be queried by time range and file.
type AuditEntry struct {
	// SubagentID identifies the subagent that made the change (the ref from
	// SubagentMeta).
	SubagentID string `json:"subagent_id"`

	// File is the workspace-relative path that was changed.
	File string `json:"file"`

	// Before is the file content before the edit. nil means the file did not
	// exist (a create).
	Before *string `json:"before,omitempty"`

	// After is the file content after the edit. nil means the file was deleted.
	After *string `json:"after,omitempty"`

	// Timestamp is when the change was recorded (UTC).
	Timestamp time.Time `json:"timestamp"`

	// Kind classifies the change: create, modify, or delete.
	Kind string `json:"kind"`

	// ChangeSize is the line count delta (+added / -removed). Positive means net
	// growth; negative means net shrinkage.
	ChangeSize int `json:"change_size"`
}

// AuditTrail is an append-only, thread-safe, searchable record of every file
// change made by subagents. The trail lives in memory for the duration of a
// session; a caller that wants persistence wraps it with a JSONL append on each
// write.
type AuditTrail struct {
	mu      sync.RWMutex
	entries []*AuditEntry
}

// NewAuditTrail creates an empty audit trail.
func NewAuditTrail() *AuditTrail {
	return &AuditTrail{}
}

// RecordChange appends a new audit entry to the trail.
func (t *AuditTrail) RecordChange(entry AuditEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	t.mu.Lock()
	t.entries = append(t.entries, &entry)
	t.mu.Unlock()
}

// RecordChangeQuick is a convenience helper that records a change without
// requiring the caller to construct a full AuditEntry.
func (t *AuditTrail) RecordChangeQuick(subagentID, file, kind string, before, after *string, changeSize int) {
	t.RecordChange(AuditEntry{
		SubagentID: subagentID,
		File:       file,
		Before:     before,
		After:      after,
		Kind:       kind,
		ChangeSize: changeSize,
		Timestamp:  time.Now().UTC(),
	})
}

// Query returns all audit entries whose timestamp falls within [start, end].
// An empty time.Time for start means "from the beginning of the trail"; an empty
// end means "to the end of the trail".
func (t *AuditTrail) Query(start, end time.Time) []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Normalize zero times to epoch and far-future so the sweep comparison is
	// simple.
	from := start
	to := end
	if to.IsZero() {
		to = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	var results []AuditEntry
	for _, e := range t.entries {
		if !e.Timestamp.Before(from) && e.Timestamp.Before(to) {
			results = append(results, *e)
		}
	}
	return results
}

// ByFile returns every audit entry for the given workspace-relative path, oldest
// first.
func (t *AuditTrail) ByFile(path string) []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []AuditEntry
	for _, e := range t.entries {
		if e.File == path {
			results = append(results, *e)
		}
	}
	return results
}

// BySubagent returns every audit entry produced by the given subagent, oldest
// first.
func (t *AuditTrail) BySubagent(subagentID string) []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var results []AuditEntry
	for _, e := range t.entries {
		if e.SubagentID == subagentID {
			results = append(results, *e)
		}
	}
	return results
}

// All returns every entry in the trail, oldest first.
func (t *AuditTrail) All() []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	results := make([]AuditEntry, len(t.entries))
	for i, e := range t.entries {
		results[i] = *e
	}
	return results
}

// Since returns entries with timestamps >= t, oldest first.
func (t *AuditTrail) Since(since time.Time) []AuditEntry {
	return t.Query(since, time.Time{})
}

// Until returns entries with timestamps <= t, oldest first.
func (t *AuditTrail) Until(until time.Time) []AuditEntry {
	return t.Query(time.Time{}, until)
}

// FilesTouched returns the set of distinct file paths appearing in the trail.
func (t *AuditTrail) FilesTouched() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	seen := make(map[string]bool)
	var paths []string
	for _, e := range t.entries {
		if !seen[e.File] {
			seen[e.File] = true
			paths = append(paths, e.File)
		}
	}
	sort.Strings(paths)
	return paths
}

// SubagentTimeline returns a chronological map of subagentID -> entries for every
// subagent that has recorded a change, sorted alphabetically by subagent ID.
func (t *AuditTrail) SubagentTimeline() map[string][]AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	m := make(map[string][]AuditEntry)
	for _, e := range t.entries {
		m[e.SubagentID] = append(m[e.SubagentID], *e)
	}
	return m
}

// ConflictCheck inspects the trail for overlapping changes: two or more
// subagents editing the same file within the given time window. Returns a map of
// file path -> subagent IDs that collided.
func (t *AuditTrail) ConflictCheck(window time.Duration) map[string][]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.entries) < 2 {
		return nil
	}

	// Group entries by file.
	byFile := make(map[string][]*AuditEntry)
	for _, e := range t.entries {
		byFile[e.File] = append(byFile[e.File], e)
	}

	conflicts := make(map[string][]string)
	for file, entries := range byFile {
		if len(entries) < 2 {
			continue
		}
		seen := make(map[string]bool)
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				// Different subagents only — same subagent edits to the same file
				// are sequential, not conflicting.
				if entries[i].SubagentID == entries[j].SubagentID {
					continue
				}
				delta := entries[i].Timestamp.Sub(entries[j].Timestamp)
				if delta < 0 {
					delta = -delta
				}
				if delta <= window {
					if !seen[entries[i].SubagentID] {
						conflicts[file] = append(conflicts[file], entries[i].SubagentID)
						seen[entries[i].SubagentID] = true
					}
					if !seen[entries[j].SubagentID] {
						conflicts[file] = append(conflicts[file], entries[j].SubagentID)
						seen[entries[j].SubagentID] = true
					}
				}
			}
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return conflicts
}

// Len returns the total number of entries in the trail.
func (t *AuditTrail) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}
