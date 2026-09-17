package main

// The reply's ฟัง button and the voice picker behind it — the screen's half of
// ตั้งค่า > เสียง (§248 B1). A reading is heard where the window is, so the
// synthesizer runs here, the installed voices are this machine's, and the one
// piece of judgement internal/tts refuses to hold — which voice to prefer when
// the user never picked one — is made here, with the UI locale the engine
// hands over in VoiceSettings. The preference itself is the engine's to keep
// (engine/voice.go says why, and what that costs on a remote engine).

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/tts"
)

// TTSVoiceInfo is one installed voice for the voice picker.
type TTSVoiceInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Lang   string `json:"lang"`
	Gender string `json:"gender"`
	Active bool   `json:"active"`
}

// ListTTSVoices enumerates what the active TTS engine can speak with. The
// slice is never nil; the error is the engine's own reason, verbatim, for the
// page to show above an empty list.
func (a *App) ListTTSVoices() ([]TTSVoiceInfo, error) {
	cfg := a.api.VoiceSettings()
	voices, err := a.ttsVoices(cfg.TTSEngine)
	out := []TTSVoiceInfo{}
	active := cfg.TTSVoice
	for _, v := range voices {
		out = append(out, TTSVoiceInfo{
			ID:     v.ID,
			Name:   v.Name,
			Lang:   v.Lang,
			Gender: v.Gender,
			Active: active != "" && strings.EqualFold(v.ID, active),
		})
	}
	return out, err
}

// RefreshTTSVoices forgets the screen-side voice roster and enumerates it
// again. Windows can add a language voice while Aetox is still open; without
// this door the install button would lead somewhere useful, but the picker
// would keep showing the pre-install cache until the whole app restarted.
func (a *App) RefreshTTSVoices() ([]TTSVoiceInfo, error) {
	cfg := a.api.VoiceSettings()
	desc, ok := tts.Lookup(strings.TrimSpace(cfg.TTSEngine))
	if !ok {
		return nil, fmt.Errorf("ไม่รู้จัก engine เสียงอ่านชื่อ %q", cfg.TTSEngine)
	}
	a.ttsVoiceMu.Lock()
	delete(a.ttsVoiceCache, desc.ID)
	a.ttsVoiceMu.Unlock()
	return a.ListTTSVoices()
}

// SetTTSVoice pins the voice replies are read with. Empty means "the engine
// decides", which resolves through defaultTTSVoice below. Checked against
// this machine's voices, then written down by the engine.
func (a *App) SetTTSVoice(id string) error {
	id = strings.TrimSpace(id)
	if id != "" {
		voices, err := a.ttsVoices(a.api.VoiceSettings().TTSEngine)
		if err != nil {
			return err
		}
		known := false
		for _, v := range voices {
			if strings.EqualFold(v.ID, id) {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("ไม่พบเสียงชื่อ %q ในเครื่อง", id)
		}
	}
	a.api.RememberTTSVoice(id)
	return nil
}

// TTSStatus is what the page shows above the TTS picker: "" when the engine is
// ready, otherwise its own reason it cannot run, in the user's language — the
// same contract as SpeechStatus.
func (a *App) TTSStatus() string {
	cfg := a.api.VoiceSettings()
	if _, err := tts.New(tts.Options{Engine: cfg.TTSEngine, Model: cfg.TTSModelName}); err != nil {
		return err.Error()
	}
	return ""
}

// SpeakText synthesizes a SHORT, fixed phrase in one call and hands back a
// data: URL the webview plays directly. The audio never touches the workspace
// — it is a rendering, not a deliverable.
//
// This used to read replies too, and that is what made a long answer take
// forever to start: nothing was heard until the whole thing had been
// synthesized and base64'd across the binding. Replies go through StartSpeech
// now (speak.go), which cuts them into pieces and streams them as
// URLs. What is left here is the one case the old shape is right for — ลองฟัง
// on ตั้งค่า > เสียง, one sentence of preview text, where a queue would be
// machinery around a single piece.
func (a *App) SpeakText(text string) (string, error) {
	cfg := a.api.VoiceSettings()
	voice := cfg.TTSVoice
	if voice == "" {
		voice = a.defaultTTSVoice(cfg.TTSEngine, cfg.UILocale)
	}
	engine, err := tts.New(tts.Options{
		Engine: cfg.TTSEngine,
		Voice:  voice,
		Model:  cfg.TTSModelName,
	})
	if err != nil {
		return "", err
	}
	// Bounded, because SAPI on a wedged audio driver can sit forever and the
	// button this serves has no other way home. Three minutes covers a very
	// long reply many times over at synthesis speed.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	tmpDir, err := os.MkdirTemp("", "aetox-speak-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	outPath := filepath.Join(tmpDir, "reply.audio")
	if err := engine.Synthesize(ctx, text, outPath); err != nil {
		return "", err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", err
	}
	// The engine's own MIME, not a hardcoded wav: the cloud vendors hand back
	// MP3 and the player must not be lied to about it.
	return "data:" + engine.Mime() + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// ttsVoices reads the engine's installed voices through the process cache —
// see the field comment on App.ttsVoiceCache for why a stale-until-restart
// list is the right trade here.
func (a *App) ttsVoices(engineID string) ([]tts.Voice, error) {
	desc, ok := tts.Lookup(strings.TrimSpace(engineID))
	if !ok {
		return nil, fmt.Errorf("ไม่รู้จัก engine เสียงอ่านชื่อ %q", engineID)
	}
	a.ttsVoiceMu.Lock()
	cached, hit := a.ttsVoiceCache[desc.ID]
	a.ttsVoiceMu.Unlock()
	if hit {
		return cached, nil
	}
	engine, err := newTTSEngine(tts.Options{Engine: desc.ID})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	voices, err := engine.Voices(ctx)
	if err != nil {
		return nil, err
	}
	a.ttsVoiceMu.Lock()
	if a.ttsVoiceCache == nil {
		a.ttsVoiceCache = map[string][]tts.Voice{}
	}
	a.ttsVoiceCache[desc.ID] = voices
	a.ttsVoiceMu.Unlock()
	return voices, nil
}

// defaultTTSVoice is the policy internal/tts refuses to hold: with no voice
// picked, prefer one that speaks the UI's language, so a Thai machine's first
// ฟัง press answers in Thai rather than in SAPI's English default. No match —
// or no way to enumerate — falls back to "", the engine's own default, and
// speaking with the wrong accent beats refusing to speak.
func (a *App) defaultTTSVoice(engineID, locale string) string {
	lang := strings.ToLower(strings.TrimSpace(locale))
	if lang == "" {
		return ""
	}
	voices, err := a.ttsVoices(engineID)
	if err != nil {
		return ""
	}
	for _, v := range voices {
		if strings.HasPrefix(strings.ToLower(v.Lang), lang) {
			return v.ID
		}
	}
	return ""
}
