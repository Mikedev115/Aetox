package main

// A local model has to be read off the disk before it can answer, and on a
// 20B model on a laptop that is thirty seconds of nothing.
//
// Owner, 6 ก.ย., with a screenshot of another app: "อยากให้มีแบบนี้ด้วย ตอน
// โหลดโมเดลผ่าน Ollama หรือ LM studio". The screenshot showed a percentage. A
// percentage is the one thing neither runtime will give: Ollama answers
// /api/ps and LM Studio /api/v0/models with a model that is resident or a
// model that is not, the chat request itself simply blocks until the weights
// are in, and no endpoint on either counts the bytes on their way into memory.
//
// So what is reported here is the state and the clock, not an invented bar —
// the same rule the context meter follows when it refuses to draw a "free"
// slice for a window nobody has measured. "กำลังโหลดโมเดล... 12 วินาที" is a
// true sentence that answers the question the spinner is being asked (is this
// thing stuck, or is it working), and a bar filling at a rate this process
// made up would not be.
//
// The poll costs one request to localhost per 700ms, only while a local
// provider is answering, and only until the model shows up resident.

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

// ModelLoading is what the live row is drawn from.
//
// Loading, not a non-empty Model, is what says a wait is on: a session that has
// not pinned a model name waits exactly as long as one that has, and reading
// the flag off the name would have left that wait invisible.
type ModelLoading struct {
	Loading  bool   `json:"loading"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// Seconds since the wait began — the only number here that is measured.
	Secs int `json:"secs"`
}

// How often the runtime is asked, and how long a wait must last before it is
// worth telling anyone about. A model already in memory answers the first
// question instantly and the row is never drawn; the grace covers the small
// load that would otherwise flash a spinner for a third of a second.
// Vars, not consts, so the test can run the whole wait in a few milliseconds
// instead of sleeping through a grace period written for human eyes.
var (
	modelLoadPoll  = 700 * time.Millisecond
	modelLoadGrace = 1200 * time.Millisecond
)

// runsWeightsLocally reports whether this provider brings a model into THIS
// machine's memory to answer.
//
// Two callers, one question, and they use the answer in opposite directions:
// the load row exists only for these (§the block above), and the queued-switch
// preflight (§232) exists only for everyone else — pinging a local runtime to
// prove the next model would pull it into the same VRAM the model that is
// still answering is sitting in, which can evict or OOM the turn the user is
// waiting on. One spelling, so the two can never disagree about which
// providers those are.
func runsWeightsLocally(canonical string) bool {
	return canonical == "ollama" || canonical == "lmstudio"
}

// watchModelLoad reports, for as long as it takes, that this turn is waiting on
// a local runtime to bring its model into memory. The returned func ends the
// watch and clears the row; it is safe to call more than once, and every caller
// must call it — the first token arriving is what ends most waits, not the
// model going resident, because a runtime lists a model as loaded a beat after
// it starts answering.
func (a *App) watchModelLoad(ctx context.Context, conv *conversation) func() {
	canonical := model.NormalizeProvider(conv.cfg.ModelProvider)
	// Only the two runtimes that run the weights on this machine. Everyone
	// else's model is already in memory somewhere else, and a wait on them is
	// the network's, which this row would misname.
	if !runsWeightsLocally(canonical) {
		return func() {}
	}
	base := strings.TrimSpace(conv.cfg.ModelBaseURL)
	if base == "" {
		base = resolveBaseURLForProvider(canonical)
	}
	if base == "" {
		return func() {}
	}
	name := strings.TrimSpace(conv.cfg.ModelName)
	key := resolveAPIKeyForProvider(canonical)

	done := make(chan struct{})
	var once sync.Once
	stop := func() { once.Do(func() { close(done) }) }

	go func() {
		started := time.Now()
		announced := false
		defer func() {
			if announced {
				a.emitEvent("model:loading", sessionEvent[ModelLoading]{SessionID: conv.id})
			}
		}()
		for {
			// Resident is the end of the wait however it is reached — the model
			// was already in memory when the turn started, or it has just
			// finished arriving.
			if model.LocalModelResident(canonical, base, key, name) {
				return
			}
			if !announced && time.Since(started) >= modelLoadGrace {
				announced = true
			}
			if announced {
				a.emitEvent("model:loading", sessionEvent[ModelLoading]{
					SessionID: conv.id,
					Data: ModelLoading{
						Loading:  true,
						Provider: canonical,
						Model:    name,
						Secs:     int(time.Since(started).Seconds()),
					},
				})
			}
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-time.After(modelLoadPoll):
			}
		}
	}()

	return stop
}
