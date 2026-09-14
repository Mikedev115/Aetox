//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows"
)

func setUTF8Console() {
	const utf8CP = 65001
	_ = windows.SetConsoleOutputCP(utf8CP)
	_ = windows.SetConsoleCP(utf8CP)
}

// readInterrupted reports a console read that Ctrl-C cut short: Windows
// answers the pending ReadFile with ERROR_OPERATION_ABORTED when the
// console generates the event, and that is not the terminal closing.
func readInterrupted(err error) bool {
	return errors.Is(err, windows.ERROR_OPERATION_ABORTED)
}
