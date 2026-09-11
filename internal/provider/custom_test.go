package provider

import (
	"testing"
)

func TestSlugCustomID(t *testing.T) {
	cases := map[string]string{
		"My vLLM":           "my-vllm",
		"  DeepSeek 2  ":    "deepseek-2",
		"a__b..c//d":        "a-b-c-d",
		"-lead-and-trail-":  "lead-and-trail",
		"ภาษาไทยล้วน":       "",
		"ไทย mixed English": "mixed-english",
	}
	for in, want := range cases {
		if got := SlugCustomID(in); got != want {
			t.Errorf("SlugCustomID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateCustomIDRefusesCatalogNames(t *testing.T) {
	for _, taken := range []string{"deepseek", "openai-compatible", "custom", "compatible", "chatgpt"} {
		if err := ValidateCustomID(taken); err == nil {
			t.Errorf("ValidateCustomID(%q) accepted a name the catalog owns", taken)
		}
	}
	for _, bad := range []string{"", "-x", "Has Space", "UPPER"} {
		if err := ValidateCustomID(bad); err == nil {
			t.Errorf("ValidateCustomID(%q) accepted a malformed id", bad)
		}
	}
	if err := ValidateCustomID("my-vllm"); err != nil {
		t.Errorf("ValidateCustomID(my-vllm) = %v", err)
	}
}

func TestCustomRowAnswersLikeACatalogRow(t *testing.T) {
	t.Cleanup(func() { SetCustom(nil) })
	SetCustom([]Custom{
		{ID: "my-vllm", BaseURL: "http://10.0.0.2:8000/v1/"},
		{ID: "deepseek", BaseURL: "https://shadow.example"}, // catalog name: dropped
	})

	spec, ok := Lookup("my-vllm")
	if !ok {
		t.Fatal("Lookup(my-vllm) = not found after SetCustom")
	}
	if spec.Runtime != RuntimeOpenAICompatible {
		t.Errorf("Runtime = %q, want openai-compatible", spec.Runtime)
	}
	if spec.BaseURL != "http://10.0.0.2:8000/v1" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", spec.BaseURL)
	}
	if !spec.RequiresAPIKey || !spec.AcceptsAPIKey {
		t.Errorf("a custom row takes a key like the openai-compatible row: requires=%v accepts=%v", spec.RequiresAPIKey, spec.AcceptsAPIKey)
	}
	if len(spec.EnvKeys) != 0 {
		t.Errorf("EnvKeys = %v, want none: another endpoint's env key must not be read", spec.EnvKeys)
	}
	if !IsCustom("my-vllm") || IsCustom("deepseek") || IsCustom("openai-compatible") {
		t.Error("IsCustom answers wrong for one of my-vllm / deepseek / openai-compatible")
	}
	if got := DefaultBaseURL("deepseek"); got == "https://shadow.example" {
		t.Error("a custom row named after a catalog provider shadowed it")
	}

	all := SupportedProviders()
	if all[len(all)-1] != "my-vllm" {
		t.Errorf("SupportedProviders ends with %q, want the custom row last", all[len(all)-1])
	}
	if Customs()[0].ID != "my-vllm" || len(Customs()) != 1 {
		t.Errorf("Customs() = %+v, want just my-vllm", Customs())
	}

	SetCustom(nil)
	if _, ok := Lookup("my-vllm"); ok {
		t.Error("Lookup(my-vllm) still found after SetCustom(nil)")
	}
}
