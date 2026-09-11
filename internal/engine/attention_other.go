//go:build !windows

package engine

// FlashOwnWindow has nothing to flash here yet: the taskbar signal is a Win32
// call (attention_windows.go), and the other platforms get it with their
// browser hosts (ARCHITECTURE.md §48). The switch, the chime and the marks in
// the window all work without it.
func FlashOwnWindow() {}
