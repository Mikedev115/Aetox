package config

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/provider"
)

// A custom endpoint is a row in the preference file, and the catalog only
// answers from a copy of that file — so the copy has to follow every write
// and every read, or a row added a moment ago is "unknown provider" to the
// very next call (SetProviderEnabled normalizes through the catalog).
func TestCustomProvidersReachTheCatalogOnSaveAndLoad(t *testing.T) {
	isolateUserDirs(t)
	t.Cleanup(func() { provider.SetCustom(nil) })

	if err := SaveModelPreference(ModelPreference{
		CustomProviders:  []CustomProvider{{ID: "my-vllm", BaseURL: "http://10.0.0.2:8000/v1"}},
		EnabledProviders: []string{"aetox", "my-vllm"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// The save alone registers it.
	if _, ok := provider.Lookup("my-vllm"); !ok {
		t.Fatal("custom row unknown to the catalog right after the save")
	}

	provider.SetCustom(nil) // a fresh process
	pref, ok, err := LoadModelPreference()
	if err != nil || !ok {
		t.Fatalf("reload: ok=%v err=%v", ok, err)
	}
	spec, found := provider.Lookup("my-vllm")
	if !found {
		t.Fatal("custom row unknown to the catalog after a load")
	}
	if spec.BaseURL != "http://10.0.0.2:8000/v1" || spec.Runtime != provider.RuntimeOpenAICompatible {
		t.Errorf("spec = %+v", spec)
	}
	if got := ResolvedEnabledProviders(pref.EnabledProviders, "aetox"); len(got) != 2 || got[1] != "my-vllm" {
		t.Errorf("enabled = %v, want [aetox my-vllm]", got)
	}

	// The whole point: the engine can be built on it.
	p, err := model.NewProvider(model.ProviderOptions{Provider: "my-vllm", Model: "x", APIKey: "sk-vllm"})
	if err != nil {
		t.Fatalf("NewProvider(my-vllm): %v", err)
	}
	if p.Name() != "my-vllm" {
		t.Errorf("Name() = %q", p.Name())
	}
}
