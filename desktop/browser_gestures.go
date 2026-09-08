package main

// The rest of the mouse and the rest of the keyboard.
//
// Owner, 6 ก.ย., after the typing fix (§226) had gone live on Docs, Sheets,
// Slides and NotebookLM: *"การลาก คลุม หรืออะไรพวกนี้อีกล่ะ … ถ้าจะทำให้มันควบคุมได้
// เต็มเบ็ดเสร็จ จะต้องมีอะไรเพิ่มอีก"*. The model had already told him the same
// thing in its own words — *"hover / ลากเมาส์ / คลิกขวา — ไม่มีคำสั่งตรง ๆ"* — and
// tool_runs 1547 that night showed the cost of aiming by number alone: a stale
// ref sent a `type` to the Sheets logo and the page navigated away.
//
// This file is the answer, in three parts that share one door.
//
//   - **A target is a ref, a text, or a point.** `browserTarget` and
//     resolveTarget are the one way every pointer action turns "what the model
//     said" into a place on the page: a ref through pointScript, a text through
//     the filtered read `read` already has, a point through underScript. Text
//     is what makes a batch of steps possible (no ref is known before a read
//     inside the batch) and what retires the stale-ref mistake for the common
//     case — a person clicks the button that says ปิด, not element 43.
//   - **The gestures.** hover, drag, right/double/triple click, named keys and
//     chords, upload, all through the engine (browser_keys.go), all answering
//     with what they pressed and where, and what they cannot know.
//   - **The change note.** Every action that acts ends with what changed —
//     URL, title, focus, how many things there are to press — instead of
//     "ใช้ read ดูผลลัพธ์". A separate read after every action was the second
//     half of "1 call ได้แค่ 1 แอ็คชั่น", and the half that measured largest.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// browserTarget is where a pointer action is aimed: one of a ref from the last
// read, a text the element's label contains, or a viewport point in CSS px
// read off a capture.
type browserTarget struct {
	Ref   int
	Find  string
	X, Y  float64
	HasXY bool
}

func (t browserTarget) empty() bool { return t.Ref == 0 && t.Find == "" && !t.HasXY }

// String is how the target is named back in an answer before it resolved.
func (t browserTarget) String() string {
	switch {
	case t.Find != "":
		return fmt.Sprintf("element ที่มีข้อความ %q", t.Find)
	case t.Ref > 0:
		return fmt.Sprintf("ref %d", t.Ref)
	case t.HasXY:
		return point{t.X, t.Y}.String()
	}
	return "(ไม่ได้ระบุเป้า)"
}

// targetFrom reads a target out of the call's arguments under the given
// names, so `drag` can read two of them from one argument map.
func targetFrom(args map[string]any, refKey, findKey, xKey, yKey string) browserTarget {
	t := browserTarget{Ref: intArg(args[refKey]), Find: strings.TrimSpace(str(args[findKey]))}
	x, okX := skill.FloatArg(args[xKey])
	y, okY := skill.FloatArg(args[yKey])
	if okX && okY {
		t.X, t.Y, t.HasXY = x, y, true
	}
	return t
}

// maxFindListed is how many matches an ambiguous `find` lists before it stops.
const maxFindListed = 12

// resolveTarget is the one door: what the model said, to a place on the page,
// or an error that says why not.
//
// Text first, because it may turn into a ref. A filtered read tags exactly the
// elements whose text contains it and numbers them from 1, so one match IS ref
// 1 of that read, and the ref path takes it from there. Zero and several are
// both refusals — never "the first one": a person asked to click "ปิด" when
// there are three does not pick one, they ask which.
//
// A point is checked against the viewport before anything is pressed. The
// engine would happily press outside it, which is how a coordinate read off a
// full-page capture (whose y is a document offset) turns into a click on
// nothing.
func (a *App) resolveTarget(id AgentTabID, t browserTarget) (browserActResult, error) {
	ref := t.Ref
	if t.Find != "" {
		found, err := a.findByText(id, t.Find)
		if err != nil {
			return browserActResult{}, err
		}
		ref = found
	}
	if ref > 0 {
		res, answered, err := a.browserActOn(string(id), func(token string) string { return pointScript(token, ref) })
		if err != nil {
			return browserActResult{}, err
		}
		if !answered {
			return browserActResult{}, fmt.Errorf("หน้าไม่ตอบว่า ref %d อยู่ตรงไหน (หน้ากำลังโหลด หรือไม่ยอมรันสคริปต์)", ref)
		}
		if !res.Found {
			return browserActResult{}, fmt.Errorf("ไม่มี element ref %d บนหน้านี้%s", ref, a.browserWhyRefMissed(id, ref))
		}
		res.Ref = ref
		return res, nil
	}
	if t.HasXY {
		res, answered, err := a.browserActOn(string(id), func(token string) string { return underScript(token, t.X, t.Y) })
		if err != nil {
			return browserActResult{}, err
		}
		if !answered {
			return browserActResult{}, fmt.Errorf("หน้าไม่ตอบว่ามีอะไรอยู่ที่ %s", point{t.X, t.Y})
		}
		if !res.Found {
			return browserActResult{}, fmt.Errorf("จุด %s อยู่นอก viewport (%d×%d CSS px) — พิกัดต้องมาจากภาพ capture แบบไม่ full และคูณอัตราส่วนที่ capture บอก", point{t.X, t.Y}, res.VW, res.VH)
		}
		return res, nil
	}
	return browserActResult{}, errors.New("ต้องระบุเป้า: ref (จาก read), find (ข้อความบน element) หรือ x,y (พิกัดจากภาพ capture)")
}

// findWait is how long `find` gives a page to show the text before refusing.
//
// The owner's first batch on 6 ก.ย. opened NotebookLM and clicked "สร้าง
// Notebook ใหม่" in the next step, and the step refused: the page's shell had
// loaded and its buttons had not. A person looking for a button waits for it
// to appear; three seconds is longer than any page that is going to show it
// takes, and short enough that a page that never will is refused in words
// rather than in a timeout.
const findWait = 3 * time.Second

// findByText turns a text into the ref of the one element that carries it.
//
// Controls first — the filtered read `read` already has, which tags exactly
// the interactive elements whose label contains the text and numbers them
// from 1, so one match IS ref 1. Then any visible text at all, through
// findTextScript, so a word in a paragraph can be hovered, double-clicked or
// dragged over. Zero and several are both refusals — never "the first one":
// a person asked to click "ปิด" when there are three does not pick one, they
// ask which. Zero is retried until findWait runs out, because zero right
// after a navigation usually means "not yet".
func (a *App) findByText(id AgentTabID, text string) (int, error) {
	deadline := time.Now().Add(findWait)
	for {
		snap, err := a.browserSnapshot(string(id), text)
		if err != nil {
			return 0, err
		}
		if n := len(snap.Elements); n > 1 {
			var b strings.Builder
			fmt.Fprintf(&b, "มี %d element ที่มีข้อความ %q เลือกไม่ได้ว่าตัวไหน ใช้ ref จากรายการนี้แทน หรือใช้ข้อความที่ยาวขึ้น:", n, text)
			for i, el := range snap.Elements {
				if i >= maxFindListed {
					fmt.Fprintf(&b, "\n… และอีก %d", n-maxFindListed)
					break
				}
				b.WriteString("\n" + elementLine(el))
			}
			return 0, errors.New(b.String())
		} else if n == 1 {
			return 1, nil
		}
		res, answered, err := a.browserActOn(string(id), func(token string) string { return findTextScript(token, text) })
		if err != nil {
			return 0, err
		}
		if answered && res.Count == 1 {
			if tab := a.browsers.tab(string(id)); tab != nil {
				tab.noteRefs(1, text)
			}
			return 1, nil
		}
		if answered && res.Count > 1 {
			var b strings.Builder
			fmt.Fprintf(&b, "มี %d ที่บนหน้าที่มีข้อความ %q ไม่มีตัวไหนเป็นปุ่มหรือช่องกรอก และเลือกไม่ได้ว่าตัวไหน — ใช้ข้อความที่ยาวขึ้นให้เหลือหนึ่ง หรือ capture marks=true แล้วกดด้วย ref ที่เห็นในภาพ (ถ้าสิ่งที่ต้องการไม่มีเลขกำกับ แปลว่าหน้านี้วาดเอง ค่อยเล็งด้วย x,y):", res.Count, text)
			for _, m := range res.Matches {
				b.WriteString("\n- " + m)
			}
			return 0, errors.New(b.String())
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("ไม่มีข้อความ %q บนหน้านี้ (รอ %s แล้ว) — read ดูว่ามีอะไรบ้าง ใช้ข้อความอื่น หรือ capture marks=true เพื่อเห็นของที่กดได้พร้อมเลข ref", text, findWait)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// targetSaid names a resolved target the way an answer should: the element
// by tag and label when there was one, the point and what was under it when
// there was not.
func targetSaid(t browserTarget, res browserActResult) string {
	if t.HasXY && t.Ref == 0 && t.Find == "" {
		s := point{res.CX, res.CY}.String()
		if res.Under != "" {
			s += " (ใต้จุดนั้นคือ " + res.Under + ")"
		}
		return s
	}
	return browserActLabel(res.Ref, res, true)
}

// canvasNote is the sentence a painted page adds to every answer (§226).
func canvasNote(res browserActResult) string {
	if res.CanvasShare >= canvasAppShare {
		return " หน้านี้วาดเนื้อหาบน canvas: read มองไม่เห็นผล ยืนยันด้วย capture"
	}
	return ""
}

// clickSaid is the verb for a press: which button, how many times.
func clickSaid(button string, count int) string {
	switch {
	case button == "right":
		return "คลิกขวา"
	case button == "middle":
		return "คลิกกลาง"
	case count == 2:
		return "ดับเบิลคลิก"
	case count >= 3:
		return "คลิกสามครั้ง"
	}
	return "คลิก"
}

// pointerMessage is the answer of a press by the engine: what was pressed,
// where, and the two things it cannot see — a native context menu, and paint.
func pointerMessage(button string, count int, t browserTarget, res browserActResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s แล้ว", clickSaid(button, count), targetSaid(t, res))
	if !(t.HasXY && t.Ref == 0 && t.Find == "") {
		fmt.Fprintf(&b, " ที่ %s", point{res.CX, res.CY})
	}
	if button == "right" {
		b.WriteString(" — ถ้าหน้ามีเมนูของตัวเองจะเห็นใน read; เมนูของระบบไม่อยู่ใน DOM และไม่อยู่ในภาพ กด key Escape เพื่อปิด")
	}
	b.WriteString(canvasNote(res))
	return b.String()
}

// pointerFailed is the one shape every engine refusal takes: the gesture did
// not happen, and the engine's own words say why.
func pointerFailed(what string, err error) string {
	return fmt.Sprintf("%s ไม่สำเร็จ: เอนจินตอบว่า %v — ถือว่ายังไม่ได้ทำ", what, err)
}

// pointerContext is the patience a pointer gesture gets from the engine.
func pointerContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), deckEngineTimeout)
}

// aim resolves a target and sends the cursor there, waiting for it to arrive
// so the press lands when the arrow does. The wait is zero when the layer is
// off.
func (a *App) aim(id AgentTabID, t browserTarget, press bool) (browserActResult, error) {
	res, err := a.resolveTarget(id, t)
	if err != nil {
		return res, err
	}
	time.Sleep(a.markCursorMove(id, point{res.CX, res.CY}, press))
	return res, nil
}

// --- hover ------------------------------------------------------------------

type browserHoverSkill struct{ app *App }

func (s *browserHoverSkill) hover(t browserTarget) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "browser_hover", Command: "browser hover " + t.String()}
	id, err := s.app.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	res, err := s.app.aim(id, t, false)
	if err != nil {
		out.Content, out.Stderr = "hover ไม่สำเร็จ: "+err.Error(), err.Error()
		out.DurationMs = time.Since(start).Milliseconds()
		return out, err
	}
	ctx, cancel := pointerContext()
	defer cancel()
	if err := s.app.hoverByMouse(ctx, string(id), point{res.CX, res.CY}); err != nil {
		msg := pointerFailed("hover", err) + s.app.browserWhere(id)
		out.Content, out.Stderr = msg, msg
		return out, err
	}
	// A menu that opens on hover opens in its own time.
	time.Sleep(200 * time.Millisecond)
	out.Success = true
	out.Content = fmt.Sprintf("เลื่อนเมาส์ไปที่ %s แล้ว%s%s", targetSaid(t, res), canvasNote(res), s.app.browserWhere(id))
	out.RawOutput = out.Content
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// --- drag -------------------------------------------------------------------

type browserDragSkill struct{ app *App }

func (s *browserDragSkill) drag(from, to browserTarget) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "browser_drag", Command: "browser drag " + from.String() + " → " + to.String()}
	id, err := s.app.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	if to.empty() {
		err := errors.New("ต้องระบุปลายทางของการลาก: toRef, toFind หรือ toX,toY")
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	// Both ends are resolved before anything is pressed: a drag that pressed
	// and then could not find where to go would leave the button down.
	a, err := s.app.aim(id, from, false)
	if err != nil {
		out.Content, out.Stderr = "ลากไม่สำเร็จ (จุดเริ่ม): "+err.Error(), err.Error()
		return out, err
	}
	b, err := s.app.resolveTarget(id, to)
	if err != nil {
		out.Content, out.Stderr = "ลากไม่สำเร็จ (ปลายทาง): "+err.Error(), err.Error()
		return out, err
	}
	fp, tp := point{a.CX, a.CY}, point{b.CX, b.CY}
	s.app.markCursorDrag(id, fp, tp, dragDuration(fp, tp))
	ctx, cancel := pointerContext()
	defer cancel()
	if err := s.app.dragByMouse(ctx, string(id), fp, tp); err != nil {
		msg := pointerFailed("ลาก", err) + s.app.browserWhere(id)
		out.Content, out.Stderr = msg, msg
		return out, err
	}
	time.Sleep(300 * time.Millisecond)
	out.Success = true
	out.Content = fmt.Sprintf("ลากจาก %s ไป %s แล้ว (กดค้างที่ %s กวาดไปปล่อยที่ %s) — สิ่งที่ถูกกวาดผ่านจะถูกเลือกไว้ถ้าเป็นข้อความ ถ้าเป็นของที่ลากได้ก็ย้ายไปแล้ว%s%s",
		targetSaid(from, a), targetSaid(to, b), fp, tp, canvasNote(a), s.app.browserWhere(id))
	out.RawOutput = out.Content
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// --- key --------------------------------------------------------------------

type browserKeySkill struct{ app *App }

// keyActMessage is the answer of a chord: what was pressed and where the
// page's keyboard was when it landed (§226's sentence), and what it cannot
// see on a painted page.
func keyActMessage(said []string, res browserActResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "กด %s แล้ว", strings.Join(said, " "))
	if res.Focus != "" {
		fmt.Fprintf(&b, " — คีย์ไปที่ focus ของหน้า: %s", res.Focus)
	}
	b.WriteString(canvasNote(res))
	return b.String()
}

func (s *browserKeySkill) key(keys string) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "browser_key", Command: "browser key " + keys}
	// The plan is built first, so a bad chord is refused before the engine —
	// or the page — hears anything.
	if _, _, err := chordPlan(keys); err != nil {
		msg := "กดไม่ได้: " + err.Error()
		out.Content, out.Stderr = msg, msg
		return out, err
	}
	id, err := s.app.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	res, _, _ := s.app.browserActOn(string(id), focusScript)
	ctx, cancel := pointerContext()
	defer cancel()
	said, err := s.app.sendKeys(ctx, string(id), keys)
	if err != nil {
		msg := pointerFailed("กด "+keys, err) + s.app.browserWhere(id)
		out.Content, out.Stderr = msg, msg
		return out, err
	}
	time.Sleep(200 * time.Millisecond)
	out.Success = true
	out.Content = keyActMessage(said, res) + s.app.browserWhere(id)
	out.RawOutput = out.Content
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// --- upload -----------------------------------------------------------------

type browserUploadSkill struct{ app *App }

// sandboxFile resolves a path the way `open` does for a page — sandbox-
// relative, steered through the output folder the way `write` places new
// files — and refuses what every other tool refuses: a credential store, or
// anywhere outside the sandbox.
func (a *App) sandboxFile(request string) (string, error) {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		return "", errors.New("no working folder is set")
	}
	placed := skill.PlacedPath(root, a.outputSubdir, request)
	abs, err := skill.SandboxFile(root, placed)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("ไม่มีไฟล์ %s", request)
	}
	return abs, nil
}

// uploadMessage says what happened and, more to the point, what has not: the
// page has been handed the file, and nothing has been uploaded anywhere until
// the page does it.
func uploadMessage(name string, size int64, res browserActResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ใส่ไฟล์ %s (%s) ลง %s แล้ว — หน้าได้รับ change แล้ว แต่ยังไม่ได้อัปโหลดไปไหนจนกว่าหน้าจะทำเอง ใช้ wait/read ดูข้อความยืนยันของหน้า",
		name, humanBytes(size), browserActLabel(res.Ref, res, true))
	if res.Accept != "" {
		fmt.Fprintf(&b, " (input นี้รับ %s)", res.Accept)
	}
	if res.Multiple {
		b.WriteString(" (รับหลายไฟล์ ใส่ไปหนึ่งไฟล์)")
	}
	return b.String()
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func (s *browserUploadSkill) upload(t browserTarget, path string) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "browser_upload", Command: "browser upload " + path}
	if strings.TrimSpace(path) == "" {
		err := errors.New("ต้องระบุ path ของไฟล์ใน sandbox")
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	abs, err := s.app.sandboxFile(path)
	if err != nil {
		out.Content, out.Stderr = "อัปโหลดไม่ได้: "+err.Error(), err.Error()
		return out, err
	}
	info, _ := os.Stat(abs)
	id, err := s.app.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	if t.HasXY && t.Ref == 0 && t.Find == "" {
		err := errors.New("upload ต้องเล็งด้วย ref หรือ find (input type=file) ไม่ใช่พิกัด")
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	res, err := s.app.aim(id, t, false)
	if err != nil {
		out.Content, out.Stderr = "อัปโหลดไม่ได้: "+err.Error(), err.Error()
		return out, err
	}
	kind, answered, err := s.app.browserActOn(string(id), func(token string) string { return fileInputScript(token, res.Ref) })
	if err == nil && (!answered || !kind.FileInput) {
		err = fmt.Errorf("%s ไม่ใช่ input type=file — ใส่ไฟล์ได้เฉพาะช่องเลือกไฟล์ ถ้าหน้ามีปุ่ม 'อัปโหลด' ที่เปิดหน้าต่างของระบบ ให้หา input ที่ซ่อนอยู่ใน read", browserActLabel(res.Ref, res, true))
	}
	if err != nil {
		out.Content, out.Stderr = "อัปโหลดไม่ได้: "+err.Error(), err.Error()
		return out, err
	}
	kind.Ref = res.Ref
	s.app.markPageClick(id, point{res.CX, res.CY})
	ctx, cancel := pointerContext()
	defer cancel()
	if err := s.app.setFileInput(ctx, string(id), res.Ref, abs); err != nil {
		msg := pointerFailed("ใส่ไฟล์", err) + " ให้ผู้ใช้เลือกไฟล์เอง" + s.app.browserWhere(id)
		out.Content, out.Stderr = msg, msg
		return out, err
	}
	time.Sleep(300 * time.Millisecond)
	out.Success = true
	out.Content = uploadMessage(filepath.Base(abs), info.Size(), kind) + s.app.browserWhere(id)
	out.RawOutput = out.Content
	out.DurationMs = time.Since(start).Milliseconds()
	return out, nil
}

// setFileInput hands a file to a file input through the engine: the node is
// found by its ref as a remote object, and DOM.setFileInputFiles sets it the
// way the OS picker would. DOM.enable first, because the DOM agent answers
// nothing until it is.
func (a *App) setFileInput(ctx context.Context, id string, ref int, abs string) error {
	host, err := a.browserHostLazy()
	if err != nil {
		return err
	}
	if !host.live(id) {
		return fmt.Errorf("no browser tab %q", id)
	}
	expr := fmt.Sprintf(`(function(){%s return aetoxFind(%d);})()`, aetoxScanJS, ref)
	evalParams, _ := jsonObject(map[string]any{"expression": expr, "returnByValue": false})
	raw, err := callEngineOn(ctx, host, id, "Runtime.evaluate", evalParams)
	if err != nil {
		return fmt.Errorf("Runtime.evaluate: %w", err)
	}
	objectID := jsonPath(raw, "result", "objectId")
	if objectID == "" {
		return errors.New("Runtime.evaluate: the engine returned no object for the input")
	}
	// Enabling is idempotent and cheap; a refusal here is not fatal because
	// some engines answer setFileInputFiles without it.
	_, _ = callEngineOn(ctx, host, id, "DOM.enable", "{}")
	setParams, _ := jsonObject(map[string]any{"files": []string{abs}, "objectId": objectID})
	if _, err := callEngineOn(ctx, host, id, "DOM.setFileInputFiles", setParams); err != nil {
		return fmt.Errorf("DOM.setFileInputFiles: %w", err)
	}
	return nil
}

// jsonObject and jsonPath are the two shapes the engine speaks in: a map out,
// a nested string field back.
func jsonObject(m map[string]any) (string, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func jsonPath(raw string, keys ...string) string {
	var cur any
	if err := json.Unmarshal([]byte(raw), &cur); err != nil {
		return ""
	}
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[k]
	}
	s, _ := cur.(string)
	return s
}

// --- the change note --------------------------------------------------------

// stateWait is how long the page gets to say what it is after an action.
// Short, because silence here means "navigating away", and that is an answer.
const stateWait = 800 * time.Millisecond

// pageState is the page in four numbers and a name (stateScript). ok is false
// when the page did not answer in time.
func (a *App) pageState(id AgentTabID, wait time.Duration) (browserActResult, bool) {
	res, answered, err := a.browserActOnFor(string(id), stateScript, wait)
	return res, err == nil && answered
}

// settleAfterAct waits for a page that an action may have sent somewhere else.
// A page mid-navigation does not answer the state script; when that happens
// the tab's own idea of its URL is watched until it moves or time runs out,
// and the state is asked for again on the page that arrived.
func (a *App) settleAfterAct(id AgentTabID, before browserActResult) (browserActResult, bool) {
	if after, ok := a.pageState(id, stateWait); ok {
		return after, true
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		if tab := a.browsers.tab(string(id)); tab != nil {
			if _, url := tab.meta(); url != before.URL {
				break
			}
		}
	}
	return a.pageState(id, 2*time.Second)
}

// changeNoteRefs is how many of a new page's elements ride on the note. Enough
// to act on a page that just arrived without a separate read; few enough that
// a note stays a note.
const changeNoteRefs = 40

// changeNote is what an action hands back in place of "ใช้ read ดูผลลัพธ์":
// the difference between the page before and after. When the page moved, the
// first refs of the new one come along, which is the observation the model
// would otherwise spend a call on.
func (a *App) changeNote(id AgentTabID, before browserActResult) string {
	after, ok := a.settleAfterAct(id, before)
	if !ok {
		return "\n→ หน้าไม่ตอบหลังทำ (กำลังโหลด หรือเปลี่ยนหน้าอยู่) ใช้ wait หรือ read ก่อนทำต่อ"
	}
	if after.URL != before.URL {
		var b strings.Builder
		fmt.Fprintf(&b, "\n→ หน้าเปลี่ยนเป็น %s", browserPageRef(after.Title, after.URL))
		if snap, err := a.browserSnapshot(string(id), ""); err == nil && len(snap.Elements) > 0 {
			b.WriteString(" — refs ของหน้าใหม่:")
			for i, el := range snap.Elements {
				if i >= changeNoteRefs {
					fmt.Fprintf(&b, "\n… และอีก %d (read เพื่อดูทั้งหมด)", snap.ElementsTotal-changeNoteRefs)
					break
				}
				b.WriteString("\n" + elementLine(el))
			}
		}
		return b.String()
	}
	var parts []string
	if after.Title != before.Title {
		parts = append(parts, fmt.Sprintf("ชื่อหน้า %q → %q", before.Title, after.Title))
	}
	if after.Count != before.Count {
		parts = append(parts, fmt.Sprintf("element ที่กดได้ %d → %d (refs เดิมหมดอายุ read ใหม่ก่อนใช้ ref)", before.Count, after.Count))
	}
	if after.Focus != before.Focus {
		parts = append(parts, fmt.Sprintf("focus %s → %s", before.Focus, after.Focus))
	}
	if len(parts) == 0 {
		return "\n→ หน้าไม่เปลี่ยน (URL, จำนวน element และ focus เท่าเดิม) ถ้าคาดว่าจะเปลี่ยน ใช้ wait หรือ read"
	}
	return "\n→ " + strings.Join(parts, "; ")
}
