package imagegen

// The OpenAI-shaped images endpoint (POST /images/generations), worn by more
// than one catalog row — the same arrangement internal/tts/openai.go uses for
// speech, and for the same reason: these vendors disagree about which models
// they serve and what sizes they accept, and about nothing else. One
// implementation, one spec per row.
//
// **Keys are not new keys.** Every row here reads the credential the user
// already entered on ตั้งค่า > โมเดล for that same provider (credentials.ProviderAPIKey),
// so turning on picture-making for OpenAI costs the user nothing they have not
// already done. The per-provider base URL override is honored too, which is
// what lets the OpenAI row serve any self-hosted server speaking the same API.
//
// **Two response shapes, and we ask for neither.** gpt-image-1 always answers
// with base64 in `b64_json`; dall-e-3 answers with a short-lived `url` unless
// told otherwise — and the `response_format` parameter that would settle it is
// REJECTED by gpt-image-1. So nothing is sent, and both shapes are accepted on
// the way back: base64 is decoded, a url is fetched. Sending the parameter
// would make one of the two models fail on a flag that exists to help.
//
// **One row wears a sign-in instead of a key.** The ChatGPT backend that serves
// a Codex subscription (oauth.CodexBaseURL) answers the very same
// POST /images/generations — measured 13 ก.ย. 2026 on a Plus account: 200,
// `b64_json`, a 1254×1254 PNG in 15 s, and the x-codex-* quota headers on the
// way back, so the picture is drawn from the SAME 5-hour / weekly windows the
// chat draws from, not from a separate credit. What differs from the OpenAI
// row is the credential (a bearer token that expires and is refreshed, plus
// the account-id header the sign-in carries) and nothing on the wire. Two
// things that backend does NOT do: it ignores `model` — a made-up name draws
// just the same, the server picks its own picture model — and it ignores
// `size`. So the row offers no model and declares no size support, which is
// what makes both facts true on the page as well as on the wire.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/apierr"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/credentials"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// apiImageSpec is one vendor's wearing of the shared wire format.
type apiImageSpec struct {
	defaultBase string
	provider    string   // credential-store name — the models page's own row id
	envVars     []string // key fallbacks
	vendor      string   // shown in errors
	official    string   // keyless calls allowed only off this host
	// signIn names the oauth provider whose sign-in this row rides on, for the
	// one vendor that is reached with a session token rather than a key. Empty
	// for every key-bearing row. When set, envVars and the key store are not
	// consulted at all: the token comes from oauth.TokenSource, the base URL
	// from the sign-in's own Endpoint, and the request carries oauth.Headers.
	signIn string
	// supportsSize says whether this vendor accepts a `size` parameter at all.
	// xAI's image endpoint does not, and sending one there is a 400 on a
	// request that would otherwise have worked — so width/height are dropped
	// with the picture still made, rather than turned into a failure.
	supportsSize bool
	// ext is what this vendor's bytes usually are. A hint only: image_make
	// re-reads the file it wrote and renames it to whatever the bytes really
	// are, so a wrong guess here costs nothing.
	ext string
}

var apiImageSpecs = map[string]apiImageSpec{
	"openai": {
		defaultBase:  "https://api.openai.com/v1",
		provider:     "openai",
		envVars:      []string{"OPENAI_API_KEY"},
		vendor:       "OpenAI",
		official:     "https://api.openai.com/v1",
		supportsSize: true,
		ext:          ".png",
	},
	"xai": {
		defaultBase: "https://api.x.ai/v1",
		provider:    "xai",
		envVars:     []string{"XAI_API_KEY"},
		vendor:      "xAI",
		official:    "https://api.x.ai/v1",
		// grok's image endpoint takes a prompt and a count, and nothing else.
		supportsSize: false,
		ext:          ".jpg",
	},
	// The three below are read straight off internal/provider/catalog.go: each
	// declares RuntimeOpenAICompatible with these exact base URLs, which is
	// what makes one runtime enough for all of them. `official` is the same
	// string, so pointing any of them at a local server drops the key check
	// the way the OpenAI row already allows.
	"alibaba": {
		defaultBase: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		provider:    "alibaba",
		envVars:     []string{"DASHSCOPE_API_KEY", "QWEN_API_KEY"},
		vendor:      "Alibaba Cloud",
		official:    "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		// DashScope spells a size "1024*1024". Sending the OpenAI spelling is
		// a 400, so none is sent and the vendor's default stands.
		supportsSize: false,
		ext:          ".png",
	},
	"zai": {
		defaultBase:  "https://api.z.ai/api/paas/v4",
		provider:     "zai",
		envVars:      []string{"ZAI_API_KEY"},
		vendor:       "Z.ai",
		official:     "https://api.z.ai/api/paas/v4",
		supportsSize: true,
		ext:          ".png",
	},
	"modelscope": {
		defaultBase: "https://api-inference.modelscope.cn/v1",
		provider:    "modelscope",
		envVars:     []string{"MODELSCOPE_API_KEY"},
		vendor:      "ModelScope",
		official:    "https://api-inference.modelscope.cn/v1",
		// Its roster moves and the accepted sizes move with it.
		supportsSize: false,
		ext:          ".png",
	},
	// The ChatGPT subscription. Same wire as the openai row; see the package
	// comment for what was measured and what the backend ignores.
	"codex": {
		defaultBase: oauth.CodexBaseURL,
		provider:    "codex",
		vendor:      "ChatGPT",
		official:    oauth.CodexBaseURL,
		signIn:      "codex",
		// Sent 1024x1024, got 1254x1254: the backend chooses. Not sending one
		// is the honest version of that.
		supportsSize: false,
		ext:          ".png",
	},
}

type apiImages struct {
	id      string
	baseURL string
	apiKey  string
	model   string
	spec    apiImageSpec
	client  *http.Client
	// The sign-in trio, set only for a spec with signIn. tokenSource is asked
	// per request (a token near expiry is renewed on the way); tokenRefresh is
	// the one retry a 401 gets, the same discipline internal/model applies —
	// the recorded expiry is the client's belief and the 401 is the fact.
	tokenSource  func(context.Context) (string, error)
	tokenRefresh func(context.Context) (string, error)
	headers      map[string]string
}

func newAPIImages(desc Descriptor, opts Options) (Engine, error) {
	spec, ok := apiImageSpecs[desc.ID]
	if !ok {
		return nil, fmt.Errorf("engine %q อยู่ในรายการแต่ยังไม่มีตัวรัน", desc.ID)
	}
	model, err := resolveNamedModel(desc, opts.Model)
	if err != nil {
		return nil, err
	}
	if spec.signIn != "" {
		return newSignedInImages(desc, spec, model)
	}
	key := credentials.ProviderAPIKey(spec.provider, spec.envVars...)
	base := strings.TrimRight(config.ProviderBaseURL(spec.provider), "/")
	if base == "" {
		base = spec.defaultBase
	}
	// No key is allowed only off the official host: a local clone usually
	// wants none, the real service always does — and failing here beats
	// failing after a turn has already been spent composing the prompt.
	if key == "" && base == spec.official {
		// The fact only. Where to go is a BUTTON on the page that shows this
		// (Settings.svelte) — a sentence spelling out a path the app could
		// simply walk you down is a worse version of the same thing (owner,
		// 7 ก.ย.: "แทนที่จะเขียนคำอธิบายแบบนี้ สร้างทางลัดให้ก็จบละ").
		return nil, fmt.Errorf("ยังไม่มี API key ของ %s", spec.vendor)
	}
	return &apiImages{
		id: desc.ID, baseURL: base, apiKey: key, model: model, spec: spec,
		client: &http.Client{Timeout: 3 * time.Minute},
	}, nil
}

// newSignedInImages builds the row that rides a sign-in. No sign-in is the
// same kind of fact as no key: stated plainly, and the page that shows it
// carries the button that walks the user to the models page (Settings.svelte).
func newSignedInImages(desc Descriptor, spec apiImageSpec, model string) (Engine, error) {
	source := oauth.TokenSource(spec.signIn)
	if source == nil {
		return nil, fmt.Errorf("ยังไม่ได้ล็อกอิน %s", spec.vendor)
	}
	// The sign-in's own endpoint first, then the per-provider override the
	// models page allows, then the catalog default — the order internal/model
	// resolves the chat's base URL in, so the picture goes where the chat goes.
	base := strings.TrimRight(oauth.Endpoint(spec.signIn), "/")
	if base == "" {
		base = strings.TrimRight(config.ProviderBaseURL(spec.provider), "/")
	}
	if base == "" {
		base = spec.defaultBase
	}
	return &apiImages{
		id: desc.ID, baseURL: base, model: model, spec: spec,
		client:       &http.Client{Timeout: 3 * time.Minute},
		tokenSource:  source,
		tokenRefresh: oauth.RefreshSource(spec.signIn),
		headers:      oauth.Headers(spec.signIn),
	}, nil
}

func (a *apiImages) ID() string  { return a.id }
func (a *apiImages) Ext() string { return a.spec.ext }

func (a *apiImages) Generate(ctx context.Context, prompt string, req Request, outPath string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("ไม่มีคำสั่งวาด — บอกด้วยว่าจะให้วาดอะไร")
	}

	body := map[string]any{"prompt": prompt, "n": 1}
	// A row with no model concept sends none, rather than "model": "" — which
	// a strict vendor would refuse and a lax one would silently accept, and
	// either way says something the row does not mean.
	if a.model != "" {
		body["model"] = a.model
	}
	if a.spec.supportsSize && req.Width > 0 && req.Height > 0 {
		// Sent verbatim. Which sizes are legal is the vendor's list and it
		// changes; guessing it here would mean refusing a size that works.
		body["size"] = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}
	payload, _ := json.Marshal(body)

	resp, err := a.post(ctx, payload)
	if err != nil {
		return err
	}
	// Whatever the answer was, the picture came out of a rate-limit window,
	// and a 429 states the same headers a 200 does — the moment the number
	// matters most. NoteQuotas is a no-op for a provider that states none.
	model.NoteQuotas(a.spec.provider, resp)
	if resp.StatusCode == http.StatusUnauthorized && a.tokenRefresh != nil {
		// One renewal, then the request again with whatever the token source
		// yields next. If the renewal itself fails, the original 401 stands
		// and is shown: the sign-in is gone and the user must make it again.
		if _, refreshErr := a.tokenRefresh(ctx); refreshErr == nil {
			resp.Body.Close()
			if resp, err = a.post(ctx, payload); err != nil {
				return err
			}
			model.NoteQuotas(a.spec.provider, resp)
		}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Through apierr, never verbatim: a 401 body quotes the redacted key
		// back and a 500 can be a whole HTML page.
		return apierr.HTTP(a.spec.vendor, resp.StatusCode, raw)
	}

	var answer struct {
		Data []struct {
			B64  string `json:"b64_json"`
			URL  string `json:"url"`
			Text string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &answer); err != nil {
		return fmt.Errorf("%s ตอบมาเป็นรูปแบบที่อ่านไม่ออก", a.spec.vendor)
	}
	if len(answer.Data) == 0 {
		return fmt.Errorf("%s ตอบสำเร็จแต่ไม่มีรูปมาด้วย", a.spec.vendor)
	}

	first := answer.Data[0]
	var pic []byte
	switch {
	case first.B64 != "":
		pic, err = base64.StdEncoding.DecodeString(first.B64)
		if err != nil {
			return fmt.Errorf("%s ส่ง base64 ที่ถอดไม่ได้", a.spec.vendor)
		}
	case first.URL != "":
		pic, err = a.fetch(ctx, first.URL)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s ตอบสำเร็จแต่ไม่มีทั้งไบต์และลิงก์ของรูป", a.spec.vendor)
	}
	if len(pic) == 0 {
		return fmt.Errorf("%s ส่งรูปเปล่ามา", a.spec.vendor)
	}
	return os.WriteFile(outPath, pic, 0o644)
}

// post sends one generation request, with whichever credential this row
// carries: a fixed key, or a token asked for fresh each time — so a token the
// store renewed between two calls is the one that goes out.
func (a *apiImages) post(ctx context.Context, payload []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/images/generations", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for name, value := range a.headers {
		httpReq.Header.Set(name, value)
	}
	switch {
	case a.tokenSource != nil:
		token, err := a.tokenSource(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s sign-in: %w", a.spec.vendor, err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
	case a.apiKey != "":
		httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	resp, err := doWithRetry(a.client, httpReq)
	if err != nil {
		return nil, fmt.Errorf("ต่อ %s ไม่ได้: %w", a.spec.vendor, err)
	}
	return resp, nil
}

// fetch pulls the picture from the short-lived URL dall-e-3 hands back instead
// of bytes. Same client and the same context, so a cancelled turn cancels this
// too — the download is part of the call, not a background errand.
func (a *apiImages) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("โหลดรูปจาก %s ไม่ได้: %w", a.spec.vendor, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s ให้ลิงก์รูปมาแต่โหลดแล้วได้ %d — ลิงก์แบบนี้หมดอายุเร็ว", a.spec.vendor, resp.StatusCode)
	}
	if err := mustBeImage(resp.Header.Get("Content-Type")); err != nil {
		return nil, err
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}
