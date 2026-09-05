package tts

// Microsoft Edge's online voices, spoken to directly — the protocol Edge's own
// Read Aloud uses, as worked out and kept current by the edge-tts project
// (github.com/rany2/edge-tts; this file mirrors its 7.2.8 release) — with no
// edge-tts CLI, and so no Python, in between.
//
// The CLI was the first version of this file, and it cost the listener a
// second per piece: every piece started a Python interpreter (about 1 s on
// the owner's machine, measured 2026-09-06) before a byte went to Microsoft,
// and the first ฟัง of a session paid `edge-tts --list-voices` on top of that
// to find the Thai voice. The owner's kiosk (AI-Robot-Guide) never paid
// either, because its edge_tts lives in a Python process that stays up; Aetox
// has no such process, so the protocol lives here. What it buys: the first
// sound in about a second, and a cloud voice that needs nothing installed.
//
// Cloud means cloud: the TEXT goes to Microsoft and MP3 comes back. The
// descriptor's Install text says so; this file just does the work.
//
// WHAT MICROSOFT MAY CHANGE, AND WHERE IT LIVES. The trusted client token,
// the Sec-MS-GEC clock token and the Chromium version string are the
// constants below; the message shapes are in synthesize, the voice list in
// Voices. The day the service answers 403 to every request, diff these
// against edge-tts's current constants.py / drm.py / communicate.py and
// update — that is the whole maintenance story, and it has come up about
// once a year over there. TestEdgeLive (AETOX_EDGE_LIVE=1) is the check.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	edgeTrustedToken = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	// The Edge build the service is told it is talking to: in the User-Agent
	// and as Sec-MS-GEC-Version. edge-tts bumps it now and then.
	edgeChromium     = "143.0.3650.75"
	edgeOutputFormat = "audio-24khz-48kbitrate-mono-mp3"
	// The voice asked for when none was picked — edge-tts's own default.
	// desktop/voice.go picks one for the UI language before it gets here.
	edgeDefaultVoice = "en-US-EmmaMultilingualNeural"
	// The Windows file-time epoch (1601-01-01) as a Unix offset. The clock
	// token is the time on that epoch, in 100 ns ticks, rounded down to five
	// minutes.
	edgeWindowsEpoch = 11644473600
)

// The endpoints, as variables so a test can stand a fake service up.
var (
	edgeSocketURL = "wss://speech.platform.bing.com/consumer/speech/synthesize/readaloud/edge/v1"
	edgeVoicesURL = "https://speech.platform.bing.com/consumer/speech/synthesize/readaloud/voices/list"
	edgeNow       = time.Now
	// edgeTrace, when set, is told the moments of a synthesis ("dial",
	// "connected", "first-audio", "done"). Timing only; nil in production.
	edgeTrace func(event string)
)

// edgeClockSkew is the correction, in seconds, from this machine's clock to
// the service's. The clock token is good for five minutes either side, so a
// PC whose clock is off gets 403 for everything until the skew is learned
// from the Date header of the refusal. Package-wide, as in edge-tts: the skew
// is the machine's, not the voice's.
var edgeClockSkew atomic.Int64

type edgeVoice struct {
	voice string // ShortName like "th-TH-PremwadeeNeural"; "" = edgeDefaultVoice
}

func newEdge(_ Descriptor, opts Options) (Engine, error) {
	return &edgeVoice{voice: strings.TrimSpace(opts.Voice)}, nil
}

func (*edgeVoice) ID() string { return "edge" }

func (*edgeVoice) Mime() string { return "audio/mpeg" }

// Voices fetches the service's list — a JSON array with a ShortName, Locale
// and Gender per voice. The ShortName ("th-TH-PremwadeeNeural") is the ID
// config pins and what the SSML asks for.
func (e *edgeVoice) Voices(ctx context.Context) ([]Voice, error) {
	body, err := edgeGet(ctx, edgeVoicesURL, "trustedclienttoken")
	if err != nil {
		return nil, err
	}
	return parseEdgeVoices(body)
}

func parseEdgeVoices(body []byte) ([]Voice, error) {
	var rows []struct {
		ShortName string `json:"ShortName"`
		Locale    string `json:"Locale"`
		Gender    string `json:"Gender"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("รายชื่อเสียงจาก Microsoft อ่านไม่ออก (%v)", err)
	}
	voices := make([]Voice, 0, len(rows))
	for _, r := range rows {
		if r.ShortName == "" {
			continue
		}
		voices = append(voices, Voice{ID: r.ShortName, Name: r.ShortName, Lang: r.Locale, Gender: r.Gender})
	}
	return voices, nil
}

func (e *edgeVoice) Synthesize(ctx context.Context, text, outPath string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("ไม่มีข้อความให้อ่าน")
	}
	audio, err := e.synthesize(ctx, text)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, audio, 0o600)
}

// synthesize is one turn on the service's socket: a config message, an SSML
// message, then audio frames until turn.end.
func (e *edgeVoice) synthesize(ctx context.Context, text string) ([]byte, error) {
	trace := func(event string) {
		if edgeTrace != nil {
			edgeTrace(event)
		}
	}
	trace("dial")
	conn, err := edgeDial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	trace("connected")
	// A stop, or PieceTimeout, closes the socket under the read loop; that is
	// how a blocking read learns the caller has gone.
	defer context.AfterFunc(ctx, func() { conn.Close() })()

	// The timestamp is Edge's JavaScript Date string, and the "Z" appended to
	// the SSML one is not a mistake: it is what Edge sends.
	stamp := edgeNow().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)")
	config := "X-Timestamp:" + stamp + "\r\nContent-Type:application/json; charset=utf-8\r\nPath:speech.config\r\n\r\n" +
		`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"true"},"outputFormat":"` + edgeOutputFormat + `"}}}}` + "\r\n"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(config)); err != nil {
		return nil, edgeOffline(err)
	}
	ssml := "X-RequestId:" + edgeHexID() + "\r\nContent-Type:application/ssml+xml\r\nX-Timestamp:" + stamp + "Z\r\nPath:ssml\r\n\r\n" + edgeSSML(e.voiceName(), text)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(ssml)); err != nil {
		return nil, edgeOffline(err)
	}

	var audio []byte
	for {
		kind, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, edgeOffline(err)
		}
		switch kind {
		case websocket.TextMessage:
			// turn.start, response, audio.metadata (word timings), turn.end.
			// Only the last one matters here.
			headers, _ := edgeSplit(msg, bytes.Index(msg, []byte("\r\n\r\n")))
			if headers["Path"] != "turn.end" {
				continue
			}
			if len(audio) == 0 {
				// The service's way of saying no: a turn with nothing in it.
				// Seen from th-TH-PremwadeeNeural five times in seven on
				// 2026-09-05, while th-TH-NiwatNeural answered every time.
				return nil, fmt.Errorf("Microsoft ไม่ส่งเสียงกลับมาสำหรับเสียง %s — ลองเสียงอื่น", e.shortName())
			}
			trace("done")
			return audio, nil
		case websocket.BinaryMessage:
			// Two bytes of header length, the headers, then the bytes. The
			// stream's last frame is a header-only one with no Content-Type.
			if len(msg) < 2 {
				continue
			}
			n := int(msg[0])<<8 | int(msg[1])
			if 2+n > len(msg) {
				continue
			}
			headers, data := edgeSplit(msg[2:], n)
			if headers["Path"] != "audio" || len(data) == 0 {
				continue
			}
			if len(audio) == 0 {
				trace("first-audio")
			}
			audio = append(audio, data...)
		}
	}
}

// edgeDial opens the socket with the headers Edge sends, learning the clock
// skew from a 403 and dialing once more.
func edgeDial(ctx context.Context) (*websocket.Conn, error) {
	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 20 * time.Second}
	for attempt := 0; ; attempt++ {
		url := edgeSocketURL + "?" + edgeQuery("TrustedClientToken") + "&ConnectionId=" + edgeHexID()
		conn, resp, err := dialer.DialContext(ctx, url, edgeHeaders(map[string]string{
			"Pragma":        "no-cache",
			"Cache-Control": "no-cache",
			"Origin":        "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold",
		}))
		if err == nil {
			return conn, nil
		}
		if resp != nil && resp.StatusCode == http.StatusForbidden && attempt == 0 && edgeLearnSkew(resp.Header) {
			continue
		}
		if resp != nil {
			return nil, fmt.Errorf("Microsoft ปฏิเสธการเชื่อมต่อเสียง Edge (%s) — ถ้าเป็นทุกครั้ง โปรโตคอลอาจเปลี่ยน ดูหัวไฟล์ internal/tts/edge.go", resp.Status)
		}
		return nil, edgeOffline(err)
	}
}

// edgeGet is one GET with the headers Edge sends, and one retry after learning
// the clock skew from a 403. The query is rebuilt per attempt because the
// token in it is what the skew changes.
func edgeGet(ctx context.Context, base, tokenKey string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+edgeQuery(tokenKey), nil)
		if err != nil {
			return nil, err
		}
		req.Header = edgeHeaders(map[string]string{
			"Accept":         "*/*",
			"Sec-Fetch-Site": "none",
			"Sec-Fetch-Mode": "cors",
			"Sec-Fetch-Dest": "empty",
		})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, edgeOffline(err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden && attempt == 0 && edgeLearnSkew(resp.Header) {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Microsoft ตอบ %s ตอนขอรายชื่อเสียง Edge", resp.Status)
		}
		if readErr != nil {
			return nil, edgeOffline(readErr)
		}
		return body, nil
	}
}

// edgeQuery is the query every request carries: the trusted token (under the
// name each endpoint spells it), the clock token, and the version it was
// minted for.
func edgeQuery(tokenKey string) string {
	return fmt.Sprintf("%s=%s&Sec-MS-GEC=%s&Sec-MS-GEC-Version=1-%s", tokenKey, edgeTrustedToken, edgeSecMSGEC(edgeNow()), edgeChromium)
}

// edgeSecMSGEC is the clock token: SHA-256 of the time as Windows file time,
// rounded down to five minutes, followed by the trusted token, in upper-case
// hex. edge-tts worked this out (rany2/edge-tts#290); the test pins this port
// to the Python original for a fixed instant.
func edgeSecMSGEC(now time.Time) string {
	return edgeTokenAt(now.Unix() + edgeClockSkew.Load())
}

// edgeTokenAt is the token for a Unix second, skew already applied.
func edgeTokenAt(unix int64) string {
	sec := unix + edgeWindowsEpoch
	sec -= sec % 300
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d%s", sec*10_000_000, edgeTrustedToken)))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// edgeLearnSkew reads the service's clock from a refusal and remembers the
// difference. Reports whether there was a date to learn from.
func edgeLearnSkew(h http.Header) bool {
	server, err := http.ParseTime(h.Get("Date"))
	if err != nil {
		return false
	}
	edgeClockSkew.Store(server.Unix() - edgeNow().Unix())
	return true
}

// edgeHeaders are what Edge itself sends; the service is known to refuse a
// bare client. The cookie is a fresh MUID per request, as edge-tts sends.
func edgeHeaders(extra map[string]string) http.Header {
	major := strings.SplitN(edgeChromium, ".", 2)[0]
	h := http.Header{}
	h.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36 Edg/%s.0.0.0", major, major))
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Cookie", "muid="+strings.ToUpper(edgeHexID())+";")
	for k, v := range extra {
		h.Set(k, v)
	}
	return h
}

// edgeHexID is 16 random bytes as lower-case hex — a UUID without dashes,
// which is how the service wants its connection and request IDs.
func edgeHexID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// edgeSplit reads "Key:Value" lines out of the first n bytes of buf and hands
// back the rest. A negative n means no header block was found.
func edgeSplit(buf []byte, n int) (map[string]string, []byte) {
	headers := map[string]string{}
	if n < 0 || n > len(buf) {
		return headers, nil
	}
	for _, line := range bytes.Split(buf[:n], []byte("\r\n")) {
		if k, v, ok := bytes.Cut(line, []byte(":")); ok {
			headers[strings.TrimSpace(string(k))] = strings.TrimSpace(string(v))
		}
	}
	rest := buf[n:]
	rest = bytes.TrimPrefix(rest, []byte("\r\n\r\n"))
	return headers, rest
}

// edgeSSML wraps text for the service: the control characters it rejects
// replaced with spaces, the three XML specials escaped, the voice named the
// long way. Pitch, rate and volume are Edge's defaults — the pace of a read is
// the player's business, not the synthesizer's.
func edgeSSML(voice, text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r <= 8, r == 11, r == 12, r >= 14 && r <= 31:
			b.WriteRune(' ')
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'>" +
		"<voice name='" + voice + "'><prosody pitch='+0Hz' rate='+0%' volume='+0%'>" +
		b.String() + "</prosody></voice></speak>"
}

func (e *edgeVoice) shortName() string {
	if e.voice == "" {
		return edgeDefaultVoice
	}
	return e.voice
}

// voiceName is the long form the service wants: "th-TH-PremwadeeNeural"
// becomes "Microsoft Server Speech Text to Speech Voice (th-TH,
// PremwadeeNeural)". A name with a fourth part ("zh-CN-liaoning-XiaobeiNeural")
// folds it into the region, as edge-tts does — that is what Edge sends.
func (e *edgeVoice) voiceName() string {
	short := e.shortName()
	if strings.HasPrefix(short, "Microsoft Server Speech") {
		return short
	}
	parts := strings.SplitN(short, "-", 3)
	if len(parts) != 3 {
		return short
	}
	lang, region, name := parts[0], parts[1], parts[2]
	if i := strings.Index(name, "-"); i >= 0 {
		region, name = region+"-"+name[:i], name[i+1:]
	}
	return fmt.Sprintf("Microsoft Server Speech Text to Speech Voice (%s-%s, %s)", lang, region, name)
}

func edgeOffline(err error) error {
	return fmt.Errorf("เสียง Edge ไม่สำเร็จ (%v) — ตัวนี้ต้องต่อเน็ต เพราะเสียงสังเคราะห์บนคลาวด์ของ Microsoft", err)
}
