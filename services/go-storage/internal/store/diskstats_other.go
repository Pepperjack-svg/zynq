//go:build !linux

package store

// diskStats is not implemented on non-Linux platforms.
// Returns (0, 0) — callers must treat this as "stats unavailable",
// not "disk full".
func diskStats(_ string) (avail, total uint64) { return 0, 0 }
