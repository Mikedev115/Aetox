package engine

// The voice page's bindings, the engine's half: two vendor pickers, the
// named-model picks, and the mic. The voice picker and the reading are the
// screen's (desktop/voice_test.go).

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// A picker with no marked row reads as "not configured" — the default engine
// must come back active before anybody has picked anything.
func TestListEnginesMarkTheDefaultWhenNothingIsPicked(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	for name, rows := range map[string][]VoiceEngineInfo{
		"stt": a.ListSpeechEngines(),
		"tts": a.ListTTSEngines(),
	} {
		if len(rows) == 0 {
			t.Fatalf("%s: no engines listed", name)
		}
		active := 0
		for _, r := range rows {
			if r.Active {
				active++
			}
			if strings.TrimSpace(r.Label) == "" || strings.TrimSpace(r.Install) == "" {
				t.Errorf("%s: engine %q missing label or install hint", name, r.ID)
			}
		}
		if active != 1 {
			t.Errorf("%s: %d engines marked active, want exactly 1", name, active)
		}
	}

	for _, row := range a.ListTTSEngines() {
		if row.Active && row.ID != "edge" {
			t.Errorf("untouched TTS picker activates %q, want Microsoft Edge", row.ID)
		}
	}
}

func TestSetSpeechEnginePersistsAndRejectsUnknown(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	if err := a.SetSpeechEngine("no-such-vendor"); err == nil {
		t.Fatal("an unknown engine id must be refused")
	}
	if err := a.SetSpeechEngine("whisper-cpp"); err != nil {
		t.Fatalf("SetSpeechEngine: %v", err)
	}
	if a.cur().cfg.SpeechEngine != "whisper-cpp" {
		t.Errorf("cfg.SpeechEngine = %q", a.cur().cfg.SpeechEngine)
	}
	pref, ok, err := config.LoadModelPreference()
	if err != nil || !ok {
		t.Fatalf("preference not saved: ok=%v err=%v", ok, err)
	}
	if pref.SpeechEngine != "whisper-cpp" {
		t.Errorf("saved preference = %q — the pick would not survive a restart", pref.SpeechEngine)
	}
}

// Switching vendor clears the voice pick: a voice id is one engine's private
// naming, and the new engine has never heard of it. The pick itself arrives
// from the screen, which checked the voice exists (desktop/voice.go).
func TestSetTTSEngineClearsTheVoicePick(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	a.RememberTTSVoice("Microsoft Pattara - Thai (Thailand)")
	if got := a.VoiceSettings().TTSVoice; got != "Microsoft Pattara - Thai (Thailand)" {
		t.Fatalf("the pick was not written down: %q", got)
	}
	if err := a.SetTTSEngine("windows"); err != nil {
		t.Fatalf("SetTTSEngine: %v", err)
	}
	if got := a.cur().cfg.TTSVoice; got != "" {
		t.Errorf("voice pick survived a vendor switch: %q", got)
	}
	pref, _, _ := config.LoadModelPreference()
	if pref.TTSEngine != "windows" || pref.TTSVoice != "" {
		t.Errorf("saved preference engine=%q voice=%q", pref.TTSEngine, pref.TTSVoice)
	}
}

func TestTranscribeMicAudioRefusesAMalformedRecording(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	if _, err := a.TranscribeMicAudio("not a data url"); err == nil {
		t.Error("a payload without ;base64, must be refused")
	}
	if _, err := a.TranscribeMicAudio("data:audio/webm;base64,%%%"); err == nil {
		t.Error("invalid base64 must be refused")
	}
	if _, err := a.TranscribeMicAudio("data:audio/webm;base64,"); err == nil {
		t.Error("an empty recording must be refused")
	}
}

// The ติดตั้ง button may run only what the catalog says. An engine with no
// runnable install, an unknown engine, and an unknown side each refuse with a
// sentence — there is no path from the webview to an arbitrary command.
func TestInstallVoiceEngineRunsOnlyCatalogCommands(t *testing.T) {
	if _, err := voiceInstallArgv("tts", "windows"); err == nil {
		t.Error("windows TTS has no install command but was allowed")
	}
	if _, err := voiceInstallArgv("stt", "no-such-engine"); err == nil {
		t.Error("unknown engine was allowed")
	}
	if _, err := voiceInstallArgv("chat", "edge"); err == nil {
		t.Error("unknown side was allowed")
	}
	if _, err := voiceInstallArgv("tts", "edge"); err == nil {
		t.Error("edge needs nothing installed any more, and must say so rather than run something")
	}
	argv, err := voiceInstallArgv("tts", "gtts")
	if err != nil || strings.Join(argv, " ") != "pip install gTTS" {
		t.Errorf("gtts argv = %v, %v — must be exactly the catalog's command", argv, err)
	}
}

// Rows carry the command for the page to display next to the button. Absent
// means empty, never null (ARCHITECTURE.md §34 — the frontend does .length).
func TestEngineRowsCarryNonNilInstallCommand(t *testing.T) {
	a, _ := newSpeechTestApp(t)
	for _, rows := range [][]VoiceEngineInfo{a.ListSpeechEngines(), a.ListTTSEngines()} {
		for _, r := range rows {
			if r.InstallCommand == nil {
				t.Errorf("engine %q: InstallCommand is nil", r.ID)
			}
		}
	}
}

// The named-model pick: sticks, validates against the active vendor's roster,
// and clears on a vendor switch — a model name is one vendor's vocabulary.
func TestSetSpeechModelNameValidatesPersistsAndClears(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	if err := a.SetSpeechEngine("openai"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetSpeechModelName("no-such-model"); err == nil {
		t.Error("a model the vendor does not serve must be refused")
	}
	if err := a.SetSpeechModelName("gpt-4o-mini-transcribe"); err != nil {
		t.Fatalf("SetSpeechModelName: %v", err)
	}
	pref, _, _ := config.LoadModelPreference()
	if pref.SpeechModelName != "gpt-4o-mini-transcribe" {
		t.Errorf("saved preference = %q — the pick would not survive a restart", pref.SpeechModelName)
	}
	for _, row := range a.ListSpeechEngines() {
		if row.ID == "openai" && row.ActiveModel != "gpt-4o-mini-transcribe" {
			t.Errorf("row.ActiveModel = %q", row.ActiveModel)
		}
		if row.Models == nil {
			t.Errorf("engine %q: Models is nil", row.ID)
		}
	}
	// Switching vendor clears the pick.
	if err := a.SetSpeechEngine("whisper-cpp"); err != nil {
		t.Fatal(err)
	}
	if got := a.cur().cfg.SpeechModelName; got != "" {
		t.Errorf("model pick survived a vendor switch: %q", got)
	}
}

func TestSetTTSModelNameFollowsTheSameRules(t *testing.T) {
	a, _ := newSpeechTestApp(t)

	if err := a.SetTTSEngine("openai"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetTTSModelName("tts-1-hd"); err != nil {
		t.Fatalf("SetTTSModelName: %v", err)
	}
	if err := a.SetTTSModelName("whisper-1"); err == nil {
		t.Error("an STT model name on the TTS side must be refused")
	}
	if err := a.SetTTSEngine("windows"); err != nil {
		t.Fatal(err)
	}
	if got := a.cur().cfg.TTSModelName; got != "" {
		t.Errorf("model pick survived a vendor switch: %q", got)
	}
}
