package engine

import (
	"os"
	"strings"

	"github.com/Mikedev115/Aetox/internal/command"
	"github.com/Mikedev115/Aetox/internal/model"
)

// ListPromptPresets reports every prompt preset for the Settings page —
// the ones bundled with Aetox plus anything in <DataRoot>/prompts/*.md.
func (a *Engine) ListPromptPresets() []command.Preset {
	return jsonSlice(command.ListPresets())
}

// GuideTopics lists the questions Aetox's built-in engine can answer about
// itself, in the UI's language. Clicking one sends it as an ordinary message —
// there is no separate guide path any more (ARCHITECTURE.md §42).
func (a *Engine) GuideTopics() []model.GuideTopic {
	return jsonSlice(model.GuideTopics(a.cur().cfg.UILocale))
}

// SetUILocale records the language the UI is showing so Aetox's own built-in
// provider can answer in it — the only part of the engine that has any business
// with language (ARCHITECTURE.md §40). Persisted next to the other preferences
// and applied through the same path a model change takes, so there is no new
// machinery here at all.
func (a *Engine) SetUILocale(locale string) error {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" || locale == a.cur().cfg.UILocale {
		return nil // nothing to do; never re-bootstrap for no reason
	}
	cfg := a.cfg
	cfg.UILocale = locale
	a.applyConfig(a.cur(), cfg)
	return nil
}

// SavePromptPreset writes (or overwrites) a user preset. Saving under a
// bundled preset's name creates an override — see internal/command.
func (a *Engine) SavePromptPreset(name, body string) error {
	return command.SavePreset(name, body)
}

// DeletePromptPreset removes a user preset and its cover. Deleting an override
// restores the bundled preset it was hiding.
func (a *Engine) DeletePromptPreset(name string) error {
	return command.DeletePreset(name)
}

// SetPresetImageFrom copies the image at path in as that preset's cover and
// returns the cover as a data URI, so the card updates without re-reading the
// whole list. The engine's half of PickPresetImage (screen_doors.go).
func (a *Engine) SetPresetImageFrom(name, path string) (string, error) {
	if err := command.SavePresetImage(name, path); err != nil {
		return "", err
	}
	for _, p := range command.ListPresets() {
		if p.Name == name {
			return p.Image, nil
		}
	}
	return "", nil
}

// RemovePresetImage drops a preset's cover, keeping the prompt.
func (a *Engine) RemovePresetImage(name string) error {
	return command.RemovePresetImage(name)
}

// PromptsFolderPath creates the prompts directory if needed and answers with
// it, so adding a preset is "drop a .md file here" (OpenPromptsFolder,
// screen_doors.go, reveals it). Creating it on demand is why the folder does
// not need to exist at install time.
func (a *Engine) PromptsFolderPath() (string, error) {
	dir, err := command.PresetsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
