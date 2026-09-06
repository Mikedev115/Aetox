package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// The bug these pin: a provider enabled in the desktop's picker was gone a
// while later (owner, 5 ก.ย.; DECISIONS §225). Every writer of the preference
// file read it, set its own field and wrote the whole struct back — the CLI's
// writer from a fresh struct, the desktop's from an empty one whenever the read
// failed — and either way the fields only the file remembered were dropped.
// UpdateModelPreference is the one door now, and these are its promises.

func TestUpdateModelPreferenceKeepsWhatItWasNotAskedAbout(t *testing.T) {
	isolateUserDirs(t)
	if err := SaveModelPreference(ModelPreference{
		ModelProvider:    "opencode-go",
		EnabledProviders: []string{"opencode-go", "deepseek"},
		ModelNames:       map[string]string{"deepseek": "deepseek-v4-flash"},
		UserName:         "Mike",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := UpdateModelPreference(func(p *ModelPreference) error {
		p.ModelName = "glm-5.3-flash"
		return nil
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	pref, ok, err := LoadModelPreference()
	if err != nil || !ok {
		t.Fatalf("reload: ok=%v err=%v", ok, err)
	}
	if pref.ModelName != "glm-5.3-flash" {
		t.Fatalf("the change itself did not land: %+v", pref)
	}
	if got := ResolvedEnabledProviders(pref.EnabledProviders, pref.ModelProvider); len(got) != 2 {
		t.Fatalf("the enabled providers did not survive an unrelated save: %v", got)
	}
	if pref.ModelForProvider("deepseek") != "deepseek-v4-flash" || pref.UserName != "Mike" {
		t.Fatalf("fields nobody touched were lost: %+v", pref)
	}
}

func TestUpdateModelPreferenceLeavesAFileItCannotReadAlone(t *testing.T) {
	isolateUserDirs(t)
	path, err := PreferencePath()
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
	err = UpdateModelPreference(func(p *ModelPreference) error {
		p.ModelName = "glm-5.3-flash"
		return nil
	})
	if err == nil {
		t.Fatal("an unreadable file was treated as a missing one")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != halfWritten {
		t.Fatalf("a file that could not be read was overwritten:\n%s", raw)
	}
}

func TestUpdateModelPreferenceStartsFromEmptyWhenThereIsNoFile(t *testing.T) {
	isolateUserDirs(t)
	if err := UpdateModelPreference(func(p *ModelPreference) error {
		p.UserName = "Mike"
		return nil
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	pref, ok, _ := LoadModelPreference()
	if !ok || pref.UserName != "Mike" {
		t.Fatalf("the first save did not create the file: ok=%v %+v", ok, pref)
	}
}

func TestUpdateModelPreferenceUnchangedWritesNothing(t *testing.T) {
	isolateUserDirs(t)
	if err := UpdateModelPreference(func(p *ModelPreference) error {
		return ErrPreferenceUnchanged
	}); err != nil {
		t.Fatalf("unchanged is not an error: %v", err)
	}
	path, _ := PreferencePath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("a no-op still wrote the file (stat err=%v)", err)
	}
}

// Two settings saved at once used to keep only the second one's change.
func TestConcurrentUpdatesLoseNothing(t *testing.T) {
	isolateUserDirs(t)
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- UpdateModelPreference(func(p *ModelPreference) error {
				p.EnabledProviders = append(p.EnabledProviders, fmt.Sprintf("provider-%02d", i))
				return nil
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("update: %v", err)
		}
	}
	pref, _, _ := LoadModelPreference()
	if len(pref.EnabledProviders) != n {
		t.Fatalf("%d of %d concurrent saves survived: %v", len(pref.EnabledProviders), n, pref.EnabledProviders)
	}
}
