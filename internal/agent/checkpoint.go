// Package agent — checkpoint.go: per-subagent snapshots for granular undo.
//
// Extends the workspace-level checkpointing in internal/checkpoint with snapshots
// that are scoped to a single subagent invocation. When a subagent touches a file,
// its pre-edit content is captured here so that a single subagent's work can be
// rolled back independently of other subagents. Before restoring, the system
// checks for overlapping changes made by other subagents and emits a conflict
// warning rather than silently overwriting work.
package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"reasonix/internal/diff"
	"reasonix/internal/fileutil"
	fileenc "reasonix/internal/fileutil/encoding"
	"reasonix/internal/nilutil"
)

// SubagentFileSnap is one file's state at the moment a subagent first touched it.
// Content == nil means the file did not exist then, so a restore deletes it.
type SubagentFileSnap struct {
	Path     string        `json:"path"`
	Content  *string       `json:"content"`
	Encoding *fileenc.Kind `json:"encoding,omitempty"`
}

// PerSubagentSnapshot captures the pre-edit state of every distinct file touched
// by a single subagent during one invocation. It is the unit of granular undo:
// restoring this snapshot rolls back only this subagent's changes.
type PerSubagentSnapshot struct {
	// SubagentID is the subagent ref (from SubagentMeta.Ref).
	SubagentID string `json:"subagent_id"`

	// CreatedAt is when the snapshot was first created (the subagent's start time).
	CreatedAt time.Time `json:"created_at"`

	// ParentTurn is the user turn number that spawned this subagent.
	ParentTurn int `json:"parent_turn"`

	// Files is the set of pre-edit file contents captured before the subagent
	// touched them.
	Files []SubagentFileSnap `json:"files"`

	// WorkspaceRoot is the absolute workspace root at snapshot time (needed for
	// path-escape guards during restore).
	WorkspaceRoot string `json:"workspace_root"`
}

// PerSubagentSnapshotStore holds per-subagent snapshots in memory and provides
// conflict detection on restore. All methods are safe for concurrent use.
type PerSubagentSnapshotStore struct {
	mu        sync.Mutex
	snapshots []*PerSubagentSnapshot
	seen      map[string]map[string]bool // subagentID → path → already snapshotted
}

// NewPerSubagentSnapshotStore creates an empty per-subagent snapshot store.
func NewPerSubagentSnapshotStore() *PerSubagentSnapshotStore {
	return &PerSubagentSnapshotStore{
		seen: make(map[string]map[string]bool),
	}
}

// Begin starts a new snapshot for a subagent. If a snapshot for this subagent
// already exists (resumed invocation), it is returned instead.
func (s *PerSubagentSnapshotStore) Begin(subagentID string, parentTurn int, workspaceRoot string) *PerSubagentSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether this subagent already has a snapshot (fork / resume).
	for _, snap := range s.snapshots {
		if snap.SubagentID == subagentID {
			return snap
		}
	}

	snap := &PerSubagentSnapshot{
		SubagentID:    subagentID,
		CreatedAt:     time.Now().UTC(),
		ParentTurn:    parentTurn,
		WorkspaceRoot: workspaceRoot,
	}
	s.snapshots = append(s.snapshots, snap)
	if _, ok := s.seen[subagentID]; !ok {
		s.seen[subagentID] = make(map[string]bool)
	}
	return snap
}

// Snapshot records the pre-edit state of a file the subagent is about to change.
// Only the first touch of a path by this subagent is kept.
func (s *PerSubagentSnapshotStore) Snapshot(subagentID string, ch diff.Change) {
	if ch.Path == "" || subagentID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var snap *PerSubagentSnapshot
	for _, sn := range s.snapshots {
		if sn.SubagentID == subagentID {
			snap = sn
			break
		}
	}
	if snap == nil {
		return
	}

	// Only first touch per subagent per path is recorded.
	paths, ok := s.seen[subagentID]
	if !ok {
		paths = make(map[string]bool)
		s.seen[subagentID] = paths
	}
	if paths[ch.Path] {
		return
	}
	paths[ch.Path] = true

	var content *string
	if ch.Kind != diff.Create {
		old := ch.OldText
		content = &old
	}

	var enc *fileenc.Kind
	if ch.Kind != diff.Create {
		kind := detectFileEncodingSafe(snap.WorkspaceRoot, ch.Path)
		enc = &kind
	}

	snap.Files = append(snap.Files, SubagentFileSnap{
		Path:     ch.Path,
		Content:  content,
		Encoding: enc,
	})
}

// Get returns the snapshot for a given subagent, or nil if none exists.
func (s *PerSubagentSnapshotStore) Get(subagentID string) *PerSubagentSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, snap := range s.snapshots {
		if snap.SubagentID == subagentID {
			return snap
		}
	}
	return nil
}

// Remove discards a subagent's snapshot. Used when a subagent's results are
// accepted and no rollback is needed.
func (s *PerSubagentSnapshotStore) Remove(subagentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, snap := range s.snapshots {
		if snap.SubagentID == subagentID {
			s.snapshots = append(s.snapshots[:i], s.snapshots[i+1:]...)
			break
		}
	}
	delete(s.seen, subagentID)
}

// List returns all snapshots sorted by creation time, oldest first.
func (s *PerSubagentSnapshotStore) List() []*PerSubagentSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]*PerSubagentSnapshot, len(s.snapshots))
	copy(copied, s.snapshots)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].CreatedAt.Before(copied[j].CreatedAt)
	})
	return copied
}

// FilesTouched returns the set of paths touched by a specific subagent.
func (s *PerSubagentSnapshotStore) FilesTouched(subagentID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var paths []string
	if seen, ok := s.seen[subagentID]; ok {
		for p := range seen {
			paths = append(paths, p)
		}
		sort.Strings(paths)
	}
	return paths
}

// OverlapReport describes which other subagents have modified files in common
// with the given subagent since its snapshot was taken. This is used to warn
// before a granular undo.
type OverlapReport struct {
	// HasConflict is true when at least one overlapping change was found.
	HasConflict bool `json:"has_conflict"`

	// Conflicts maps file path → list of subagent IDs that edited that file after
	// the target subagent's snapshot.
	Conflicts map[string][]string `json:"conflicts,omitempty"`

	// Message is a human-readable summary of the conflicts.
	Message string `json:"message,omitempty"`
}

// CheckOverlap inspects whether any other subagent has modified the same files
// as the given subagent after its snapshot was created. Returns an OverlapReport
// describing the conflicts.
//
// The comparison uses the store's own data: each snapshot's CreatedAt timestamp
// determines ordering. This means only snapshots recorded in THIS store are
// considered — external changes (e.g., user edits) are not detected here.
func (s *PerSubagentSnapshotStore) CheckOverlap(subagentID string) *OverlapReport {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target *PerSubagentSnapshot
	for _, snap := range s.snapshots {
		if snap.SubagentID == subagentID {
			target = snap
			break
		}
	}
	if target == nil {
		return nil
	}

	// Collect the set of paths the target subagent touched.
	targetPaths := make(map[string]bool)
	if seen, ok := s.seen[subagentID]; ok {
		for p := range seen {
			targetPaths[p] = true
		}
	}
	if len(targetPaths) == 0 {
		return &OverlapReport{HasConflict: false}
	}

	// Find newer subagents that touched the same paths.
	conflicts := make(map[string][]string)
	for _, other := range s.snapshots {
		if other.SubagentID == subagentID {
			continue
		}
		// Only consider subagents that started after the target.
		if !other.CreatedAt.After(target.CreatedAt) {
			continue
		}
		otherSeen, ok := s.seen[other.SubagentID]
		if !ok {
			continue
		}
		for otherPath := range otherSeen {
			if targetPaths[otherPath] {
				conflicts[otherPath] = append(conflicts[otherPath], other.SubagentID)
			}
		}
	}

	if len(conflicts) == 0 {
		return &OverlapReport{HasConflict: false}
	}

	var msgLines []string
	for p, subs := range conflicts {
		for _, sid := range subs {
			msgLines = append(msgLines, fmt.Sprintf("  %s was also edited by subagent %s", p, sid))
		}
	}
	sort.Strings(msgLines)

	return &OverlapReport{
		HasConflict: true,
		Conflicts:   conflicts,
		Message: fmt.Sprintf(
			"Undoing subagent %s will overwrite changes made by %d other subagent(s):\n%s",
			subagentID, len(conflicts),
			joinLines(msgLines),
		),
	}
}

// Restore reverts the workspace files touched by this snapshot to their
// pre-subagent state. It returns the paths written and deleted, and any errors
// encountered. Restore does NOT remove the snapshot from the store — call Remove
// separately if the rollback is final.
//
// Restore uses safe-path resolution (see checkpoint.safePath) to prevent writes
// outside the recorded workspace root.
func (snap *PerSubagentSnapshot) Restore() (written, deleted []string, err error) {
	root := snap.WorkspaceRoot

	for _, f := range snap.Files {
		abs, pathErr := safeResolvePath(root, f.Path)
		if pathErr != nil {
			err = errors.Join(err, pathErr)
			continue
		}

		// Content nil → file didn't exist → delete it.
		if f.Content == nil {
			if rmErr := os.Remove(abs); rmErr == nil {
				deleted = append(deleted, f.Path)
			} else if !os.IsNotExist(rmErr) {
				err = errors.Join(err, rmErr)
			}
			continue
		}

		if mkErr := os.MkdirAll(filepath.Dir(abs), 0o755); mkErr != nil {
			err = errors.Join(err, mkErr)
			continue
		}

		enc := fileenc.UTF8
		if f.Encoding != nil {
			enc = *f.Encoding
		}

		if wErr := os.WriteFile(abs, fileenc.Encode(*f.Content, enc), 0o644); wErr != nil {
			err = errors.Join(err, wErr)
			continue
		}
		written = append(written, f.Path)
	}
	return written, deleted, err
}

// safeResolvePath resolves p against root and rejects anything escaping it.
func safeResolvePath(root, p string) (string, error) {
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
			return "", fmt.Errorf("path %q escapes workspace %q", p, root)
		}
	}
	return abs, nil
}

// detectFileEncodingSafe tries to detect encoding; errors are silent (returns
// UTF8 as default).
func detectFileEncodingSafe(root, p string) fileenc.Kind {
	abs, err := safeResolvePath(root, p)
	if err != nil {
		return fileenc.UTF8
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return fileenc.UTF8
	}
	enc, ok := fileenc.Detect(b)
	if !ok {
		return fileenc.UTF8
	}
	return enc
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// Ensure nilutil import is used.
var _ = nilutil.Nil
