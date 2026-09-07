package imagegen

// Gemini, which does not have an images endpoint at all: a picture comes back
// from the ordinary generateContent call, as an inline part, when the request
// asks for an image modality. That is why this is its own file rather than one
// more row in apiImageSpecs — the URL, the request body and the place the bytes
// arrive are all different, and only the credential is shared.
//
// internal/tts/gemini.go reaches the same API for sound on the same key. The
// two files are deliberately not merged: what they have in common is one line
// of auth, and what they differ in is everything that follows it.

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

const geminiImageBase = "https://generativelanguage.googleapis.com/v1beta"

type geminiImages struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func newGeminiImages(desc Descriptor, opts Options) (Engine, error) {
	model, err := resolveNamedModel(desc, opts.Model)
	if err != nil {
		return nil, err
	}
	key := config.ProviderAPIKey("gemini", "GEMINI_API_KEY", "GOOGLE_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ยังไม่มี API key ของ Gemini — ใส่ได้ที่ ตั้งค่า > โมเดล > Gemini แล้วการสร้างภาพจะใช้ key เดียวกัน")
	}
	base := strings.TrimRight(config.ProviderBaseURL("gemini"), "/")
	if base == "" {
		base = geminiImageBase
	}
	return &geminiImages{
		baseURL: base, apiKey: key, model: model,
		client: &http.Client{Timeout: 3 * time.Minute},
	}, nil
}

func (*geminiImages) ID() string { return "gemini" }

// PNG is what the inline part carries. A hint only — image_make renames the
// file to match the bytes it actually finds.
func (*geminiImages) Ext() string { return ".png" }

func (g *geminiImages) Generate(ctx context.Context, prompt string, req Request, outPath string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("ไม่มีคำสั่งวาด — บอกด้วยว่าจะให้วาดอะไร")
	}

	// There is no width/height on this API. Rather than drop the request
	// silently, the size is asked for in the prompt itself, which is the only
	// channel the model has for it — and is honest about being a request
	// rather than a setting.
	if req.Width > 0 && req.Height > 0 {
		prompt = fmt.Sprintf("%s (aspect ratio %d:%d)", prompt, req.Width, req.Height)
	}

	body := map[string]any{
		"contents": []any{map[string]any{
			"parts": []any{map[string]any{"text": prompt}},
		}},
		// The whole reason a chat endpoint returns a picture. Both modalities
		// are listed because the model may narrate alongside the image, and
		// asking for image alone is rejected on some of these models.
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}
	payload, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/models/%s:generateContent", g.baseURL, g.model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// The header form, not ?key= in the query: a URL with a credential in it
	// ends up in logs and error messages.
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := doWithRetry(g.client, httpReq)
	if err != nil {
		return fmt.Errorf("ต่อ Gemini ไม่ได้: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apierr.HTTP("Gemini", resp.StatusCode, raw)
	}

	pic, err := geminiInlineImage(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, pic, 0o644)
}

// geminiInlineImage digs the first image part out of a generateContent answer.
//
// A refusal arrives here as a perfectly successful response whose only part is
// text — the model explaining why it will not draw the thing. That is the case
// worth naming rather than reporting as "no image": the user needs the model's
// own sentence, which is the only place the reason exists.
func geminiInlineImage(raw []byte) ([]byte, error) {
	var answer struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text"`
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &answer); err != nil {
		return nil, fmt.Errorf("Gemini ตอบมาเป็นรูปแบบที่อ่านไม่ออก")
	}
	if len(answer.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini ตอบสำเร็จแต่ไม่มีคำตอบมาด้วย")
	}

	var said []string
	for _, c := range answer.Candidates {
		for _, part := range c.Content.Parts {
			if data := strings.TrimSpace(part.InlineData.Data); data != "" {
				pic, err := base64.StdEncoding.DecodeString(data)
				if err != nil {
					return nil, fmt.Errorf("Gemini ส่ง base64 ที่ถอดไม่ได้")
				}
				if len(pic) == 0 {
					return nil, fmt.Errorf("Gemini ส่งรูปเปล่ามา")
				}
				return pic, nil
			}
			if t := strings.TrimSpace(part.Text); t != "" {
				said = append(said, t)
			}
		}
	}
	if len(said) > 0 {
		return nil, fmt.Errorf("Gemini ไม่ได้วาดให้ และตอบมาว่า: %s", oneLine(strings.Join(said, " ")))
	}
	return nil, fmt.Errorf("Gemini ตอบสำเร็จแต่ไม่มีรูปมาด้วย")
}
