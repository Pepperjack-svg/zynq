//go:build linux

package store

import "syscall"

// diskStats returns the available and total bytes on the filesystem that
// contains path. Uses Bavail (blocks available to unprivileged processes)
// rather than Bfree (root-reserved blocks included) so we report the space
// that the storage service — running as non-root — can actually use.
func diskStats(path string) (avail, total uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	bsize := uint64(st.Bsize)
	return st.Bavail * bsize, st.Blocks * bsize
}
