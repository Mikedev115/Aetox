package engine

// The Settings page's voice section, the engine's half: the two vendor
// pickers and the picks they persist, and the composer's mic (speak instead of
// type), which is transcribed by the same engine `audio_transcribe` runs on —
// the host's.
//
// The reply's ฟัง button is the screen's (desktop/voice.go, desktop/speak.go):
// a reading is heard where the window is, so the synthesizer runs there. What
// it reads with is a preference, and in v1 every voice preference — STT and
// TTS alike — lives in the engine's model-preference.json and is read back
// through VoiceSettings (design doc §2 records the wart: a remote engine's
// voice pick follows the host, until screen-preferences.json in phase 5).

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/stt"
	"github.com/Mikedev115/Aetox/internal/tts"
)

// VoiceSettings is what the screen reads to speak: the read-aloud vendor,
// voice and model the user picked, and the UI language a default voice is
// chosen for when they picked none.
type VoiceSettings struct {
	TTSEngine    string `json:"ttsEngine"`
	TTSVoice     string `json:"ttsVoice"`
	TTSModelName string `json:"ttsModelName"`
	UILocale     string `json:"uiLocale"`
}

// VoiceSettings answers with the read-aloud preferences of the chat on screen.
func (a *Engine) VoiceSettings() VoiceSettings {
	cfg := a.cur().cfg
	return VoiceSettings{
		TTSEngine:    strings.TrimSpace(cfg.TTSEngine),
		TTSVoice:     strings.TrimSpace(cfg.TTSVoice),
		TTSModelName: strings.TrimSpace(cfg.TTSModelName),
		UILocale:     strings.TrimSpace(cfg.UILocale),
	}
}

// RememberTTSVoice writes the voice pick down. Whether the voice exists is the
// screen's to check first (SetTTSVoice, desktop/voice.go) — the voices are
// installed where the reading is heard. Empty means "the engine decides".
func (a *Engine) RememberTTSVoice(id string) {
	conv := a.cur()
	next := a.dialBase(conv)
	next.TTSVoice = strings.TrimSpace(id)
	a.applyConfig(conv, next)
}

// VoiceEngineInfo is one vendor row, shaped for either picker — the STT list
// and the TTS list render the same way on purpose.
type VoiceEngineInfo struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Install string `json:"install"`
	Active  bool   `json:"active"`
	// HasModels says whether this vendor reads model files the picker can
	// point at. faster-whisper stores its own weights by name, so drawing a
	// file picker for it would be a control over nothing.
	HasModels bool `json:"hasModels"`
	// InstallCommand is the catalog's runnable install, argv-shaped, for the
	// ติดตั้ง button to display — and for InstallVoiceEngine to run. Empty for
	// a vendor with nothing runnable. Never nil (§34).
	InstallCommand []string `json:"installCommand"`
	// Models are the vendor's NAMED models, first = default, and ActiveModel
	// is the one this vendor would run right now. One entry or none means the
	// page draws no picker — a picker with a single entry is not a choice.
	// Never nil (§34).
	Models      []string `json:"models"`
	ActiveModel string   `json:"activeModel"`
}

// speechOptions is the one translation from config to internal/stt options —
// SpeechStatus, the mic, and bootstrap-by-way-of-applyConfig must not each
// assemble their own and drift.
func (a *Engine) speechOptions() stt.Options {
	cfg := a.cur().cfg
	return stt.Options{
		Engine:    strings.TrimSpace(cfg.SpeechEngine),
		Model:     strings.TrimSpace(cfg.SpeechModelName),
		ModelPath: strings.TrimSpace(cfg.SpeechModelPath),
	}
}

// ListSpeechEngines enumerates the STT vendors the catalog knows. Never nil
// (ARCHITECTURE.md §34).
func (a *Engine) ListSpeechEngines() []VoiceEngineInfo {
	cfg := a.cur().cfg
	return engineRows(sttDescriptors(), cfg.SpeechEngine, cfg.SpeechModelName)
}

// SetSpeechEngine picks the STT vendor. Empty goes back to the catalog
// default. Through applyConfig because audio_transcribe is handed its engine
// at construction — the same path SetSpeechModel takes, for the same reason.
func (a *Engine) SetSpeechEngine(id string) error {
	id = strings.TrimSpace(id)
	if _, ok := stt.Lookup(id); !ok {
		return fmt.Errorf("ไม่รู้จัก engine ถอดเสียงชื่อ %q", id)
	}
	conv := a.cur()
	next := a.dialBase(conv)
	next.SpeechEngine = id
	// A model name is one vendor's private vocabulary — same rule as the TTS
	// voice on a vendor switch.
	next.SpeechModelName = ""
	a.applyConfig(conv, next)
	return nil
}

// ListTTSEngines enumerates the read-aloud vendors. Never nil.
func (a *Engine) ListTTSEngines() []VoiceEngineInfo {
	cfg := a.cur().cfg
	return engineRows(ttsDescriptors(), cfg.TTSEngine, cfg.TTSModelName)
}

// SetTTSEngine picks the read-aloud vendor. The voice pick is cleared with it:
// a voice id is one engine's private naming, and keeping it across a vendor
// switch would pin the new engine to a name it has never heard of.
func (a *Engine) SetTTSEngine(id string) error {
	id = strings.TrimSpace(id)
	if _, ok := tts.Lookup(id); !ok {
		return fmt.Errorf("ไม่รู้จัก engine เสียงอ่านชื่อ %q", id)
	}
	conv := a.cur()
	next := a.dialBase(conv)
	next.TTSEngine = id
	next.TTSVoice = ""
	next.TTSModelName = ""
	a.applyConfig(conv, next)
	return nil
}

// SetSpeechModelName / SetTTSModelName pin a NAMED model on the active vendor
// (whisper-1 vs gpt-4o-transcribe, tts-1 vs tts-1-hd). Empty is a real choice
// — back to the vendor's first. Validation is against the catalog roster, so
// the webview can only ever store a name the vendor actually serves.
func (a *Engine) SetSpeechModelName(name string) error {
	name = strings.TrimSpace(name)
	if err := validateNamedModel(sttDescriptors(), a.cur().cfg.SpeechEngine, name); err != nil {
		return err
	}
	conv := a.cur()
	next := a.dialBase(conv)
	next.SpeechModelName = name
	a.applyConfig(conv, next)
	return nil
}

func (a *Engine) SetTTSModelName(name string) error {
	name = strings.TrimSpace(name)
	if err := validateNamedModel(ttsDescriptors(), a.cur().cfg.TTSEngine, name); err != nil {
		return err
	}
	conv := a.cur()
	next := a.dialBase(conv)
	next.TTSModelName = name
	a.applyConfig(conv, next)
	return nil
}

func validateNamedModel(descs []descriptorRow, activeEngine, name string) error {
	if name == "" {
		return nil
	}
	for _, row := range engineRows(descs, activeEngine, "") {
		if !row.Active {
			continue
		}
		for _, m := range row.Models {
			if strings.EqualFold(m, name) {
				return nil
			}
		}
		return fmt.Errorf("engine %s ไม่มีโมเดลชื่อ %q", row.ID, name)
	}
	return fmt.Errorf("ไม่พบ engine ที่กำลังใช้อยู่")
}

// TranscribeMicAudio turns a composer recording (a data: URL from
// MediaRecorder) into plain text for the input box. ffmpeg normalizes whatever
// the webview recorded into the 16kHz mono WAV every internal/stt engine
// expects — the same shape audio_transcribe produces, minus the timestamps: a
// dictated sentence has no use for "[m:ss]" in front of it.
func (a *Engine) TranscribeMicAudio(dataURL string) (string, error) {
	_, payload, ok := strings.Cut(dataURL, ";base64,")
	if !ok {
		return "", fmt.Errorf("รูปแบบเสียงที่อัดมาไม่ถูกต้อง")
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("อ่านเสียงที่อัดมาไม่ได้ (%v)", err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("ไม่มีเสียงในบันทึก ลองพูดอีกครั้ง")
	}
	// The engine first: a missing whisper binary or model is the likeliest
	// failure, and finding out before spending ffmpeg time gives the same
	// error faster — the order audio_transcribe.go settled on.
	engine, err := stt.New(a.speechOptions())
	if err != nil {
		return "", err
	}
	tmpDir, err := os.MkdirTemp("", "aetox-mic-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	// ffmpeg probes the container from content, so the recording needs no
	// correct extension — only a name.
	srcPath := filepath.Join(tmpDir, "mic.audio")
	if err := os.WriteFile(srcPath, raw, 0o600); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	wavPath := filepath.Join(tmpDir, "mic.wav")
	if err := micToWav(ctx, srcPath, wavPath); err != nil {
		return "", err
	}
	segments, err := engine.Transcribe(ctx, wavPath)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(segments))
	for _, seg := range segments {
		if text := strings.TrimSpace(seg.Text); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("ไม่ได้ยินเสียงพูดในบันทึก ลองพูดใกล้ไมค์ขึ้นอีกนิด")
	}
	return strings.Join(parts, " "), nil
}

// micToWav is extractAudioTrack's shape with the desktop's own ffmpeg
// resolution: findProgram already knows every address a real ffmpeg lives at
// on this machine (videotooling.go), including the copy the capability
// install downloads.
func micToWav(ctx context.Context, srcPath, wavPath string) error {
	ffmpeg := findProgram("ffmpeg")
	if ffmpeg == "" {
		return fmt.Errorf("ไม่พบโปรแกรม ffmpeg ซึ่งใช้แปลงเสียงจากไมค์ — ติดตั้งชุดเครื่องมือวิดีโอในหน้าเอเจน หรือ winget install ffmpeg แล้วลองใหม่")
	}
	cmd := exec.CommandContext(ctx, ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", srcPath,
		"-vn", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le",
		wavPath,
	)
	proc.HideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("แปลงเสียงที่อัดมาไม่ได้ (%s)", msg)
	}
	return nil
}

func engineRows(descs []descriptorRow, active, activeModel string) []VoiceEngineInfo {
	out := []VoiceEngineInfo{}
	active = strings.TrimSpace(active)
	activeModel = strings.TrimSpace(activeModel)
	for _, d := range descs {
		isActive := strings.EqualFold(d.id, active) || (active == "" && d.def)
		row := VoiceEngineInfo{
			ID:      d.id,
			Label:   d.label,
			Install: d.install,
			// The default engine is active when nothing is picked — a picker
			// with no marked row reads as "not configured", and that is not
			// what an empty value means.
			Active:         isActive,
			HasModels:      d.models,
			InstallCommand: append([]string{}, d.cmd...),
			Models:         append([]string{}, d.modelNames...),
		}
		// The model actually running: the pick when this vendor is the one
		// picked on, its first otherwise — so the page never shows a stale
		// name from another vendor's vocabulary.
		if len(d.modelNames) > 0 {
			row.ActiveModel = d.modelNames[0]
			if isActive && activeModel != "" {
				row.ActiveModel = activeModel
			}
		}
		out = append(out, row)
	}
	return out
}

// descriptorRow is the two catalogs' common shape — internal/stt and
// internal/tts deliberately do not import each other, so the flattening
// happens here.
type descriptorRow struct {
	id, label, install string
	cmd                []string
	modelNames         []string
	def, models        bool
}

func sttDescriptors() []descriptorRow {
	out := make([]descriptorRow, 0)
	for _, d := range stt.Catalog() {
		out = append(out, descriptorRow{id: d.ID, label: d.Label, install: d.Install, cmd: d.InstallCommand, modelNames: d.Models, def: d.Default, models: d.ModelGlob != ""})
	}
	return out
}

func ttsDescriptors() []descriptorRow {
	out := make([]descriptorRow, 0)
	for _, d := range tts.Catalog() {
		out = append(out, descriptorRow{id: d.ID, label: d.Label, install: d.Install, cmd: d.InstallCommand, modelNames: d.Models, def: d.Default})
	}
	return out
}
