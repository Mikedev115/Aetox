//go:build windows

package main

import (
	"sync"
	"syscall"
)

// enumWindows walks the top-level windows and hands each to visit until visit
// answers false. The return is EnumWindows's own, for the caller that wants
// it: r is zero when the call failed AND when a visitor stopped the walk, so
// a caller that stops early reads err only beside its own evidence.
//
// One callback for the life of the process. syscall.NewCallback does not
// register a closure with Windows; it burns a slot in the Go runtime's
// callback table, and that table holds 2000 for the whole process and never
// gives one back. findOwnMainWindow used to register a fresh closure per call,
// and the attention watcher calls it twice a second for as long as a
// notification hangs on the tray with the user elsewhere — sixteen minutes of
// being away and the runtime threw "too many callback functions" (crash dump
// 15 ก.ย. 2026 14:42, DECISIONS.md §295). So the callback is registered once
// and told which visitor it is serving through lParam: a key into a table,
// not a Go pointer cast to an integer, so a visitor's closure lives wherever
// the GC likes and two callers on two threads walk at once without seeing
// each other's state.
func enumWindows(visit func(hwnd uintptr) (keepGoing bool)) (r uintptr, err error) {
	enumWindowsOnce.Do(func() {
		enumWindowsCB = syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
			enumVisitsMu.Lock()
			visit := enumVisits[lparam]
			enumVisitsMu.Unlock()
			if visit == nil || !visit(hwnd) {
				return 0 // stop
			}
			return 1 // keep enumerating
		})
	})
	enumVisitsMu.Lock()
	enumVisitsNext++
	key := enumVisitsNext
	enumVisits[key] = visit
	enumVisitsMu.Unlock()
	defer func() {
		enumVisitsMu.Lock()
		delete(enumVisits, key)
		enumVisitsMu.Unlock()
	}()
	r, _, err = procEnumWindows.Call(enumWindowsCB, key)
	return r, err
}

var (
	enumWindowsOnce sync.Once
	enumWindowsCB   uintptr

	enumVisitsMu   sync.Mutex
	enumVisitsNext uintptr
	enumVisits     = map[uintptr]func(uintptr) bool{}
)
