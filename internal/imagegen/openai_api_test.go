package imagegen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// apiServer stands in for an OpenAI-shaped endpoint and records the body it
// was sent, so a test can assert on the request that was actually built.
func apiServer(t *testing.T, handle func(body map[string]any, w http.ResponseWriter)) (*httptest.Server, *map[string]any) {
	t.Helper()
	seen := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Errorf("called %s, want /images/generations", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&seen)
		handle(seen, w)
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func b64Answer(w http.ResponseWriter, pic []byte) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(pic)}},
	})
}

func TestOpenAIShapedEngineDecodesBase64AndSendsTheSize(t *testing.T) {
	srv, seen := apiServer(t, func(_ map[string]any, w http.ResponseWriter) {
		b64Answer(w, []byte("\x89PNG\r\n\x1a\npretend"))
	})
	eng := &apiImages{
		id: "openai", baseURL: srv.URL, apiKey: "k", model: "gpt-image-1",
		spec: apiImageSpecs["openai"], client: srv.Client(),
	}
	out := filepath.Join(t.TempDir(), "a.png")
	if err := eng.Generate(context.Background(), "a cat", Request{Width: 1024, Height: 1024}, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil || !strings.HasPrefix(string(got), "\x89PNG") {
		t.Fatalf("the decoded picture did not land: %q %v", got, err)
	}
	if (*seen)["size"] != "1024x1024" {
		t.Errorf("size = %v, want 1024x1024", (*seen)["size"])
	}
	if (*seen)["model"] != "gpt-image-1" {
		t.Errorf("model = %v", (*seen)["model"])
	}
	// response_format must never be sent: gpt-image-1 rejects it outright, and
	// both shapes are handled on the way back instead.
	if _, sent := (*seen)["response_format"]; sent {
		t.Error("response_format was sent — gpt-image-1 refuses the request that carries it")
	}
}

func TestAVendorWithNoSizeSupportIsNotSentOne(t *testing.T) {
	// xAI's image endpoint takes a prompt and a count. Sending a size there is
	// a 400 on a request that would otherwise have worked, so the size is
	// dropped and the picture still gets made.
	srv, seen := apiServer(t, func(_ map[string]any, w http.ResponseWriter) {
		b64Answer(w, []byte("\xff\xd8\xffpretend"))
	})
	eng := &apiImages{
		id: "xai", baseURL: srv.URL, apiKey: "k", model: "grok-2-image",
		spec: apiImageSpecs["xai"], client: srv.Client(),
	}
	if err := eng.Generate(context.Background(), "a cat", Request{Width: 512, Height: 512}, filepath.Join(t.TempDir(), "a.jpg")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, sent := (*seen)["size"]; sent {
		t.Error("size was sent to a vendor that does not accept one")
	}
}

func TestAUrlAnswerIsFetchedRatherThanRefused(t *testing.T) {
	// dall-e-3's shape: a short-lived link instead of bytes.
	pic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nfrom-a-url"))
	}))
	defer pic.Close()

	srv, _ := apiServer(t, func(_ map[string]any, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"url": pic.URL + "/x.png"}}})
	})
	eng := &apiImages{
		id: "openai", baseURL: srv.URL, apiKey: "k", model: "dall-e-3",
		spec: apiImageSpecs["openai"], client: srv.Client(),
	}
	out := filepath.Join(t.TempDir(), "a.png")
	if err := eng.Generate(context.Background(), "a cat", Request{}, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got, _ := os.ReadFile(out)
	if !strings.Contains(string(got), "from-a-url") {
		t.Errorf("the linked picture did not land: %q", got)
	}
}

func TestAFailureBodyReachesTheUserThroughApierr(t *testing.T) {
	srv, _ := apiServer(t, func(_ map[string]any, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided: sk-****"}}`))
	})
	eng := &apiImages{
		id: "openai", baseURL: srv.URL, apiKey: "bad", model: "gpt-image-1",
		spec: apiImageSpecs["openai"], client: srv.Client(),
	}
	dir := t.TempDir()
	err := eng.Generate(context.Background(), "a cat", Request{}, filepath.Join(dir, "a.png"))
	if err == nil {
		t.Fatal("a 401 was accepted")
	}
	// apierr's 401 sentence, not the vendor's body — which would quote the
	// redacted key back at the user.
	if !strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "sk-") {
		t.Errorf("the 401 was not turned into the calm sentence: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Error("a failed call left a file behind")
	}
}

func TestAnEmptyAnswerIsAnErrorRatherThanAnEmptyFile(t *testing.T) {
	for _, body := range []string{`{"data":[]}`, `{"data":[{}]}`, `not json at all`} {
		srv, _ := apiServer(t, func(_ map[string]any, w http.ResponseWriter) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		})
		eng := &apiImages{
			id: "openai", baseURL: srv.URL, apiKey: "k", model: "gpt-image-1",
			spec: apiImageSpecs["openai"], client: srv.Client(),
		}
		dir := t.TempDir()
		if err := eng.Generate(context.Background(), "x", Request{}, filepath.Join(dir, "a.png")); err == nil {
			t.Errorf("accepted %q", body)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("%q left a file behind", body)
		}
	}
}

func TestGeminiReadsTheInlinePartAndQuotesARefusal(t *testing.T) {
	pic := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\ngemini"))
	got, err := geminiInlineImage([]byte(`{"candidates":[{"content":{"parts":[
		{"text":"here you go"},
		{"inlineData":{"mimeType":"image/png","data":"` + pic + `"}}]}}]}`))
	if err != nil {
		t.Fatalf("geminiInlineImage: %v", err)
	}
	if !strings.HasPrefix(string(got), "\x89PNG") {
		t.Errorf("the inline picture did not come back: %q", got)
	}

	// A refusal is a 200 whose only part is the model's own sentence. That
	// sentence is the only place the reason exists, so it has to reach the user
	// rather than becoming "no image".
	_, err = geminiInlineImage([]byte(`{"candidates":[{"content":{"parts":[{"text":"I can't draw that."}]}}]}`))
	if err == nil {
		t.Fatal("a text-only answer was treated as success")
	}
	if !strings.Contains(err.Error(), "I can't draw that.") {
		t.Errorf("the model's own reason was dropped: %v", err)
	}
}

func TestEveryCloudRowSaysThePromptLeavesTheMachine(t *testing.T) {
	// §31: a cloud vendor is only ever reached because the user picked it by
	// name, and the row that offers it must say what that costs. Checked for
	// all of them at once so a vendor added later cannot skip it.
	for _, d := range Catalog() {
		if len(d.Binaries) > 0 {
			continue // a local engine sends nothing anywhere
		}
		if !strings.Contains(d.Install, "คลาวด์") && !strings.Contains(d.Install, "เซิร์ฟเวอร์") && !strings.Contains(d.Install, "ปลายทาง") {
			t.Errorf("cloud row %q does not say the prompt leaves the machine: %q", d.ID, d.Install)
		}
	}
}

func TestEveryCatalogRowHasARuntime(t *testing.T) {
	// The switch in newEngine and the catalog are two lists that must not
	// drift. A row nobody can build is a picker entry that fails on click.
	for _, d := range Catalog() {
		_, err := newEngine(d, Options{})
		// A missing key is a legitimate answer here — it means the row was
		// recognised and refused for a reason the user can act on. "no runtime"
		// is the failure this test is for.
		if err != nil && strings.Contains(err.Error(), "ยังไม่มีตัวรัน") {
			t.Errorf("catalog row %q has no runtime", d.ID)
		}
	}
}
