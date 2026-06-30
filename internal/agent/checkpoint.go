// Package agent — per-subagent checkpoint snapshots.
//
// PerSubagentCheckpoint provides revert-button support for individual
// subagent contributions. Before a subagent writes a file, the system
// takes a CheckpointSnapshot of the file's current state. If the user
// chooses to revert a subagent's work, the snapshot is restored.
//
// Overlapping changes (two subagents editing the same file) are detected
// and flagged with conflict warnings so the user can decide which version
// to keep.
//
// This builds on the existing checkpoint.Store (per-turn snapshots) by
// adding subagent-scoped granularity. Each PerSubagentCheckpoint is owned
// by a specific subagent run; when the subagent finishes, its snapshots
// are finalised and can no longer be extended.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"reasonix/internal/diff"
	"reasonix/internal/fileutil"
	fileenc "reasonix/internal/fileutil/encoding"
)

// ---------------------------------------------------------------------------
// Checkpoint snapshot
// ---------------------------------------------------------------------------

// CheckpointSnapshot is one file's state captured before a subagent writes
// to it. Content == nil means the file did not exist at snapshot time, so
// a revert deletes it (or leaves the delete in place if it was a creation).
type CheckpointSnapshot struct {
	// Path is the workspace-relative file path.
	Path string `json:"path"`

	// Content is the file's full text at snapshot time. nil means the
	// file didn't exist — reverting deletes whatever the subagent wrote.
	Content *string `json:"content,omitempty"`

	// Encoding is the detected file encoding at snapshot time so the
	// restore writes with the same encoding.
	Encoding *fileenc.Kind `json:"encoding,omitempty"`

	// SnapshotAt is when this snapshot was taken.
	SnapshotAt time.Time `json:"snapshot_at"`

	// Hash is a content fingerprint for quick change detection.
	Hash string `json:"hash"`
}

// ---------------------------------------------------------------------------
// Per-subagent checkpoint
// ---------------------------------------------------------------------------

// SubagentCheckpointStatus tracks the lifecycle of a subagent's snapshots.
type SubagentCheckpointStatus string

const (
	// CheckpointActive — the subagent is still running and may take more
	// snapshots. Snapshots can be added.
	CheckpointActive SubagentCheckpointStatus = "active"

	// CheckpointFinalised — the subagent has finished (succeeded or failed).
	// Snapshots are frozen and can only be read, not extended.
	CheckpointFinalised SubagentCheckpointStatus = "finalised"

	// CheckpointReverted — the user has reverted this subagent's work.
	// The snapshots are kept for forensic purposes but the workspace has
	// been restored to pre-subagent state.
	CheckpointReverted SubagentCheckpointStatus = "reverted"
)

// PerSubagentCheckpoint manages the pre-write snapshots for a single
// subagent invocation. It is safe for concurrent use — the agent takes
// snapshots from tool goroutines while the controller may read the list
// for the UI.
type PerSubagentCheckpoint struct {
	// SubagentID is the subagent reference (sa_...) or tool-call ID.
	SubagentID string `json:"subagent_id"`

	// SubagentName is a human-readable label (e.g. the task description
	// or workflow stage name).
	SubagentName string `json:"subagent_name"`

	// Status is the current lifecycle state.
	Status SubagentCheckpointStatus `json:"status"`

	// CreatedAt is when the subagent was spawned.
	CreatedAt time.Time `json:"created_at"`

	// FinalisedAt is when Finalise was called.
	FinalisedAt time.Time `json:"finalised_at,omitempty"`

	// RevertedAt is when Revert was called.
	RevertedAt time.Time `json:"reverted_at,omitempty"`

	mu        sync.Mutex
	snapshots []CheckpointSnapshot
	seen      map[string]bool // dedup paths within this subagent
	root      string          // workspace root for path-safe restore
	dir       string          // optional persistence directory
}

// NewPerSubagentCheckpoint creates a checkpoint tracker for a subagent.
// dir is the optional persistence directory; "" disables disk persistence.
// root is the workspace root for safe path resolution during restore.
func NewPerSubagentCheckpoint(subagentID, subagentName, root, dir string) *PerSubagentCheckpoint {
	c := &PerSubagentCheckpoint{
		SubagentID:   strings.TrimSpace(subagentID),
		SubagentName: strings.TrimSpace(subagentName),
		Status:       CheckpointActive,
		CreatedAt:    time.Now(),
		seen:         make(map[string]bool),
		root:         root,
		dir:          dir,
	}
	if dir != "" {
		c.load()
	}
	return c
}

func (c *PerSubagentCheckpoint) load() {
	if c == nil || c.dir == "" {
		return
	}
	path := c.persistencePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var loaded struct {
		SubagentID   string                    `json:"subagent_id"`
		SubagentName string                    `json:"subagent_name"`
		Status       SubagentCheckpointStatus  `json:"status"`
		CreatedAt    time.Time                 `json:"created_at"`
		FinalisedAt  time.Time                 `json:"finalised_at,omitempty"`
		RevertedAt   time.Time                 `json:"reverted_at,omitempty"`
		Snapshots    []CheckpointSnapshot      `json:"snapshots"`
	}
	if json.Unmarshal(data, &loaded) != nil {
		return
	}
	c.SubagentName = loaded.SubagentName
	c.Status = loaded.Status
	c.CreatedAt = loaded.CreatedAt
	c.FinalisedAt = loaded.FinalisedAt
	c.RevertedAt = loaded.RevertedAt
	c.snapshots = loaded.Snapshots
	for _, s := range loaded.Snapshots {
		c.seen[s.Path] = true
	}
}

// Snapshot records the current on-disk state of a file before the
// subagent writes to it. Only the first Snapshot for a given path is kept
// (that is the file's state before ANY subagent edit, which is what a
// revert must restore). A no-op when the checkpoint is not Active.
func (c *PerSubagentCheckpoint) Snapshot(ch diff.Change) {
	if ch.Path == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Status != CheckpointActive {
		return
	}
	if c.seen[ch.Path] {
		return
	}
	c.seen[ch.Path] = true

	if ch.Kind != diff.Create {
		// File existed before the subagent's write — capture its current
		// content from disk so we can restore it on revert.
		text, enc, err := c.readFile(ch.Path)
		if err != nil {
			slog.Warn("per-subagent checkpoint: read failed, snapshot will delete on revert",
				"subagent", c.SubagentID, "path", ch.Path, "err", err)
		}
		snap := CheckpointSnapshot{
			Path:       ch.Path,
			Content:    text,
			Encoding:   enc,
			SnapshotAt: time.Now(),
			Hash:       contentHashPtrCheckpoint(text),
		}
		c.snapshots = append(c.snapshots, snap)
	} else {
		// File is being created — pre-subagent state is "doesn't exist"
		// (nil Content). Reverting deletes the created file.
		c.snapshots = append(c.snapshots, CheckpointSnapshot{
			Path:       ch.Path,
			Content:    nil,
			SnapshotAt: time.Now(),
		})
	}

	c.persistLocked()
}

// readFile reads a file's text and encoding from disk. Returns nil
// pointers on error.
func (c *PerSubagentCheckpoint) readFile(path string) (*string, *fileenc.Kind, error) {
	abs, err := safeCheckpointPath(c.root, path)
	if err != nil {
		return nil, nil, err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil, err
	}
	enc, _ := fileenc.Detect(b)
	text := string(fileenc.Decode(b, enc))
	return &text, &enc, nil
}

// safeCheckpointPath resolves a workspace-relative path against the root
// and rejects anything that escapes the workspace.
func safeCheckpointPath(root, p string) (string, error) {
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, p)
	}
	abs = filepath.Clean(abs)
	if root != "" {
		r := filepath.Clean(root)
		if realRoot, eErr := filepath.EvalSymlinks(r); eErr == nil {
			r = realRoot
		}
		if realAbs, eErr := filepath.EvalSymlinks(abs); eErr == nil {
			abs = realAbs
		}
		rel, relErr := filepath.Rel(r, abs)
		if relErr != nil || !filepath.IsLocal(rel) {
			return "", fmt.Errorf("subagent checkpoint path %q escapes workspace %q", p, root)
		}
	}
	return abs, nil
}

// Finalise marks this checkpoint as complete. After this, Snapshot is a
// no-op. Call when the subagent finishes (success or failure).
func (c *PerSubagentCheckpoint) Finalise() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Status == CheckpointActive {
		c.Status = CheckpointFinalised
		c.FinalisedAt = time.Now()
		c.persistLocked()
	}
}

// Revert restores every file this subagent touched to its pre-subagent
// state. Files that were created by the subagent (nil Content) are deleted.
// Files touched by other subagents AFTER this snapshot was taken are NOT
// reverted — they are flagged as conflicts.
//
// laterSnapshots maps file paths to the earliest snapshot time by any
// other subagent after this one. If a file was touched later, it is
// skipped and returned in the skipped slice.
//
// Returns the paths written, the paths deleted, a list of overlapping
// paths that were skipped (touched by a later subagent), and any error.
func (c *PerSubagentCheckpoint) Revert(laterSnapshots map[string]time.Time) (written, deleted, skipped []string, err error) {
	c.mu.Lock()
	if c.Status == CheckpointReverted {
		c.mu.Unlock()
		return nil, nil, nil, fmt.Errorf("subagent %q has already been reverted", c.SubagentID)
	}
	snaps := make([]CheckpointSnapshot, len(c.snapshots))
	copy(snaps, c.snapshots)
	c.mu.Unlock()

	for _, snap := range snaps {
		// Check for overlapping changes: if another subagent touched
		// this file after our snapshot, skip it and flag a conflict.
		if t, ok := laterSnapshots[snap.Path]; ok && t.After(snap.SnapshotAt) {
			skipped = append(skipped, snap.Path)
			continue
		}

		abs, pathErr := safeCheckpointPath(c.root, snap.Path)
		if pathErr != nil {
			err = errors.Join(err, pathErr)
			continue
		}

		if snap.Content == nil {
			// File didn't exist before — delete it.
			if rmErr := os.Remove(abs); rmErr == nil {
				deleted = append(deleted, snap.Path)
			} else if !os.IsNotExist(rmErr) {
				err = errors.Join(err, rmErr)
			}
			continue
		}

		// Restore the pre-subagent content.
		if mkErr := os.MkdirAll(filepath.Dir(abs), 0o755); mkErr != nil {
			err = errors.Join(err, mkErr)
			continue
		}
		enc := fileenc.UTF8
		if snap.Encoding != nil {
			enc = *snap.Encoding
		}
		if wErr := os.WriteFile(abs, fileenc.Encode(*snap.Content, enc), 0o644); wErr != nil {
			err = errors.Join(err, wErr)
			continue
		}
		written = append(written, snap.Path)
	}

	c.mu.Lock()
	c.Status = CheckpointReverted
	c.RevertedAt = time.Now()
	c.persistLocked()
	c.mu.Unlock()

	return written, deleted, skipped, err
}

// Snapshots returns a copy of all snapshots in this checkpoint.
func (c *PerSubagentCheckpoint) Snapshots() []CheckpointSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]CheckpointSnapshot, len(c.snapshots))
	copy(out, c.snapshots)
	return out
}

// TouchedPaths returns the set of file paths this subagent has touched
// (i.e. has taken a snapshot for).
func (c *PerSubagentCheckpoint) TouchedPaths() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.seen))
	for p := range c.seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// IsEmpty reports whether the checkpoint has no snapshots.
func (c *PerSubagentCheckpoint) IsEmpty() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.snapshots) == 0
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

func (c *PerSubagentCheckpoint) persistencePath() string {
	return filepath.Join(c.dir, "subagent-"+sanitiseCheckpointFilename(c.SubagentID)+".ckpt.json")
}

func (c *PerSubagentCheckpoint) persistLocked() {
	if c.dir == "" {
		return
	}
	type persisted struct {
		SubagentID   string                   `json:"subagent_id"`
		SubagentName string                   `json:"subagent_name"`
		Status       SubagentCheckpointStatus `json:"status"`
		CreatedAt    time.Time                `json:"created_at"`
		FinalisedAt  time.Time                `json:"finalised_at,omitempty"`
		RevertedAt   time.Time                `json:"reverted_at,omitempty"`
		Snapshots    []CheckpointSnapshot     `json:"snapshots"`
	}
	p := persisted{
		SubagentID:   c.SubagentID,
		SubagentName: c.SubagentName,
		Status:       c.Status,
		CreatedAt:    c.CreatedAt,
		FinalisedAt:  c.FinalisedAt,
		RevertedAt:   c.RevertedAt,
		Snapshots:    c.snapshots,
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	if mkErr := os.MkdirAll(c.dir, 0o755); mkErr != nil {
		slog.Warn("per-subagent checkpoint: create dir failed", "dir", c.dir, "err", mkErr)
		return
	}
	if wErr := fileutil.AtomicWriteFile(c.persistencePath(), data, 0o644); wErr != nil {
		slog.Warn("per-subagent checkpoint: persist failed", "subagent", c.SubagentID, "err", wErr)
	}
}

func sanitiseCheckpointFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Subagent checkpoint manager (orchestrates all per-subagent checkpoints
// for a session)
// ---------------------------------------------------------------------------

// SubagentCheckpointManager tracks all PerSubagentCheckpoint instances
// for the current session. It coordinates overlap detection across
// subagents, persists checkpoints to disk, and provides the top-level
// revert API.
type SubagentCheckpointManager struct {
	mu          sync.RWMutex
	checkpoints map[string]*PerSubagentCheckpoint // subagentID → checkpoint
	root        string
	dir         string // persistence directory
}

// NewSubagentCheckpointManager creates a manager. dir is the persistence
// directory ("" for in-memory only). root is the workspace root.
func NewSubagentCheckpointManager(root, dir string) *SubagentCheckpointManager {
	m := &SubagentCheckpointManager{
		checkpoints: make(map[string]*PerSubagentCheckpoint),
		root:        root,
		dir:         dir,
	}
	if dir != "" {
		m.loadAll()
	}
	return m
}

func (m *SubagentCheckpointManager) loadAll() {
	if m.dir == "" {
		return
	}
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "subagent-") || !strings.HasSuffix(e.Name(), ".ckpt.json") {
			continue
		}
		// Extract subagent ID from filename.
		id := strings.TrimPrefix(e.Name(), "subagent-")
		id = strings.TrimSuffix(id, ".ckpt.json")
		if id == "" {
			continue
		}
		if _, exists := m.checkpoints[id]; exists {
			continue
		}
		cp := NewPerSubagentCheckpoint(id, "", m.root, m.dir)
		m.checkpoints[id] = cp
	}
}

// BeginCheckpoint creates a new PerSubagentCheckpoint for a subagent
// that is about to start work. Returns an error if a checkpoint already
// exists for this subagent ID with active status.
func (m *SubagentCheckpointManager) BeginCheckpoint(subagentID, subagentName string) (*PerSubagentCheckpoint, error) {
	subagentID = strings.TrimSpace(subagentID)
	if subagentID == "" {
		return nil, fmt.Errorf("subagent id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.checkpoints[subagentID]; ok {
		if existing.Status == CheckpointActive {
			return nil, fmt.Errorf("subagent %q already has an active checkpoint", subagentID)
		}
		// Replace a finished/reverted checkpoint with a fresh one.
	}

	cp := NewPerSubagentCheckpoint(subagentID, subagentName, m.root, m.dir)
	m.checkpoints[subagentID] = cp
	return cp, nil
}

// GetCheckpoint returns the checkpoint for a given subagent, or nil if
// none exists.
func (m *SubagentCheckpointManager) GetCheckpoint(subagentID string) *PerSubagentCheckpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checkpoints[strings.TrimSpace(subagentID)]
}

// FinaliseCheckpoint marks a subagent's checkpoint as complete. No-op if
// the checkpoint doesn't exist.
func (m *SubagentCheckpointManager) FinaliseCheckpoint(subagentID string) {
	cp := m.GetCheckpoint(subagentID)
	if cp != nil {
		cp.Finalise()
	}
}

// RevertSubagent reverts all file changes made by the given subagent.
// It checks for overlapping changes (files also touched by other
// subagents AFTER this one's snapshots) and returns them as conflicts.
func (m *SubagentCheckpointManager) RevertSubagent(subagentID string) (written, deleted, skipped []string, err error) {
	cp := m.GetCheckpoint(subagentID)
	if cp == nil {
		return nil, nil, nil, fmt.Errorf("no checkpoint for subagent %q", subagentID)
	}

	// Build the laterSnapshots map: for every other subagent's files,
	// record the earliest snapshot time so we can detect overlaps.
	laterSnapshots := m.laterSnapshotsFor(subagentID)

	return cp.Revert(laterSnapshots)
}

// laterSnapshotsFor collects the earliest snapshot time for each file
// touched by any subagent other than excludeID, after the excludeID's
// snapshot times. Used for overlap detection during revert.
func (m *SubagentCheckpointManager) laterSnapshotsFor(excludeID string) map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get the excluded subagent's snapshot times to compare against.
	excluded, ok := m.checkpoints[excludeID]
	if !ok {
		return nil
	}
	excludedSnaps := excluded.Snapshots()
	excludedTimes := make(map[string]time.Time, len(excludedSnaps))
	for _, s := range excludedSnaps {
		excludedTimes[s.Path] = s.SnapshotAt
	}

	// Collect later snapshots from other subagents.
	later := make(map[string]time.Time)
	for id, cp := range m.checkpoints {
		if id == excludeID {
			continue
		}
		for _, s := range cp.Snapshots() {
			excludedTime, hasExcluded := excludedTimes[s.Path]
			if hasExcluded && s.SnapshotAt.After(excludedTime) {
				// Keep the earliest later snapshot time.
				if existing, exists := later[s.Path]; !exists || s.SnapshotAt.Before(existing) {
					later[s.Path] = s.SnapshotAt
				}
			}
		}
	}
	return later
}

// AllCheckpoints returns a snapshot of all tracked subagent checkpoints
// and their statuses, ordered by creation time.
func (m *SubagentCheckpointManager) AllCheckpoints() []*PerSubagentCheckpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*PerSubagentCheckpoint, 0, len(m.checkpoints))
	for _, cp := range m.checkpoints {
		out = append(out, cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// OverlapReport describes conflicting edits across subagents.
type OverlapReport struct {
	File      string
	Subagents []string // subagent IDs that touched this file, in order
	Snapshots []CheckpointSnapshot
	Conflict  bool // true when edits overlap temporally
}

// DetectOverlaps returns all files that were touched by more than one
// subagent, with the list of subagents involved and whether a conflict
// exists (overlapping time windows).
func (m *SubagentCheckpointManager) DetectOverlaps() []OverlapReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all snapshots by file.
	type agentSnap struct {
		subagentID string
		snapshot   CheckpointSnapshot
	}

	byFile := make(map[string][]agentSnap)
	for id, cp := range m.checkpoints {
		for _, s := range cp.Snapshots() {
			byFile[s.Path] = append(byFile[s.Path], agentSnap{id, s})
		}
	}

	var reports []OverlapReport
	for file, snaps := range byFile {
		if len(snaps) <= 1 {
			continue
		}

		// Collect distinct subagents in snapshot-time order.
		sort.Slice(snaps, func(i, j int) bool {
			return snaps[i].snapshot.SnapshotAt.Before(snaps[j].snapshot.SnapshotAt)
		})

		seen := make(map[string]bool)
		var agents []string
		var orderedSnaps []CheckpointSnapshot
		for _, as := range snaps {
			orderedSnaps = append(orderedSnaps, as.snapshot)
			if !seen[as.subagentID] {
				seen[as.subagentID] = true
				agents = append(agents, as.subagentID)
			}
		}

		reports = append(reports, OverlapReport{
			File:      file,
			Subagents: agents,
			Snapshots: orderedSnaps,
			Conflict:  len(agents) > 1,
		})
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].File < reports[j].File
	})
	return reports
}

// Cleanup removes all checkpoints that are finalised or reverted and
// whose persistence files are older than the given duration. Returns the
// number of checkpoints removed.
func (m *SubagentCheckpointManager) Cleanup(olderThan time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	removed := 0
	for id, cp := range m.checkpoints {
		if cp.Status == CheckpointActive {
			continue
		}
		if cp.FinalisedAt.After(cutoff) && cp.RevertedAt.After(cutoff) {
			continue
		}
		// Remove from memory.
		delete(m.checkpoints, id)
		// Remove from disk.
		path := filepath.Join(m.dir, "subagent-"+sanitiseCheckpointFilename(id)+".ckpt.json")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("per-subagent checkpoint: cleanup remove failed", "path", path, "err", err)
		}
		removed++
	}
	return removed
}

// ---------------------------------------------------------------------------
// Content hash helper
// ---------------------------------------------------------------------------

// contentHashCheckpoint returns a short hex fingerprint of s using a
// fast djb2-like hash. This is NOT cryptographic — it is a cheap, stable
// identifier so checkpoints can detect change without full comparison.
func contentHashCheckpoint(s string) string {
	if s == "" {
		return ""
	}
	var h uint64 = 5381
	for i := 0; i < len(s); i++ {
		h = ((h << 5) + h) + uint64(s[i])
	}
	return fmtHexCheckpoint(h)
}

func contentHashPtrCheckpoint(s *string) string {
	if s == nil {
		return ""
	}
	return contentHashCheckpoint(*s)
}

const hexDigitsCheckpoint = "0123456789abcdef"

func fmtHexCheckpoint(v uint64) string {
	b := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		b[i] = hexDigitsCheckpoint[v&0xf]
		v >>= 4
	}
	return string(b)
}
