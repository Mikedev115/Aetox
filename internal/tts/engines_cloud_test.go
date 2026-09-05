package tts

// The cloud vendors, tested without a cloud: gTTS through its parser and a
// swapped runner, Edge against a fake Read Aloud service that demands what
// the real one demands, the API pair against httptest servers reached through
// the same per-provider base-URL override a user would set. TestEdgeLive is
// the one exception, and it asks before it dials.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Mikedev115/Aetox/internal/config"
)

func isolateCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ELEVENLABS_API_KEY", "")
	t.Setenv("ELEVEN_API_KEY", "")
}

func saveBaseURL(t *testing.T, provider, url string) {
	t.Helper()
	pref := config.ModelPreference{}
	pref.SetBaseURLForProvider(provider, url)
	if err := config.SaveModelPreference(pref); err != nil {
		t.Fatal(err)
	}
}

// fakeEdge is a Read Aloud service on localhost: the voice list at
// /voices/list, the socket at /edge/v1, both refusing a request without the
// tokens the real one demands. refuseFirstDial makes the first handshake a
// 403 carrying the server's clock — how the real service answers a client
// whose clock is off — and silent makes a turn come back with no audio, the
// way th-TH-PremwadeeNeural did on 2026-09-05.
type fakeEdge struct {
	srv *httptest.Server

	mu              sync.Mutex
	config, ssml    string
	dials           int
	refuseFirstDial bool
	serverClock     time.Time // zero = the machine's own
	silent          bool
}

func (f *fakeEdge) now() time.Time {
	if f.serverClock.IsZero() {
		return time.Now()
	}
	return f.serverClock
}

// tokenIsCurrent accepts the token for the server's current five-minute
// bucket or the one before it, so a test straddling a boundary still passes.
func (f *fakeEdge) tokenIsCurrent(token string) bool {
	now := f.now()
	return token == edgeTokenAt(now.Unix()) || token == edgeTokenAt(now.Unix()-300)
}

func newFakeEdge(t *testing.T) *fakeEdge {
	t.Helper()
	f := &fakeEdge{}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch r.URL.Path {
		case "/voices/list":
			if q.Get("trustedclienttoken") != edgeTrustedToken || !f.tokenIsCurrent(q.Get("Sec-MS-GEC")) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`[{"ShortName":"th-TH-NiwatNeural","Locale":"th-TH","Gender":"Male"},` +
				`{"ShortName":"th-TH-PremwadeeNeural","Locale":"th-TH","Gender":"Female"},{"Name":"no short name"}]`))
		case "/edge/v1":
			f.mu.Lock()
			f.dials++
			refuse := f.refuseFirstDial && f.dials == 1
			f.mu.Unlock()
			if refuse || q.Get("TrustedClientToken") != edgeTrustedToken || q.Get("ConnectionId") == "" || !f.tokenIsCurrent(q.Get("Sec-MS-GEC")) {
				w.Header().Set("Date", f.now().UTC().Format(http.TimeFormat))
				w.WriteHeader(http.StatusForbidden)
				return
			}
			c, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer c.Close()
			_, m1, err := c.ReadMessage()
			if err != nil {
				return
			}
			_, m2, err := c.ReadMessage()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.config, f.ssml = string(m1), string(m2)
			silent := f.silent
			f.mu.Unlock()
			text := func(path string) []byte {
				return []byte("X-RequestId:abc\r\nContent-Type:application/json; charset=utf-8\r\nPath:" + path + "\r\n\r\n{}")
			}
			binary := func(headers string, data string) []byte {
				h := []byte(headers)
				frame := append([]byte{byte(len(h) >> 8), byte(len(h))}, h...)
				return append(frame, data...)
			}
			_ = c.WriteMessage(websocket.TextMessage, text("turn.start"))
			_ = c.WriteMessage(websocket.TextMessage, text("response"))
			if !silent {
				for _, part := range []string{"MP3-", "DATA"} {
					_ = c.WriteMessage(websocket.BinaryMessage, binary("X-RequestId:abc\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n", part))
				}
			}
			// The real stream's last frame: headers only, no Content-Type.
			_ = c.WriteMessage(websocket.BinaryMessage, binary("X-RequestId:abc\r\nPath:audio\r\n", ""))
			_ = c.WriteMessage(websocket.TextMessage, text("turn.end"))
		default:
			http.NotFound(w, r)
		}
	}))
	oldSocket, oldVoices := edgeSocketURL, edgeVoicesURL
	edgeSocketURL = "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/edge/v1"
	edgeVoicesURL = f.srv.URL + "/voices/list"
	edgeClockSkew.Store(0)
	t.Cleanup(func() {
		edgeSocketURL, edgeVoicesURL = oldSocket, oldVoices
		edgeClockSkew.Store(0)
		f.srv.Close()
	})
	return f
}

// The protocol, end to end against the fake: the voice list read as JSON, a
// turn made of the config message and the SSML message, the audio frames
// glued into the file, the text escaped and the voice named the long way.
func TestEdgeSpeaksTheReadAloudProtocol(t *testing.T) {
	f := newFakeEdge(t)
	eng, err := New(Options{Engine: "edge", Voice: "th-TH-PremwadeeNeural"})
	if err != nil {
		t.Fatal(err)
	}
	voices, err := eng.Voices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(voices) != 2 || voices[0].ID != "th-TH-NiwatNeural" || voices[0].Lang != "th-TH" || voices[0].Gender != "Male" {
		t.Errorf("voices = %+v", voices)
	}
	out := filepath.Join(t.TempDir(), "piece.mp3")
	if err := eng.Synthesize(context.Background(), "สวัสดี <b> & ครับ", out); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(out); err != nil || string(body) != "MP3-DATA" {
		t.Errorf("file holds %q (%v), want the audio frames glued together", body, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !strings.Contains(f.config, "Path:speech.config") || !strings.Contains(f.config, edgeOutputFormat) {
		t.Errorf("config message = %q", f.config)
	}
	for _, want := range []string{"Path:ssml", "X-RequestId:", "Microsoft Server Speech Text to Speech Voice (th-TH, PremwadeeNeural)", "สวัสดี &lt;b&gt; &amp; ครับ"} {
		if !strings.Contains(f.ssml, want) {
			t.Errorf("SSML message lacks %q:\n%s", want, f.ssml)
		}
	}
}

func TestEdgeATurnWithNoAudioIsAnError(t *testing.T) {
	f := newFakeEdge(t)
	f.silent = true
	eng, _ := New(Options{Engine: "edge", Voice: "th-TH-PremwadeeNeural"})
	err := eng.Synthesize(context.Background(), "สวัสดี", filepath.Join(t.TempDir(), "p.mp3"))
	if err == nil || !strings.Contains(err.Error(), "ไม่ส่งเสียง") || !strings.Contains(err.Error(), "th-TH-PremwadeeNeural") {
		t.Errorf("a silent turn gave %v, want an error naming the voice", err)
	}
}

// A PC whose clock is off is refused with the server's time in the refusal;
// the client learns the difference and the second dial goes through.
func TestEdgeLearnsTheClockFromA403(t *testing.T) {
	f := newFakeEdge(t)
	f.refuseFirstDial = true
	f.serverClock = time.Now().Add(time.Hour)
	eng, _ := New(Options{Engine: "edge", Voice: "th-TH-NiwatNeural"})
	if err := eng.Synthesize(context.Background(), "สวัสดี", filepath.Join(t.TempDir(), "p.mp3")); err != nil {
		t.Fatalf("second dial should have gone through: %v", err)
	}
	if f.dials != 2 {
		t.Errorf("dialed %d times, want the refusal and the retry", f.dials)
	}
	if skew := edgeClockSkew.Load(); skew < 3590 || skew > 3610 {
		t.Errorf("learned a skew of %d s, want about 3600", skew)
	}
}

// The clock token, pinned to edge-tts 7.2.8's Python for a fixed instant
// (computed with the installed package on 2026-09-06). Two instants in the
// same five-minute bucket give the same token.
func TestEdgeClockTokenMatchesThePythonOriginal(t *testing.T) {
	edgeClockSkew.Store(0)
	const want = "FFEA56EAF25EACF7FB329478EB0B4E3344D87E6EFF31C04132DC3F8E20A5C57F"
	for _, unix := range []int64{1757100000, 1757100299} {
		if got := edgeSecMSGEC(time.Unix(unix, 0)); got != want {
			t.Errorf("token at %d = %s, want %s", unix, got, want)
		}
	}
}

func TestEdgeVoiceNameIsTheLongForm(t *testing.T) {
	cases := map[string]string{
		"th-TH-PremwadeeNeural":        "Microsoft Server Speech Text to Speech Voice (th-TH, PremwadeeNeural)",
		"zh-CN-liaoning-XiaobeiNeural": "Microsoft Server Speech Text to Speech Voice (zh-CN-liaoning, XiaobeiNeural)",
		"":                             "Microsoft Server Speech Text to Speech Voice (en-US, EmmaMultilingualNeural)",
		"Microsoft Server Speech Text to Speech Voice (cy-GB, NiaNeural)": "Microsoft Server Speech Text to Speech Voice (cy-GB, NiaNeural)",
	}
	for short, want := range cases {
		if got := (&edgeVoice{voice: short}).voiceName(); got != want {
			t.Errorf("voiceName(%q) = %q, want %q", short, got, want)
		}
	}
}

// Nothing to install: the descriptor says so, and New does not go looking for
// a program.
func TestEdgeNeedsNothingInstalled(t *testing.T) {
	desc, _ := Lookup("edge")
	if len(desc.Binaries) != 0 || len(desc.InstallCommand) != 0 {
		t.Errorf("edge still asks for a program: %+v", desc)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := New(Options{Engine: "edge"}); err != nil {
		t.Errorf("edge without any program on PATH: %v", err)
	}
}

// TestEdgeLive talks to the real service — the check to run when Microsoft
// changes something, and the number behind "how fast can the first sound
// be". Skipped unless AETOX_EDGE_LIVE=1. Prints, per run: time to connect,
// to the first audio byte, and to the whole first piece of the owner's reply.
func TestEdgeLive(t *testing.T) {
	if os.Getenv("AETOX_EDGE_LIVE") != "1" {
		t.Skip("set AETOX_EDGE_LIVE=1 to talk to Microsoft")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	eng, err := New(Options{Engine: "edge", Voice: "th-TH-NiwatNeural"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	voices, err := eng.Voices(ctx)
	if err != nil {
		t.Fatalf("voice list: %v", err)
	}
	t.Logf("voices: %d in %d ms", len(voices), time.Since(start).Milliseconds())

	var marks []string
	edgeTrace = func(event string) {
		marks = append(marks, fmt.Sprintf("%s=%dms", event, time.Since(start).Milliseconds()))
	}
	defer func() { edgeTrace = nil }()
	piece := "ได้ครับ ผมจะทำไฟล์ตัวอย่างแล้วเปิดให้ดูในเบราว์เซอร์ โดยจะแสดงพื้นฐานของเว็บสามส่วน เช่น"
	for run := 1; run <= 3; run++ {
		out := filepath.Join(t.TempDir(), fmt.Sprintf("run%d.mp3", run))
		start, marks = time.Now(), nil
		if err := eng.Synthesize(ctx, piece, out); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		info, _ := os.Stat(out)
		t.Logf("run %d: %s total=%dms bytes=%d (~%.1f s of audio)", run, strings.Join(marks, " "), time.Since(start).Milliseconds(), info.Size(), float64(info.Size())*8/48000)
	}
}

func TestParseGTTSLanguages(t *testing.T) {
	raw := "  af: Afrikaans\n  th: Thai\n  en: English\nnot a language line\n"
	got := parseGTTSLanguages(raw)
	if len(got) != 3 {
		t.Fatalf("expected 3 languages, got %d: %+v", len(got), got)
	}
	// Sorted by code, and the code doubles as the Lang the locale-default
	// policy matches on.
	if got[2].ID != "th" || got[2].Lang != "th" || !strings.Contains(got[2].Name, "Thai") {
		t.Errorf("thai row parsed wrong: %+v", got[2])
	}
}

func TestOpenAISpeechWithoutAKeyNamesTheFix(t *testing.T) {
	isolateCredentials(t)
	if _, err := New(Options{Engine: "openai"}); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Errorf("no key on the official host must name the fix, got: %v", err)
	}
}

func TestOpenAISpeechSpeaksThroughTheBaseURLOverride(t *testing.T) {
	isolateCredentials(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}
		// A keyless local clone: no Authorization header expected.
		if r.Header.Get("Authorization") != "" {
			t.Error("keyless base-URL override must not send a bearer header")
		}
		_, _ = w.Write([]byte("RIFFfake-wav"))
	}))
	defer server.Close()
	saveBaseURL(t, "openai", server.URL)

	engine, err := New(Options{Engine: "openai", Voice: "nova"})
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "out.wav")
	if err := engine.Synthesize(context.Background(), "สวัสดี", outPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil || string(data) != "RIFFfake-wav" {
		t.Errorf("audio not written through: %q err=%v", data, err)
	}
	if engine.Mime() != "audio/wav" {
		t.Errorf("openai mime = %q", engine.Mime())
	}
}

func TestElevenLabsListsAndSpeaksWithTheAccountsVoices(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("ELEVENLABS_API_KEY", "xi-test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "xi-test-key" {
			t.Errorf("key header missing on %s", r.URL.Path)
		}
		switch {
		case r.URL.Path == "/voices":
			_, _ = w.Write([]byte(`{"voices":[{"voice_id":"v1","name":"Rachel","labels":{"gender":"female"}}]}`))
		case strings.HasPrefix(r.URL.Path, "/text-to-speech/v1"):
			_, _ = w.Write([]byte("fake-mp3"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	saveBaseURL(t, "elevenlabs", server.URL)

	engine, err := New(Options{Engine: "elevenlabs"})
	if err != nil {
		t.Fatal(err)
	}
	voices, err := engine.Voices(context.Background())
	if err != nil || len(voices) != 1 || voices[0].Name != "Rachel" {
		t.Fatalf("voices = %+v err=%v", voices, err)
	}
	// No voice pinned: the account's first voice is the engine's default.
	outPath := filepath.Join(t.TempDir(), "out.mp3")
	if err := engine.Synthesize(context.Background(), "hello", outPath); err != nil {
		t.Fatal(err)
	}
	if engine.Mime() != "audio/mpeg" {
		t.Errorf("elevenlabs mime = %q", engine.Mime())
	}
}

func TestElevenLabsWithoutAKeyNamesTheFix(t *testing.T) {
	isolateCredentials(t)
	if _, err := New(Options{Engine: "elevenlabs"}); err == nil || !strings.Contains(err.Error(), "ELEVENLABS_API_KEY") {
		t.Errorf("no key must name the env var, got: %v", err)
	}
}

func TestGroqSpeechUsesItsOwnRosterAndModel(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("GROQ_API_KEY", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "playai-tts" || body["voice"] != "Fritz-PlayAI" {
			t.Errorf("groq payload wrong: %+v", body)
		}
		_, _ = w.Write([]byte("RIFFgroq"))
	}))
	defer server.Close()
	saveBaseURL(t, "groq", server.URL)

	engine, err := New(Options{Engine: "groq"})
	if err != nil {
		t.Fatal(err)
	}
	voices, _ := engine.Voices(context.Background())
	if len(voices) == 0 || voices[0].Lang != "en" {
		t.Errorf("playai voices must be marked English-only: %+v", voices[:1])
	}
	outPath := filepath.Join(t.TempDir(), "out.wav")
	if err := engine.Synthesize(context.Background(), "hello", outPath); err != nil {
		t.Fatal(err)
	}
}

func TestGeminiSpeechWrapsThePCMItIsHanded(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("GEMINI_API_KEY", "g-test-key")
	pcm := []byte{1, 2, 3, 4}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "gemini-2.5-flash-preview-tts") {
			t.Errorf("wrong model path: %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "g-test-key" {
			t.Error("gemini key header missing")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		raw, _ := json.Marshal(body)
		if !strings.Contains(string(raw), "AUDIO") || !strings.Contains(string(raw), "Kore") {
			t.Errorf("payload missing modality or default voice: %s", raw)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"audio/L16;codec=pcm;rate=24000","data":"` +
			base64.StdEncoding.EncodeToString(pcm) + `"}}]}}]}`))
	}))
	defer server.Close()
	oldBase := geminiBase
	geminiBase = server.URL
	defer func() { geminiBase = oldBase }()

	engine, err := New(Options{Engine: "gemini"})
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "out.wav")
	if err := engine.Synthesize(context.Background(), "สวัสดี", outPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:4]) != "RIFF" || len(data) != 44+len(pcm) {
		t.Errorf("not a WAV wrap of the PCM: %d bytes, head %q", len(data), data[:4])
	}
}

func TestPcmRateReadsTheMime(t *testing.T) {
	if got := pcmRate("audio/L16;codec=pcm;rate=16000"); got != 16000 {
		t.Errorf("rate = %d", got)
	}
	if got := pcmRate("audio/L16"); got != 24000 {
		t.Errorf("missing rate must fall back to the documented 24000, got %d", got)
	}
}
