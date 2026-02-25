package main

import "os"

// shutdownSignals lists the OS signals that trigger graceful shutdown.
// os.Interrupt (SIGINT / Ctrl-C) is the portable baseline available on every OS.
// SIGTERM is appended by signals_unix.go on non-Windows platforms.
var shutdownSignals = []os.Signal{os.Interrupt}
