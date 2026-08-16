//go:build bbbquiet

package main

// The quiet half of the logging switch; see log_verbose.go.
const verboseLog = false

func logInfo(string) {}
