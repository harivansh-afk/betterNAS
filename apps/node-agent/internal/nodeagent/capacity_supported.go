//go:build android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package nodeagent

import (
	"math"
	"syscall"
)

func detectCapacityBytes(path string) *int64 {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return nil
	}

	capacity := uint64(stats.Blocks) * uint64(stats.Bsize)
	if capacity > math.MaxInt64 {
		return nil
	}

	value := int64(capacity)
	return &value
}
