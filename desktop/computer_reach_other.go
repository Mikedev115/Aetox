//go:build !windows

package main

// The reach on every platform that is not Windows: a clear no, never a silent
// nothing.
//
// PLATFORM-SUPPORT.md's phases put macOS and Linux after this, and the
// direction doc says why the row cannot simply be ported — UI Automation is a
// Windows API, and the equivalent on each other platform is a different one
// with a different permission model (macOS wants Accessibility and Screen
// Recording granted in System Settings before anything works at all).
//
// The old tool's non-Windows half did exactly this and it was the right call
// then too: computer_other.go, six stubs, one error, "รองรับเฉพาะ Windows
// ตอนนี้". A no-op that returns an empty list instead would tell a model the
// machine has no windows, which is a lie it would then act on.

var errReachUnsupported = refuse(
	"การใช้คอมพิวเตอร์รองรับเฉพาะ Windows ตอนนี้",
	"บนเครื่องนี้ใช้ `shell` สำหรับงานเครื่อง และ `browser` สำหรับงานเว็บแทน")

func reachListWindows() ([]reachTarget, error) { return nil, errReachUnsupported }

func reachFindWindow(uintptr) (reachTarget, error) { return reachTarget{}, errReachUnsupported }

func reachReadWindow(uintptr, string, int) ([]reachNode, int, error) {
	return nil, 0, errReachUnsupported
}

func reachCaptureWindow(uintptr) ([]byte, error) { return nil, errReachUnsupported }
