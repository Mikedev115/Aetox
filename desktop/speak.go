package main

// The ฟัง button, as a stream rather than as one long wait.
//
// SpeakText (voice.go) is what this replaces for anything longer than a
// sentence: it synthesized the entire reply, read the whole file into memory,
// base64'd it, and handed a single data: URL across the Wails binding. Nothing
// was heard until all of that had finished, so the wait grew with the answer —
// and the string in the DOM grew with it at 4/3 the size of an uncompressed
// WAV. filehost.go's opening comment is the argument against that shape,
// written for the file panes; the reply-reading button was still doing it.
//
// Three parts meet here, and only the third is a judgement:
//
//   - internal/tts.Segment cuts the reply into speakable pieces.
//   - internal/tts.Read synthesizes them into files, a few at a time.
//   - THIS file decides how far ahead of the listener that is allowed to run,
//     and how many pieces may be in the engine at once.
//
// That last one is the policy internal/tts refuses to hold, for the same
// reason it refuses to pick a voice for the UI language: the listener lives up
// here. The webview reports which piece it has started playing (SpeechPlaying)
// and the reader is released one piece at a time against that report, so a
// forty-paragraph answer never has more than a handful of pieces of audio made
// for it in advance — speechLookahead released, plus the speechParallel the
// reader may still have in the engine — which matters on disk for the local
// engines and in money for the cloud ones.
//
// A fourth judgement joined on 2026-09-05: which voices stand in when the one
// the user picked fails a piece (speechFallbacks). The kiosk's rule again —
// its backend lists two voices per language and then another vendor — and
// policy for the same reason as the others: the list depends on the UI
// language and on what this machine has installed, neither of which
// internal/tts may read.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/tts"
)

// newTTSEngine builds the engine a read speaks with. Swapped in tests, which
// have no business starting PowerShell to find out whether a queue drains;
// production always goes to the catalog.
var newTTSEngine = tts.New

// speechLookahead is how many pieces may be synthesized past the one the
// listener is on. Two is enough that the next piece is always ready before the
// current one ends (synthesis is faster than speech on every engine here, by a
// wide margin), and small enough that pressing stop wastes at most two pieces
// of work rather than the rest of the reply.
const speechLookahead = 2

// speechParallel is how many pieces the reader may have in the engine at
// once. The idea is the owner's, from his kiosk (AI-Robot-Guide,
// AvatarManager.speak, 2026-09-05: "ลองไปศึกษาจาก ... วิธีการแบ่งโหลด"): every
// chunk's request goes out the moment the text is cut, so one slow chunk
// costs one wait instead of a wait per chunk behind it. Measured the same
// evening against edge-tts, the reason it matters here: Niwat answered a
// 214-rune piece in 1.6–3.0 s, Premwadee took 7–20 s for the same piece or
// returned nothing, and one such piece in a one-at-a-time reader is a gap in
// the reading.
//
// Three, not all of them, is the adaptation. A local engine is a process per
// piece (PowerShell for SAPI, a Python start for edge-tts), and a forty-piece
// reply fired at once would be forty processes on the listener's machine; a
// paid engine bills every piece whether or not anyone hears it, and a stop
// press after the first sentence should not have bought the whole reply.
// With speechLookahead=2 the most that is ever made ahead of the listener is
// speechLookahead + speechParallel + 1, about two minutes of audio.
const speechParallel = 3

// speechChunkEvent is one piece announced to the webview. It carries a URL,
// never bytes: the audio element fetches it from ttsHost and only the seconds
// being played are ever in memory.
type speechChunkEvent struct {
	Job   string `json:"job"`
	Seq   int    `json:"seq"`
	URL   string `json:"url"`
	Mime  string `json:"mime"`
	Last  bool   `json:"last"`
	Error string `json:"error,omitempty"`
}

// speechFile is one finished piece: where it is, and what type it is — the
// type of the engine that made THAT piece, which after a fallback is not the
// same for every piece of a read.
type speechFile struct {
	path, mime string
}

// speechJob is one press of ฟัง.
type speechJob struct {
	id     string
	dir    string
	cancel context.CancelFunc

	mu sync.Mutex
	// files is seq -> the file that piece was written to, and it is also the
	// whole of ttsHost's authorization: a URL can only ever reach a path this
	// map already holds, so there is no path to validate and nothing to
	// traverse. It is not a directory listing because a cache hit points
	// outside the job's own folder.
	files map[int]speechFile
	// played is the highest piece the webview says it has started. -1 until
	// the first one, which is what gives the read its head start.
	played int
	// wake is signalled on every progress report. Buffered by one so a report
	// that arrives while the reader is not waiting is not lost.
	wake chan struct{}
	// done closes when runSpeech has returned and the reader with it — the
	// moment after which nothing will write into dir.
	done chan struct{}
}

// StartSpeech begins reading text aloud and returns the job id the webview
// listens for. It returns as soon as the engine is resolved — before any audio
// exists — because the whole point is that the caller is not waiting on
// synthesis.
//
// One job at a time, matching the UI: a second press stops the first, the way
// the single audio element always did.
func (a *App) StartSpeech(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("ไม่มีข้อความให้อ่าน")
	}
	cfg := a.cur().cfg
	voice := strings.TrimSpace(cfg.TTSVoice)
	if voice == "" {
		voice = a.defaultTTSVoice(cfg.TTSEngine, cfg.UILocale)
	}
	engineID := strings.TrimSpace(cfg.TTSEngine)
	model := strings.TrimSpace(cfg.TTSModelName)
	engine, err := newTTSEngine(tts.Options{Engine: engineID, Voice: voice, Model: model})
	if err != nil {
		return "", err
	}

	a.stopAllSpeech()

	dir, err := os.MkdirTemp("", "aetox-speak-*")
	if err != nil {
		return "", err
	}
	id, err := newSpeechID()
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &speechJob{
		id:     id,
		dir:    dir,
		cancel: cancel,
		files:  map[int]speechFile{},
		played: -1,
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	a.speakMu.Lock()
	if a.speakJobs == nil {
		a.speakJobs = map[string]*speechJob{}
	}
	a.speakJobs[id] = job
	a.speakMu.Unlock()

	// The engine's identity AND its configuration: two voices reading the same
	// sentence are two different sounds, and a key that ignored the voice
	// would hand back the wrong one from the cache.
	opts := tts.ReadOptions{
		Dir:      dir,
		CacheDir: speechCacheDir(),
		CacheKey: speechCacheKey(engineID, voice, model),
		Parallel: speechParallel,
		// Built only once a piece has failed — speechFallbacks says why that
		// matters to the first piece's timing.
		Fallbacks: func(context.Context) []tts.Fallback {
			return a.speechFallbacks(cfg.UILocale, engineID, voice, model)
		},
	}
	go a.runSpeech(ctx, job, engine, text, opts)
	return id, nil
}

// runSpeech pumps the reader into the webview, one piece at a time, holding
// the reader back to speechLookahead pieces past whatever is playing.
func (a *App) runSpeech(ctx context.Context, job *speechJob, engine tts.Engine, text string, opts tts.ReadOptions) {
	defer close(job.done)
	pieces := tts.Read(ctx, engine, text, opts)
	// The channel closing is Read's promise that no piece is still being
	// written. Draining it on every way out is what lets `done` mean the
	// folder can go.
	defer func() {
		for range pieces {
		}
	}()
	for piece := range pieces {
		if piece.Err != nil {
			a.emitEvent("speech:chunk", speechChunkEvent{Job: job.id, Seq: piece.Seq, Last: true, Error: piece.Err.Error()})
			return
		}
		if piece.Fallback > 0 {
			// The answer to "why did the voice change halfway through",
			// written where it can be found.
			debuglog.Msg("speak: piece %d of %s read by stand-in voice %d", piece.Seq, job.id, piece.Fallback)
		}
		job.mu.Lock()
		job.files[piece.Seq] = speechFile{path: piece.Path, mime: piece.Mime}
		job.mu.Unlock()
		a.emitEvent("speech:chunk", speechChunkEvent{
			Job:  job.id,
			Seq:  piece.Seq,
			URL:  fmt.Sprintf("%s%s/%d%s", ttsHostPrefix, job.id, piece.Seq, filepath.Ext(piece.Path)),
			Mime: piece.Mime,
			Last: piece.Last,
		})
		if piece.Last {
			return
		}
		// Hold here until the listener is within the window. Ranging over the
		// channel again is what releases the reader for the next piece, so not
		// reading is the throttle — there is no second mechanism to keep in
		// step with this one.
		if !job.waitFor(ctx, piece.Seq-speechLookahead) {
			return
		}
	}
}

// waitFor blocks until the webview reports it has started piece `seq`, or the
// job is cancelled. A seq below zero is already satisfied — that is the head
// start the first speechLookahead pieces get.
func (job *speechJob) waitFor(ctx context.Context, seq int) bool {
	for {
		job.mu.Lock()
		caughtUp := job.played >= seq
		job.mu.Unlock()
		if caughtUp {
			return true
		}
		select {
		case <-job.wake:
		case <-ctx.Done():
			return false
		}
	}
}

// SpeechPlaying is the webview saying it has started playing a piece. It is
// what releases the next one, so it must be called for every piece — a webview
// that stops calling it stops the synthesizer within speechLookahead +
// speechParallel pieces, which is the intended behaviour and not a stall.
func (a *App) SpeechPlaying(jobID string, seq int) {
	job := a.speechJob(jobID)
	if job == nil {
		return
	}
	job.mu.Lock()
	if seq > job.played {
		job.played = seq
	}
	job.mu.Unlock()
	select {
	case job.wake <- struct{}{}:
	default: // a wake already pending does the same work
	}
}

// StopSpeech ends a read: cancels the synthesis in flight and deletes the
// pieces. The webview calls it both when the user presses stop and when the
// last piece finishes playing, because "the audio is over" and "the files can
// go" are the same moment and only the webview knows it.
//
// Cancelling actually stops work now. The old SpeakText could not: its context
// was cancelled by its own return, so a stop press left the engine synthesizing
// a reply nobody would hear.
func (a *App) StopSpeech(jobID string) {
	a.speakMu.Lock()
	job := a.speakJobs[jobID]
	delete(a.speakJobs, jobID)
	a.speakMu.Unlock()
	closeSpeechJob(job)
}

// stopAllSpeech ends every read in flight. Called when a new one starts, and
// at shutdown — a job holds a temp folder, and the process exiting is not a
// reason to leave it behind.
func (a *App) stopAllSpeech() {
	a.speakMu.Lock()
	jobs := make([]*speechJob, 0, len(a.speakJobs))
	for id, job := range a.speakJobs {
		jobs = append(jobs, job)
		delete(a.speakJobs, id)
	}
	a.speakMu.Unlock()
	for _, job := range jobs {
		closeSpeechJob(job)
	}
}

func closeSpeechJob(job *speechJob) {
	if job == nil {
		return
	}
	job.cancel()
	// Deleting the folder waits for the work to have stopped: a piece still
	// being written into a folder being removed leaves the folder behind.
	// Bounded, because an engine that ignores its context is PieceTimeout's
	// problem and must not become the stop button's.
	select {
	case <-job.done:
	case <-time.After(2 * time.Second):
	}
	// Only the job's own folder. A cached piece lives in the shared cache and
	// is the next press's head start; deleting it here would make the cache a
	// thing that never hits.
	_ = os.RemoveAll(job.dir)
}

func (a *App) speechJob(id string) *speechJob {
	a.speakMu.Lock()
	defer a.speakMu.Unlock()
	return a.speakJobs[strings.TrimSpace(id)]
}

// speechChunkFile answers ttsHost: the file for one piece of one job, and the
// type to serve it as. An unknown job or piece is simply not found — there is
// no path to check because no path was ever accepted.
func (a *App) speechChunkFile(jobID string, seq int) (string, string, bool) {
	job := a.speechJob(jobID)
	if job == nil {
		return "", "", false
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	f, ok := job.files[seq]
	return f.path, f.mime, ok
}

// speechCacheKey names one voice for the cache: the engine's identity AND its
// configuration, through the catalog so that "" and "windows" — the same
// engine asked for two ways — are one key.
func speechCacheKey(engineID, voice, model string) string {
	if desc, ok := tts.Lookup(engineID); ok {
		engineID = desc.ID
	}
	return engineID + "\x00" + voice + "\x00" + model
}

// speechFallbacks is the policy internal/tts leaves up here: which voices to
// try, in order, when the one the user picked fails a piece. The owner's kiosk
// (AI-Robot-Guide, speech_handler.synthesize_speech_stream) lists two voices
// per language and then a different vendor, and falls down the list until one
// answers. Measured against edge-tts on 2026-09-05, that list is the
// difference between a reading and an error: Premwadee returned nothing five
// times in seven, Niwat answered every time.
//
// So: one more voice of the same engine in the same language, then the local
// Windows engine, which needs no network and is on every machine this ships
// to. Same engine first because it is the same kind of voice; Windows last
// because Pattara after Premwadee is a change of speaker — and a change of
// speaker still beats a reading that stops. One same-engine voice, not all of
// them: a third voice of a vendor that just failed twice fails for the same
// reason, and every try is seconds the listener spends in silence.
//
// The list is built only once a piece has failed (ReadOptions.Fallbacks):
// finding Windows' Thai voice means starting PowerShell, and a read whose
// voice works must not pay that in front of its first piece.
func (a *App) speechFallbacks(locale, engineID, voice, model string) []tts.Fallback {
	var out []tts.Fallback
	add := func(engine, v, m string) {
		eng, err := newTTSEngine(tts.Options{Engine: engine, Voice: v, Model: m})
		if err != nil {
			return // an engine this machine cannot run is no stand-in
		}
		out = append(out, tts.Fallback{Engine: eng, CacheKey: speechCacheKey(engine, v, m)})
	}
	// The stand-in speaks the language of the voice that failed — the UI's
	// when that voice is not in the engine's own list to ask.
	lang := strings.ToLower(strings.TrimSpace(locale))
	voices, _ := a.ttsVoices(engineID)
	for _, v := range voices {
		if strings.EqualFold(v.ID, voice) && strings.TrimSpace(v.Lang) != "" {
			lang = strings.ToLower(v.Lang)
		}
	}
	for _, v := range voices {
		if strings.EqualFold(v.ID, voice) || (lang != "" && !strings.HasPrefix(strings.ToLower(v.Lang), lang)) {
			continue
		}
		add(engineID, v.ID, model)
		break
	}
	if desc, ok := tts.Lookup(engineID); !ok || desc.ID != "windows" {
		add("windows", a.defaultTTSVoice("windows", locale), "")
	}
	return out
}

// speechCacheDir is where finished pieces are kept so a second press of ฟัง on
// the same reply plays instantly. Under the data root rather than in the OS
// temp folder: temp is swept by the OS on its own schedule, which would make
// the cache hit or miss for reasons the user cannot see.
//
// An unreachable data root returns "" and internal/tts skips the cache — a
// missing cache is slower, never wrong.
func speechCacheDir() string {
	root, err := config.DataRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(root, "cache", "speech")
}

func newSpeechID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
