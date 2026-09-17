// Package botbridge connects chat platforms to Aetox without making either
// platform part of the conversation engine.  It owns bot credentials, the
// small pairing gate in front of them, and the wire clients that deliver an
// incoming text message to the engine.
package botbridge

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/atrest"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

const (
	Telegram = "telegram"
	Discord  = "discord"
)

type Account struct {
	Login string
	Name  string
}

type Status struct {
	Connected bool
	Login     string
}

var apiHTTP = &http.Client{Timeout: 25 * time.Second}

func Token(platform string) string {
	cred, ok := oauth.Get(platform)
	if !ok {
		return ""
	}
	return strings.TrimSpace(cred.Key)
}

func CurrentStatus(platform string) Status {
	cred, ok := oauth.Get(platform)
	if !ok || strings.TrimSpace(cred.Key) == "" {
		return Status{}
	}
	return Status{Connected: true, Login: cred.Account}
}

func Connect(ctx context.Context, platform, token string) (Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Account{}, errors.New("ต้องใส่ bot token")
	}
	account, err := verifyToken(ctx, platform, token)
	if err != nil {
		return Account{}, err
	}
	if platform == Telegram {
		if err := telegramPollingAvailable(ctx, token); err != nil {
			return Account{}, err
		}
	}
	if err := oauth.Set(platform, oauth.Credential{
		Type: "api", Key: token, Account: account.Login,
		Label: displayName(platform) + " · " + account.Login,
	}); err != nil {
		return Account{}, fmt.Errorf("บันทึกการเชื่อมต่อ %s ไม่สำเร็จ: %w", displayName(platform), err)
	}
	if err := resetPairing(platform); err != nil {
		_ = oauth.Delete(platform)
		return Account{}, fmt.Errorf("สร้างรหัสจับคู่ไม่สำเร็จ: %w", err)
	}
	return account, nil
}

func telegramPollingAvailable(ctx context.Context, token string) error {
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			URL string `json:"url"`
		} `json:"result"`
	}
	if err := getJSON(ctx, telegramAPI(token, "getWebhookInfo"), "", &result); err != nil {
		return fmt.Errorf("ตรวจ Telegram webhook ไม่สำเร็จ: %w", err)
	}
	if strings.TrimSpace(result.Result.URL) != "" {
		return errors.New("Telegram bot นี้ตั้ง webhook ไว้ จึงใช้ getUpdates กับ Aetox พร้อมกันไม่ได้ — ลบ webhook หรือสร้าง bot ใหม่ก่อน")
	}
	return nil
}

func Verify(ctx context.Context, platform string) (Account, error) {
	token := Token(platform)
	if token == "" {
		return Account{}, fmt.Errorf("ยังไม่ได้เชื่อม %s", displayName(platform))
	}
	return verifyToken(ctx, platform, token)
}

func Disconnect(platform string) error {
	if err := oauth.Delete(platform); err != nil {
		return err
	}
	return deletePairing(platform)
}

func verifyToken(ctx context.Context, platform, token string) (Account, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	switch platform {
	case Telegram:
		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
			Result      struct {
				ID        int64  `json:"id"`
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
			} `json:"result"`
		}
		if err := getJSON(ctx, telegramAPI(token, "getMe"), "", &result); err != nil {
			return Account{}, fmt.Errorf("ติดต่อ Telegram ไม่ได้: %w", err)
		}
		if !result.OK || strings.TrimSpace(result.Result.Username) == "" {
			if result.Description == "" {
				result.Description = "token ไม่ถูกต้อง"
			}
			return Account{}, fmt.Errorf("Telegram ปฏิเสธ bot token: %s", result.Description)
		}
		return Account{Login: "@" + result.Result.Username, Name: result.Result.FirstName}, nil
	case Discord:
		var result struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Global   string `json:"global_name"`
		}
		if err := getJSON(ctx, discordAPIURL+"/users/@me", token, &result); err != nil {
			return Account{}, fmt.Errorf("Discord ปฏิเสธ bot token หรือเชื่อมต่อไม่ได้: %w", err)
		}
		if result.ID == "" || result.Username == "" {
			return Account{}, errors.New("Discord ตอบกลับบัญชีบอทไม่ครบ")
		}
		return Account{Login: "@" + result.Username, Name: strings.TrimSpace(result.Global)}, nil
	default:
		return Account{}, fmt.Errorf("unknown bot platform: %q", platform)
	}
}

func getJSON(ctx context.Context, endpoint, discordToken string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if discordToken != "" {
		req.Header.Set("Authorization", "Bot "+discordToken)
	}
	resp, err := apiHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &envelope)
		if envelope.Message != "" {
			return fmt.Errorf("%s (%d)", envelope.Message, resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func displayName(platform string) string {
	if platform == Telegram {
		return "Telegram"
	}
	return "Discord"
}

var (
	telegramBaseURL = "https://api.telegram.org"
	discordAPIURL   = "https://discord.com/api/v10"
)

func telegramAPI(token, method string) string {
	return telegramBaseURL + "/bot" + url.PathEscape(token) + "/" + method
}

// Pairing is deliberately separate from the bot token.  Knowing a public bot
// username is not permission to operate an agent that can touch the machine.
// The first chat/channel that sends the one-time code shown in Aetox becomes
// the only endpoint the bot accepts until it is reconnected.
type pairingState struct {
	Code     string `json:"code,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

var pairingMu sync.Mutex

func pairingPath() string { return filepath.Join(oauth.Root(), "bot-pairings.json") }

func loadPairings() map[string]pairingState {
	raw, err := os.ReadFile(pairingPath())
	if err != nil {
		return map[string]pairingState{}
	}
	out := map[string]pairingState{}
	if json.Unmarshal(atrest.Unprotect(raw), &out) != nil {
		return map[string]pairingState{}
	}
	return out
}

func savePairings(items map[string]pairingState) error {
	path := pairingPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, atrest.Protect(raw), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func newPairingCode() string {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	n := (uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])) % 1000000
	return fmt.Sprintf("%06d", n)
}

func resetPairing(platform string) error {
	pairingMu.Lock()
	defer pairingMu.Unlock()
	items := loadPairings()
	items[platform] = pairingState{Code: newPairingCode()}
	return savePairings(items)
}

func deletePairing(platform string) error {
	pairingMu.Lock()
	defer pairingMu.Unlock()
	items := loadPairings()
	delete(items, platform)
	return savePairings(items)
}

func Pairing(platform string) (code string, paired bool) {
	pairingMu.Lock()
	defer pairingMu.Unlock()
	items := loadPairings()
	state, ok := items[platform]
	if !ok && Token(platform) != "" {
		state = pairingState{Code: newPairingCode()}
		items[platform] = state
		_ = savePairings(items)
	}
	return state.Code, state.Endpoint != ""
}

// Authorize admits an already paired endpoint, or consumes `/pair 123456`.
// It never reveals the code over the public platform.
func Authorize(platform, endpoint, text string) (allowed, pairedNow bool) {
	pairingMu.Lock()
	defer pairingMu.Unlock()
	items := loadPairings()
	state, ok := items[platform]
	if !ok {
		return false, false
	}
	if state.Endpoint != "" {
		return state.Endpoint == endpoint, false
	}
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) != 2 || !strings.EqualFold(fields[0], "/pair") || fields[1] != state.Code {
		return false, false
	}
	state.Endpoint, state.Code = endpoint, ""
	items[platform] = state
	if savePairings(items) != nil {
		return false, false
	}
	return true, true
}
