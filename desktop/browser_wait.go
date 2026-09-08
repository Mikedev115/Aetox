package main

// Waiting for the page to catch up, and hearing what it says while you do.
//
// These two arrived together because they are the same defect seen twice: the
// page is doing something and the agent has no way to know. One is the page
// being slow, the other is the page asking a question. Both used to end as a
// timeout and a sentence about the network.
//
// ## wait
//
// `open` waits for a navigation to complete, and for a page that renders on the
// server that is the whole story. For everything built in this decade it is the
// beginning of one: the document arrives, then scripts fetch the actual content
// and put it in. An agent that reads immediately reads an empty shell, and — the
// part that makes it a real defect rather than a slow path — it reads it
// SUCCESSFULLY. There is nothing in the answer to suggest waiting; the page
// genuinely has no results in it yet, so the model concludes there are none.
//
// Waiting is therefore not a convenience. It is the difference between "no
// results" and "not yet", which nothing else in this tool can tell apart.
//
// The condition is text rather than a CSS selector on purpose. A model reliably
// knows what it expects to SEE — a name, a total, the word "Results" — and
// unreliably knows what the page calls its own divs.
//
// ## dialog
//
// `alert()`, `confirm()` and `prompt()` stop a page dead until somebody answers.
// Nobody could: the agent has no hands for a native dialog, so a page that
// raised one hung every later action until its timeout and reported that the
// page was not responding. True, and useless.
//
// The overrides below mean a dialog can no longer block anything. The page gets
// an answer immediately, from a standing policy the agent sets, and what was
// said is recorded so the next answer the agent receives can mention it.
//
// **The default is dismiss**, and that is a safety position rather than a
// convenience one: a `confirm()` sitting in front of a destructive action is the
// commonest kind there is, and answering yes by default would make the browser
// tool quietly agree to things on the user's behalf. Saying yes has to be a
// thing the agent chose, one dialog at a time.
//
// What a dialog says is quoted back and never obeyed. It is text from a page,
// which this subsystem has treated as untrusted since browser-security-2026-07-21.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

// waitScript polls for text and answers exactly once, either when it turns up or
// when its own deadline passes. The deadline lives in the page as well as in Go
// so a script whose tab is abandoned stops on its own instead of polling for the
// life of the document.
func waitScript(token, needle string, ms int) string {
	tok, _ := json.Marshal(token)
	want, _ := json.Marshal(needle)
	return fmt.Sprintf(`(function(){%s%s
  var fired=false,iv=null,deadline=Date.now()+%d;
  function done(ok){
    if(fired)return; fired=true; if(iv)clearInterval(iv);
    %s(JSON.stringify({__aetox:"wait",token:%s,url:location.href,found:ok}));
  }
  function check(){
    var body=aetoxText();
    if(body.indexOf(%s)>=0){done(true);return}
    if(Date.now()>deadline)done(false);
  }
  check();
  if(!fired)iv=setInterval(check,200);
})()`, aetoxScanJS, aetoxTextJS, ms, bridgePost, tok, want)
}

// dialogScript replaces the three blocking dialogs. Installed at document
// creation so it is in place before any page script can call one.
//
// window.__aetox_dlg is the standing policy, and living in the page is what
// makes it survive into every frame and every same-document navigation. It is
// reset on each new document, which is the right default: an accept the agent
// granted for one page should not still be granted three pages later.
func dialogScript() string {
	return `(function(){
  if(window.__aetox_dlg)return;
  window.__aetox_dlg={accept:false,text:null};
  function report(kind,msg,answer){
    try{` + bridgePost + `(JSON.stringify({__aetox:"dialog",url:location.href,dialog:kind,message:String(msg==null?"":msg).slice(0,300),answer:answer}));}catch(e){}
  }
  window.alert=function(m){report("alert",m,"ok");};
  window.confirm=function(m){var a=!!window.__aetox_dlg.accept;report("confirm",m,a?"ok":"cancel");return a;};
  window.prompt=function(m,d){
    if(!window.__aetox_dlg.accept){report("prompt",m,"cancel");return null;}
    var v=window.__aetox_dlg.text;
    if(v===null||v===undefined)v=(d===undefined?"":d);
    report("prompt",m,String(v));
    return String(v);
  };
})()`
}

// waitForText blocks until the page contains needle, or gives up.
func (a *App) waitForText(ctx context.Context, id AgentTabID, needle string, timeout time.Duration) (bool, error) {
	host := a.browsers
	if host == nil {
		return false, fmt.Errorf("no browser")
	}
	t := host.tab(string(id))
	if t == nil {
		return false, fmt.Errorf("no browser tab %q", id)
	}

	token := newMessageToken()
	ch := make(chan bool, 1)
	t.waitMu.Lock()
	t.waitCh, t.waitToken = ch, token
	t.waitMu.Unlock()

	host.onTab(string(id), func(v tabView, _ *browserTab) {
		v.eval(waitScript(token, needle, int(timeout/time.Millisecond)))
	})

	select {
	case found := <-ch:
		return found, nil
	case <-ctx.Done():
		return false, ctx.Err()
	// Slack over the script's own deadline: the page may navigate mid-wait, and
	// a script that went with its document will never answer.
	case <-time.After(timeout + 3*time.Second):
		t.waitMu.Lock()
		t.waitCh, t.waitToken = nil, ""
		t.waitMu.Unlock()
		return false, statereport.New("the page stopped answering while waiting (did it navigate?)")
	}
}

type browserWaitSkill struct{ app *App }

// How long a wait may run, and the shape of the promise it makes.
//
// waitMax was 60 seconds for as long as this tool has existed, and it was the
// wrong number for the commonest reason anybody waits on a page. Sixty seconds
// is the right ceiling for the race `wait` was written for — the document is
// here, the scripts are still fetching what goes in it — and it is far too
// short for the other thing pages do: hand a job to a server that takes
// minutes. A podcast being generated, a video rendering, a deploy running, a
// report building. The agent could see the page saying so and had no way to
// wait for it.
//
// What it did instead is the failure this whole subsystem already has a name
// for. With no wait long enough, the only remaining move is read, say "still
// going, I will wait a moment", read again — and every one of those is a full
// model round paid for learning that nothing has happened yet. The comment
// above shell_output (internal/skill/shell_background.go) rejects a sleep tool
// in exactly those words; it did not occur to anyone that the browser was
// being forced into the polling loop anyway, by a constant.
//
// 600 seconds, matching shell's maxWaitTimeout, because they are the same
// promise: the model names a condition and the machine waits for it, bounded,
// with the Stop button cutting through. Two ceilings had to move together —
// internal/turn.toolCallDeadline held the turn to 60 seconds regardless, so
// raising this one alone would have changed nothing. That they were both 60
// is what hid the problem: the call died at its deadline and the timeout
// looked like the tool's own.
//
// waitDefault stays at 10 seconds. A caller that names no duration is waiting
// on the load race, which is what the old default was right about.
const (
	waitDefault = 10 * time.Second
	waitMax     = 600 * time.Second
)

// waitProgressEvery is how often a wait says where it has got to.
//
// A ten-minute wait with a still screen is indistinguishable from a hung one,
// and a user who cannot tell the difference presses Stop — which would make
// the raised ceiling worth nothing. So the wait reports, and the workbench's
// busy bar reads the report (frontend/src/lib/stores/busySignal.svelte.ts).
//
// Five seconds because the bar is a line of text rather than a spinner: it has
// to change often enough to read as alive and rarely enough not to flicker.
// Nothing is emitted for a wait that is over before the first tick, which is
// nearly all of them.
const waitProgressEvery = 5 * time.Second

func (s *browserWaitSkill) wait(ctx context.Context, text string, seconds int) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "browser_wait", Command: "browser wait " + text}
	a := s.app

	text = strings.TrimSpace(text)
	if text == "" {
		err := fmt.Errorf("wait needs the text you expect to appear on the page")
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	timeout := waitDefault
	if seconds > 0 {
		timeout = time.Duration(seconds) * time.Second
		if timeout > waitMax {
			timeout = waitMax
		}
	}

	id, err := a.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}

	stop := a.reportWaiting(ctx, id, text, timeout)
	found, err := a.waitForText(ctx, id, text, timeout)
	stop()
	out.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}
	out.Success = true
	if found {
		out.Content = fmt.Sprintf("เจอ %q แล้วใน %.1f วินาที", text, time.Since(start).Seconds()) + a.dialogNote(id)
	} else {
		// Not an error. The page may simply not be going to say it, and that is
		// a fact about the page rather than a fault in the call — the same
		// distinction §128.4 drew between weather and a defect. What the answer
		// must not do is stay silent about the difference, because "still not
		// there after 10s" and "there are no results" lead somewhere different.
		out.Content = fmt.Sprintf("รอ %.0f วินาทีแล้วยังไม่เจอ %q บนหน้านี้ อาจยังโหลดไม่เสร็จ หรือหน้านี้ไม่มีสิ่งนั้นจริง ๆ อ่านหน้าดูก่อนตัดสิน",
			timeout.Seconds(), text) + a.dialogNote(id)
	}
	out.RawOutput = out.Content
	return out, nil
}

// reportWaiting tells the workbench where a wait has got to, until the returned
// function is called.
//
// It exists because the ceiling above moved from one minute to ten. A tool call
// that blocks for ten minutes behind a screen that never changes is a hang as
// far as anybody watching is concerned, and the reasonable thing to do about a
// hang is press Stop — so an unreported wait would have handed the ceiling back
// the moment it was raised.
//
// The report goes out as a workbench event rather than as a tool event, and
// that seam is the decision worth recording. `internal/turn` has a vocabulary
// of three things a tool call can say — it was called, it produced a result,
// it failed — and none of them is "still going". Adding a fourth would put a
// browser's pacing problem into the executor every desk shares. The busy bar,
// meanwhile, is already a live reader of what the browser is doing and already
// draws a subject line per action; this is that line, kept true for longer.
//
// Nothing is emitted before the first tick. A wait that ends in under five
// seconds is the load race this tool was built for, and it should come and go
// without the bar saying anything.
func (a *App) reportWaiting(ctx context.Context, id AgentTabID, text string, timeout time.Duration) func() {
	done := make(chan struct{})
	go func() {
		tick := time.NewTicker(waitProgressEvery)
		defer tick.Stop()
		start := time.Now()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				a.emitEvent("browser:waiting", map[string]any{
					"tab":  string(id),
					"text": text,
					// Whole seconds both. The bar renders them as "47/600 วิ",
					// and a fraction there is noise in a number nobody is
					// measuring with.
					"elapsed": int(time.Since(start).Seconds()),
					"total":   int(timeout.Seconds()),
				})
			}
		}
	}()
	// Idempotent: the caller closes it once on the normal path, and a second
	// close would panic on a path somebody adds later.
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			// One last event with the light off, so the bar stops saying a
			// number that has stopped moving. The frontend reads elapsed<0 as
			// "this wait is over" rather than needing a second event name.
			a.emitEvent("browser:waiting", map[string]any{"tab": string(id), "text": text, "elapsed": -1, "total": int(timeout.Seconds())})
		})
	}
}

type browserDialogSkill struct{ app *App }

// dialog sets what the page's next alert/confirm/prompt will be answered with,
// and reports anything already said.
func (s *browserDialogSkill) dialog(accept bool, text string) (skill.Output, error) {
	out := skill.Output{Name: "browser_dialog", Command: fmt.Sprintf("browser dialog accept=%v", accept)}
	a := s.app

	id, err := a.agentTab()
	if err != nil {
		out.Content, out.Stderr = err.Error(), err.Error()
		return out, err
	}

	answer, _ := json.Marshal(text)
	js := fmt.Sprintf("window.__aetox_dlg={accept:%v,text:%s}", accept, answer)
	if text == "" {
		js = fmt.Sprintf("window.__aetox_dlg={accept:%v,text:null}", accept)
	}
	a.onTab(string(id), func(v tabView, _ *browserTab) { v.eval(js) })

	out.Success = true
	if accept {
		out.Content = "กล่องข้อความถัดไปบนหน้านี้จะถูกตอบว่าตกลง"
		if text != "" {
			out.Content += fmt.Sprintf(" และ prompt จะกรอกว่า %q", text)
		}
	} else {
		out.Content = "กล่องข้อความถัดไปบนหน้านี้จะถูกตอบว่ายกเลิก"
	}
	out.Content += " ตั้งค่านี้อยู่กับหน้านี้ เปลี่ยนหน้าแล้วกลับเป็นยกเลิกเหมือนเดิม"
	out.Content += a.dialogNote(id)
	out.RawOutput = out.Content
	return out, nil
}
