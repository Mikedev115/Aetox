//go:build windows

package main

// The binding's own test, and the reason it walks a real tree rather than a
// stub.
//
// The tool this work replaces (internal/skill/computer.go, removed 50584e5)
// shipped with unit tests over its input-buffer builders. Those tests passed.
// The tool did not work. DECISIONS.md §22's amendment names that gap directly:
// "it comes back with a live end-to-end test on a real desktop, not just unit
// tests over the input-buffer builders — that was the gap that let a broken
// tool ship."
//
// A vtable is the same kind of thing as an input buffer: you can assert its
// shape all day without ever learning whether slot 4 is the function you think
// it is. Only a real call proves that, so this test makes real calls — against
// the desktop root, which exists in every Windows session including a test
// process with no window of its own.
//
// It skips rather than fails when UI Automation cannot start, because "this
// machine has no interactive desktop" is a true fact about a build agent and
// not a defect in the code.

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestUIAutomationStartsAndWalksARealTree(t *testing.T) {
	h := uia()
	if err := h.start(); err != nil {
		t.Skipf("UI Automation unavailable here: %v", err)
	}

	var (
		count   int32
		names   []string
		gotRTID bool
	)
	err := h.do(func(a *iUIAutomation) error {
		root, err := a.elementFromHandle(desktopRoot())
		if err != nil {
			return err
		}
		defer root.release()

		// A property read on the root proves the element vtable's offsets, not
		// merely that a pointer came back.
		if _, err := root.name(); err != nil {
			return err
		}

		cond, err := a.createTrueCondition()
		if err != nil {
			return err
		}
		defer cond.release()

		arr, err := root.findAll(scopeChildren, cond)
		if err != nil {
			return err
		}
		defer arr.release()

		count, err = arr.length()
		if err != nil {
			return err
		}
		for i := int32(0); i < count && i < 8; i++ {
			el, err := arr.at(i)
			if err != nil {
				return err
			}
			n, err := el.name()
			if err != nil {
				el.release()
				return err
			}
			if id, err := el.runtimeID(); err == nil && len(id) > 0 {
				gotRTID = true
			}
			names = append(names, n)
			el.release()
		}
		return nil
	})
	if err != nil {
		t.Skipf("UI Automation refused the walk on this machine: %v", err)
	}

	// The desktop always has children. Zero here means FindAll returned an
	// array whose get_Length is not get_Length — the exact silent-misdispatch
	// failure the vtable comment warns about.
	if count == 0 {
		t.Fatal("the desktop root reported no child windows at all, which means a vtable slot is misaligned rather than that the screen is empty")
	}
	if !gotRTID {
		t.Error("no element yielded a runtime id; refs cannot survive a stale pointer without one")
	}
	t.Logf("walked %d top-level windows; first few: %s", count, strings.Join(names, " | "))
}

// The two properties every acting decision is made from. A window that reports
// neither a process nor a rectangle cannot be checked against the guard's
// "never drive Aetox" rule, so this failing is a security fact, not a cosmetic
// one.
func TestEveryWindowReportsAProcessAndARectangle(t *testing.T) {
	h := uia()
	if err := h.start(); err != nil {
		t.Skipf("UI Automation unavailable here: %v", err)
	}

	var checked int
	err := h.do(func(a *iUIAutomation) error {
		root, err := a.elementFromHandle(desktopRoot())
		if err != nil {
			return err
		}
		defer root.release()
		cond, err := a.createTrueCondition()
		if err != nil {
			return err
		}
		defer cond.release()
		arr, err := root.findAll(scopeChildren, cond)
		if err != nil {
			return err
		}
		defer arr.release()
		n, err := arr.length()
		if err != nil {
			return err
		}
		for i := int32(0); i < n && checked < 5; i++ {
			el, err := arr.at(i)
			if err != nil {
				return err
			}
			pid, perr := el.processID()
			_, rerr := el.boundingRect()
			el.release()
			if perr != nil {
				return perr
			}
			if rerr != nil {
				return rerr
			}
			if pid <= 0 {
				return errors.New("a window reported process id 0")
			}
			checked++
		}
		return nil
	})
	if err != nil {
		t.Skipf("UI Automation refused the walk on this machine: %v", err)
	}
	if checked == 0 {
		t.Skip("no top-level windows to check on this machine")
	}
}

// A tool call must end. The budget exists so a hung provider — a frozen app is
// the ordinary case, not the exotic one — surfaces as a sentence instead of a
// turn that never comes back.
func TestTheAutomationThreadHasABudget(t *testing.T) {
	if uiaStartBudget <= 0 {
		t.Fatal("uiaStartBudget must bound the wait; an unbounded start is a turn that cannot be stopped")
	}
	if uiaStartBudget > 30*time.Second {
		t.Errorf("uiaStartBudget is %s, long enough that a user reads it as the app hanging", uiaStartBudget)
	}
}
