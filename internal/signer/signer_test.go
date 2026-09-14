package signer

import (
	"net/http"
	"net/url"
	"testing"
)

type recording struct{ got *http.Request }

func (r *recording) RoundTrip(req *http.Request) (*http.Response, error) {
	r.got = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

// A pasted key rides on the header the wire format names, read at request
// time — and the request handed in is not the one sent.
func TestKeyRidesOnTheWireFormatsHeader(t *testing.T) {
	net := &recording{}
	rt := Transport(Options{
		Provider:   "deepseek",
		WireFormat: "openai-compatible",
		Key:        func(p string) string { return "k-" + p },
	})(net)
	req, _ := http.NewRequest(http.MethodPost, "https://api.deepseek.com/v1/chat/completions", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if got := net.got.Header.Get("Authorization"); got != "Bearer k-deepseek" {
		t.Fatalf("Authorization = %q", got)
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("the caller's request was edited; it must be cloned")
	}
}

// A destination the screen does not know for the provider goes out bare —
// the key never rides to a server the engine's host picked.
func TestMayRideFalseSendsUnsigned(t *testing.T) {
	net := &recording{}
	rt := Transport(Options{
		Provider:   "deepseek",
		WireFormat: "openai-compatible",
		Key:        func(string) string { return "secret" },
		MayRide:    func(_ string, u *url.URL) bool { return u.Host == "api.deepseek.com" },
	})(net)
	req, _ := http.NewRequest(http.MethodPost, "https://evil.example/v1/chat/completions", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if got := net.got.Header.Get("Authorization"); got != "" {
		t.Fatalf("a credential rode to a stranger: %q", got)
	}
}

// A wire format with no key header sends nothing, whatever Key answers.
func TestNoHeaderNoKey(t *testing.T) {
	net := &recording{}
	rt := Transport(Options{
		Provider: "no-such-provider",
		Key:      func(string) string { return "secret" },
	})(net)
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:1234/v1/models", nil)
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if len(net.got.Header) != 0 {
		t.Fatalf("headers on a keyless wire: %v", net.got.Header)
	}
}
