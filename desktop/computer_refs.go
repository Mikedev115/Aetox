package main

// Refs, and the one design decision in here that is not copied from the browser.
//
// browser_read tags elements by writing `data-aetox-ref` into the page, so the
// map IS the DOM and Go stores only bookkeeping (browser.go:1175). There is no
// equivalent here — a foreign window has nowhere to write an attribute — so the
// table lives in Go.
//
// What it stores is a **runtime id, not a COM pointer.** The plan for this work
// said keep both and fall back to the id when the pointer dies; building it
// showed that keeping the pointer buys nothing and costs a whole class of bug.
// A UIA element pointer is apartment-bound, has a lifetime nobody on this side
// controls, and is exactly the kind of thing that is valid right up until the
// window repaints. A runtime id is a small array of ints that identifies the
// element inside its provider, and re-finding by it is one tree walk — the same
// walk `read` already did, which took a fraction of a second on a real desktop.
// So: no pointer outlives the call that made it, and "the pointer died" is not
// a state this code can be in.
//
// Everything else is browser_read's rules, kept deliberately identical so a
// model that has learned one has learned the other:
//
//   - every read renumbers from 1
//   - a filtered read numbers only what it kept
//   - a ref belongs to the window it was read from, and to the read that made it
//   - a miss is diagnosed, never blamed on "the window changed"

import (
	"fmt"
	"strings"
	"sync"
)

// reachReadCap bounds one read. Same shape of number as browser_read's 150, set
// lower because a desktop tree is flatter and noisier than a page: a file
// dialog exposes every item in the folder, and 400 list rows is a context
// window spent on a directory listing.
const reachReadCap = 120

// reachNode is one row of a read.
type reachNode struct {
	Ref  int
	Name string
	// Role is the control type in the USER's language, because it is written
	// for a model that is answering a user: "ปุ่ม" on a Thai Windows and
	// "button" on an English one, from the same control.
	//
	// Which is exactly why nothing may branch on it. Kind below is the raw
	// UIA control-type id, and it is the same integer on every Windows in
	// every language. A test looking for the editable field by searching Role
	// for "edit" found nothing on a Thai machine against a window that plainly
	// had one, which is how this field came to exist.
	Kind      int32
	Role      string
	Value     string
	Enabled   bool
	Password  bool
	RuntimeID []int32
}

// reachRefs is one chat's last read: which window, with what filter, and what
// each number meant.
type reachRefs struct {
	mu     sync.Mutex
	hwnd   uintptr
	title  string
	filter string
	read   bool
	total  int
	nodes  []reachNode
}

func (r *reachRefs) remember(hwnd uintptr, title, filter string, total int, nodes []reachNode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hwnd, r.title, r.filter, r.total, r.read = hwnd, title, normalizeFilter(filter), total, true
	r.nodes = append(r.nodes[:0], nodes...)
}

// forget drops the table. Called when the window the refs belong to is gone, so
// a later miss is diagnosed as "nothing has been read" rather than as a number
// out of range in a table describing a window that closed.
func (r *reachRefs) forget() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hwnd, r.title, r.filter, r.total, r.read, r.nodes = 0, "", "", 0, false, nil
}

// lookup answers a ref, or says why it cannot.
//
// The diagnosis is the point, and it is the same five-branch answer
// browserWhyRefMissed gives, for the reason its own tests pin: telling a model
// "the window changed" when in truth it never read the window, or used a number
// from a filtered round, sends it to re-read something that was never the
// problem. Each branch names the actual cause and the actual next step.
func (r *reachRefs) lookup(hwnd uintptr, ref int) (reachNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch {
	case !r.read:
		return reachNode{}, refuse(
			"ยังไม่ได้ `read` หน้าต่างไหนเลยในแชตนี้",
			"อ่านหน้าต่างก่อน แล้วใช้เลข ref ที่ได้จากการอ่านครั้งนั้น")
	case r.hwnd != hwnd:
		return reachNode{}, refuse(
			fmt.Sprintf("เลข ref ที่ใช้เป็นของหน้าต่าง %q ไม่ใช่หน้าต่างนี้", r.title),
			"ref ผูกกับหน้าต่างที่อ่าน — `read` หน้าต่างนี้ก่อน")
	case len(r.nodes) == 0 && r.filter != "":
		return reachNode{}, refuse(
			fmt.Sprintf("การอ่านครั้งล่าสุดกรองด้วย %q แล้วไม่เจออะไรเลย", r.filter),
			"อ่านใหม่โดยไม่ใส่ filter")
	case len(r.nodes) == 0:
		return reachNode{}, refuse(
			"การอ่านครั้งล่าสุดไม่พบสิ่งที่กดหรือพิมพ์ได้ในหน้าต่างนี้",
			"ลอง `capture` เพื่อดูว่าหน้าต่างกำลังแสดงอะไรอยู่")
	case ref < 1 || ref > len(r.nodes):
		if r.filter != "" {
			return reachNode{}, refuse(
				fmt.Sprintf("การอ่านครั้งล่าสุดกรองด้วย %q และให้เลข 1-%d เท่านั้น ref %d จึงเป็นของรอบก่อน",
					r.filter, len(r.nodes), ref),
				"อ่านใหม่แล้วใช้เลขจากรอบล่าสุด — ทุกครั้งที่อ่าน เลขจะเริ่มที่ 1 ใหม่")
		}
		return reachNode{}, refuse(
			fmt.Sprintf("การอ่านครั้งล่าสุดให้เลข 1-%d ไม่มี ref %d", len(r.nodes), ref),
			"อ่านใหม่แล้วใช้เลขจากรอบล่าสุด — ทุกครั้งที่อ่าน เลขจะเริ่มที่ 1 ใหม่")
	}
	return r.nodes[ref-1], nil
}

// window reports which window the current refs belong to, for the caller that
// wants to act without naming one again.
func (r *reachRefs) window() (uintptr, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hwnd, r.title
}

func normalizeFilter(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func containsFold(haystack, needleLower string) bool {
	if needleLower == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needleLower)
}

// renderRead is what the model actually sees. One line per ref, in the order
// they were numbered, with the count and the ceiling stated when the ceiling
// was hit — a truncated list that does not say it is truncated is a list that
// gets treated as complete.
func renderRead(target reachTarget, nodes []reachNode, total int, filter string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "หน้าต่าง: %s\n", target.Label())
	if f := normalizeFilter(filter); f != "" {
		fmt.Fprintf(&b, "กรองด้วย: %q\n", f)
	}
	if len(nodes) == 0 {
		b.WriteString("ไม่พบสิ่งที่กดหรือพิมพ์ได้\n")
		return b.String()
	}
	for _, n := range nodes {
		fmt.Fprintf(&b, "[%d] %s", n.Ref, n.Role)
		if n.Name != "" {
			fmt.Fprintf(&b, " %q", n.Name)
		}
		if n.Password {
			// Named rather than hidden. A model that cannot see the field
			// exists will hunt for it; one that knows it is there and is
			// refused stops asking.
			b.WriteString(" [ช่องรหัสผ่าน — พิมพ์ไม่ได้]")
		} else if n.Value != "" {
			fmt.Fprintf(&b, " = %q", clip(n.Value, 200))
		}
		if !n.Enabled {
			b.WriteString(" [ปิดอยู่]")
		}
		b.WriteByte('\n')
	}
	if total > len(nodes) {
		fmt.Fprintf(&b, "\n(แสดง %d จาก %d — ใส่ filter เพื่อแคบลง)\n", len(nodes), total)
	}
	return b.String()
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
