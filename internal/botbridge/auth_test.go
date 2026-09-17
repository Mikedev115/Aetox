package botbridge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func isolateBridge(t *testing.T) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
}

func TestTelegramConnectValidatesStoresAndPairsOneEndpoint(t *testing.T) {
	isolateBridge(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			fmt.Fprint(w, `{"ok":true,"result":{"id":42,"username":"aetox_bot","first_name":"Aetox"}}`)
		case strings.HasSuffix(r.URL.Path, "/getWebhookInfo"):
			fmt.Fprint(w, `{"ok":true,"result":{"url":""}}`)
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()
	old := telegramBaseURL
	telegramBaseURL = server.URL
	t.Cleanup(func() { telegramBaseURL = old })

	account, err := Connect(context.Background(), Telegram, "123:secret")
	if err != nil {
		t.Fatal(err)
	}
	if account.Login != "@aetox_bot" || Token(Telegram) != "123:secret" {
		t.Fatalf("account/token = %+v / %q", account, Token(Telegram))
	}
	code, paired := Pairing(Telegram)
	if paired || len(code) != 6 {
		t.Fatalf("pairing = %q, %v", code, paired)
	}
	if allowed, _ := Authorize(Telegram, "telegram:7", "hello"); allowed {
		t.Fatal("unpaired endpoint was admitted")
	}
	allowed, pairedNow := Authorize(Telegram, "telegram:7", "/pair "+code)
	if !allowed || !pairedNow {
		t.Fatal("valid one-time code did not pair")
	}
	if allowed, _ := Authorize(Telegram, "telegram:8", "/pair "+code); allowed {
		t.Fatal("pairing code was reusable by a second endpoint")
	}
	if allowed, _ := Authorize(Telegram, "telegram:7", "hello"); !allowed {
		t.Fatal("paired endpoint was refused")
	}
}

func TestDiscordConnectUsesBotAuthorization(t *testing.T) {
	isolateBridge(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bot discord-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		fmt.Fprint(w, `{"id":"99","username":"Aetox","global_name":"Aetox Bot"}`)
	}))
	defer server.Close()
	old := discordAPIURL
	discordAPIURL = server.URL
	t.Cleanup(func() { discordAPIURL = old })

	account, err := Connect(context.Background(), Discord, "discord-secret")
	if err != nil {
		t.Fatal(err)
	}
	if account.Login != "@Aetox" || account.Name != "Aetox Bot" {
		t.Fatalf("account = %+v", account)
	}
}

func TestTelegramConnectRefusesExistingWebhook(t *testing.T) {
	isolateBridge(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			fmt.Fprint(w, `{"ok":true,"result":{"id":42,"username":"aetox_bot"}}`)
			return
		}
		fmt.Fprint(w, `{"ok":true,"result":{"url":"https://example.com/hook"}}`)
	}))
	defer server.Close()
	old := telegramBaseURL
	telegramBaseURL = server.URL
	t.Cleanup(func() { telegramBaseURL = old })

	_, err := Connect(context.Background(), Telegram, "123:secret")
	if err == nil || !strings.Contains(err.Error(), "webhook") {
		t.Fatalf("err = %v, want actionable webhook refusal", err)
	}
	if Token(Telegram) != "" {
		t.Fatal("token was stored even though long polling cannot run")
	}
}

func TestSplitTextKeepsUnicodeAndLimitsChunks(t *testing.T) {
	parts := splitText(strings.Repeat("ไทย ", 20), 15)
	if len(parts) < 2 {
		t.Fatal("long text was not split")
	}
	for _, part := range parts {
		if len([]rune(part)) > 15 {
			t.Fatalf("chunk has %d runes: %q", len([]rune(part)), part)
		}
	}
}
