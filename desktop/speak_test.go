package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/tts"
)

// fakeSpeaker is a vendor that writes the text it was given and counts the
// calls. Real engines here mean starting PowerShell, which a test about queue
// pacing has no business doing.
type fakeSpeaker struct {
	calls atomic.Int64
	mime  string
}

func (*fakeSpeaker) ID() string { return "fake" }

func (f *fakeSpeaker) Mime() string {
	if f.mime == "" {
		return "audio/wav"
	}
	return f.mime
}

func (*fakeSpeaker) Voices(context.Context) ([]tts.Voice, error) { return nil, nil }

func (f *fakeSpeaker) Synthesize(_ context.Context, text, outPath string) error {
	f.calls.Add(1)
	return os.WriteFile(outPath, []byte(text), 0o600)
}

const speakSample = "ผมกำลังทดสอบเสียงอ่านของแอปนี้ซึ่งต้องถูกแบ่งออกเป็นชิ้นก่อนสังเคราะห์ทีละชิ้นตามลำดับ "

// speakApp seeds an app whose speech goes to a fake vendor, and returns the
// channel every speech:chunk event lands on.
func speakApp(t *testing.T, eng tts.Engine) (*App, <-chan speechChunkEvent) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	prev := newTTSEngine
	newTTSEngine = func(tts.Options) (tts.Engine, error) { return eng, nil }
	t.Cleanup(func() { newTTSEngine = prev })

	app := seed(&App{cfg: config.Config{SandboxRoot: t.TempDir()}}, newConversation())
	t.Cleanup(func() {
		app.stopAllSpeech()
		if app.db != nil {
			_ = app.db.Close()
		}
	})
	events := make(chan speechChunkEvent, 512)
	app.emit = func(event string, data ...any) {
		if event != "speech:chunk" || len(data) == 0 {
			return
		}
		if c, ok := data[0].(speechChunkEvent); ok {
			select {
			case events <- c:
			default:
			}
		}
	}
	return app, events
}

func nextChunk(t *testing.T, events <-chan speechChunkEvent) speechChunkEvent {
	t.Helper()
	select {
	case c := <-events:
		return c
	case <-time.After(10 * time.Second):
		t.Fatal("no speech chunk arrived")
		return speechChunkEvent{}
	}
}

// The shape of the fix: pieces arrive one at a time, as URLs, and the first
// one is announced long before the last one exists.
func TestStartSpeechAnnouncesPiecesAsURLs(t *testing.T) {
	app, events := speakApp(t, &fakeSpeaker{})
	job, err := app.StartSpeech(strings.Repeat(speakSample, 10))
	if err != nil {
		t.Fatal(err)
	}
	first := nextChunk(t, events)
	if first.Error != "" {
		t.Fatalf("first piece failed: %s", first.Error)
	}
	if first.Job != job || first.Seq != 0 {
		t.Errorf("first event is job %q seq %d, want %q seq 0", first.Job, first.Seq, job)
	}
	if first.Last {
		t.Error("the first piece of a long reply claims to be the last")
	}
	if want := ttsHostPrefix + job + "/0"; !strings.HasPrefix(first.URL, want) {
		t.Errorf("URL is %q, want it to start %q", first.URL, want)
	}
	// The whole reason this replaced SpeakText: no audio crosses the binding.
	if strings.Contains(first.URL, "base64") || strings.HasPrefix(first.URL, "data:") {
		t.Errorf("audio is still being carried as a value: %.40q", first.URL)
	}
	if first.Mime != "audio/wav" {
		t.Errorf("Mime is %q, want the engine's own", first.Mime)
	}
}

// The listener sets the pace. A webview that plays every piece gets every
// piece, in order, ending with one marked last.
func TestReportingProgressWalksTheReadToTheEnd(t *testing.T) {
	eng := &fakeSpeaker{}
	app, events := speakApp(t, eng)
	text := strings.Repeat(speakSample, 12)
	job, err := app.StartSpeech(text)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for {
		c := nextChunk(t, events)
		if c.Error != "" {
			t.Fatalf("piece %d failed: %s", c.Seq, c.Error)
		}
		if c.Seq != seen {
			t.Fatalf("got piece %d, want %d — pieces must arrive in order", c.Seq, seen)
		}
		seen++
		app.SpeechPlaying(job, c.Seq)
		if c.Last {
			break
		}
	}
	if want := len(tts.Segment(text)); seen != want {
		t.Errorf("the read delivered %d pieces, the text has %d", seen, want)
	}
}

// The look-ahead bound, which is the reason SpeechPlaying exists at all: a
// listener still on the first piece must not have the whole reply synthesized
// behind their back.
func TestSynthesisStopsAheadOfASilentListener(t *testing.T) {
	eng := &fakeSpeaker{}
	app, events := speakApp(t, eng)
	text := strings.Repeat(speakSample, 40)
	if pieces := len(tts.Segment(text)); pieces < 10 {
		t.Fatalf("test text only makes %d pieces — too few to show a bound", pieces)
	}
	if _, err := app.StartSpeech(text); err != nil {
		t.Fatal(err)
	}
	nextChunk(t, events) // the listener starts the first piece and says nothing more
	time.Sleep(300 * time.Millisecond)
	// The window, plus the one piece parked in the reader's channel, plus the
	// pieces the reader may have in the engine at once.
	most := int64(speechLookahead + 2 + speechParallel)
	if n := eng.calls.Load(); n > most {
		t.Errorf("engine ran %d pieces for a listener still on the first, want at most %d", n, most)
	}
}

func TestStopSpeechCancelsTheReadAndTakesTheFilesWithIt(t *testing.T) {
	app, events := speakApp(t, &fakeSpeaker{})
	job, err := app.StartSpeech(strings.Repeat(speakSample, 10))
	if err != nil {
		t.Fatal(err)
	}
	first := nextChunk(t, events)
	if _, _, ok := app.speechChunkFile(job, first.Seq); !ok {
		t.Fatal("a piece that was announced cannot be served")
	}
	dir := app.speechJob(job).dir

	app.StopSpeech(job)

	if _, _, ok := app.speechChunkFile(job, first.Seq); ok {
		t.Error("a stopped job can still be served from")
	}
	if app.speechJob(job) != nil {
		t.Error("a stopped job is still in the registry")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the job folder survived the stop: %v", err)
	}
}

// One voice at a time, the way the single audio element always behaved — and
// now the first read's synthesis actually stops rather than running on unheard.
func TestASecondReadEndsTheFirst(t *testing.T) {
	app, events := speakApp(t, &fakeSpeaker{})
	first, err := app.StartSpeech(strings.Repeat(speakSample, 10))
	if err != nil {
		t.Fatal(err)
	}
	nextChunk(t, events)
	second, err := app.StartSpeech(strings.Repeat(speakSample, 10))
	if err != nil {
		t.Fatal(err)
	}
	if app.speechJob(first) != nil {
		t.Error("the first read is still registered after a second one started")
	}
	if app.speechJob(second) == nil {
		t.Error("the second read did not register")
	}
}

func TestStartSpeechRefusesAnEmptyReply(t *testing.T) {
	app, _ := speakApp(t, &fakeSpeaker{})
	if _, err := app.StartSpeech("   \n  "); err == nil {
		t.Error("an empty reply should say why it cannot be read, not start a silent job")
	}
}

// ttsHost is reachable only through the registry, so these are the whole of
// its surface: a piece that exists, and everything else.
func TestTTSHostServesOnlyRegisteredPieces(t *testing.T) {
	app, events := speakApp(t, &fakeSpeaker{mime: "audio/mpeg"})
	job, err := app.StartSpeech(strings.Repeat(speakSample, 10))
	if err != nil {
		t.Fatal(err)
	}
	first := nextChunk(t, events)

	passedThrough := false
	handler := app.assetMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		passedThrough = true
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, first.URL, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("serving an announced piece gave %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Errorf("Content-Type is %q — an MP3 engine's output must not be served as WAV", got)
	}
	if rec.Body.Len() == 0 {
		t.Error("the piece was served empty")
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("pieces must be range-served, so playback starts before the file is in hand")
	}

	misses := []string{
		ttsHostPrefix + "deadbeef/0.wav",             // no such job
		ttsHostPrefix + job + "/99.wav",              // no such piece
		ttsHostPrefix + job + "/notanumber.wav",      // not a piece at all
		ttsHostPrefix + job + "/../../../etc/passwd", // a path, which this space does not accept
		ttsHostPrefix, // nothing addressed
	}
	for _, url := range misses {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s gave %d, want 404", url, rec.Code)
		}
	}

	// The rule every middleware in this chain owes the app: anything outside
	// its own prefix leaves untouched, or the window comes up blank.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if !passedThrough || rec.Code != http.StatusTeapot {
		t.Error("a request for the app's own assets did not reach them")
	}
}
