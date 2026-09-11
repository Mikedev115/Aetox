// Package credentials is the store of provider API keys, and nothing else.
//
// It is its own package, below the desktop and above config, because of who
// must NOT be able to import it (§248 A4): the engine half of the app. The
// engine builds its model clients without a key — bootstrap hands them a
// model.Transport that the screen signs each request through — and a package
// the engine could link would be a key the engine could read. So this one is
// imported by the screen, by the CLI, and by the tools that still read a key
// of their own (image and speech engines, per host by decision), and by no
// package under internal/engine. A test there checks the dependency list.
//
// What used to live in internal/config: the same file, the same at-rest
// wrapping, the same scrubbing. config no longer reads or writes it, and
// config.Config and config.ModelPreference no longer carry a key, so nothing
// that travels through config can carry one either.
package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/atrest"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/provider"
)

// Credentials are the secrets that used to live in model-preference.json, in
// their own file (owner's call, 2026-08-06).
//
// The reason is not that a preference file cannot be given tight permissions —
// it already had 0600 and an atomic write. It is that secrets and settings have
// different *handling*, and one file cannot have two. `ui_locale`, `last_desk`,
// `user_name` and `speech_model_path` are things people open the file to check,
// paste into a bug report, and screenshot. The API keys sat in the same object,
// so every one of those ordinary acts leaked them — which is exactly how a key
// ended up in a debugging transcript on the day this was written.
//
// Splitting them means the ordinary file can stay ordinary: readable, quotable,
// safe to show. Everything that is genuinely a secret is in one place, with one
// set of rules — encrypted at rest where the platform offers it (atrest),
// refused by every file tool (skill.ownSecretFiles), and scrubbed from the
// debug log (debuglog.Redact).
type Credentials struct {
	// ModelAPIKeys is keyed by provider, the same shape the preference file
	// held, so nothing above this layer had to learn a new one.
	ModelAPIKeys map[string]string `json:"provider_api_keys,omitempty"`
}

// Path is <DataRoot>/credentials.json.
func Path() (string, error) {
	root, err := config.DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "credentials.json"), nil
}

// mu serializes every load-modify-save in this process, for the reason
// config.prefMu exists: two writers that each read the file, change their own
// entry and write the whole thing back keep only the second one's change.
var mu sync.Mutex

// Load reads the secrets file. A missing file is not an error — it is a fresh
// install, or one whose keys have not been split out yet.
//
// Every key that comes back is registered with debuglog, which is the one place
// that can guarantee it never appears in a log line: doing it at the read means
// no caller can forget, and a key that was never loaded cannot leak.
//
// Keys still sitting in an old preference file are moved here on the way past,
// so that a store from before the split (2026-08-06) is read whole by whoever
// asks first. config itself no longer knows the field, so this is the one
// reader that can.
func Load() (Credentials, error) {
	absorbLegacyPreferenceKeys()
	return load()
}

func load() (Credentials, error) {
	var creds Credentials
	path, err := Path()
	if err != nil {
		return creds, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return creds, nil
		}
		return creds, err
	}
	if err := json.Unmarshal(atrest.Unprotect(raw), &creds); err != nil {
		return creds, err
	}
	for _, key := range creds.ModelAPIKeys {
		debuglog.Redact(key)
	}
	return creds, nil
}

// Save writes the secrets file: encrypted at rest where the platform offers
// it, 0600, atomically.
//
// Write-then-rename for the same reason the preference file uses it — a
// truncate that dies mid-write loses every key at once — and the temp file is
// created 0600 too, since a secret briefly world-readable is still a secret
// that was world-readable.
func Save(creds Credentials) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	for _, key := range creds.ModelAPIKeys {
		debuglog.Redact(key)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, atrest.Protect(payload), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// KeyFor is the key a provider is reached with: the one the user saved for it,
// else the provider's usual environment variable (provider.ResolveAPIKey),
// which the .env file config loads at startup feeds too. Empty means the user
// has genuinely not provided one.
//
// Both sides of the lookup are normalized, and that is the whole migration
// path for a provider rename. The stored map is keyed by whatever the app was
// calling the provider on the day the key was saved ("qwen" before
// 2026-08-24, "alibaba" after); normalizing only the lookup name would turn
// every one of those files into a key the owner can see in credentials.json
// and the app swears is not there.
func KeyFor(providerName string) string {
	if key := StoredKeyFor(providerName); key != "" {
		return key
	}
	return strings.TrimSpace(provider.ResolveAPIKey(providerName))
}

// StoredKeyFor is KeyFor without the environment fallback: only what the user
// saved. The settings page asks this one, because "there is a key" and "there
// is a key in the file" are different answers there.
func StoredKeyFor(providerName string) string {
	key := normalize(providerName)
	if key == "" {
		return ""
	}
	creds, err := Load()
	if err != nil {
		return ""
	}
	for stored, value := range creds.ModelAPIKeys {
		if normalize(stored) == key {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Set records a key for a provider. An empty key is not a deletion — see
// Forget — because a form that submits its field blank must not be able to
// erase a credential by accident.
//
// One provider, one row: an entry saved under an older spelling of the same
// provider is dropped, or the reader above would walk a map with two answers
// and return whichever Go handed it first.
func Set(providerName, apiKey string) error {
	key := normalize(providerName)
	trimmed := strings.TrimSpace(apiKey)
	if key == "" || trimmed == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	creds, err := Load()
	if err != nil {
		return err
	}
	if creds.ModelAPIKeys == nil {
		creds.ModelAPIKeys = map[string]string{}
	}
	dropAliases(creds.ModelAPIKeys, key)
	creds.ModelAPIKeys[key] = trimmed
	return Save(creds)
}

// Forget drops the key filed under a provider, however it was spelled. A
// custom provider row being removed is the usual caller: its name is about to
// mean nothing, and a key left under it would sit in the secrets file forever,
// readable by nobody and deletable from nowhere.
func Forget(providerName string) error {
	key := normalize(providerName)
	if key == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	creds, err := Load()
	if err != nil {
		return err
	}
	if creds.ModelAPIKeys == nil {
		return nil
	}
	dropAliases(creds.ModelAPIKeys, key)
	delete(creds.ModelAPIKeys, key)
	// Written even when now empty, or the file keeps the key the user just
	// removed.
	return Save(creds)
}

// ProviderAPIKey is how the image and speech engines (internal/imagegen,
// internal/stt, internal/tts) read the same store the model layer's screen
// does: the key the user entered on the models page serves their voice too,
// with the engine's own environment variables as the fallback. Empty means the
// user has genuinely not provided one — the engines turn that into their own
// actionable error.
func ProviderAPIKey(providerName string, envVars ...string) string {
	if key := StoredKeyFor(providerName); key != "" {
		return key
	}
	for _, name := range envVars {
		if key := strings.TrimSpace(os.Getenv(name)); key != "" {
			return key
		}
	}
	return ""
}

func normalize(providerName string) string {
	return strings.ToLower(strings.TrimSpace(model.NormalizeProvider(providerName)))
}

// dropAliases removes every entry that means the same provider as key but is
// spelled differently, leaving key itself alone.
func dropAliases(m map[string]string, key string) {
	for stored := range m {
		if stored != key && normalize(stored) == key {
			delete(m, stored)
		}
	}
}

// absorbLegacyPreferenceKeys moves keys still sitting in model-preference.json
// into the credentials file and rewrites the preference file without them.
//
// Ordered so a crash cannot lose a key: the credentials file is written first
// and only a successful write leads to stripping the original. The worst
// interruption leaves the key in both files, which the next run resolves — the
// opposite order would leave a user unable to reach their own provider.
//
// Keys already in the credentials file win. That file is the one the app writes
// to now, so anything in the preference file is by definition the older copy.
//
// config.ModelPreference has no field for these any more, which is what makes
// the strip a plain load-and-save there: the struct cannot carry what it does
// not know.
func absorbLegacyPreferenceKeys() {
	path, err := config.PreferencePath()
	if err != nil {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var legacy struct {
		Keys map[string]string `json:"provider_api_keys"`
	}
	if json.Unmarshal(raw, &legacy) != nil || len(legacy.Keys) == 0 {
		return
	}
	creds, err := load()
	if err != nil {
		return // unreadable secrets file: leave the old copy where it is
	}
	if creds.ModelAPIKeys == nil {
		creds.ModelAPIKeys = map[string]string{}
	}
	moved := false
	for providerName, key := range legacy.Keys {
		if _, taken := creds.ModelAPIKeys[providerName]; taken {
			continue
		}
		creds.ModelAPIKeys[providerName] = key
		moved = true
	}
	if moved {
		if err := Save(creds); err != nil {
			return // could not store them safely: do not remove the only copy
		}
	}
	if pref, ok, err := config.LoadModelPreference(); ok && err == nil {
		_ = config.SaveModelPreference(pref)
	}
}
