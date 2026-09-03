package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/tts"
)

// The whole path, with nothing faked: the real speech engine on this machine,
// the real asset-server middleware, and the real bytes the audio element would
// fetch. Every other test here swaps the vendor out — this is the one that
// would notice if SAPI wrote a file the URL host then served as something the
// player cannot open.
//
// Opt-in (AETOX_VOICE_E2E=1) because it runs a real synthesizer: ~25 seconds
// and a PowerShell launch per piece, which is not something to put on every
// `go test ./desktop/`. It also needs a Thai voice installed, since the text
// it reads is Thai — an English voice produces near-silence for it in about a
// tenth of the time, which would let the test pass while measuring nothing.
func TestSpeechEndToEndThroughTheRealEngine(t *testing.T) {
	if os.Getenv("AETOX_VOICE_E2E") == "" {
		t.Skip("set AETOX_VOICE_E2E=1 to run against the machine's real speech engine")
	}
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())

	probe, err := tts.New(tts.Options{})
	if err != nil {
		t.Skip(err)
	}
	voices, err := probe.Voices(t.Context())
	if err != nil {
		t.Skip(err)
	}
	thai := ""
	for _, v := range voices {
		if strings.HasPrefix(strings.ToLower(v.Lang), "th") {
			thai = v.ID
			break
		}
	}
	if thai == "" {
		t.Skip("no Thai voice installed — the Thai test text would measure silence")
	}
	t.Logf("อ่านด้วยเสียง %s", thai)

	app := seed(&App{cfg: config.Config{SandboxRoot: t.TempDir(), TTSVoice: thai}}, newConversation())
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
			events <- c
		}
	}
	handler := app.assetMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	// Every sentence different, and that is not decoration. A repeated one
	// hashes to the same cache entry, so the second piece onward would be a
	// cache hit and the run would report the cache's speed as the pipeline's.
	// (It did, the first time this test was written: 10 pieces in 523ms.)
	var sb strings.Builder
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&sb, "ประโยคที่ %d ของการทดสอบเสียงอ่าน ผมกำลังดูว่าเสียงแรกออกเร็วแค่ไหน และชิ้นถัดไปมาทันก่อนชิ้นนี้จะพูดจบหรือเปล่า ", i)
	}
	text := sb.String()
	start := time.Now()
	job, err := app.StartSpeech(text)
	if err != nil {
		t.Fatal(err)
	}

	seen, bytesServed := 0, 0
	var audio, shortest = 0.0, math.MaxFloat64
	var firstAt, gap time.Duration
	last := time.Now()
	for {
		var c speechChunkEvent
		select {
		case c = <-events:
		case <-time.After(2 * time.Minute):
			t.Fatalf("the read stalled after %d pieces", seen)
		}
		if c.Error != "" {
			t.Fatalf("piece %d failed: %s", c.Seq, c.Error)
		}
		if c.Seq != seen {
			t.Fatalf("got piece %d, want %d", c.Seq, seen)
		}
		if seen == 0 {
			firstAt = time.Since(start)
		} else if since := time.Since(last); since > gap {
			gap = since
		}
		last = time.Now()

		// Exactly what the audio element does with the URL it was handed.
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.URL, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s gave %d", c.URL, rec.Code)
		}
		body := rec.Body.Bytes()
		// A real engine wrote this, so the check is that it is real audio and
		// not, say, an error page or a zero-length file the player would fail
		// on silently mid-queue.
		if len(body) < 1024 {
			t.Errorf("piece %d served %d bytes — too small to be speech", c.Seq, len(body))
		}
		if !bytes.HasPrefix(body, []byte("RIFF")) || !bytes.Contains(body[:16], []byte("WAVE")) {
			t.Errorf("piece %d is not a WAV: % x", c.Seq, body[:min(16, len(body))])
		}
		bytesServed += len(body)
		secs, err := wavSeconds(body)
		if err != nil {
			t.Fatalf("piece %d: %v", c.Seq, err)
		}
		audio += secs
		if secs < shortest {
			shortest = secs
		}

		seen++
		app.SpeechPlaying(job, c.Seq)
		if c.Last {
			break
		}
	}

	if want := len(tts.Segment(text)); seen != want {
		t.Errorf("delivered %d pieces, the text has %d", seen, want)
	}
	t.Logf("เสียงแรกออกที่ %v · ครบ %d ชิ้น %.1f MB = เสียง %.0f วินาที สังเคราะห์เสร็จใน %v",
		firstAt.Round(time.Millisecond), seen, float64(bytesServed)/1e6, audio,
		time.Since(start).Round(time.Millisecond))
	t.Logf("ชิ้นสั้นสุดยาว %.1f วินาที · ช่องว่างรอชิ้นถัดไปนานสุด %v — เหลือระยะ %.1f เท่า",
		shortest, gap.Round(time.Millisecond), shortest/gap.Seconds())

	// The claim the pipeline actually rests on, and the one thing a bounded
	// look-ahead could get wrong: the next piece must be ready before the one
	// playing runs out, or the listener hears a hole instead of a sentence.
	// Measured against the SHORTEST piece, because that is the least cover any
	// gap ever gets.
	if gap.Seconds() >= shortest {
		t.Errorf("waited %v for a piece while the shortest one only covers %.1fs — playback would starve", gap, shortest)
	}

	// The promise the whole change was made for. Generous on purpose: it is a
	// guard against the first piece growing with the reply again, not a
	// benchmark of this machine.
	if firstAt > 3*time.Second {
		t.Errorf("first audio took %v — time-to-first-audio is the number this exists to hold down", firstAt)
	}

	// A finished read is cleaned up by the webview, and after that its URLs
	// must stop resolving — the files are gone.
	app.StopSpeech(job)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ttsHostPrefix+job+"/0.wav", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("a stopped read still serves its pieces (%d)", rec.Code)
	}
}

// wavSeconds reads how long a piece will play from its own header — the byte
// rate the engine wrote, not an assumption about sample rates. A gap between
// pieces only matters relative to this number.
func wavSeconds(b []byte) (float64, error) {
	if len(b) < 44 || !bytes.HasPrefix(b, []byte("RIFF")) {
		return 0, fmt.Errorf("not a WAV (%d bytes)", len(b))
	}
	// Walk the chunks rather than assuming the canonical 44-byte layout: SAPI
	// writes a fact chunk on some voices, which shifts everything after it.
	var byteRate uint32
	for i := 12; i+8 <= len(b); {
		id := string(b[i : i+4])
		size := binary.LittleEndian.Uint32(b[i+4 : i+8])
		body := i + 8
		switch {
		case id == "fmt " && body+16 <= len(b):
			byteRate = binary.LittleEndian.Uint32(b[body+8 : body+12])
		case id == "data":
			if byteRate == 0 {
				return 0, fmt.Errorf("data chunk before fmt")
			}
			n := float64(size)
			if rest := float64(len(b) - body); rest < n {
				n = rest
			}
			return n / float64(byteRate), nil
		}
		i = body + int(size) + int(size&1) // chunks are word-aligned
	}
	return 0, fmt.Errorf("no data chunk")
}
