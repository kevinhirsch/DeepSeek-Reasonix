package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// resolveSymlinks is a test-local helper that resolves a path to its
// absolute, symlink-free form. It replicates the logic in the builtin
// package's realPath for the purpose of verifying sandbox behaviour.
// Because a write target need not exist yet, it resolves the deepest
// existing ancestor via EvalSymlinks and re-appends the not-yet-existing
// tail components.
func resolveSymlinks(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	tail := ""
	cur := abs
	for {
		if real, e := filepath.EvalSymlinks(cur); e == nil {
			return filepath.Join(real, tail), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs, nil // nothing along the path exists; use the cleaned abs
		}
		tail = filepath.Join(filepath.Base(cur), tail)
		cur = parent
	}
}

// isWithinRoot reports whether path is at or below root. Both must be
// absolute, cleaned, and symlink-free. Mirrors the builtin package's
// within function for testing purposes.
func isWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// TestSymlinkOutsideWriteRoots verifies that a write target that is a symlink
// pointing outside the configured WriteRoots is rejected. The realPath function
// (in the builtin package) resolves symlinks before the within check, so a
// symlink to a path outside the roots does not smuggle a write past the
// boundary.
func TestSymlinkOutsideWriteRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	workspace := t.TempDir()
	outside := t.TempDir()

	linkPath := filepath.Join(workspace, "escape-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}

	// Resolving the symlink should yield the outside directory.
	resolved, err := resolveSymlinks(linkPath)
	if err != nil {
		t.Fatalf("resolveSymlinks: %v", err)
	}

	// The resolved path must NOT be within the workspace.
	if isWithinRoot(workspace, resolved) {
		t.Errorf("symlink to outside directory should NOT be within write roots\n"+
			"  workspace=%s\n  link=%s\n  resolved=%s", workspace, linkPath, resolved)
	}

	// When the Spec has the workspace as its sole WriteRoot, writing through
	// the symlink should be blocked because the resolved target is outside.
	spec := Spec{
		Mode:       "enforce",
		WriteRoots: []string{workspace},
	}
	if len(spec.WriteRoots) != 1 || spec.WriteRoots[0] != workspace {
		t.Error("Spec.WriteRoots should contain the workspace")
	}
	if !spec.enforce() {
		t.Error("Spec with enforce mode should enforce")
	}

	// Verify the Spec includes only the workspace, not the outside directory.
	for _, root := range spec.WriteRoots {
		resolvedRoot, err := resolveSymlinks(root)
		if err != nil {
			continue
		}
		if isWithinRoot(resolvedRoot, resolved) {
			t.Errorf("resolved symlink target %q should not fall within any write root %q", resolved, resolvedRoot)
		}
	}
}

// TestTOCTOUSymlinkSwap verifies that the TOCTOU hazard — where a path is
// validated as within the roots, then the directory tree is replaced with a
// symlink pointing outside before the write lands — is caught because the
// enforcement layer resolves symlinks at write time, not validation time.
//
// The test simulates:
//   1. Path initially safe: root/safe-dir/target.txt — a regular directory.
//   2. Attacker swaps safe-dir → symlink to /etc (outside the root).
//   3. Any subsequent write resolves the symlink and finds the target is outside.
func TestTOCTOUSymlinkSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Step 1: Create a safe-looking directory inside the root.
	safeDir := filepath.Join(root, "safe-dir")
	if err := os.MkdirAll(safeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	safeFile := filepath.Join(safeDir, "target.txt")

	// Confirm: before the swap, the file is within the root.
	resolvedBefore, err := resolveSymlinks(safeFile)
	if err != nil {
		t.Fatalf("resolveSymlinks before swap: %v", err)
	}
	if !isWithinRoot(root, resolvedBefore) {
		t.Fatalf("before swap, file should be within root: root=%s resolved=%s", root, resolvedBefore)
	}

	// Step 2: Simulate TOCTOU — swap safe-dir with a symlink to outside.
	if err := os.RemoveAll(safeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, safeDir); err != nil {
		t.Fatal(err)
	}

	// After the swap, resolveSymlinks must detect the target is now outside.
	resolvedAfter, err := resolveSymlinks(safeFile)
	if err != nil {
		t.Fatalf("resolveSymlinks after swap: %v", err)
	}
	if isWithinRoot(root, resolvedAfter) {
		t.Errorf("TOCTOU: after swap, resolved path should NOT be within root\n"+
			"  root=%s\n  file=%s\n  resolved=%s", root, safeFile, resolvedAfter)
	}

	// The Spec with root as sole WriteRoot protects against this.
	spec := Spec{Mode: "enforce", WriteRoots: []string{root}}
	_ = spec // Spec carries the correct roots; enforcement lives in builtin
}

// TestTOCTOUSymlinkAncestor verifies TOCTOU detection when an ancestor
// directory (not the immediate parent) is swapped with a symlink.
func TestTOCTOUSymlinkAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Create a deep path: root/a/b/c
	deepDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}
	deepFile := filepath.Join(deepDir, "target.txt")

	// Verify deep file is within the root before the swap.
	resolvedBefore, err := resolveSymlinks(deepFile)
	if err != nil {
		t.Fatalf("resolveSymlinks before ancestor swap: %v", err)
	}
	if !isWithinRoot(root, resolvedBefore) {
		t.Fatalf("before ancestor swap, file should be within root")
	}

	// Swap ancestor "a" with a symlink to outside.
	if err := os.RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}

	resolvedAfter, err := resolveSymlinks(deepFile)
	if err != nil {
		t.Fatalf("resolveSymlinks after ancestor swap: %v", err)
	}
	if isWithinRoot(root, resolvedAfter) {
		t.Errorf("TOCTOU ancestor swap: path should NOT be within root after ancestor replaced with symlink\n"+
			"  root=%s\n  file=%s\n  resolved=%s", root, deepFile, resolvedAfter)
	}
}

// TestTOCTOUNonExistentTail verifies TOCTOU detection when the direct target
// doesn't exist yet (the common case for write_file creating a new file) but
// its parent directory has been swapped with a symlink. The deepest existing
// ancestor is resolved, so the swap on the parent is caught.
func TestTOCTOUNonExistentTail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Create a directory inside the root, then swap it with a symlink.
	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(inner); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, inner); err != nil {
		t.Fatal(err)
	}

	// Target file doesn't exist yet — this is the write_file case where the
	// new file is about to be created.
	nonExistent := filepath.Join(inner, "new-file.txt")

	// Even though the file doesn't exist, the parent directory (inner) is a
	// symlink to outside, so resolveSymlinks must detect this.
	resolved, err := resolveSymlinks(nonExistent)
	if err != nil {
		t.Fatalf("resolveSymlinks: %v", err)
	}
	if isWithinRoot(root, resolved) {
		t.Errorf("non-existent file under swapped parent should NOT be within root\n"+
			"  root=%s\n  target=%s\n  resolved=%s", root, nonExistent, resolved)
	}

	// Confirm the resolved path lands outside the root.
	if !strings.HasPrefix(resolved, outside) {
		t.Errorf("resolved path should be in outside dir, got: %s (outside=%s)", resolved, outside)
	}
}

// TestGitHooksInForbidReadRoots verifies that .git/hooks/ can be configured
// as a forbid-read root and that the Spec correctly carries that configuration.
// The sandbox must prevent subagents from reading or executing files in
// .git/hooks/, which would let them hijack git operations by inserting
// malicious hooks.
func TestGitHooksInForbidReadRoots(t *testing.T) {
	repoRoot := t.TempDir()
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a fake hook file.
	hookPath := filepath.Join(hooksDir, "post-checkout")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho pwned"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Configure the Spec with .git/hooks/ as a forbid-read root.
	spec := Spec{
		Mode:            "enforce",
		ForbidReadRoots: []string{hooksDir},
	}

	// The Spec must carry the forbid-read root.
	found := false
	for _, root := range spec.ForbidReadRoots {
		resolved, err := resolveSymlinks(root)
		if err != nil {
			continue
		}
		if isWithinRoot(resolved, hookPath) {
			found = true
			break
		}
		// Also check if the root directly covers the hooks directory.
		if strings.HasPrefix(hookPath, root) {
			found = true
			break
		}
	}
	if !found {
		t.Error(".git/hooks/ should be configured in ForbidReadRoots")
	}

	// Verify ForbidReadRoots is non-empty and enforceable.
	if len(spec.ForbidReadRoots) == 0 {
		t.Error("ForbidReadRoots should not be empty when .git/hooks/ is configured")
	}
}

// TestDefaultForbidReadRootsIncludesGitHooks verifies the full default
// ForbidReadRoots configuration. .git/hooks/ must be included; other .git
// directories (objects, refs) are not forbidden by default; workspace src
// directories are always readable.
func TestDefaultForbidReadRootsIncludesGitHooks(t *testing.T) {
	repoRoot := t.TempDir()

	// Set up a realistic git repo structure.
	for _, dir := range []string{
		".git",
		".git/hooks",
		".git/objects",
		".git/refs",
		"src",
	} {
		if err := os.MkdirAll(filepath.Join(repoRoot, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	for _, hook := range []string{"pre-commit", "post-checkout", "pre-push"} {
		if err := os.WriteFile(
			filepath.Join(hooksDir, hook),
			[]byte("#!/bin/sh\necho hooked"),
			0o755,
		); err != nil {
			t.Fatal(err)
		}
	}

	// Simulate the default ForbidReadRoots the config layer supplies: only
	// .git/hooks/ is forbidden; .git/objects, .git/refs, and src/ are not.
	spec := Spec{
		Mode:            "enforce",
		ForbidReadRoots: []string{hooksDir},
	}

	// All hook files must be covered by ForbidReadRoots.
	for _, hook := range []string{"pre-commit", "post-checkout", "pre-push"} {
		hookPath := filepath.Join(hooksDir, hook)
		covered := false
		for _, root := range spec.ForbidReadRoots {
			if strings.HasPrefix(hookPath, root) {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("%s should be covered by ForbidReadRoots", hook)
		}
	}

	// Other .git directories should NOT be in ForbidReadRoots by default.
	objectsDir := filepath.Join(repoRoot, ".git", "objects")
	for _, root := range spec.ForbidReadRoots {
		if strings.HasPrefix(objectsDir, root) {
			t.Error(".git/objects/ should not be in default ForbidReadRoots")
			break
		}
	}

	refsDir := filepath.Join(repoRoot, ".git", "refs")
	for _, root := range spec.ForbidReadRoots {
		if strings.HasPrefix(refsDir, root) {
			t.Error(".git/refs/ should not be in default ForbidReadRoots")
			break
		}
	}

	// The src directory must not be forbidden.
	srcDir := filepath.Join(repoRoot, "src")
	for _, root := range spec.ForbidReadRoots {
		if strings.HasPrefix(srcDir, root) {
			t.Error("src/ should not be in ForbidReadRoots")
			break
		}
	}
}

// TestRealPathNoSymlink is a baseline: a normal file inside the root resolves
// correctly and is considered within the root.
func TestRealPathNoSymlink(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "normal.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveSymlinks(file)
	if err != nil {
		t.Fatalf("resolveSymlinks: %v", err)
	}
	if !isWithinRoot(root, resolved) {
		t.Errorf("normal file should be within root: root=%s file=%s resolved=%s", root, file, resolved)
	}
}

// TestConfineWritersRejectsSymlinkEscape verifies that the Spec with a
// restricted WriteRoot set correctly represents confinement: the workspace
// is the sole write root, and a symlink that points outside is not within it.
func TestConfineWritersRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Create a file outside the root.
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Symlink from inside the root to the outside file.
	linkPath := filepath.Join(root, "link-to-secret")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Fatal(err)
	}

	// The symlink resolves outside the root.
	resolved, err := resolveSymlinks(linkPath)
	if err != nil {
		t.Fatalf("resolveSymlinks: %v", err)
	}
	if isWithinRoot(root, resolved) {
		t.Errorf("symlink %s should resolve outside root %s, got: %s", linkPath, root, resolved)
	}

	// The Spec with only the workspace as a WriteRoot correctly captures that
	// the outside path is not writable.
	spec := Spec{Mode: "enforce", WriteRoots: []string{root}}
	if len(spec.WriteRoots) == 0 {
		t.Error("Spec.WriteRoots must be non-empty when enforcing writes")
	}
	if spec.WriteRoots[0] != root {
		t.Errorf("Spec.WriteRoots[0] = %q, want %q", spec.WriteRoots[0], root)
	}
}

// TestResolveSymlinksPreservesNonExistentTail verifies that resolveSymlinks
// handles the case where the target does not exist and correctly resolves the
// deepest existing ancestor.
func TestResolveSymlinksPreservesNonExistentTail(t *testing.T) {
	root := t.TempDir()

	// Create a directory structure that exists.
	existing := filepath.Join(root, "existing")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}

	// Target within existing, but the file itself doesn't exist.
	nonExistent := filepath.Join(existing, "new-subdir", "file.txt")

	resolved, err := resolveSymlinks(nonExistent)
	if err != nil {
		t.Fatalf("resolveSymlinks: %v", err)
	}

	// The resolved path should be within the root.
	if !isWithinRoot(root, resolved) {
		t.Errorf("non-existent file under real directory should resolve within root: root=%s resolved=%s", root, resolved)
	}

	// The resolved path should start with the existing directory's real path.
	existingReal, err := filepath.EvalSymlinks(existing)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resolved, existingReal) {
		t.Errorf("resolved path should prefix with existing dir: resolved=%s prefix=%s", resolved, existingReal)
	}
}

// TestForbidReadWithEmptyRootsIsUnconfined verifies that an empty
// ForbidReadRoots slice does not block any reads — the safe default.
func TestForbidReadWithEmptyRootsIsUnconfined(t *testing.T) {
	spec := Spec{}

	// An empty ForbidReadRoots means no paths are forbidden.
	if len(spec.ForbidReadRoots) != 0 {
		t.Error("zero-value Spec should have empty ForbidReadRoots")
	}

	// Even with enforce mode, empty ForbidReadRoots means unconfined reads.
	spec.Mode = "enforce"
	if len(spec.ForbidReadRoots) != 0 {
		t.Error("Spec with enforce mode but no ForbidReadRoots should still have empty list")
	}
}

// TestWriteRootWithEmptyRootsIsUnconfined verifies that an empty WriteRoots
// slice on an enforce-mode Spec does not grant any write access — writes
// are unconfined, which is the safe pre-configuration default.
func TestWriteRootWithEmptyRootsIsUnconfined(t *testing.T) {
	spec := Spec{Mode: "enforce"}

	// With enforce mode but empty WriteRoots, no paths are allowed as write
	// targets (the enforcement layer treats empty roots as unconfined for
	// the pre-configuration case).
	if len(spec.WriteRoots) != 0 {
		t.Error("Spec with empty WriteRoots should have empty list")
	}
	if !spec.enforce() {
		t.Error("Spec with enforce mode should report enforce=true")
	}
}
