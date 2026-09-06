package main

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The CLI's startup save used to rebuild the preference file from the five
// fields it cares about, and the desktop then read the result as a picker
// nobody had customized — one provider where there had been several (owner,
// 5 ก.ย.; DECISIONS §225).
func TestPersistModelPreferenceKeepsTheDesktopsFields(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if err := config.SaveModelPreference(config.ModelPreference{
		ModelProvider:    "opencode-go",
		ModelName:        "glm-5.3-flash",
		EnabledProviders: []string{"opencode-go", "deepseek"},
		ModelNames:       map[string]string{"deepseek": "deepseek-v4-flash"},
		UserName:         "Mike",
		LastDesk:         "assistant",
		UILocale:         "th",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := persistModelPreference(config.Config{
		ModelProvider: "deepseek",
		ModelName:     "deepseek-v4-flash",
		ThinkLevel:    "high",
		ApprovalMode:  "full-access",
	}); err != nil {
		t.Fatalf("persist: %v", err)
	}

	pref, ok, err := config.LoadModelPreference()
	if err != nil || !ok {
		t.Fatalf("reload: ok=%v err=%v", ok, err)
	}
	if pref.ModelProvider != "deepseek" || pref.ModelName != "deepseek-v4-flash" {
		t.Fatalf("the CLI's own choice did not land: %+v", pref)
	}
	if got := config.ResolvedEnabledProviders(pref.EnabledProviders, pref.ModelProvider); len(got) != 2 {
		t.Fatalf("the picker's providers did not survive a CLI launch: %v", got)
	}
	if pref.ModelForProvider("deepseek") != "deepseek-v4-flash" || pref.UserName != "Mike" || pref.LastDesk != "assistant" || pref.UILocale != "th" {
		t.Fatalf("desktop-only fields were dropped: %+v", pref)
	}
}
