package botbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type telegramEnvelope struct {
	OK          bool             `json:"ok"`
	Description string           `json:"description"`
	Result      []telegramUpdate `json:"result"`
}

type telegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID       int64  `json:"id"`
			Type     string `json:"type"`
			Username string `json:"username"`
		} `json:"chat"`
		From struct {
			ID        int64  `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			IsBot     bool   `json:"is_bot"`
		} `json:"from"`
	} `json:"message"`
}

func listenTelegram(ctx context.Context, token string, handler Handler) error {
	if token == "" {
		return fmt.Errorf("Telegram bot token is missing")
	}
	client := &http.Client{Timeout: 40 * time.Second}
	var offset int64
	for ctx.Err() == nil {
		payload, _ := json.Marshal(map[string]any{
			"offset": offset, "timeout": 25, "allowed_updates": []string{"message"},
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramAPI(token, "getUpdates"), bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var envelope telegramEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || !envelope.OK {
			return fmt.Errorf("Telegram getUpdates: %s", envelope.Description)
		}
		for _, update := range envelope.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			m := update.Message
			if m == nil || m.From.IsBot || strings.TrimSpace(m.Text) == "" {
				continue
			}
			chatID := strconv.FormatInt(m.Chat.ID, 10)
			sender := strings.TrimSpace(m.From.Username)
			if sender == "" {
				sender = strings.TrimSpace(m.From.FirstName)
			}
			msg := Message{
				Platform: Telegram, Endpoint: Telegram + ":" + chatID,
				Sender: sender, Text: strings.TrimSpace(m.Text),
				Reply: func(callCtx context.Context, text string) error {
					return telegramSend(callCtx, token, chatID, text)
				},
				Typing: func(callCtx context.Context) error {
					return telegramCall(callCtx, token, "sendChatAction", map[string]any{"chat_id": chatID, "action": "typing"})
				},
			}
			handler(ctx, msg)
		}
	}
	return nil
}

func telegramSend(ctx context.Context, token, chatID, text string) error {
	parts := splitText(text, 4000)
	for _, part := range parts {
		if err := telegramCall(ctx, token, "sendMessage", map[string]any{
			"chat_id": chatID, "text": part, "disable_web_page_preview": true,
		}); err != nil {
			return err
		}
	}
	return nil
}

func telegramCall(ctx context.Context, token, method string, payload any) error {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramAPI(token, method), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := apiHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Telegram %s: HTTP %d", method, resp.StatusCode)
	}
	return nil
}

func splitText(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"(ไม่มีข้อความตอบกลับ)"}
	}
	var out []string
	for len([]rune(text)) > limit {
		r := []rune(text)
		cut := limit
		for cut > limit/2 && r[cut] != '\n' && r[cut] != ' ' {
			cut--
		}
		if cut <= limit/2 {
			cut = limit
		}
		out = append(out, strings.TrimSpace(string(r[:cut])))
		text = strings.TrimSpace(string(r[cut:]))
	}
	if text != "" {
		out = append(out, text)
	}
	return out
}
