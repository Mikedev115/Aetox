package main

// เลขกำกับบนภาพ — the refs a `read` hands out, drawn into the photograph.
//
// The asymmetry this closes was the owner's, 8 ก.ย.: *"เราทำระบบให้โมเดลที่อ่านภาพ
// ไม่ได้ควบคุมเบราว์เซอร์ได้ดีมาก แต่โมเดลที่อ่านภาพได้ เรากลับทำให้ได้ไม่ดี"* — and he is
// right about where the seam is. A blind model gets `read`, which walks the DOM,
// stamps `data-aetox-ref` on everything clickable and hands back a numbered
// table; it then clicks `ref 12` and hits exactly that node. A model with eyes
// got a clean photograph and a multiplication to do: pixel × ratio = x,y
// (captureScaleNote). The refs were on the page the whole time. They were simply
// invisible, because an attribute is not a pixel.
//
// So the seeing model was made to work HARDER than the blind one, and to work
// in the weaker unit. A coordinate has to survive CSS px → device pixel ratio →
// zoom preset → the provider's own downscale (Anthropic to a 1568px long edge,
// DeepSeek to about the pixel count of 800×800), be read off the shrunken
// picture by eye, and be multiplied back — and it dies the moment the page
// scrolls or relayouts, which the tool already admits by treating x,y as a blind
// aim (stepAimsBlind). A ref is bound to a node and survives all of that.
//
// ## The rules this obeys
//
//   - **One ref space, never two.** The numbers drawn here are read straight off
//     `data-aetox-ref`, which only textScript ever writes. Nothing renumbers,
//     nothing counts. A second numbering that happened to agree most of the time
//     would produce the worst bug this feature can have: the model reads the
//     right number off the picture and presses a different control.
//   - **Drawn in the page, like every other thing Aetox draws on somebody's
//     page.** A tab is a native WebView2 window composited ABOVE the app, so
//     anything the app draws over that rectangle is behind it (browser_marks.go
//     says this at length). documentElement, position:fixed, pointer-events
//     none — the same three rules pickScript and the busy marks follow.
//   - **Only what is in the viewport.** Refs are stamped across the whole
//     document; a picture only shows one screen of it. Marking what is not in
//     the frame would put numbers on nothing.
//   - **The numbers are addresses, not ranks.** A viewport holding refs 3, 7 and
//     22 is drawn as 3, 7 and 22. Renumbering them 1, 2, 3 to look tidy is
//     exactly the second ref space above, arrived at by good intentions.
//   - **Taken off before anything else looks.** capture photographs the page and
//     then removes the layer; the user is left with the page they had. It is the
//     one deliberate exception to "nothing of Aetox's own in the photograph",
//     and it is an exception because here the annotation IS the answer.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// refMarkLayerID is the overlay's identity: one element holding every box and
// every chip, so taking the marks off is one removal and cannot half-finish.
//
// Deliberately NOT markElementID. That id belongs to the busy marks, whose
// mount helper takes down whatever wore it before and whose timer removes it
// after a second and a half — both fatal here, where the layer has to outlive
// nothing at all and then be removed on the exact beat the photograph is taken.
const refMarkLayerID = "__aetox-refmarks"

// refMarkCap is how many controls one picture is allowed to be labelled with.
//
// Past this the page stops being a page with numbers on it and becomes a wall
// of numbers, which answers no question — and the model cannot tell a crowded
// picture from a complete one, so the count that was cut is reported instead of
// being left to be noticed. 150 (textScript's cap) in one viewport is a page
// where nothing could be read anyway; 60 is a dense app screen fully marked.
const refMarkCap = 60

// refMarkAccent is the ink. The same blue every other thing Aetox draws on
// someone's page uses (markAccent), for the reason stated there: a user who has
// seen one should recognise the other. A number in white on solid blue is what
// carries the meaning, and it stays legible on a page that is itself blue.
const refMarkAccent = markAccent

// refMarkChipH is the label's height in CSS px, and the font is 13px bold.
//
// Sized for what the MODEL will see, not for what looks right on screen. The
// picture is the viewport times DPR and is then shrunk again by the provider to
// its own budget, so a 13px label on a 1280px-wide viewport arrives at roughly
// 13px — about the size of ordinary body text on the same page, which every
// vision model reads. A smaller, tidier chip is the one change to this file
// that would quietly break it.
const refMarkChipH = 18

// refMarkScript draws a box round every in-viewport ref and a numbered chip at
// its corner, then reports which refs it drew and how many were in view.
//
// **The chip sits INSIDE its own box, never above it.** The first version put it
// above, on the reasoning that a number should not cover the word it names —
// which is right in isolation and wrong on a real page. Tried against a
// Wikipedia article, 8 ก.ย.: in the table of contents, a column of 24px rows,
// every chip landed squarely on the text of the row ABOVE its own. A label
// covering a little of the thing it names is unambiguous; a label covering a
// DIFFERENT element is a lie, and the model has no way to tell which it is
// looking at. So the chip is attached to its box and clamped into the frame,
// and an element half off-screen still shows its number.
func refMarkScript(token string, cap int) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s
  var LAYER=%q, ACC=%q, CH=%d, CAP=%d;
  var old=document.getElementById(LAYER);
  if(old&&old.parentNode)old.parentNode.removeChild(old);
  var root=document.documentElement;
  if(!root){aetoxReport(%s,0,null,{marks:[],count:0});return;}
  var vw=window.innerWidth||0, vh=window.innerHeight||0;
  /* Every root textScript tagged, shadow trees included: a ref it handed out
     and this could not find would be a ref the picture silently omits. */
  var roots=aetoxScan().roots, seen={}, boxes=[];
  for(var r=0;r<roots.length;r++){
    var els;
    try{els=roots[r].querySelectorAll('[data-aetox-ref]');}catch(e){continue;}
    for(var i=0;i<els.length;i++){
      var el=els[i];
      var ref=parseInt(el.getAttribute('data-aetox-ref'),10);
      if(!ref||seen[ref])continue;
      var b=el.getBoundingClientRect();
      if(b.width<=0||b.height<=0)continue;
      /* Out of frame entirely. Partly in stays: half a button is still the
         button, and the number tells the model which one. */
      if(b.bottom<=0||b.top>=vh||b.right<=0||b.left>=vw)continue;
      seen[ref]=1;
      boxes.push({ref:ref,x:b.left,y:b.top,w:b.width,h:b.height});
    }
  }
  /* By ref, so the cap keeps the same controls a read would have listed first
     rather than whichever root happened to be scanned first. */
  boxes.sort(function(p,q){return p.ref-q.ref;});
  var inview=boxes.length;
  if(boxes.length>CAP)boxes=boxes.slice(0,CAP);

  var layer=document.createElement("div");
  layer.id=LAYER;
  layer.style.cssText="position:fixed;left:0;top:0;width:100%%;height:100%%;margin:0;padding:0;border:0;background:none;pointer-events:none;z-index:2147483602;";
  root.appendChild(layer);

  var drawn=[];
  for(var k=0;k<boxes.length;k++){
    var bx=boxes[k];
    drawn.push(bx.ref);
    var o=document.createElement("div");
    o.style.cssText="position:absolute;box-sizing:border-box;margin:0;padding:0;background:none;left:"+bx.x+"px;top:"+bx.y+"px;width:"+bx.w+"px;height:"+bx.h+"px;border:2px solid "+ACC+";border-radius:2px;";
    layer.appendChild(o);
    /* Attached to its own box, then clamped into the frame: a control scrolled
       half off the top still gets a readable number, and no chip is ever drawn
       over a neighbour it does not name. */
    var ly=bx.y, lx=bx.x;
    if(ly<0)ly=0;
    if(ly>vh-CH)ly=vh-CH;
    if(lx<0)lx=0;
    if(lx>vw-34)lx=vw-34;
    var chip=document.createElement("div");
    /* textContent, never innerHTML: a page with Trusted Types enforced refuses
       the second one, and this has to work on exactly those pages. */
    chip.textContent=String(bx.ref);
    chip.style.cssText="position:absolute;box-sizing:border-box;margin:0;left:"+lx+"px;top:"+ly+"px;height:"+CH+"px;line-height:"+(CH-2)+"px;padding:0 5px;background:"+ACC+";color:#fff;font-family:ui-sans-serif,system-ui,-apple-system,'Segoe UI',sans-serif;font-size:13px;font-weight:700;font-style:normal;letter-spacing:0;text-transform:none;white-space:nowrap;border-radius:3px;box-shadow:0 0 0 1px rgba(0,0,0,.45);";
    layer.appendChild(chip);
  }
  aetoxReport(%s,0,null,{marks:drawn,count:inview});
})()`, aetoxScanJS, aetoxActJS(), refMarkLayerID, refMarkAccent, refMarkChipH, cap, string(tok), string(tok))
}

// clearRefMarksScript takes the whole layer off in one removal.
func clearRefMarksScript(token string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s
  var old=document.getElementById(%q);
  if(old&&old.parentNode)old.parentNode.removeChild(old);
  aetoxReport(%s,0,null);
})()`, aetoxActJS(), refMarkLayerID, string(tok))
}

// drawRefMarks puts the numbers on the page and says what it managed to label:
// the refs drawn, and how many were in the frame before the cap.
//
// Three seconds rather than the default two: this runs a querySelectorAll and
// a getBoundingClientRect per ref across every root, which on a heavy app is
// real layout work, and the alternative to waiting is a photograph of a page
// halfway through being labelled.
func (a *App) drawRefMarks(id AgentTabID, cap int) (drawn []int, inView int, err error) {
	res, answered, err := a.browserActOnFor(string(id), func(token string) string {
		return refMarkScript(token, cap)
	}, 3*time.Second)
	if err != nil {
		return nil, 0, err
	}
	if !answered {
		return nil, 0, fmt.Errorf("หน้าเว็บไม่ตอบตอนวางเลขกำกับ")
	}
	return res.Marks, res.Count, nil
}

// clearRefMarks takes the layer off, and every return value is dropped for the
// same reason clearPageMarks drops its own: there is nothing a caller could do
// differently, and a capture that failed because a mark could not be confirmed
// gone would be a courtesy grown into a gate. It WAITS, so the removal has
// landed before the next thing looks at the page.
func (a *App) clearRefMarks(id AgentTabID) {
	_, _, _ = a.browserActOn(string(id), clearRefMarksScript)
}

// refMarkLegend is the picture's key: what each number in it is.
//
// This is the half that makes marks better than either of the two answers that
// existed before, rather than merely different. `read` gives a table with no
// picture; a bare capture gives a picture with no names. One call now returns
// both, and they cannot disagree, because the numbers in the picture and the
// numbers in this list are the same attribute read twice.
//
// Elements come from the read that stamped the refs, so a ref drawn on the page
// and missing from that list is not possible — but it is checked anyway, and a
// number with nothing known about it is simply left out of the key rather than
// printed as an empty quote.
func refMarkLegend(drawn []int, els []browserElement, inView int) string {
	if len(drawn) == 0 {
		return "\nไม่มีปุ่มหรือช่องกรอกที่กำกับได้ในจอนี้ (หน้านี้อาจวาดเนื้อหาเองทั้งหมด) ใช้พิกัด x,y ตามสูตรด้านบนแทน"
	}
	byRef := make(map[int]browserElement, len(els))
	for _, el := range els {
		byRef[el.Ref] = el
	}
	sort.Ints(drawn)

	var b strings.Builder
	b.WriteString("\nเลขในภาพคือ ref ใช้กับ click/type/hover ได้ตรง ๆ ไม่ต้องแปลงพิกัด:")
	for _, ref := range drawn {
		el, ok := byRef[ref]
		if !ok {
			continue
		}
		name := el.Role
		if name == "" {
			name = el.Tag
		}
		line := fmt.Sprintf("\n[%d] %s", ref, name)
		if text := strings.TrimSpace(el.Text); text != "" {
			if len([]rune(text)) > 40 {
				text = string([]rune(text)[:40]) + "…"
			}
			line += fmt.Sprintf(" %q", text)
		}
		if el.Focused {
			line += " (คีย์บอร์ดอยู่ที่นี่)"
		}
		b.WriteString(line)
	}
	if inView > len(drawn) {
		b.WriteString(fmt.Sprintf("\nในจอนี้มีของกดได้ %d อย่าง กำกับให้ %d อย่างแรก ที่เหลือดูจาก read", inView, len(drawn)))
	}
	b.WriteString("\nสิ่งที่อยู่นอกจอไม่ได้ถูกกำกับ เลื่อนหน้าแล้วแคปใหม่ถ้าต้องกดของที่ยังไม่เห็น")
	return b.String()
}
