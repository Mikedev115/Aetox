package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/provider"
	"github.com/Mikedev115/Aetox/internal/safety"
)

// isolateUserDirs points every "where does this user's data live" lookup at a
// temp directory, on every platform.
//
// Setting APPDATA/LOCALAPPDATA alone only isolates Windows. os.UserConfigDir
// reads XDG_CONFIG_HOME on Linux and $HOME/Library/Application Support on
// macOS, so on those platforms these tests were reading and writing the real
// ~/.config/aetox — which is how a "missing file returns nil" test started
// failing the moment another package's tests ran first and left a file there.
func isolateUserDirs(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("APPDATA", base)         // windows: os.UserConfigDir
	t.Setenv("LOCALAPPDATA", base)    // windows: the legacy preference path
	t.Setenv("USERPROFILE", base)     // windows: os.UserHomeDir
	t.Setenv("XDG_CONFIG_HOME", base) // linux: os.UserConfigDir
	t.Setenv("HOME", base)            // linux + macos: os.UserHomeDir, and macOS' config dir
	return base
}

func TestLoadDefaults(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	cfg := Load(ConfigOptions{
		ModelProvider: "openrouter",
	})

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		t.Fatalf("abs cwd failed: %v", err)
	}
	absRoot, err := filepath.Abs(cfg.SandboxRoot)
	if err != nil {
		t.Fatalf("abs root failed: %v", err)
	}

	if absRoot != absCwd {
		t.Fatalf("expected root fallback to cwd, got %q", cfg.SandboxRoot)
	}
	if cfg.MaxRetries != 2 {
		t.Fatalf("expected max retries 2, got %d", cfg.MaxRetries)
	}
	if cfg.MaxPlanRetries != 0 {
		t.Fatalf("expected max plan retries 0, got %d", cfg.MaxPlanRetries)
	}
	if cfg.ApprovalTimeoutSec != 60 {
		t.Fatalf("expected approval timeout 60, got %d", cfg.ApprovalTimeoutSec)
	}
	if cfg.ModelTimeoutSec != 30 {
		t.Fatalf("expected model timeout 30, got %d", cfg.ModelTimeoutSec)
	}
	if cfg.ModelProvider != "openrouter" {
		t.Fatalf("expected model provider openrouter, got %q", cfg.ModelProvider)
	}
	if cfg.ThinkLevel != "low" {
		t.Fatalf("expected default think level low, got %q", cfg.ThinkLevel)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	root := t.TempDir()

	cfg := Load(ConfigOptions{
		RootPath:        root,
		MaxRetries:      0,
		MaxPlanRetries:  -1,
		ApprovalTimeout: 0,
		ModelTimeout:    0,
		ModelProvider:   "",
	})

	if cfg.SandboxRoot != root {
		t.Fatalf("expected configured root %q, got %q", root, cfg.SandboxRoot)
	}
	if cfg.MaxRetries != 2 {
		t.Fatalf("expected fallback max retries 2, got %d", cfg.MaxRetries)
	}
	if cfg.MaxPlanRetries != 0 {
		t.Fatalf("expected fallback max plan retries 0, got %d", cfg.MaxPlanRetries)
	}
	if cfg.ApprovalTimeoutSec != 60 {
		t.Fatalf("expected fallback approval timeout 60, got %d", cfg.ApprovalTimeoutSec)
	}
	if cfg.ModelTimeoutSec != 30 {
		t.Fatalf("expected fallback model timeout 30, got %d", cfg.ModelTimeoutSec)
	}
	if cfg.ModelProvider != "aetox" {
		t.Fatalf("expected model provider fallback aetox, got %q", cfg.ModelProvider)
	}
	if cfg.ThinkLevel != "low" {
		t.Fatalf("expected fallback think level low, got %q", cfg.ThinkLevel)
	}
}

func TestSaveAndLoadModelPreferenceThinkLevel(t *testing.T) {
	isolateUserDirs(t)

	want := ModelPreference{
		ModelProvider: "openrouter",
		ModelName:     "deepseek/deepseek-r1",
		ThinkLevel:    "high",
	}
	if err := SaveModelPreference(want); err != nil {
		t.Fatalf("save preference failed: %v", err)
	}

	got, ok, err := LoadModelPreference()
	if err != nil {
		t.Fatalf("load preference failed: %v", err)
	}
	if !ok {
		t.Fatal("expected saved preference to exist")
	}
	if got.ThinkLevel != want.ThinkLevel {
		t.Fatalf("expected think level %q, got %q", want.ThinkLevel, got.ThinkLevel)
	}
}

func TestResolvedEnabledProvidersDefaultsToActiveProvider(t *testing.T) {
	// Never customized (empty slice) — an install must still show something,
	// and it must be exactly the provider already configured, not the whole catalog.
	got := ResolvedEnabledProviders(nil, "deepseek")
	if len(got) != 1 || got[0] != "deepseek" {
		t.Fatalf("ResolvedEnabledProviders(nil, deepseek) = %v, want [deepseek]", got)
	}
}

// A genuinely fresh install (never customized, never picked a real provider)
// runs on "aetox" (Aetox's own built-in engine) — it must show up by default,
// not be hidden, since that's exactly what removing an active provider falls
// back to and what the onboarding reply points a user back at.
func TestResolvedEnabledProvidersShowsAetoxByDefault(t *testing.T) {
	got := ResolvedEnabledProviders(nil, "aetox")
	if len(got) != 1 || got[0] != "aetox" {
		t.Fatalf("ResolvedEnabledProviders(nil, aetox) = %v, want [aetox]", got)
	}
	// "noop" is a backward-compat alias — must resolve the same way.
	got = ResolvedEnabledProviders(nil, "noop")
	if len(got) != 1 || got[0] != "aetox" {
		t.Fatalf("ResolvedEnabledProviders(nil, noop) = %v, want [aetox] (noop normalizes to aetox)", got)
	}
}

// A provider retired from the catalog does not outlive itself in the file.
// Antigravity (§242) was deleted in v1.5.25 and the owner's enabled_providers
// still named it three releases on, so the picker drew a row with a letter
// for an icon that nothing behind it could answer. The user's own rows are not
// that case: they are in the catalog the moment the file loads.
func TestResolvedEnabledProvidersDropsARetiredProvider(t *testing.T) {
	provider.SetCustom([]provider.Custom{{ID: "my-box", BaseURL: "http://127.0.0.1:1"}})
	t.Cleanup(func() { provider.SetCustom(nil) })
	got := ResolvedEnabledProviders([]string{"zai", "antigravity", "my-box", "codex"}, "codex")
	want := []string{"zai", "my-box", "codex"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ResolvedEnabledProviders = %v, want %v — the retired row must go, the user's own must stay", got, want)
	}
}

func TestResolvedEnabledProvidersRespectsCustomSetEvenWithoutActiveProvider(t *testing.T) {
	// A customized set wins outright — the active provider is NOT force-appended.
	// Otherwise explicitly disabling the currently-active provider (to switch
	// away and hide it) would be silently undone on every subsequent read.
	got := ResolvedEnabledProviders([]string{"openai", "anthropic"}, "deepseek")
	want := []string{"openai", "anthropic"}
	if len(got) != len(want) {
		t.Fatalf("ResolvedEnabledProviders = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ResolvedEnabledProviders = %v, want %v", got, want)
		}
	}
}

func TestResolvedEnabledProvidersDedupesAndNormalizes(t *testing.T) {
	got := ResolvedEnabledProviders([]string{"OpenAI", "openai", " anthropic "}, "openai")
	want := []string{"openai", "anthropic"}
	if len(got) != len(want) {
		t.Fatalf("ResolvedEnabledProviders = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ResolvedEnabledProviders = %v, want %v", got, want)
		}
	}
}

func TestLoadPermissionsMissingFileReturnsEmpty(t *testing.T) {
	isolateUserDirs(t)

	got, err := LoadPermissions()
	if err != nil {
		t.Fatalf("load permissions failed: %v", err)
	}
	if len(got.Rules) != 0 {
		t.Fatalf("expected no rules when file is missing, got %v", got.Rules)
	}
}

func TestLoadMCPServersMissingFileReturnsNil(t *testing.T) {
	isolateUserDirs(t)

	got, err := LoadMCPServers()
	if err != nil {
		t.Fatalf("load mcp servers failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no servers when file is missing, got %v", got)
	}
}

func TestSaveAndLoadMCPServers(t *testing.T) {
	isolateUserDirs(t)

	want := []MCPServerConfig{
		{Name: "fs", Command: []string{"npx", "-y", "server-filesystem", "/tmp"}, TimeoutMs: 5000},
		{Name: "git", Command: []string{"uvx", "mcp-git"}, Environment: map[string]string{"TOKEN": "x"}},
	}
	if err := SaveMCPServers(want); err != nil {
		t.Fatalf("save mcp servers failed: %v", err)
	}

	got, err := LoadMCPServers()
	if err != nil {
		t.Fatalf("load mcp servers failed: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d servers, got %d", len(want), len(got))
	}
	if got[0].Name != "fs" || len(got[0].Command) != 4 || got[0].TimeoutMs != 5000 {
		t.Fatalf("server 0 round-trip mismatch: %+v", got[0])
	}
	if got[1].Environment["TOKEN"] != "x" {
		t.Fatalf("server 1 environment not preserved: %+v", got[1])
	}
}

func TestLoadPermissions(t *testing.T) {
	isolateUserDirs(t)

	want := safety.PermissionConfig{Rules: []safety.PermissionRule{
		{Tool: "shell", Pattern: "rm *", Action: safety.PermissionDeny},
		{Tool: "git", Pattern: "status", Action: safety.PermissionAllow},
	}}
	path, err := PermissionsPath()
	if err != nil {
		t.Fatalf("permissions path: %v", err)
	}
	payload, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write permissions file: %v", err)
	}

	got, err := LoadPermissions()
	if err != nil {
		t.Fatalf("load permissions failed: %v", err)
	}
	if len(got.Rules) != len(want.Rules) {
		t.Fatalf("expected %d rules, got %d", len(want.Rules), len(got.Rules))
	}
	for i, rule := range want.Rules {
		if got.Rules[i] != rule {
			t.Fatalf("rule %d mismatch: want %+v, got %+v", i, rule, got.Rules[i])
		}
	}
}

// A server the user configured used to connect, register, and then be filtered
// off every desk, because a desk manifest ships compiled in and could not name
// a server that did not exist yet. The assistant reported having no MCP tools,
// which from where it stood was true. Ownership now sits on the server.
func TestMCPServersReachTheDesksTheyNameAndNoOthers(t *testing.T) {
	withServers(t, `[
	  {"name":"sequential-thinking","command":["npx","x"],"for":["assistant","coding"]},
	  {"name":"video-tools","command":["npx","y"],"for":["agent:วิดีโอ"]},
	  {"name":"off","command":["npx","z"],"for":["assistant"],"disabled":true},
	  {"name":"shelved","command":["npx","w"],"for":[]}
	]`)

	if got := MCPServersForDesk("assistant"); !slices.Equal(got, []string{"sequential-thinking"}) {
		t.Errorf("assistant desk got %v, want just the server that names it", got)
	}
	// The separation this field exists for: a server installed for an agent is
	// never on the main assistant's tool block.
	for _, desk := range []string{"assistant", "coding", "specialized"} {
		if slices.Contains(MCPServersForDesk(desk), "video-tools") {
			t.Errorf("an agent's server leaked onto the %s desk", desk)
		}
	}
	// Two switches, both real: disabled is never connected, empty `for` is
	// connected and attached nowhere.
	if got := MCPServersForDesk("assistant"); slices.Contains(got, "off") || slices.Contains(got, "shelved") {
		t.Errorf("a switched-off server was carried: %v", got)
	}
	// The pre-modes full desk still carries everything it can reach.
	if got := MCPServersForDesk(""); len(got) != 3 {
		t.Errorf("the full desk got %v, want every enabled server", got)
	}
}

// An install written before the field existed must not go dark, and the value
// it gets must be visible in the file rather than applied invisibly — this
// field's whole purpose is to be a switch the user can flip.
func TestMCPOwnersMigrateOnceAndAreWrittenBack(t *testing.T) {
	path := withServers(t, `[{"name":"sequential-thinking","command":["npx","x"]}]`)

	if got := MCPServersForDesk("assistant"); !slices.Contains(got, "sequential-thinking") {
		t.Fatalf("a pre-field server went dark: %v", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(raw), `"for"`) {
		t.Fatalf("the default was applied invisibly — the user cannot switch what they cannot see:\n%s", raw)
	}

	// An explicit empty list is a switch the user turned off. Absent and empty
	// decode to the same nil, so the migration reads the raw JSON; if it did
	// not, turning a server off would silently turn it back on.
	withServers(t, `[{"name":"shelved","command":["npx","x"],"for":[]}]`)
	if got := MCPServersForDesk("assistant"); len(got) != 0 {
		t.Errorf("an explicitly emptied `for` was overwritten by the default: %v", got)
	}
}

// A server saved with a nil placement reaches the file as `"for": null`,
// because the field is written without omitempty. The migration skipped it —
// the key IS there — so a server added from the shelf sat placed on no desk at
// all, and the entry looked settled while nothing carried its tools. Absent and
// null are the same statement: nobody has said where this goes.
func TestANullForIsTreatedAsNeverPlaced(t *testing.T) {
	withServers(t, `[{"name":"canva","url":"https://mcp.canva.com/mcp","for":null}]`)

	if got := MCPServersForDesk("assistant"); !slices.Contains(got, "canva") {
		t.Fatalf("a server saved with a null `for` is carried by no desk: %v", got)
	}
}

// withServers points the data root at a temp dir holding one servers file, and
// resets the once-guard so each test migrates its own fixture.
func withServers(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", root)
	path := filepath.Join(root, "mcp-servers.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write servers: %v", err)
	}
	migratedMCPOwners = sync.Once{}
	return path
}

// Turning a server off for everyone must survive a round trip through the
// file. An empty `for` written with omitempty disappears, the migration reads
// absence as "predates the field", and the switch the user just turned off is
// back on at the next launch — a setting that silently undoes itself.
func TestSwitchingAServerOffForEveryoneSurvivesASave(t *testing.T) {
	withServers(t, `[{"name":"s","command":["npx","x"],"for":["assistant"]}]`)

	servers, err := LoadMCPServers()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	servers[0].For = []string{}
	if err := SaveMCPServers(servers); err != nil {
		t.Fatalf("save: %v", err)
	}

	migratedMCPOwners = sync.Once{} // the next launch
	if got := MCPServersForDesk("assistant"); len(got) != 0 {
		t.Errorf("the server switched itself back on: %v", got)
	}
}

// Same rule for the endpoint override, including the reset: "back to the
// catalog default" has to clear a value saved under the older name, or the
// button does nothing on exactly the machines this rename touched.
func TestBaseURLOverrideFollowsTheRename(t *testing.T) {
	pref := ModelPreference{ModelBaseURLs: map[string]string{"qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1"}}
	if got := pref.BaseURLForProvider("alibaba"); got == "" {
		t.Error("a China-region endpoint saved as \"qwen\" is no longer read")
	}
	pref.SetBaseURLForProvider("alibaba", "")
	if len(pref.ModelBaseURLs) != 0 {
		t.Errorf("reset left %v behind", pref.ModelBaseURLs)
	}
}
