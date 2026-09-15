//go:build windows

package main

import (
	"sync"
	"testing"
)

// The Go runtime holds 2000 callback slots per process and never frees one.
// Before enumwindows_windows.go every walk registered its own, so the 2001st
// walk threw "too many callback functions" and took the desktop down (crash
// dump 15 ก.ย. 2026 14:42). Walk past that line, and from several goroutines
// at once, since the attention watcher and a computer-use listing can overlap.
func TestEnumWindowsRegistersOneCallback(t *testing.T) {
	const walks = 2100
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < walks/4; i++ {
				seen := 0
				enumWindows(func(uintptr) bool {
					seen++
					return false // one window is enough to prove the slot was reused
				})
				if seen != 1 {
					t.Errorf("walk saw %d windows, want 1", seen)
					return
				}
			}
		}()
	}
	wg.Wait()
	if len(enumVisits) != 0 {
		t.Errorf("%d visitors left in the table after every walk returned", len(enumVisits))
	}
}

// Each walk sees its own visitor: the key travels through lParam, so two
// concurrent walks must not hand a window to the other's closure.
func TestEnumWindowsKeepsVisitorsApart(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				got := -1
				enumWindows(func(uintptr) bool {
					got = id
					return false
				})
				if got != id {
					t.Errorf("walk %d served visitor %d", id, got)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}
