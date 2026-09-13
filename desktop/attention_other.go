//go:build !windows

package main

// The taskbar flash and the notification are Win32 calls
// (attention_windows.go); the other platforms get them with their browser
// hosts (ARCHITECTURE.md §48). With no way to ask the desktop, the window is
// never "away" here — the frontend's own focus guess carries the chime, and
// the switches and the marks in the window all work without the rest.
func windowAway() bool { return false }

func flashOwnWindow() {}

func showAttentionToast(kind, title, text string) {}

func windowsNotificationsOff() bool { return false }
