//go:build !windows

package main

func setUTF8Console() {}

// readInterrupted: a Unix read returns EINTR-and-retry inside Go's runtime,
// so a read that errors here is the terminal going away.
func readInterrupted(error) bool { return false }
