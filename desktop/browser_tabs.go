package main

// The agent's tabs, plural.
//
// It had exactly one from 2026-08-10 until now, and that rule was really two
// rules wearing one coat:
//
//   - **Ownership.** The agent works tabs it opened; the user's are the user's.
//     This is the one that matters, it is what §127 spent a week getting right,
//     and it is untouched here — the `web-agent-` prefix separates the two at
//     any number, and nothing below can reach a tab the agent did not open.
//   - **Count.** There is one of them. This came from fixing "เปิดใหม่ ๆ รัว ๆ"
//     (tab after tab, each stranded), and it fixed that by removing the ability
//     rather than by managing it.
//
// The agent found the seam itself while being tested, 17 ส.ค.: *"ยังไม่พบความ
// สามารถสร้างแท็บใหม่หรือควบคุมหลายแท็บพร้อมกัน"*. Owner: *"ควรทำเหมือน skill ที่
// ห่อแท็ปแบบนั้น มันจะได้ดูได้"* — and the "ดูได้" half already exists, because a
// tab the agent opens is a real tab in the strip, flagged `mine`, that the user
// can see and close.
//
// What the count rule was protecting is still protected, differently: reuse
// stays the default, so a plain `open` still replaces the current page and a
// session that never asks for a second tab behaves exactly as it did.
//
// ## The hazard, named
//
// refs live in the page, as `data-aetox-ref` attributes textScript stamps on
// during a read. They are therefore per-tab and always were — what changes is
// that the agent can now be on a different tab than the one it read. Selecting
// is a page change like any other, so the answer says so in the words the tool
// description already uses: read, act, read again.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// owner: see browserSkill.owner — the session a select's desk event names.
type browserTabsSkill struct {
	app   *App
	owner string
}

func (s *browserTabsSkill) run(ctx context.Context, action, id string) (skill.Output, error) {
	out := skill.Output{Name: "browser_tabs", Command: strings.TrimSpace("browser tabs " + action + " " + id)}
	a := s.app

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", "list":
		out.Content = a.agentTabList()
		out.Success = true
	case "select":
		if err := a.selectAgentTab(id, s.owner); err != nil {
			out.Content, out.Stderr = err.Error(), err.Error()
			return out, err
		}
		out.Success = true
		out.Content = fmt.Sprintf("ทำงานกับแท็บ %s แล้ว%s refs จากการ read ก่อนหน้าใช้ไม่ได้กับหน้านี้ อ่านใหม่ก่อนคลิก",
			id, a.browserWhere(AgentTabID(id)))
		out.Content += a.selectRaiseNote(ctx, id)
	case "close":
		// No id means this one. The model has just been working a page and
		// wants it shut; making it call `list` first to learn an id the tool
		// already knows is friction that produces a refusal instead of a close
		// — which is exactly what happened the first time the guidance told a
		// model to tidy up after itself (owner's run, 24 ส.ค.).
		//
		// `select` keeps needing an id, and that is not an inconsistency: "go to
		// the tab I am already on" is not a thing anybody asks for.
		if strings.TrimSpace(id) == "" {
			current, curErr := a.agentTab()
			if curErr != nil {
				err := fmt.Errorf("ไม่มีแท็บที่เปิดอยู่ให้ปิด")
				out.Content, out.Stderr = err.Error(), err.Error()
				return out, err
			}
			id = string(current)
			// Named in the row too, or the timeline would show a close with no
			// subject for the one case where the tool worked out the subject.
			out.Command = "browser tabs close " + id
		}
		if err := a.closeAgentTab(id); err != nil {
			out.Content, out.Stderr = err.Error(), err.Error()
			return out, err
		}
		out.Success = true
		out.Content = "ปิดแท็บ " + id + " แล้ว\n" + a.agentTabList()
	default:
		err := fmt.Errorf("browser tabs %q is not one of list, select, close", action)
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	out.RawOutput = out.Content
	return out, nil
}

// agentTabs names every tab the agent owns, oldest first, and never one of the
// user's — the prefix check happens once, at open, and this reads the list that
// produced.
func (a *App) agentTabs() []string {
	h := a.browsers
	if h == nil {
		return nil
	}
	h.mu.Lock()
	ids := append([]string(nil), h.agentOrder...)
	h.mu.Unlock()
	// A tab the user closed is gone from the map before the list is pruned in
	// some orderings, and a list is read by a model that will try to select
	// what it names.
	live := ids[:0]
	for _, id := range ids {
		if h.live(id) {
			live = append(live, id)
		}
	}
	return live
}

// agentTabOn finds the agent's own tab already showing a page, if it has one.
//
// Compared case-insensitively because a file URL is the same file whichever way
// Windows spelled the drive letter, and the newest match wins: agentTabs is
// oldest first, so walking backwards lands on the tab most recently opened on
// that page, which is the one a caller asking for it again means.
func (a *App) agentTabOn(pageURL string) (AgentTabID, bool) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" || a.browsers == nil {
		return "", false
	}
	ids := a.agentTabs()
	for i := len(ids) - 1; i >= 0; i-- {
		t := a.browsers.tab(ids[i])
		if t == nil {
			continue
		}
		if _, url := t.meta(); strings.EqualFold(strings.TrimSpace(url), pageURL) {
			return AgentTabID(ids[i]), true
		}
	}
	return "", false
}

func (a *App) agentTabList() string {
	ids := a.agentTabs()
	if len(ids) == 0 {
		return "ยังไม่ได้เปิดหน้าไหนเลย ใช้ open"
	}
	current, _ := a.agentTab()
	var b strings.Builder
	b.WriteString("แท็บของคุณ:\n")
	for _, id := range ids {
		mark := "  "
		if AgentTabID(id) == current {
			mark = "* " // the one every other action works
		}
		ref := "(ยังไม่ได้ไปไหน)"
		if t := a.browsers.tab(id); t != nil {
			if named := browserPageRef(t.meta()); named != "" {
				ref = named
			}
			// Said because it changes what the user can see. A tab in the panel is
			// on one desk, in one chat; a detached one is a window of its own that
			// outlives the chat it came from, and a raise brings a whole window
			// forward rather than switching a chip.
			if t.isDetached() {
				ref += " (หน้าต่างแยก)"
			}
		}
		fmt.Fprintf(&b, "%s%s — %s\n", mark, id, ref)
	}
	b.WriteString("* คือแท็บที่ open/read/click/type/capture ทำงานด้วยตอนนี้")
	return b.String()
}

// selectAgentTab makes one of the agent's own tabs the current one, and raises
// it so the user sees the page the agent moved to — the same promise `open`
// makes, for the same reason. owner is whose conversation moved: stamped on
// the event, so a background chat's select parks on its own desk instead of
// raising over somebody else's (§187).
func (a *App) selectAgentTab(id, owner string) error {
	if err := a.mustOwn(id); err != nil {
		return err
	}
	h := a.browsers
	h.mu.Lock()
	h.agentID = id
	h.mu.Unlock()

	var url string
	if t := h.tab(id); t != nil {
		_, url = t.meta()
	}
	if a.detachedTab(id) {
		// A tab in a window of its own is raised there. See raiseDetached.
		a.raiseDetached(id)
	} else {
		a.deskEvent(owner, "open-browser", map[string]string{"id": id, "url": url})
	}
	return nil
}

// selectRaiseNote is the other half of the sentence a select gives back, said
// only when the raise it just asked for did not land.
//
// `selectAgentTab` sends an `open-browser` desk event and returns — the event
// makes the tab's chip active, and BrowserPane turns that into a real
// BrowserSetVisible(true) only if the pane also has a box on screen with no
// room drawn over it. A user in ตั้งค่า or โปรเจกต์, or with the โต๊ะ collapsed,
// leaves the select perfectly successful and the tab exactly as hidden as it
// was, and the old answer ("ทำงานกับแท็บ ... แล้ว") reported the first half
// without the second. A model that then tried to photograph the tab was
// refused, told to select it, told it worked, refused again — five minutes of
// that on 8 ก.ย. 19:04–19:09, and both halves of the loop were sentences this
// app wrote.
//
// The wait is short and only ever paid by a tab that is already hidden: a raise
// that is going to land does so in a frame or two, so a select onto a visible
// desk costs nothing and returns without ever looking at the clock.
func (a *App) selectRaiseNote(ctx context.Context, id string) string {
	if a.browsers == nil || a.detachedTab(id) {
		return ""
	}
	tab := a.browsers.tab(id)
	if tab == nil || !tab.isHidden() {
		return ""
	}
	a.waitShown(ctx, tab, 600*time.Millisecond)
	if !tab.isHidden() {
		return ""
	}
	return "\nแต่แท็บนี้ยังไม่ถูกวาดบนจอ (ผู้ใช้กำลังดูหน้าอื่นของแอป หรือพาเนลโต๊ะถูกยุบอยู่) — read/click/type ยังใช้ได้ตามปกติ แต่ capture จะไม่มีเฟรมให้ถ่าย และ select ซ้ำก็เปลี่ยนเรื่องนี้ไม่ได้"
}

func (a *App) closeAgentTab(id string) error {
	if err := a.mustOwn(id); err != nil {
		return err
	}
	// closeTab, not BrowserClose: this IS the agent closing its own tab, and
	// telling it afterwards that the user closed the page would be a lie it
	// would then act on.
	a.closeTab(id, closedByAgent)
	return nil
}

// mustOwn is the whole of the ownership rule as the tabs action sees it: an id
// the agent did not open is not refused for being unknown, it is refused for
// being somebody else's. Saying which is the difference between a model that
// tries a different id and one that tries harder.
func (a *App) mustOwn(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("which tab? use list first")
	}
	for _, own := range a.agentTabs() {
		if own == id {
			return nil
		}
	}
	if isAgentTabID(id) {
		return fmt.Errorf("แท็บ %s ปิดไปแล้ว ใช้ list เพื่อดูว่าเหลืออะไร", id)
	}
	return fmt.Errorf("แท็บ %s เป็นของผู้ใช้ ไม่ใช่ของคุณ คุณทำงานได้เฉพาะแท็บที่คุณเปิดเอง (list)", id)
}
