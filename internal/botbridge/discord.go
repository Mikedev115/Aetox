package botbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type discordFrame struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  *int64          `json:"s"`
	T  string          `json:"t"`
}

type discordMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
	Content   string `json:"content"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
}

func listenDiscord(ctx context.Context, token string, handler Handler) error {
	if token == "" {
		return fmt.Errorf("Discord bot token is missing")
	}
	botID, err := discordBotID(ctx, token)
	if err != nil {
		return err
	}

	var gateway struct {
		URL string `json:"url"`
	}
	if err := getJSON(ctx, discordAPIURL+"/gateway/bot", token, &gateway); err != nil {
		return err
	}
	if gateway.URL == "" {
		return fmt.Errorf("Discord did not return a gateway URL")
	}
	u, err := url.Parse(gateway.URL)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("v", "10")
	q.Set("encoding", "json")
	u.RawQuery = q.Encode()
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), http.Header{"User-Agent": []string{"Aetox"}})
	if err != nil {
		return err
	}
	defer conn.Close()

	var hello discordFrame
	if err := conn.ReadJSON(&hello); err != nil {
		return err
	}
	if hello.Op != 10 {
		return fmt.Errorf("Discord gateway expected HELLO, got op %d", hello.Op)
	}
	var helloData struct {
		HeartbeatInterval float64 `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(hello.D, &helloData); err != nil {
		return err
	}

	var writeMu sync.Mutex
	write := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}
	// GUILD_MESSAGES + DIRECT_MESSAGES. We deliberately do not request the
	// privileged MESSAGE_CONTENT intent: Discord includes content in DMs and in
	// guild messages that mention the bot, and those are precisely the two
	// message shapes accepted below.
	if err := write(map[string]any{"op": 2, "d": map[string]any{
		"token": token, "intents": 512 | 4096,
		"properties": map[string]string{"os": "windows", "browser": "aetox", "device": "aetox"},
	}}); err != nil {
		return err
	}

	var seqMu sync.Mutex
	var seq *int64
	var heartbeatACK atomic.Bool
	heartbeatACK.Store(true)
	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go func() {
		ticker := time.NewTicker(time.Duration(helloData.HeartbeatInterval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-heartbeatDone:
				return
			case <-ticker.C:
				// A missing ACK means this is a zombie connection. Closing it wakes
				// the read loop and lets the engine reconnect instead of leaving a
				// bot that looks online but receives nothing.
				if !heartbeatACK.Swap(false) {
					_ = conn.Close()
					return
				}
				seqMu.Lock()
				current := seq
				seqMu.Unlock()
				_ = write(map[string]any{"op": 1, "d": current})
			}
		}
	}()

	for ctx.Err() == nil {
		var frame discordFrame
		if err := conn.ReadJSON(&frame); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if frame.S != nil {
			seqMu.Lock()
			value := *frame.S
			seq = &value
			seqMu.Unlock()
		}
		if frame.Op == 7 || frame.Op == 9 {
			return fmt.Errorf("Discord gateway requested reconnect")
		}
		if frame.Op == 11 {
			heartbeatACK.Store(true)
			continue
		}
		if frame.Op == 1 {
			seqMu.Lock()
			current := seq
			seqMu.Unlock()
			_ = write(map[string]any{"op": 1, "d": current})
			continue
		}
		if frame.Op != 0 || frame.T != "MESSAGE_CREATE" {
			continue
		}
		var event discordMessage
		if json.Unmarshal(frame.D, &event) != nil || event.Author.Bot || strings.TrimSpace(event.Content) == "" {
			continue
		}
		text := strings.TrimSpace(event.Content)
		if event.GuildID != "" {
			mentionA, mentionB := "<@"+botID+">", "<@!"+botID+">"
			if !strings.Contains(text, mentionA) && !strings.Contains(text, mentionB) {
				continue
			}
			text = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, mentionA, ""), mentionB, ""))
		}
		if text == "" {
			continue
		}
		channelID := event.ChannelID
		msg := Message{
			Platform: Discord, Endpoint: Discord + ":" + channelID,
			Sender: event.Author.Username, Text: text,
			Reply: func(callCtx context.Context, text string) error {
				return discordSend(callCtx, token, channelID, text)
			},
			Typing: func(callCtx context.Context) error {
				return discordPost(callCtx, token, "/channels/"+channelID+"/typing", nil)
			},
		}
		handler(ctx, msg)
	}
	return nil
}

func discordBotID(ctx context.Context, token string) (string, error) {
	var me struct {
		ID string `json:"id"`
	}
	if err := getJSON(ctx, discordAPIURL+"/users/@me", token, &me); err != nil {
		return "", err
	}
	return me.ID, nil
}

func discordSend(ctx context.Context, token, channelID, text string) error {
	for _, part := range splitText(text, 1900) {
		if err := discordPost(ctx, token, "/channels/"+channelID+"/messages", map[string]any{
			"content": part, "allowed_mentions": map[string]any{"parse": []string{}},
		}); err != nil {
			return err
		}
	}
	return nil
}

func discordPost(ctx context.Context, token, path string, payload any) error {
	var body io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordAPIURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("User-Agent", "Aetox")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := apiHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord API: HTTP %d", resp.StatusCode)
	}
	return nil
}
