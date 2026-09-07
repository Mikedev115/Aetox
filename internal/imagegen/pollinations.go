package imagegen

// Pollinations: the keyless cloud row, and the reason this package could be
// proven end to end before any credential existed. The whole protocol is one
// GET whose path IS the prompt —
//
//	https://image.pollinations.ai/prompt/<url-escaped prompt>?width=&height=&model=
//
// — answering with the image bytes directly rather than with JSON wrapping a
// URL or a base64 blob. There is nothing to install, nothing to sign into and
// no GPU involved, which is what makes it the catalog default: the first
// picture works on a machine that has only just been unzipped.
//
// What it costs is that the prompt leaves the machine, which the Descriptor's
// Install text states in the user's own language rather than leaving to be
// discovered. Same disclosure the `edge` and `gtts` rows carry on the speech
// side, for the same §31 reason.
//
// **The content-type check below is not ceremony.** A keyless public endpoint
// is exactly the kind that answers 200 with an HTML notice — rate limited, in
// maintenance, prompt refused — and writing that body to disk under a .jpg
// would hand the user a "picture" that every viewer refuses to open, with no
// error anywhere to explain it. A 200 that is not an image is a failure here.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type pollinations struct {
	base  string
	model string
	// client is swapped in tests. Production uses one with a generous timeout:
	// a diffusion model behind a free endpoint routinely takes half a minute,
	// and a caller that gives up at ten seconds produces a broken feature
	// rather than a fast one. Cancellation still belongs to the context.
	client *http.Client
}

const pollinationsBase = "https://image.pollinations.ai/prompt/"

func newPollinations(desc Descriptor, opts Options) (Engine, error) {
	model, err := resolveNamedModel(desc, opts.Model)
	if err != nil {
		return nil, err
	}
	return &pollinations{
		base:   pollinationsBase,
		model:  model,
		client: &http.Client{Timeout: 3 * time.Minute},
	}, nil
}

func (*pollinations) ID() string { return "pollinations" }

// Ext is .jpg because that is what the endpoint answers with. It is checked
// against the response's own content type on every call rather than trusted —
// see mustBeImage.
func (*pollinations) Ext() string { return ".jpg" }

func (p *pollinations) Generate(ctx context.Context, prompt string, req Request, outPath string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("ไม่มีคำสั่งวาด — บอกด้วยว่าจะให้วาดอะไร")
	}

	// PathEscape, not QueryEscape: the prompt is a path segment here, and
	// QueryEscape writes a space as "+" which a path reads as a literal plus.
	endpoint := p.base + url.PathEscape(prompt)
	q := url.Values{}
	if p.model != "" {
		q.Set("model", p.model)
	}
	if req.Width > 0 {
		q.Set("width", strconv.Itoa(req.Width))
	}
	if req.Height > 0 {
		q.Set("height", strconv.Itoa(req.Height))
	}
	if req.Seed != 0 {
		q.Set("seed", strconv.Itoa(req.Seed))
	}
	// The endpoint stamps its own mark on the picture unless asked not to.
	// A watermark the user never asked for is the vendor's advertising on the
	// user's work, and it is one query parameter to decline.
	q.Set("nologo", "true")
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := doWithRetry(p.client, httpReq)
	if err != nil {
		return fmt.Errorf("ต่อ Pollinations ไม่ได้: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Capped: an error page can be a whole document, and the user needs
		// the vendor's sentence, not its stylesheet.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("Pollinations ตอบ %d: %s", resp.StatusCode, oneLine(string(body)))
	}
	if err := mustBeImage(resp.Header.Get("Content-Type")); err != nil {
		return err
	}

	// Written through a temp file in the same directory, then renamed. A
	// half-downloaded picture at the real path is worse than none: the file
	// card in chat would offer it, the pane would try to draw it, and the
	// failure would look like the app rather than the transfer.
	tmp, err := os.CreateTemp(dirOf(outPath), ".imagegen-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below has succeeded

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("อ่านภาพจาก Pollinations ไม่ครบ: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, outPath); err != nil {
		return err
	}
	return nil
}

// mustBeImage rejects a 200 whose body is not a picture. See the file comment:
// a keyless endpoint's maintenance page is served with status 200 and would
// otherwise be written to disk wearing an image's extension.
func mustBeImage(contentType string) error {
	mime := strings.TrimSpace(strings.ToLower(contentType))
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if strings.HasPrefix(mime, "image/") {
		return nil
	}
	if mime == "" {
		return fmt.Errorf("Pollinations ตอบกลับมาโดยไม่บอกชนิดไฟล์ — ยังไม่เขียนลงดิสก์")
	}
	return fmt.Errorf("Pollinations ตอบกลับเป็น %s ไม่ใช่รูป — น่าจะติดลิมิตหรือปิดปรับปรุงอยู่", mime)
}

// oneLine flattens a vendor's error body into something that fits on a line
// next to the status code.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	if s == "" {
		return "(ไม่มีรายละเอียด)"
	}
	return s
}

// dirOf is filepath.Dir without the import, kept local so the temp file lands
// beside its destination — a temp in the OS temp dir can be on another volume,
// where rename is not atomic and sometimes not permitted at all.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
