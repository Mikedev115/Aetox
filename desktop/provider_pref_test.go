package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The composer's picker shows the enabled set, and an empty set means "never
// customized: show the active provider only" — so any save that drops the field
// shrinks the picker to one row. That is what the owner saw (5 ก.ย.; DECISIONS
// §225): a provider added in settings, gone a while later. Every writer is a
// locked load-modify-save now (config.UpdateModelPreference), and one that
// cannot read the file writes nothing rather than an empty struct plus its own
// field.
func TestAModelSwitchKeepsTheEnabledProviders(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t, t.TempDir())
	for _, p := range []string{"opencode-go", "deepseek"} {
		if _, err := a.SetProviderEnabled(p, true); err != nil {
			t.Fatalf("enable %s: %v", p, err)
		}
	}
	// Everything a model switch writes on its way through.
	persistModelPreference(config.Config{ModelProvider: "opencode-go", ModelName: "glm-5.3-flash"})
	rememberModelForProvider("opencode-go", "glm-5.3-flash")
	rememberDesk("assistant")
	saveBusySignal(config.Config{})
	// The app's own default provider is in the list too; what must not happen
	// is either added one falling out.
	got := a.EnabledProviders()
	for _, want := range []string{"opencode-go", "deepseek"} {
		found := false
		for _, p := range got {
			if p == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s fell out of the picker after a model switch: %v", want, got)
		}
	}
}

func TestAnUnreadablePreferenceFileIsNotOverwritten(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	path, err := config.PreferencePath()
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const halfWritten = `{"provider":"opencode-go","enabled_providers":["opencode-go","deepseek"`
	if err := os.WriteFile(path, []byte(halfWritten), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	persistModelPreference(config.Config{ModelProvider: "opencode-go", ModelName: "glm-5.3-flash"})
	rememberModelForProvider("opencode-go", "glm-5.3-flash")
	raw, _ := os.ReadFile(path)
	if string(raw) != halfWritten {
		t.Fatalf("a file that could not be read was overwritten:\n%s", raw)
	}
}
