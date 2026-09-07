package imagegen

// The OpenAI-shaped images endpoint (POST /images/generations), worn by more
// than one catalog row — the same arrangement internal/tts/openai.go uses for
// speech, and for the same reason: these vendors disagree about which models
// they serve and what sizes they accept, and about nothing else. One
// implementation, one spec per row.
//
// **Keys are not new keys.** Every row here reads the credential the user
// already entered on ตั้งค่า > โมเดล for that same provider (config.ProviderAPIKey),
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
)

// apiImageSpec is one vendor's wearing of the shared wire format.
type apiImageSpec struct {
	defaultBase string
	provider    string   // credential-store name — the models page's own row id
	envVars     []string // key fallbacks
	vendor      string   // shown in errors
	official    string   // keyless calls allowed only off this host
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
}

type apiImages struct {
	id      string
	baseURL string
	apiKey  string
	model   string
	spec    apiImageSpec
	client  *http.Client
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
	key := config.ProviderAPIKey(spec.provider, spec.envVars...)
	base := strings.TrimRight(config.ProviderBaseURL(spec.provider), "/")
	if base == "" {
		base = spec.defaultBase
	}
	// No key is allowed only off the official host: a local clone usually
	// wants none, the real service always does — and failing here beats
	// failing after a turn has already been spent composing the prompt.
	if key == "" && base == spec.official {
		return nil, fmt.Errorf("ยังไม่มี API key ของ %s — ใส่ได้ที่ ตั้งค่า > โมเดล > %s แล้วการสร้างภาพจะใช้ key เดียวกัน", spec.vendor, spec.vendor)
	}
	return &apiImages{
		id: desc.ID, baseURL: base, apiKey: key, model: model, spec: spec,
		client: &http.Client{Timeout: 3 * time.Minute},
	}, nil
}

func (a *apiImages) ID() string  { return a.id }
func (a *apiImages) Ext() string { return a.spec.ext }

func (a *apiImages) Generate(ctx context.Context, prompt string, req Request, outPath string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("ไม่มีคำสั่งวาด — บอกด้วยว่าจะให้วาดอะไร")
	}

	body := map[string]any{"model": a.model, "prompt": prompt, "n": 1}
	if a.spec.supportsSize && req.Width > 0 && req.Height > 0 {
		// Sent verbatim. Which sizes are legal is the vendor's list and it
		// changes; guessing it here would mean refusing a size that works.
		body["size"] = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}
	payload, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/images/generations", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ต่อ %s ไม่ได้: %w", a.spec.vendor, err)
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
