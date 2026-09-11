package main

// The voice picker and the reading, the screen's half of ตั้งค่า > เสียง. TTS
// voice enumeration costs a real PowerShell run, so every test that needs
// voices seeds App.ttsVoiceCache instead — the cache is the seam, and it keeps
// the suite off SAPI on Windows and runnable at all everywhere else. The
// preference the pick writes is the engine's, so what these read back is what
// the engine remembers.

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/tts"
)

func seedTTSVoices(a *App, voices ...tts.Voice) {
	a.ttsVoiceCache = map[string][]tts.Voice{"windows": voices}
}

// voiceApp is a screen whose engine keeps its voice preferences in a temp
// data root of its own (newTestApp shuts the engine down with the test, so
// the store the pick opens is closed before the temp dirs go).
func voiceApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	return newTestApp(t)
}

func TestSetTTSVoiceValidatesAgainstInstalledVoices(t *testing.T) {
	a := voiceApp(t)
	seedTTSVoices(a,
		tts.Voice{ID: "Microsoft Zira Desktop - English (United States)", Lang: "en-US"},
		tts.Voice{ID: "Microsoft Pattara - Thai (Thailand)", Lang: "th-TH"},
	)

	if err := a.SetTTSVoice("Microsoft Gone - Nowhere"); err == nil {
		t.Fatal("a voice not on this machine must be refused")
	}
	if err := a.SetTTSVoice("Microsoft Pattara - Thai (Thailand)"); err != nil {
		t.Fatalf("SetTTSVoice: %v", err)
	}
	voices, err := a.ListTTSVoices()
	if err != nil {
		t.Fatalf("ListTTSVoices: %v", err)
	}
	active := 0
	for _, v := range voices {
		if v.Active {
			active++
		}
	}
	if active != 1 {
		t.Errorf("%d voices marked active, want exactly 1", active)
	}
	// Empty is a real choice — back to the engine deciding.
	if err := a.SetTTSVoice(""); err != nil {
		t.Fatalf("SetTTSVoice(\"\"): %v", err)
	}
	if got := a.api.VoiceSettings().TTSVoice; got != "" {
		t.Errorf("clearing the voice did not stick: %q", got)
	}
}

// The policy internal/tts refuses to hold: no pick + Thai UI = the Thai voice,
// and a locale with no matching voice falls back to the engine's own default
// rather than refusing to speak.
func TestDefaultTTSVoicePrefersTheUILanguage(t *testing.T) {
	a := voiceApp(t)
	seedTTSVoices(a,
		tts.Voice{ID: "Microsoft Zira Desktop - English (United States)", Lang: "en-US"},
		tts.Voice{ID: "Microsoft Pattara - Thai (Thailand)", Lang: "th-TH"},
	)

	if got := a.defaultTTSVoice("", "th"); got != "Microsoft Pattara - Thai (Thailand)" {
		t.Errorf("th locale picked %q", got)
	}
	if got := a.defaultTTSVoice("", "en"); got != "Microsoft Zira Desktop - English (United States)" {
		t.Errorf("en locale picked %q", got)
	}
	if got := a.defaultTTSVoice("", "ja"); got != "" {
		t.Errorf("an unmatched locale must fall back to the engine default, got %q", got)
	}
}

// What the engine remembers is what the screen reads back: a pick made here
// lands in the engine's preference and comes back through VoiceSettings.
func TestTheVoicePickIsTheEnginesToRemember(t *testing.T) {
	a := voiceApp(t)
	seedTTSVoices(a, tts.Voice{ID: "Microsoft Pattara - Thai (Thailand)", Lang: "th-TH"})
	if err := a.SetTTSVoice("Microsoft Pattara - Thai (Thailand)"); err != nil {
		t.Fatal(err)
	}
	if got := a.api.VoiceSettings(); got.TTSVoice != "Microsoft Pattara - Thai (Thailand)" {
		t.Errorf("VoiceSettings() = %+v, want the voice just picked", got)
	}
	var _ engine.VoiceSettings = a.api.VoiceSettings()
}
