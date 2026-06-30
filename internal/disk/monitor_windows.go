//go:build windows

package disk

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceW = kernel32.NewProc("GetDiskFreeSpaceW")
)

// getDiskUsage returns the percentage of used disk space on the filesystem
// containing the current working directory. On Windows it uses the
// GetDiskFreeSpaceW API (which works without admin privileges, unlike the newer
// GetDiskFreeSpaceExW that requires certain mount-point access).
//
// Returns 0.0 when the call fails (so the monitor never fires on errors).
func getDiskUsage() float64 {
	// Get the root path of the current drive.
	path, err := syscall.UTF16PtrFromString(".")
	if err != nil {
		return 0.0
	}

	var sectorsPerCluster, bytesPerSector, freeClusters, totalClusters uint32

	ret, _, _ := procGetDiskFreeSpaceW.Call(
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&sectorsPerCluster)),
		uintptr(unsafe.Pointer(&bytesPerSector)),
		uintptr(unsafe.Pointer(&freeClusters)),
		uintptr(unsafe.Pointer(&totalClusters)),
	)
	if ret == 0 {
		return 0.0
	}

	totalBytes := uint64(totalClusters) * uint64(sectorsPerCluster) * uint64(bytesPerSector)
	freeBytes := uint64(freeClusters) * uint64(sectorsPerCluster) * uint64(bytesPerSector)

	if totalBytes == 0 {
		return 0.0
	}

	usedBytes := totalBytes - freeBytes
	pct := float64(usedBytes) / float64(totalBytes) * 100.0

	if pct < 0 {
		return 0.0
	}
	if pct > 100.0 {
		return 100.0
	}
	return pct
}
