package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/turn"
)

// The wait script has to answer exactly once and stop by itself. A poller that
// answers twice consumes the next call's token; one that never stops runs for
// the life of the document in a tab nobody is watching.
func TestWaitScriptAnswersOnceAndStopsItself(t *testing.T) {
	js := waitScript("tok-1", "Results", 5000)

	if !strings.Contains(js, "fired") || !strings.Contains(js, "clearInterval") {
		t.Error("nothing stops the poller once it has answered")
	}
	if !strings.Contains(js, `"tok-1"`) {
		t.Error("the answer carries no token, so a page could forge it")
	}
	if !strings.Contains(js, `"Results"`) {
		t.Error("the text to wait for is missing from the script")
	}
	if !strings.Contains(js, "deadline") || !strings.Contains(js, "5000") {
		t.Error("the script has no deadline of its own, so an abandoned tab polls forever")
	}
	// Checked once before the interval is armed: text already on the page must
	// not cost 200ms of nothing.
	if strings.Index(js, "check();") > strings.Index(js, "setInterval") {
		t.Error("the first check happens after the interval is set up")
	}
}

// A needle with quotes in it must not end the string literal it is embedded in
// — the page controls this text as often as the model does.
func TestWaitScriptEscapesWhatItIsLookingFor(t *testing.T) {
	js := waitScript("tok", `he said "no"`, 1000)
	if strings.Contains(js, `indexOf(he said`) {
		t.Errorf("the needle was spliced in raw:\n%s", js)
	}
	if !strings.Contains(js, `\"no\"`) {
		t.Errorf("the quotes were not escaped:\n%s", js)
	}
}

// alert/confirm/prompt stop a page dead, and nothing in this app can answer a
// native one. The overrides are what turn a hang into a fact.
func TestDialogScriptAnswersInsteadOfBlocking(t *testing.T) {
	js := dialogScript()

	for _, fn := range []string{"window.alert=", "window.confirm=", "window.prompt="} {
		if !strings.Contains(js, fn) {
			t.Errorf("%s is not replaced, so that dialog still blocks the tab", fn)
		}
	}
	if !strings.Contains(js, "__aetox_dlg={accept:false") {
		t.Error("the default is not dismiss — a confirm() guarding a deletion would be agreed to on the user's behalf")
	}
	if !strings.Contains(js, `__aetox:"dialog"`) {
		t.Error("dialogs are answered but never reported, so the agent cannot know one happened")
	}
	if !strings.Contains(js, "if(window.__aetox_dlg)return") {
		t.Error("re-installing would reset a policy the agent set for this page")
	}
	// The report is bounded: a page in a loop must not be able to post a
	// megabyte of text into the agent's next answer.
	if !strings.Contains(js, "slice(0,300)") {
		t.Error("the reported message is unbounded")
	}
}

// One dialog is reported once. Reported on every later action, it would still
// be being reported three pages later.
func TestDialogsAreReportedOnceAndBounded(t *testing.T) {
	tab := &browserTab{}
	for i := 0; i < 40; i++ {
		tab.noteDialog("- confirm(\"really?\")")
	}
	first := tab.takeDialogs()
	if len(first) == 0 {
		t.Fatal("nothing was recorded")
	}
	if len(first) > 8 {
		t.Errorf("kept %d dialogs; a page in a loop would fill the answer", len(first))
	}
	if again := tab.takeDialogs(); len(again) != 0 {
		t.Errorf("the same dialogs came back a second time: %v", again)
	}
}

func TestDialogNoteIsEmptyWhenThePageSaidNothing(t *testing.T) {
	app := &App{}
	app.browsers = &browserHost{app: app, tabs: map[string]*browserTab{"web-agent-1": {}}, views: map[string]tabView{"web-agent-1": &fakeView{}}}
	if got := app.dialogNote("web-agent-1"); got != "" {
		t.Errorf("dialogNote() = %q on a quiet page, want nothing appended", got)
	}
	if got := app.dialogNote("web-9"); got != "" {
		t.Errorf("dialogNote() on an unknown tab = %q", got)
	}
}

// Every one of the three refuses in words before it touches the engine, so a
// session with no page open is told what to do rather than waiting out a
// timeout against a webview that will never answer.
func TestTheNewActionsRefuseWithNoPageOpen(t *testing.T) {
	newApp := func() *App {
		app := &App{}
		app.browsers = &browserHost{app: app, backend: &fakeBackend{}, tabs: map[string]*browserTab{}, views: map[string]tabView{}}
		return app
	}

	if out, err := (&browserBackSkill{app: newApp()}).back(context.Background()); err == nil || out.Success {
		t.Error("back answered without a page open")
	}
	if out, err := (&browserWaitSkill{app: newApp()}).wait(context.Background(), "x", 0); err == nil || out.Success {
		t.Error("wait answered without a page open")
	}
	if out, err := (&browserDialogSkill{app: newApp()}).dialog(true, ""); err == nil || out.Success {
		t.Error("dialog answered without a page open")
	}
}

// An empty needle would match everything and return instantly, which reads as
// "it is there" for anything the model was hoping to find.
func TestWaitRefusesAnEmptyNeedle(t *testing.T) {
	app := &App{}
	app.browsers = &browserHost{app: app, backend: &fakeBackend{}, tabs: map[string]*browserTab{}, views: map[string]tabView{}}

	out, err := (&browserWaitSkill{app: app}).wait(context.Background(), "   ", 0)
	if err == nil || out.Success {
		t.Fatal("wait accepted an empty needle")
	}
	if !strings.Contains(err.Error(), "text") {
		t.Errorf("the refusal does not say what is missing: %v", err)
	}
}

// The ceiling is not decoration: a model that asks for an hour would hold the
// turn open for one.
//
// The number this test defends moved on 2026-09-08, from two minutes to ten,
// and the objection it was written around is worth keeping rather than
// deleting. It said: a long wait on a page that will never say the thing holds
// the turn open for nothing. True, and it was the right call while the only
// waits anybody had were load races.
//
// What it did not weigh is what the agent does when it CANNOT wait long enough.
// It reads, says it will wait a moment, and reads again — a full model round
// per look, on a job the page already said would take minutes, and the page
// still has to be read at the end. The short ceiling did not save the turn; it
// spent the conversation instead, which is the more expensive of the two.
//
// Three things bound the long wait, which is why it is affordable: it is
// bounded (waitMax), the Stop button cuts through it (ctx), and it reports
// every waitProgressEvery so nobody watching mistakes it for a hang. Take any
// of those away and two minutes was right.
func TestWaitIsCappedAndDefaulted(t *testing.T) {
	if waitDefault <= 0 || waitDefault > waitMax {
		t.Errorf("waitDefault = %v is not a usable default under the cap %v", waitDefault, waitMax)
	}
	if waitMax > 10*time.Minute {
		t.Errorf("waitMax = %v holds a turn open longer than any tool call may run", waitMax)
	}
	// The default is still for the load race. It is one constant away from the
	// ceiling and the two are easy to conflate — a caller that named no
	// duration is waiting on a page finishing its own load, not on a server
	// doing a job.
	if waitDefault > 30*time.Second {
		t.Errorf("waitDefault = %v is no longer a load-race default", waitDefault)
	}
}

// The two ceilings on a wait have to agree, and this test exists because for a
// year they agreed by accident.
//
// waitMax was 60 seconds and internal/turn's per-call guard was 60 seconds, so
// a wait that ran out looked like this tool's own answer and nobody had cause
// to look for a second limit. Raising one alone changes nothing: whichever is
// smaller is the real one, and what the model is told is that the page never
// said the thing — not that its wait was cut.
func TestAWaitTheToolAllowsIsAWaitTheTurnWillSitThrough(t *testing.T) {
	if waitMax > turn.MaxToolDeadline() {
		t.Fatalf("waitMax is %s but a tool call is cut at %s — a wait asked for the full %s "+
			"would be killed by the turn and reported as a page that stayed silent",
			waitMax, turn.MaxToolDeadline(), waitMax)
	}
}

// A wait long enough to be worth watching has to say so while it runs. Ten
// minutes behind a line of text that never changes is a hang as far as anybody
// watching is concerned, and the reasonable answer to a hang is Stop — which
// would hand back the ceiling the moment it was raised.
func TestALongWaitReportsOftenEnoughToReadAsAlive(t *testing.T) {
	if waitProgressEvery <= 0 || waitProgressEvery > 15*time.Second {
		t.Fatalf("waitProgressEvery is %s: too rare to read as progress", waitProgressEvery)
	}
	if ticks := waitMax / waitProgressEvery; ticks < 10 {
		t.Fatalf("a full-length wait reports only %d times", ticks)
	}
}
