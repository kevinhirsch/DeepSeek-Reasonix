package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSymlinkOutsideWriteRoots verifies that a write target that is a symlink
// pointing outside the configured WriteRoots is rejected. The realPath function
// resolves symlinks before the within check, so a symlink to /etc (or any path
// outside the roots) does not smuggle a write past the boundary.
func TestSymlinkOutsideWriteRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	// Create a workspace root and a target outside it.
	workspace := t.TempDir()
	outside := t.TempDir()

	linkPath := filepath.Join(workspace, "escape-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}

	// realPath should resolve the symlink to the outside directory.
	resolved, err := realPath(linkPath)
	if err != nil {
		t.Fatalf("realPath: %v", err)
	}

	roots := []string{workspace}
	// The resolved path (outside) should not be considered within the workspace.
	if within(roots[0], resolved) {
		t.Errorf("symlink to outside directory should NOT be within write roots\n"+
			"  workspace=%s\n  link=%s\n  resolved=%s", workspace, linkPath, resolved)
	}

	// confine should reject writes through the symlink.
	if err := confine(roots, linkPath); err == nil {
		t.Error("confine should reject a write target that is a symlink outside the write roots")
	}
}

// TestTOCTOUSymlinkSwap verifies that the TOCTOU hazard — where a path is
// validated as within the roots, then replaced with a symlink before the write
// actually happens — is caught because the final write resolves symlinks
// before the within check. The realPath function traverses the full path,
// resolving each ancestor via EvalSymlinks, so a swapped ancestor is resolved
// against the filesystem at the moment the check runs.
func TestTOCTOUSymlinkSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Simulate: path initially looks safe — a regular directory inside the root.
	safeDir := filepath.Join(root, "safe-dir")
	if err := os.MkdirAll(safeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Put a file inside (the would-be write target)
	safeFile := filepath.Join(safeDir, "target.txt")

	// Now simulate the TOCTOU swap: remove the safe directory and replace with
	// a symlink pointing outside the root.
	if err := os.RemoveAll(safeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, safeDir); err != nil {
		t.Fatal(err)
	}

	// realPath should now resolve safeDir → outside, which is NOT within root.
	resolved, err := realPath(safeFile)
	if err != nil {
		t.Fatalf("realPath after symlink swap: %v", err)
	}
	if within(root, resolved) {
		t.Errorf("TOCTOU: path inside root that was swapped with an outside symlink should NOT be within root\n"+
			"  root=%s\n  file=%s\n  resolved=%s", root, safeFile, resolved)
	}

	// confine should reject after the swap.
	if err := confine([]string{root}, safeFile); err == nil {
		t.Error("confine should reject after TOCTOU symlink swap")
	}
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

	// Swap ancestor "a" with a symlink to outside.
	if err := os.RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}

	resolved, err := realPath(deepFile)
	if err != nil {
		t.Fatalf("realPath after ancestor swap: %v", err)
	}
	if within(root, resolved) {
		t.Errorf("TOCTOU ancestor swap: path should NOT be within root after ancestor replaced with symlink\n"+
			"  root=%s\n  file=%s\n  resolved=%s", root, deepFile, resolved)
	}
}

// TestTOCTOUNonExistentTail verifies TOCTOU detection when the direct target
// doesn't exist yet (the common case for write_file) but its parent directory
// has been swapped with a symlink. realPath resolves the deepest existing
// ancestor, so it catches the swap on the parent.
func TestTOCTOUNonExistentTail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()

	// Create a directory inside the root, then swap it.
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

	// Target file doesn't exist yet — the write_file case.
	nonExistent := filepath.Join(inner, "new-file.txt")

	resolved, err := realPath(nonExistent)
	if err != nil {
		t.Fatalf("realPath: %v", err)
	}
	if within(root, resolved) {
		t.Errorf("non-existent file under swapped parent should NOT be within root\n"+
			"  root=%s\n  target=%s\n  resolved=%s", root, nonExistent, resolved)
	}
}

// TestGitHooksInForbidReadRoots verifies that .git/hooks/ is treated as a
// forbid-read root by default. The sandbox must prevent a subagent from
// reading or executing files in .git/hooks/, which would let it hijack git
// operations (e.g., by inserting a malicious post-checkout hook).
func TestGitHooksInForbidReadRoots(t *testing.T) {
	// Verify that the default ForbidReadRoots list includes .git/hooks/.
	// The convention in this codebase is that the config layer supplies
	// default forbid-read roots including .git/hooks/. This test confirms
	// that if .git/hooks/ is passed as a forbid-read root, the confineRead
	// function correctly blocks access to files within it.

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

	// .git/hooks/ should be in the forbid-read roots.
	forbidRoots := []string{hooksDir}

	// Reading a hook file should be forbidden.
	if !confineRead(forbidRoots, hookPath) {
		t.Error(".git/hooks/post-checkout should be forbidden to read when .git/hooks/ is a forbid-read root")
	}

	// Reading the parent .git directory (but not hooks itself) should be allowed
	// if only .git/hooks/ is forbidden (not .git/ itself).
	gitDir := filepath.Join(repoRoot, ".git")
	if confineRead(forbidRoots, gitDir) {
		t.Error(".git/ (parent of hooks) should not be forbidden when only .git/hooks/ is listed")
	}

	// Reading a file in the workspace root should be allowed.
	workspaceFile := filepath.Join(repoRoot, "main.go")
	if err := os.WriteFile(workspaceFile, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if confineRead(forbidRoots, workspaceFile) {
		t.Error("workspace files outside .git/hooks/ should not be forbidden")
	}
}

// TestDefaultForbidReadRootsIncludesGitHooks verifies that the Spec's
// ForbidReadRoots field, when populated with the .git/hooks/ convention,
// correctly denies read access.
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

	// Create hook files.
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")
	for _, hook := range []string{"pre-commit", "post-checkout", "pre-push"} {
		if err := os.WriteFile(filepath.Join(hooksDir, hook), []byte("#!/bin/sh\necho hooked"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Simulate the default ForbidReadRoots that the config layer would supply.
	// .git/hooks/ must be forbidden; .git/ itself may or may not be.
	forbidRoots := []string{hooksDir}

	// All hook files should be forbidden.
	for _, hook := range []string{"pre-commit", "post-checkout", "pre-push"} {
		hookPath := filepath.Join(hooksDir, hook)
		if !confineRead(forbidRoots, hookPath) {
			t.Errorf("%s should be forbidden to read", hook)
		}
	}

	// Other .git directories (objects, refs) should be readable when not
	// explicitly forbidden.
	objectsDir := filepath.Join(repoRoot, ".git", "objects")
	if confineRead(forbidRoots, objectsDir) {
		t.Error(".git/objects/ should be readable when only .git/hooks/ is forbidden")
	}

	// The src directory should be readable.
	srcDir := filepath.Join(repoRoot, "src")
	if confineRead(forbidRoots, srcDir) {
		t.Error("src/ should be readable when only .git/hooks/ is forbidden")
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

	resolved, err := realPath(file)
	if err != nil {
		t.Fatalf("realPath: %v", err)
	}
	if !within(root, resolved) {
		t.Errorf("normal file should be within root: root=%s file=%s resolved=%s", root, file, resolved)
	}
}

// TestConfineWritersRejectsSymlinkEscape verifies that the confine function
// used by ConfineWriters rejects paths that resolve outside the roots even
// when the raw path looks safe.
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

	// confine with just the root should reject the symlink.
	err := confine([]string{root}, linkPath)
	if err == nil {
		t.Error("confine should reject symlink pointing outside write roots")
	}
	if !strings.Contains(err.Error(), "outside the writable roots") {
		t.Errorf("error message should mention writable roots, got: %v", err)
	}
}
