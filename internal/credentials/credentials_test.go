package credentials

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
}

// Round trip through whatever the platform gives us. On Windows the file is
// DPAPI-wrapped and must not contain the key in the clear; everywhere else it
// is plaintext by design (atrest) and this still pins that the value survives
// the trip.
func TestRoundTrip(t *testing.T) {
	isolate(t)

	const key = "sk-round-trip"
	if err := Save(Credentials{ModelAPIKeys: map[string]string{"deepseek": key}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ModelAPIKeys["deepseek"] != key {
		t.Fatalf("round trip lost the key: %+v", got)
	}
	if runtime.GOOS == "windows" {
		path, _ := Path()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if strings.Contains(string(raw), key) {
			t.Errorf("the key is in the clear on disk despite DPAPI:\n%s", raw)
		}
	}
}

// Secrets and settings need different handling, and one file cannot have two
// (2026-08-06). Since §248 A4 the settings struct has no field for a key at
// all, so a save from config cannot put one in the settings file — and this
// package is where one goes.
func TestAKeyNeverReachesThePreferenceFile(t *testing.T) {
	isolate(t)

	const key = "sk-do-not-let-this-into-the-settings-file"
	if err := config.SaveModelPreference(config.ModelPreference{ModelProvider: "deepseek", LastDesk: "specialized"}); err != nil {
		t.Fatalf("save preference: %v", err)
	}
	if err := Set("deepseek", key); err != nil {
		t.Fatalf("set: %v", err)
	}
	prefPath, _ := config.PreferencePath()
	raw, err := os.ReadFile(prefPath)
	if err != nil {
		t.Fatalf("read preferences: %v", err)
	}
	if strings.Contains(string(raw), key) {
		t.Fatalf("the API key is in the settings file:\n%s", raw)
	}
	if !strings.Contains(string(raw), "specialized") {
		t.Fatalf("the settings did not survive:\n%s", raw)
	}
	if got := KeyFor("deepseek"); got != key {
		t.Fatalf("KeyFor = %q, want the saved key", got)
	}
}

// An install written before the split must not lose its keys — losing one
// means the user cannot reach their own provider until they find and retype
// it. config no longer knows the field, so this package reads it raw.
func TestKeysInAnOldPreferenceFileMoveOutOnFirstRead(t *testing.T) {
	isolate(t)

	const key = "sk-written-before-the-split"
	prefPath, _ := config.PreferencePath()
	if err := os.MkdirAll(filepath.Dir(prefPath), 0o700); err != nil {
		t.Fatal(err)
	}
	old := `{"provider":"deepseek","last_desk":"coding","provider_api_keys":{"deepseek":"` + key + `"}}`
	if err := os.WriteFile(prefPath, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := KeyFor("deepseek"); got != key {
		t.Fatalf("the migration lost the key: KeyFor = %q", got)
	}
	raw, err := os.ReadFile(prefPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if strings.Contains(string(raw), key) {
		t.Fatalf("the key is still in the settings file after migrating:\n%s", raw)
	}
	if !strings.Contains(string(raw), "coding") {
		t.Fatalf("the settings did not survive the strip:\n%s", raw)
	}
	// Two copies of a secret is worse than one: the stripped file must not be
	// the only thing that happened.
	creds, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if creds.ModelAPIKeys["deepseek"] != key {
		t.Fatalf("the key did not land in the credentials file: %+v", creds)
	}
}

// The upgrade that renames a provider is the one that can lose a key, and it
// loses it silently: the file still has it, the card still draws, and the app
// says there is no key. Every credentials.json written before 2026-08-24 keyed
// Alibaba Cloud as "qwen".
func TestKeySavedUnderAnOlderProviderNameStillResolves(t *testing.T) {
	isolate(t)
	if err := Save(Credentials{ModelAPIKeys: map[string]string{"qwen": "sk-from-an-older-build"}}); err != nil {
		t.Fatal(err)
	}
	if got := KeyFor("alibaba"); got != "sk-from-an-older-build" {
		t.Errorf("KeyFor(alibaba) = %q — the key saved as \"qwen\" is unreachable", got)
	}
	if got := KeyFor("qwen"); got != "sk-from-an-older-build" {
		t.Errorf("KeyFor(qwen) = %q — the old name has to keep working too", got)
	}
}

// And the write side, which is the half that turns one stale row into two live
// ones. The reader walks a map: with both "qwen" and "alibaba" present it
// returns whichever Go hands it first, so the key actually sent could change
// between two runs of the same binary.
func TestSavingAKeyClearsTheOlderSpelling(t *testing.T) {
	isolate(t)
	if err := Save(Credentials{ModelAPIKeys: map[string]string{"qwen": "sk-old"}}); err != nil {
		t.Fatal(err)
	}
	if err := Set("alibaba", "sk-new"); err != nil {
		t.Fatal(err)
	}
	creds, _ := Load()
	if _, stale := creds.ModelAPIKeys["qwen"]; stale {
		t.Error("the entry keyed \"qwen\" survived the write — two rows now answer for one provider")
	}
	if got := KeyFor("alibaba"); got != "sk-new" {
		t.Errorf("KeyFor(alibaba) = %q, want the key just saved", got)
	}
}

// Removing the last key must leave the secrets file without it — an empty
// map is a value to write, not an absence to skip.
func TestForgetWritesTheEmptiedFile(t *testing.T) {
	isolate(t)
	if err := Set("my-vllm", "sk-vllm"); err != nil {
		t.Fatal(err)
	}
	if err := Forget("my-vllm"); err != nil {
		t.Fatalf("forget: %v", err)
	}
	creds, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, still := creds.ModelAPIKeys["my-vllm"]; still {
		t.Error("the forgotten key is still in credentials.json")
	}
	if got := StoredKeyFor("my-vllm"); got != "" {
		t.Errorf("StoredKeyFor after Forget = %q", got)
	}
}

// The environment is the fallback the CLI has always had, and the .env file
// config loads feeds the same variables; a saved key outranks it.
func TestKeyForFallsBackToTheProvidersEnvironmentVariable(t *testing.T) {
	isolate(t)
	t.Setenv("OPENROUTER_API_KEY", "env-key")
	if got := KeyFor("openrouter"); got != "env-key" {
		t.Fatalf("KeyFor = %q, want the environment's", got)
	}
	if got := StoredKeyFor("openrouter"); got != "" {
		t.Fatalf("StoredKeyFor = %q, want nothing — the environment is not the store", got)
	}
	if err := Set("openrouter", "sk-saved"); err != nil {
		t.Fatal(err)
	}
	if got := KeyFor("openrouter"); got != "sk-saved" {
		t.Fatalf("KeyFor = %q, want the saved key over the environment", got)
	}
}

// A blank submission is not a deletion.
func TestSetIgnoresAnEmptyKey(t *testing.T) {
	isolate(t)
	if err := Set("deepseek", "sk-kept"); err != nil {
		t.Fatal(err)
	}
	if err := Set("deepseek", "   "); err != nil {
		t.Fatal(err)
	}
	if got := KeyFor("deepseek"); got != "sk-kept" {
		t.Fatalf("KeyFor = %q, want the key a blank form must not erase", got)
	}
}
