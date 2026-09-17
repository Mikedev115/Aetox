//go:build !windows

package main

// The acting half everywhere that is not Windows. Same rule as
// computer_reach_other.go: a clear no rather than a silent nothing, because a
// no-op that returns success tells a model it pressed something.

func reachFocusWindow(uintptr) error { return errReachUnsupported }

func reachCloseWindow(uintptr) error { return errReachUnsupported }

func reachClick(uintptr, []int32) error { return errReachUnsupported }

func reachClickAt(uintptr, int, int, int, int) error { return errReachUnsupported }

func reachScrollAt(uintptr, int, int, int, int, int) error { return errReachUnsupported }

func reachDragAt(uintptr, int, int, int, int, int, int, int) error { return errReachUnsupported }

func reachType(uintptr, []int32, string) error { return errReachUnsupported }

func reachReadBack(uintptr, []int32) (string, error) { return "", errReachUnsupported }

func reachKeys(string) error { return errReachUnsupported }

func reachCursor() (int, int, error) { return 0, 0, errReachUnsupported }

func settle() {}
