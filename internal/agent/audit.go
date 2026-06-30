// Package agent — change audit trail.
//
// AuditTrail records a timeline of every file change attributed to a
// subagent. It is the source of truth for "who changed what, when" across
// the session, enabling the frontend to display per-subagent change
// summaries, revert individual subagent contributions, and search the
// timeline for specific file edits.
//
// Every write tool that modifies a file on disk (write_file, edit_file,
// multi_edit, apply_patch) must call RecordChange after the write
// succeeds. The audit trail is append-only and thread-safe.
//
// Querying:
//   QueryAudit(timeRange) returns all entries within a time window.
//   EntriesBySubagent(id) returns all changes attributed to one subagent.
//   EntriesByFile(path) returns the edit history of a single file.
package agent

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Audit entry
// ---------------------------------------------------------------------------

// FileChangeKind classifies what a change did to a file.
type FileChangeKind string

const (
	FileCreated  FileChangeKind = "created"
	FileModified FileChangeKind = "modified"
	FileDeleted  FileChangeKind = "deleted"
	FileRenamed  FileChangeKind = "renamed"
)

// AuditEntry records one file change attributed to a specific subagent.
type AuditEntry struct {
	// ID is a unique, monotonically increasing entry identifier within
	// the trail. It is assigned by RecordChange.
	ID int64 `json:"id"`

	// SubagentID is the subagent reference (sa_...) or tool-call ID that
	// made this change. Empty for the parent agent's own changes.
	SubagentID string `json:"subagent_id"`

	// File is the workspace-relative path that was changed.
	File string `json:"file"`

	// Kind describes the type of change.
	Kind FileChangeKind `json:"kind"`

	// Before is the file content before the change (or empty for creates).
	// Stored as a hash for privacy/space; the actual content lives in
	// the checkpoint snapshot.
	BeforeHash string `json:"before_hash,omitempty"`

	// After is the file content after the change (or empty for deletes).
	// Stored as a hash for privacy/space.
	AfterHash string `json:"after_hash,omitempty"`

	// BeforeLen and AfterLen are the byte lengths of the before/after
	// content, for quick size-change display without loading snapshots.
	BeforeLen int `json:"before_len"`
	AfterLen  int `json:"after_len"`

	// ToolName is the tool that made the change (e.g. "write_file",
	// "edit_file", "apply_patch").
	ToolName string `json:"tool_name"`

	// Time is when the change was recorded (wall-clock time).
	Time time.Time `json:"time"`

	// Turn is the parent-agent turn number during which this change
	// was made. 0 if unknown.
	Turn int `json:"turn,omitempty"`

	// Metadata carries optional extra context (e.g. the subagent's
	// declared role, the workflow stage name, the patch size).
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Audit trail
// ---------------------------------------------------------------------------

// AuditTrail is an append-only, searchable timeline of file changes. It is
// safe for concurrent use — RecordChange and queries can run on different
// goroutines.
type AuditTrail struct {
	mu       sync.RWMutex
	entries  []AuditEntry
	nextID   int64
	capacity int // max entries before the oldest are evicted; 0 = unbounded
}

// NewAuditTrail creates an audit trail. capacity caps the total number of
// entries stored; when the limit is reached, the oldest entries are evicted
// to make room. capacity <= 0 means unbounded.
func NewAuditTrail(capacity int) *AuditTrail {
	return &AuditTrail{
		entries:  make([]AuditEntry, 0, min(capacity, 256)),
		capacity: capacity,
	}
}

// RecordChange appends a new audit entry. It returns the entry ID.
//
// Parameters:
//   - subagentID: the subagent reference (sa_...) or tool-call ID
//   - file: workspace-relative path
//   - kind: what kind of change was made
//   - before: file content before the change (empty string for creates)
//   - after: file content after the change (empty string for deletes)
func (t *AuditTrail) RecordChange(subagentID, file string, kind FileChangeKind, before, after string) int64 {
	return t.RecordChangeWithMeta(subagentID, file, kind, before, after, "", 0, nil)
}

// RecordChangeWithMeta is RecordChange with additional metadata.
func (t *AuditTrail) RecordChangeWithMeta(subagentID, file string, kind FileChangeKind, before, after string, toolName string, turn int, meta map[string]string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	id := t.nextID
	t.nextID++

	entry := AuditEntry{
		ID:         id,
		SubagentID: strings.TrimSpace(subagentID),
		File:       file,
		Kind:       kind,
		BeforeHash: contentHash(before),
		AfterHash:  contentHash(after),
		BeforeLen:  len(before),
		AfterLen:   len(after),
		ToolName:   toolName,
		Time:       time.Now(),
		Turn:       turn,
		Metadata:   copyMeta(meta),
	}

	t.entries = append(t.entries, entry)

	// Evict oldest entries if over capacity.
	if t.capacity > 0 && len(t.entries) > t.capacity {
		excess := len(t.entries) - t.capacity
		t.entries = append([]AuditEntry(nil), t.entries[excess:]...)
	}

	return id
}

// QueryAudit returns all audit entries whose Time falls within the given
// time range. Both start and end are inclusive. If start is zero, it
// means "from the beginning". If end is zero, it means "until now".
func (t *AuditTrail) QueryAudit(start, end time.Time) []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if start.IsZero() && end.IsZero() {
		out := make([]AuditEntry, len(t.entries))
		copy(out, t.entries)
		return out
	}

	var out []AuditEntry
	for _, e := range t.entries {
		if !start.IsZero() && e.Time.Before(start) {
			continue
		}
		if !end.IsZero() && e.Time.After(end) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// EntriesBySubagent returns all audit entries attributed to the given
// subagent ID, in chronological order.
func (t *AuditTrail) EntriesBySubagent(subagentID string) []AuditEntry {
	subagentID = strings.TrimSpace(subagentID)
	t.mu.RLock()
	defer t.mu.RUnlock()

	var out []AuditEntry
	for _, e := range t.entries {
		if e.SubagentID == subagentID {
			out = append(out, e)
		}
	}
	return out
}

// EntriesByFile returns the edit history of a single file, in
// chronological order.
func (t *AuditTrail) EntriesByFile(path string) []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var out []AuditEntry
	for _, e := range t.entries {
		if e.File == path {
			out = append(out, e)
		}
	}
	return out
}

// LatestEntry returns the most recent audit entry, or nil when empty.
func (t *AuditTrail) LatestEntry() *AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.entries) == 0 {
		return nil
	}
	e := t.entries[len(t.entries)-1]
	return &e
}

// FilesTouchedBy returns a deduplicated list of file paths touched by the
// given subagent, in order of first touch.
func (t *AuditTrail) FilesTouchedBy(subagentID string) []string {
	subagentID = strings.TrimSpace(subagentID)
	t.mu.RLock()
	defer t.mu.RUnlock()

	seen := make(map[string]bool)
	var paths []string
	for _, e := range t.entries {
		if e.SubagentID == subagentID && !seen[e.File] {
			seen[e.File] = true
			paths = append(paths, e.File)
		}
	}
	return paths
}

// Subagents returns the set of distinct subagent IDs that have recorded
// changes, sorted alphabetically.
func (t *AuditTrail) Subagents() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	seen := make(map[string]bool)
	for _, e := range t.entries {
		if e.SubagentID != "" {
			seen[e.SubagentID] = true
		}
	}

	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Count returns the number of entries in the trail.
func (t *AuditTrail) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

// Snapshot returns a point-in-time copy of all entries, useful for
// serialisation or debug.
func (t *AuditTrail) Snapshot() []AuditEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]AuditEntry, len(t.entries))
	copy(out, t.entries)
	return out
}

// TruncateBefore removes all entries with Time before the given timestamp.
// Returns the number of entries removed.
func (t *AuditTrail) TruncateBefore(before time.Time) int {
	if before.IsZero() {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	cut := 0
	for _, e := range t.entries {
		if e.Time.Before(before) {
			cut++
		} else {
			break
		}
	}
	if cut == 0 {
		return 0
	}
	t.entries = append([]AuditEntry(nil), t.entries[cut:]...)
	return cut
}

// Reset clears all entries.
func (t *AuditTrail) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = t.entries[:0]
	t.nextID = 0
}

// ---------------------------------------------------------------------------
// Overlap detection between subagents
// ---------------------------------------------------------------------------

// OverlappingChange describes a file that was changed by more than one
// subagent within a time window — a potential conflict.
type OverlappingChange struct {
	File     string
	Entries  []AuditEntry // all entries for this file in the window
	Conflict bool         // true when the changes were to the same file region
}

// DetectOverlaps finds files that were changed by multiple distinct
// subagents within the given time window. This is how the system detects
// and warns about conflicting edits.
func (t *AuditTrail) DetectOverlaps(start, end time.Time) []OverlappingChange {
	entries := t.QueryAudit(start, end)

	// Group by file.
	byFile := make(map[string][]AuditEntry)
	for _, e := range entries {
		byFile[e.File] = append(byFile[e.File], e)
	}

	// Keep only files touched by multiple subagents.
	var overlaps []OverlappingChange
	for file, fileEntries := range byFile {
		agents := make(map[string]bool)
		for _, e := range fileEntries {
			if e.SubagentID != "" {
				agents[e.SubagentID] = true
			}
		}
		if len(agents) > 1 {
			// Sort by time.
			sort.Slice(fileEntries, func(i, j int) bool {
				return fileEntries[i].Time.Before(fileEntries[j].Time)
			})
			overlaps = append(overlaps, OverlappingChange{
				File:     file,
				Entries:  fileEntries,
				Conflict: true,
			})
		}
	}

	// Sort overlaps by file name for stable output.
	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].File < overlaps[j].File
	})
	return overlaps
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// contentHash returns a short fingerprint of s. This is NOT a
// cryptographic hash — it is a cheap, stable identifier so the audit
// trail can reference content without storing full file bodies. The
// actual content lives in checkpoints.
func contentHash(s string) string {
	if s == "" {
		return ""
	}
	// Fast djb2-like hash, good enough for change dedup.
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint64(s[i])
	}
	return fmtHex(h)
}

func fmtHex(v uint64) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		b[i] = hex[v&0xf]
		v >>= 4
	}
	return string(b)
}

func copyMeta(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// RevertPlan describes the set of changes that would be undone if a
// particular subagent's contributions were reverted, along with files
// that were also touched by other subagents (potential conflicts).
type RevertPlan struct {
	SubagentID string
	Files      []string          // files this subagent changed, in order
	Conflicts  []string          // files also touched by other subagents
	Entries    []AuditEntry      // the full entry list for this subagent
}

// RevertPlanFor returns a revert plan for the given subagent, describing
// which files would be affected and flagging conflicts.
func (t *AuditTrail) RevertPlanFor(subagentID string) *RevertPlan {
	subagentID = strings.TrimSpace(subagentID)
	if subagentID == "" {
		return nil
	}

	entries := t.EntriesBySubagent(subagentID)
	if len(entries) == 0 {
		return nil
	}

	// Collect files.
	files := t.FilesTouchedBy(subagentID)

	// Detect conflicts: any of these files also touched by others?
	others := t.Subagents()
	conflictFiles := make(map[string]bool)
	for _, other := range others {
		if other == subagentID {
			continue
		}
		for _, f := range t.FilesTouchedBy(other) {
			for _, myFile := range files {
				if f == myFile {
					conflictFiles[f] = true
				}
			}
		}
	}

	conflicts := make([]string, 0, len(conflictFiles))
	for f := range conflictFiles {
		conflicts = append(conflicts, f)
	}
	sort.Strings(conflicts)

	return &RevertPlan{
		SubagentID: subagentID,
		Files:      files,
		Conflicts:  conflicts,
		Entries:    entries,
	}
}
