package main

// เมาส์ ลูกศรเลื่อน และแรงกระเพื่อมบนหน้าเว็บ — the one busy-signal layer the window
// cannot draw.
//
// The other three live in territory the frontend owns: the panel's border, the
// strip under the toolbar, the mark on a tab chip. This one does not exist
// there at all, and the reason is the workbench browser's shape. A tab is a
// native WebView2 window glued over the pane by BrowserSetBounds, so it
// composites ABOVE the app's own webview, and everything the app draws over
// that rect is behind it. A ring pointing at a button would be a ring pointing
// at the back of a window.
//
// So the only way into those pixels is the way `read` already goes: a script
// evaluated inside the page. This file is that script, and it obeys three rules
// that are not style choices.
//
//   - **Cleared before `capture`.** BrowserCapturePNG photographs the real
//     page, and the click mark lands on the very pixel the press did. A
//     picture the model then reads would have a ripple drawn across the thing
//     it was looking for, with nothing to say the circle is not the site's.
//   - **Mounted on the document element, never inside the page's own tree.**
//     A `transform` or a `filter` anywhere up the ancestor chain makes a new
//     containing block, and `position:fixed` inside one is no longer fixed to
//     the viewport — the mark drifts with the page instead of pointing at it.
//     documentElement rather than body for the same reason one step further:
//     plenty of sites put `transform:translateZ(0)` on body to force a
//     compositing layer, which is exactly the hazard. It is also where
//     pickScript mounts, so the two things Aetox draws on somebody else's page
//     agree with each other.
//   - **The previous one removed before the next is drawn.** Exactly what
//     textScript does with stale data-aetox-ref attributes, for the same
//     reason: rapid clicks and scrolls would otherwise leave a page stacked
//     with rings and arrows for actions that finished a second ago, which
//     reads as the agent still doing all of them at once.
//
// **Never a timer, always an action.** Nothing here draws on a clock. A mark is
// made by a click or a scroll and takes itself off shortly after, which is not
// the same thing: a mark on screen means an action happened, and no action
// means an empty page.
//
// **prefers-reduced-motion outranks the switch.** The switch says what Aetox
// would like to draw; the machine says whether it moves. A page whose user has
// asked for no motion gets the mark and none of the travel, which is the same
// answer the stylesheet gives every other animation in the app.

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// markElementID is the id the mark carries, and the whole of its identity: the
// clear reads it, the mount overwrites whatever wore it before, and the delayed
// removal checks it is still holding the mark it was given before taking
// anything away.
const markElementID = "__aetox-busy-mark"

// markAccent is the blue the pick overlay defaults to, spelled the same way for
// the same reason: these are the two things Aetox ever draws on somebody else's
// page, and a user who has seen one should recognise the other.
//
// Not read from the user's theme, unlike pickScript's, and that is a real
// difference rather than an oversight. Pick is started by a click in the
// window, so the frontend is right there to hand its accent down. A mark is
// made from inside a tool call on the engine's goroutine, where there is no
// frontend anywhere in the call chain to ask.
const markAccent = "#378add"

// markLifetimeMS is how long a mark stays before it takes itself off.
//
// Long enough to be caught by somebody who was looking elsewhere when it
// appeared, short enough to be gone before the next action in an ordinary
// sequence: a click already sleeps 300ms afterwards and a scroll 700ms, so a
// run of them keeps a live mark on screen continuously without ever leaving a
// stale one beside a fresh one.
//
// It is the floor rather than the rule, because one action is no longer one
// move: `scroll` takes a number of screens now (§176), and an arrow that left
// after 1.6s of a seven-second scroll would say the page had stopped while it
// was still going. A caller that knows its action is long says so.
const markLifetimeMS = 1600

// aetoxMarkJS is the half both marks share: take the old one down, put the new
// one up, and arrange for it to leave.
//
// Styles are set through cssText rather than setAttribute("style", …) because
// the CSSOM path is not what a page's Content-Security-Policy governs — the
// same reason pickScript has always been able to draw on sites that forbid
// inline styles. Motion goes through element.animate() for the same reason one
// step further: a @keyframes rule needs a <style> element, which style-src can
// refuse, while the Web Animations API needs nothing from the page at all.
func aetoxMarkJS(lifeMS int) string {
	if lifeMS < markLifetimeMS {
		lifeMS = markLifetimeMS
	}
	return fmt.Sprintf(`
  var AETOX_MARK=%q, AETOX_ACC=%q;
  function aetoxMarkClear(){
    var old=document.getElementById(AETOX_MARK);
    if(old&&old.parentNode)old.parentNode.removeChild(old);
  }
  function aetoxMarkQuiet(){
    try{return !!(window.matchMedia&&window.matchMedia("(prefers-reduced-motion: reduce)").matches);}
    catch(e){return false;}
  }
  function aetoxMarkMount(el,css){
    aetoxMarkClear();
    var root=document.documentElement;
    if(!root)return null;
    el.id=AETOX_MARK;
    el.style.cssText="position:fixed;z-index:2147483601;pointer-events:none;box-sizing:border-box;margin:0;padding:0;"+css;
    root.appendChild(el);
    var quiet=aetoxMarkQuiet();
    if(!quiet&&el.animate){
      try{el.animate([{opacity:0},{opacity:1}],{duration:120,easing:"ease-out"});}catch(e){}
    }
    setTimeout(function(){
      /* Only ever takes down the mark it put up. A later action has already
         cleared this one and hung its own under the same id, and a timer that
         fired blind would be deleting somebody else's. */
      if(document.getElementById(AETOX_MARK)!==el)return;
      if(quiet||!el.animate){if(el.parentNode)el.parentNode.removeChild(el);return;}
      try{
        var out=el.animate([{opacity:1},{opacity:0}],{duration:220,easing:"ease-in"});
        out.onfinish=function(){if(el.parentNode)el.parentNode.removeChild(el);};
      }catch(e){if(el.parentNode)el.parentNode.removeChild(el);}
    },%d);
    return el;
  }
`, markElementID, markAccent, lifeMS)
}

// markRippleScript sends a ripple out from the pixel a click just landed on.
//
// It replaces the ring that used to be drawn around the element's box. The
// owner asked for the halo off on 6 ก.ย. and then, looking at what was left,
// asked the better question: *"แรงกระเพื่อม กระจายออกจากจุดคลิกดีไหม"*. It is the
// right answer for a reason the ring could not reach. A box says WHICH control
// — and the pointer is already sitting on that control, so it was the half
// already covered. What nothing on the page said was WHERE inside it the press
// landed: an arrow's point is one or two pixels of ink under a moving sprite,
// and on a wide control that is not enough to read the spot from. The ripple
// is that pixel, drawn — and it stayed the right answer when the sprite went
// back to being an arrow, because the thing it adds was never about the shape
// of the cursor.
//
// Two waves rather than one, the second 150ms behind: one ring reads as a
// circle that appeared, two read as something leaving the point. They go out
// from a zero-sized box mounted AT the point, so the whole mark is placed by
// one pair of coordinates and the waves are children scaling about their own
// centre.
//
// A click by coordinates gets one too, which the ring never did — it needed a
// ref to measure, so `click x,y` on a canvas drew nothing at all.
func markRippleScript(x, y float64) string {
	return fmt.Sprintf(`(function(){%s
  var box=document.createElement("div");
  var mounted=aetoxMarkMount(box,"left:%gpx;top:%gpx;width:0;height:0;");
  if(!mounted)return;
  var quiet=aetoxMarkQuiet();
  function wave(delay){
    var w=document.createElement("div");
    /* Base opacity 0: element.animate() with the default fill leaves the
       element at its own style once it finishes, and a wave that came to rest
       visible would be a ring sitting on the page for the rest of the mark's
       life — exactly the halo that was just taken off. */
    w.style.cssText="position:absolute;left:-22px;top:-22px;width:44px;height:44px;box-sizing:border-box;"+
      "border:2.5px solid "+AETOX_ACC+";border-radius:50%%;opacity:0;";
    box.appendChild(w);
    if(quiet||!w.animate){
      /* No motion allowed: one still ring at the point and no travel — the
         rule the whole layer follows, that the mark happens and the movement
         does not. */
      if(delay===0){w.style.opacity=".5";w.style.transform="scale(.62)";}
      return;
    }
    /* Three stops, not two. A straight fade from .9 to 0 spends most of its
       560ms already half gone; holding most of the ink through the first
       third is what makes the wave read as a wave rather than as a ring that
       was always faint. */
    try{w.animate([{transform:"scale(.16)",opacity:.95},
      {transform:"scale(.55)",opacity:.7,offset:.35},
      {transform:"scale(1)",opacity:0}],
      {duration:560,delay:delay,easing:"cubic-bezier(.15,.7,.3,1)"});}catch(e){}
  }
  wave(0);wave(150);
  var dot=document.createElement("div");
  dot.style.cssText="position:absolute;left:-3.5px;top:-3.5px;width:7px;height:7px;border-radius:50%%;"+
    "background:"+AETOX_ACC+";opacity:0;";
  box.appendChild(dot);
  if(quiet||!dot.animate){dot.style.opacity=".75";}
  else{try{dot.animate([{opacity:.95,transform:"scale(1)"},{opacity:0,transform:"scale(.4)"}],
    {duration:460,easing:"ease-out"});}catch(e){}}
})()`, aetoxMarkJS(markLifetimeMS), x, y)
}

// markScrollScript draws an arrow in the direction the page is about to move,
// and holds it for as long as the move will take.
//
// Two chevrons for the jumps (top, bottom) and one for a screen at a time, so
// the four directions read as two distances without a word being written. The
// words are the action bar's job, and this mark is drawn on a page whose
// language is not the app's.
func markScrollScript(to string, holdMS int) string {
	up := to == "up" || to == "top"
	// The chevron is a square wearing two of its four borders, turned. Built
	// this way rather than as an SVG because innerHTML is the one thing a page
	// with Trusted Types enforced refuses outright, and a mark that vanishes on
	// exactly the strictest sites is a mark nobody can rely on.
	turn, place, travel := "rotate(45deg)", "bottom:44px", "12px"
	if up {
		turn, place, travel = "rotate(-135deg)", "top:44px", "-12px"
	}
	count := 1
	if to == "top" || to == "bottom" {
		count = 2
	}
	return fmt.Sprintf(`(function(){%s
  var box=document.createElement("div");
  var mounted=aetoxMarkMount(box,
    "left:0;right:0;%s;display:flex;flex-direction:column;align-items:center;gap:4px;"+
    "filter:drop-shadow(0 0 8px "+AETOX_ACC+"88);");
  if(!mounted)return;
  for(var i=0;i<%d;i++){
    var c=document.createElement("div");
    c.style.cssText="width:20px;height:20px;box-sizing:border-box;"+
      "border-right:5px solid "+AETOX_ACC+";border-bottom:5px solid "+AETOX_ACC+";"+
      "transform:%s;opacity:"+(i===0?"1":".55");
    box.appendChild(c);
  }
  if(box.animate&&!aetoxMarkQuiet()){
    /* One trip in the direction of travel, then still. The page is moving
       underneath it; the arrow says which way, once. */
    try{box.animate([{transform:"translateY(0)"},{transform:"translateY(%s)"}],
      {duration:420,easing:"cubic-bezier(.3,.7,.4,1)"});}catch(e){}
  }
})()`, aetoxMarkJS(holdMS), place, count, turn, travel)
}

// clearMarksScript takes down whatever is up, draws nothing, and says when it
// is done.
//
// **The report is the whole point of this one.** Every other mark script is
// fire and forget, which is right for a courtesy nobody waits on. This one has
// a caller that genuinely cannot proceed without it: `capture` photographs the
// real page, and a ring left standing goes into the picture the model then
// reads, with nothing to tell it the circle is not part of the site.
//
// Ordering by queue is not enough to prevent that, which is the part worth
// writing down. Both this and the screenshot go through host.onTab, so they are
// ENQUEUED in order — but v.eval is ExecuteScript, which hands the script to the
// page and returns; the page runs it whenever its own thread next gets to it.
// So "queued first" says nothing about "finished first", and the only thing
// standing between a stale ring and the photograph was a 400ms sleep put there
// for something else entirely. A sleep is not an ordering primitive; it is a
// guess that has been right so far.
//
// It reuses aetoxActJS's envelope rather than inventing a second one, so this
// rides the same token/channel/timeout the click report has always used
// (browserActOn). ref 0 and a null element: nothing was aimed at.
func clearMarksScript(token string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s
  var old=document.getElementById(%q);
  if(old&&old.parentNode)old.parentNode.removeChild(old);
  /* The cursor and its trail are hidden rather than removed: a capture
     must not photograph them, and the user must not see the arrow vanish
     and reappear somewhere else — it stays where it was and comes back
     there. */
  var keep=[%q,%q];
  for(var i=0;i<keep.length;i++){
    var el=document.getElementById(keep[i]);
    if(el)el.style.visibility="hidden";
  }
  aetoxReport(%s,0,null);
})()`, aetoxActJS(), markElementID, cursorElementID, trailElementID, string(tok))
}

// The cursor: the one mark that is not an action.
//
// Owner, 6 ก.ย.: *"เรายังไม่ได้ทำไอคอนเมาส์ให้มันเลย"*, and the choice that followed —
// *"เคอร์เซอร์ + เส้นทางการลาก"*. Everything above this line is a mark that an
// action makes and that leaves; the cursor is a pointer the user can watch
// travel, and it stays. It breaks the "never a timer, always an action" rule
// on purpose, because a pointer that vanished between actions would read as
// the agent letting go of the mouse, and the whole point of drawing one is
// that the agent is holding it.
//
// It is NOT mounted through aetoxMarkMount. That helper takes the previous
// mark down before putting a new one up — right for a ring, fatal for a
// cursor that has to outlive every ring drawn around it. It has its own id,
// its own mount, and the same three rules: documentElement, fixed,
// pointer-events:none. One more than the ring's z-index, so the arrow is
// over the ring it points into.
//
// The sprite is an SVG arrow built with createElementNS rather than innerHTML,
// because innerHTML is what a page with Trusted Types enforced refuses, and
// this cursor exists to be seen on Google's pages.
//
// The trail is a canvas at device pixel ratio, built the way the pick
// overlay's ink layer is, drawn under the cursor as it sweeps and faded out
// once the sweep ends. The browser's own selection highlight shows what a
// drag selected; the trail shows where the hand went.
const (
	cursorElementID = "__aetox-cursor"
	trailElementID  = "__aetox-trail"
)

// cursorTipX and cursorTipY are where the arrow's point sits inside the
// sprite's own box, and therefore the only place a click can be said to happen.
//
// Three things have to agree on this pair or the sprite lies about where it is
// pointing: the group transform that puts the point there, the transform-origin
// the press squeezes about, and the translate that moves the sprite to a page
// coordinate. They are one constant used three times for that reason.
//
// A small inset rather than zero, because the outline is stroked around the
// shape: at (0,0) the point's own casing would fall outside the sprite's box
// and be clipped, and a pointer with a flattened tip is the one part of an
// arrow the eye reads as broken.
const (
	cursorTipX = 1.6
	cursorTipY = 1.6
)

// cursorInk is the outline drawn around the arrow: near-black, not pure black,
// the same ink the window uses behind its own mark.
//
// **The sprite was the logo for one day and is an ordinary arrow again.** The
// 6 ก.ย. version leaned the app's A fourteen degrees and used its apex as the
// hotspot, on the reasoning that a pointer the user watches on somebody else's
// site may as well say whose hand it is. Seen in place the owner's answer was
// *"เอาเมาส์ปกติก็ได้ ไม่เอาแบบนี้"*, and it is the right call for a reason worth
// keeping: a cursor is not a place to sign your name. It is the one element on
// screen whose whole job is to be read instantly and without thought, and a
// letterform at 20px is read — briefly, but read — before it is understood as
// a pointer. That pause is the cost, it is paid on every single action, and
// nothing is bought with it that the ring, the ripple and the trail were not
// already saying.
//
// White body with an ink outline is what every operating system draws, and it
// is also the one pairing that survives an unknown page: a white sprite alone
// vanishes into Google's results and a dark one into a terminal theme, while a
// light body carrying a dark edge keeps a border against both.
const cursorInk = "#0e1116"

// cursorArrowPath is the pointer itself: the shape an operating system draws,
// with its point at (0,0) so the hotspot needs no arithmetic to find.
//
// Written in CSS pixels rather than in a design grid, and therefore at no
// scale: the sprite is ~20px tall because these numbers say 19.7, which is
// what a system arrow measures. The version this replaces carried a 804×762
// grid and a 0.04703 scale down from it, and every one of those numbers was a
// place the tip could drift away from where the click actually went.
//
// Read it as the outline it is, clockwise from the point: straight down the
// left edge, up into the notch, out to the tail, back up the tail's other
// side, out to the shoulder, and home along the diagonal.
const cursorArrowPath = `M 0,0 L 0,17.1 L 4.2,13.3 L 6.7,19.7 L 9.2,18.6 L 6.8,12.4 L 12.1,12.0 Z`

// cursorTravelMax is the most a pointer action waits for the sprite to arrive
// before the press goes in.
//
// It was 350 ms, and the owner watching it said *"เมาส์มันค้าง ๆ นึง ๆ พอขยับก็เหมือน
// วาปไปมา"* (6 ก.ย.): a hand crossing a screen in a third of a second is not a
// hand, it is a teleport with a tail. A person takes most of a second for a
// long reach and starts and stops gently; the sprite now does the same, and
// the click waits for it. That is the one latency this layer adds, and it is
// the price of the thing it is for.
const cursorTravelMax = 800 * time.Millisecond

// cursorTravel is how long the sprite takes to reach a target: a floor so a
// short hop is still a visible move, a per-pixel rate so a long one reads as
// travel, and a ceiling so a click never waits longer than a hand would.
// From nowhere (the first action on a fresh tab) it simply appears.
func cursorTravel(from, to point, known bool) time.Duration {
	if !known {
		return 120 * time.Millisecond
	}
	d := math.Hypot(to.X-from.X, to.Y-from.Y)
	ms := 160 + d*0.6
	if ms > float64(cursorTravelMax/time.Millisecond) {
		ms = float64(cursorTravelMax / time.Millisecond)
	}
	return time.Duration(ms) * time.Millisecond
}

// aetoxCursorJS is the sprite: find it or build it, move it, press it.
func aetoxCursorJS() string {
	return fmt.Sprintf(`
  var AETOX_CUR=%q, AETOX_TRAIL=%q, AETOX_CACC=%q, AETOX_INK=%q, AETOX_ARROW=%q;
  function aetoxCursorQuiet(){
    try{return !!(window.matchMedia&&window.matchMedia("(prefers-reduced-motion: reduce)").matches);}
    catch(e){return false;}
  }
  function aetoxCursorEl(){
    var c=document.getElementById(AETOX_CUR);
    if(c)return c;
    var root=document.documentElement;
    if(!root)return null;
    var ns="http://www.w3.org/2000/svg";
    /* Attributes rather than a template: innerHTML is what a page with
       Trusted Types enforced refuses outright, and this sprite exists to be
       seen on exactly those sites. */
    function node(name,attrs){
      var n=document.createElementNS(ns,name);
      for(var k in attrs)n.setAttribute(k,attrs[k]);
      return n;
    }
    /* The sprite is an ordinary arrow, drawn at 1:1 — the path is already in
       CSS pixels, so there is no scale and no rotation, and the only transform
       is the one that puts its point on the hotspot. That is the whole of the
       geometry now; the version this replaces stacked a scale and a rotate on
       top of a translate out of a 804x762 design grid, and each of those was a
       place the drawn tip could disagree with the pixel being clicked.
       The viewBox is the shape (12.1x19.7) plus the hotspot inset and room for
       the casing on the far side, which is why 16x24 covers it. */
    var svg=node("svg",{viewBox:"0 0 16 24",width:"16",height:"24",fill:"none"});
    var g=node("g",{transform:"translate(%g,%g)"});
    /* One path, filled white and stroked in ink: the body and its border are
       the same outline, which is what keeps the border an even weight all the
       way round a shape with a 20-degree corner at the tip. Round joins so
       that corner is a point rather than a spike, and the miter a sharp join
       would throw past the tip is exactly what the box has no room for. */
    g.appendChild(node("path",{d:AETOX_ARROW,fill:"#ffffff",stroke:AETOX_INK,
      "stroke-width":"1.5","stroke-linejoin":"round"}));
    svg.appendChild(g);
    c=document.createElement("div");
    c.id=AETOX_CUR;
    /* transform-origin at the tip, not the middle: the press below squeezes
       the sprite, and a squeeze about the centre of a 30px box walks the tip
       off the control it is pressing. */
    c.style.cssText="position:fixed;left:0;top:0;width:16px;height:24px;margin:0;padding:0;z-index:2147483602;pointer-events:none;transform-origin:%gpx %gpx;filter:drop-shadow(0 1px 2px rgba(0,0,0,.35));will-change:transform;";
    c.appendChild(svg);
    c.__aetoxX=0;c.__aetoxY=0;
    root.appendChild(c);
    return c;
  }
  /* The mark's apex is the hotspot, so the sprite is offset by where that
     apex sits inside its own box rather than by a corner. */
  function aetoxCursorAt(x,y){return "translate("+(x-%g)+"px,"+(y-%g)+"px)";}
  /* Ease in AND out: a hand accelerates away from where it was and settles
     onto where it is going. The ring's "ease-out only" is right for a
     press that already happened and wrong for a reach. */
  function aetoxCursorTo(x,y,ms,press){
    var c=aetoxCursorEl();
    if(!c)return;
    c.style.visibility="visible";
    var from=aetoxCursorAt(c.__aetoxX,c.__aetoxY),to=aetoxCursorAt(x,y);
    c.__aetoxX=x;c.__aetoxY=y;
    if(ms<=0||aetoxCursorQuiet()||!c.animate){c.style.transform=to;}
    else{
      c.style.transform=to;
      try{c.animate([{transform:from},{transform:to}],{duration:ms,easing:"cubic-bezier(.45,.05,.25,1)"});}catch(e){}
    }
    if(press&&c.animate&&!aetoxCursorQuiet()){
      setTimeout(function(){
        try{c.animate([{transform:to+" scale(1)"},{transform:to+" scale(.8)"},{transform:to+" scale(1)"}],{duration:160,easing:"ease-out"});}catch(e){}
      },ms);
    }
  }
  function aetoxTrailClear(){
    var old=document.getElementById(AETOX_TRAIL);
    if(old&&old.parentNode)old.parentNode.removeChild(old);
  }
  /* The sweep: cursor and trail driven by one clock, so the line is always
     drawn to where the arrow is. Ease-out, the same feel as the travel. */
  function aetoxCursorDrag(fx,fy,tx,ty,ms){
    aetoxTrailClear();
    var c=aetoxCursorEl();
    if(!c)return;
    var root=document.documentElement;
    var cv=document.createElement("canvas");
    cv.id=AETOX_TRAIL;
    var dpr=window.devicePixelRatio||1,W=window.innerWidth,H=window.innerHeight;
    cv.width=Math.round(W*dpr);cv.height=Math.round(H*dpr);
    cv.style.cssText="position:fixed;left:0;top:0;width:"+W+"px;height:"+H+"px;margin:0;padding:0;z-index:2147483601;pointer-events:none;";
    root.appendChild(cv);
    var ctx=cv.getContext("2d");
    if(ctx){ctx.scale(dpr,dpr);ctx.strokeStyle=AETOX_CACC;ctx.lineWidth=3;ctx.lineCap="round";ctx.lineJoin="round";ctx.globalAlpha=.85;}
    var quiet=aetoxCursorQuiet()||!window.requestAnimationFrame;
    var t0=null;
    function ease(t){return 1-Math.pow(1-t,3);}
    function at(t){return{x:fx+(tx-fx)*t,y:fy+(ty-fy)*t};}
    function done(){
      c.__aetoxX=tx;c.__aetoxY=ty;
      c.style.transform=aetoxCursorAt(tx,ty);
      setTimeout(function(){
        if(document.getElementById(AETOX_TRAIL)!==cv)return;
        if(quiet||!cv.animate){aetoxTrailClear();return;}
        try{var out=cv.animate([{opacity:1},{opacity:0}],{duration:1200,easing:"ease-in"});out.onfinish=aetoxTrailClear;}
        catch(e){aetoxTrailClear();}
      },400);
    }
    if(quiet||ms<=0){
      if(ctx){ctx.beginPath();ctx.moveTo(fx,fy);ctx.lineTo(tx,ty);ctx.stroke();}
      done();
      return;
    }
    c.style.transform=aetoxCursorAt(fx,fy);
    c.__aetoxX=fx;c.__aetoxY=fy;
    function frame(now){
      if(t0===null)t0=now;
      var t=Math.min(1,(now-t0)/ms),p=at(ease(t));
      c.style.transform=aetoxCursorAt(p.x,p.y);
      if(ctx){ctx.clearRect(0,0,W,H);ctx.beginPath();ctx.moveTo(fx,fy);ctx.lineTo(p.x,p.y);ctx.stroke();
        ctx.beginPath();ctx.arc(fx,fy,4,0,Math.PI*2);ctx.fillStyle=AETOX_CACC;ctx.fill();}
      if(t<1)requestAnimationFrame(frame);else done();
    }
    requestAnimationFrame(frame);
  }
`, cursorElementID, trailElementID, markAccent, cursorInk, cursorArrowPath,
		cursorTipX, cursorTipY, cursorTipX, cursorTipY, cursorTipX, cursorTipY)
}

// cursorShowScript puts the sprite at a point with no travel: a new document
// after a navigation, or the page right after a capture that had to take it
// down.
func cursorShowScript(x, y float64) string {
	return fmt.Sprintf(`(function(){%s
  aetoxCursorTo(%g,%g,0,false);
})()`, aetoxCursorJS(), x, y)
}

// cursorMoveScript travels to a point, and presses on arrival when asked.
func cursorMoveScript(x, y float64, travel time.Duration, press bool) string {
	return fmt.Sprintf(`(function(){%s
  aetoxCursorTo(%g,%g,%d,%t);
})()`, aetoxCursorJS(), x, y, travel.Milliseconds(), press)
}

// cursorDragScript sweeps from one point to another, drawing the trail.
func cursorDragScript(from, to point, sweep time.Duration) string {
	return fmt.Sprintf(`(function(){%s
  aetoxCursorDrag(%g,%g,%g,%g,%d);
})()`, aetoxCursorJS(), from.X, from.Y, to.X, to.Y, sweep.Milliseconds())
}

// markCursorMove sends the sprite to a point ahead of a pointer action and
// answers with how long to wait before pressing, so the press lands when the
// arrow does. Zero when the layer is off: no sprite, no wait. Remembers the
// point either way, so the sprite comes back in the right place if the layer
// is turned on later.
func (a *App) markCursorMove(id AgentTabID, p point, press bool) time.Duration {
	var tab *browserTab
	if a.browsers != nil {
		tab = a.browsers.tab(string(id))
	}
	fx, fy, known := tab.cursor()
	tab.rememberCursor(p.X, p.Y)
	if !a.pageMarksOn() {
		return 0
	}
	travel := cursorTravel(point{fx, fy}, p, known)
	a.browserEval(string(id), cursorMoveScript(p.X, p.Y, travel, press))
	return travel
}

// markCursorDrag draws the sweep of a drag, timed to the engine's own sweep.
func (a *App) markCursorDrag(id AgentTabID, from, to point, sweep time.Duration) {
	var tab *browserTab
	if a.browsers != nil {
		tab = a.browsers.tab(string(id))
	}
	tab.rememberCursor(to.X, to.Y)
	if !a.pageMarksOn() {
		return
	}
	a.browserEval(string(id), cursorDragScript(from, to, sweep))
}

// restorePageCursor puts the sprite back after a capture took it down.
func (a *App) restorePageCursor(id AgentTabID) {
	if !a.pageMarksOn() || a.browsers == nil {
		return
	}
	if x, y, ok := a.browsers.tab(string(id)).cursor(); ok {
		a.browserEval(string(id), cursorShowScript(x, y))
	}
}

// pageMarksOn is the switch, read at the moment of drawing rather than held.
//
// a.cfg is the config a new chat is born with, and SetBusyLayer writes every
// live conversation and this field together, so all of them agree and any one
// of them is the answer. This one is chosen because a tool call does not know
// which conversation it belongs to, and the honest reading of the setting is
// that it is a fact about the window rather than about a chat.
func (a *App) pageMarksOn() bool { return !a.cfg.BusyPageMarksOff }

// markPageClick and markPageScroll are the two doors, and both are no-ops when
// the layer is off — the caller does not check, so a caller cannot forget to.
//
// Fire and forget: browserEval queues the script on the thread that owns
// webviews and does not wait for it, so a page that is busy, gone, or refusing
// to run scripts costs the action nothing. A mark is a courtesy, and a courtesy
// must never be something a tool call can fail on.
func (a *App) markPageClick(id AgentTabID, p point) {
	if !a.pageMarksOn() {
		return
	}
	a.browserEval(string(id), markRippleScript(p.X, p.Y))
}

func (a *App) markPageScroll(id AgentTabID, to string, holdMS int) {
	if !a.pageMarksOn() {
		return
	}
	a.browserEval(string(id), markScrollScript(strings.ToLower(strings.TrimSpace(to)), holdMS))
}

// clearPageMarks runs whether or not the layer is on, and that is deliberate.
// The switch decides whether a mark is DRAWN; one already on the page has to
// come off regardless, or turning the layer off mid-run would leave the last
// mark sitting there for the rest of that page's life.
//
// It WAITS, unlike the two that draw. browserActOn bounds that wait at two
// seconds and treats silence as a third outcome rather than a failure — a page
// that is busy, gone, or refusing to run scripts simply never answers, and the
// photograph is still taken. Every return value is dropped on purpose: there is
// nothing a caller could do differently with any of them, and a capture that
// refused because a mark could not be confirmed gone would be a courtesy that
// had grown into a gate.
func (a *App) clearPageMarks(id AgentTabID) {
	_, _, _ = a.browserActOn(string(id), func(token string) string {
		return clearMarksScript(token)
	})
}
