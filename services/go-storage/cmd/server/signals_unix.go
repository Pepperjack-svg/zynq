//go:build !windows

package main

import "syscall"

func init() {
	// SIGTERM is the standard graceful-shutdown signal on Linux/macOS.
	// It is not wired to the Windows job-object model, so we only register it here.
	shutdownSignals = append(shutdownSignals, syscall.SIGTERM)
}
