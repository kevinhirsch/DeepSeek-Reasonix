//go:build !windows

package disk

import (
	"syscall"
)

// getDiskUsage returns the percentage of used disk space on the filesystem
// containing the current working directory. It uses syscall.Statfs on Linux,
// macOS, and other Unix systems.
//
// Returns 0.0 when the statfs call fails (so the monitor never fires on errors).
func getDiskUsage() float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		return 0.0
	}

	// Total blocks and free blocks available to unprivileged users.
	totalBlocks := stat.Blocks
	freeBlocks := stat.Bavail
	if stat.Bfree < stat.Bavail {
		freeBlocks = stat.Bfree
	}

	if totalBlocks == 0 {
		return 0.0
	}

	usedBlocks := totalBlocks - freeBlocks
	pct := float64(usedBlocks) / float64(totalBlocks) * 100.0

	if pct < 0 {
		return 0.0
	}
	if pct > 100.0 {
		return 100.0
	}
	return pct
}
