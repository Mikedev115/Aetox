package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/botbridge"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/prompt"
)

// channelInbox serializes one Telegram chat or Discord channel. A platform
// listener must keep receiving while a model turn runs, but one conversation
// must never run two turns at once.
type channelInbox struct {
	messages chan botbridge.Message
	conv     *conversation
}

// connectedBotDesk is the single source of truth for both the conversation
// created below and the route reported on the Connections page. A bot channel
// is another doorway to the main assistant, never a separate assistant hidden
// in this adapter.
const connectedBotDesk = mode.Default

func (a *Engine) connectedBotAssistantName() string {
	name := strings.TrimSpace(a.HeadName(config.IdentityHeadFor(connectedBotDesk)))
	if name == "" {
		return "Aetox"
	}
	return name
}

func (a *Engine) startConnectedBots() {
	for _, platform := range []string{botbridge.Telegram, botbridge.Discord} {
		if botbridge.Token(platform) != "" {
			a.restartBot(platform)
		}
	}
}

func (a *Engine) restartBot(platform string) {
	a.stopBot(platform)
	if botbridge.Token(platform) == "" || a.isClosing() {
		return
	}
	ctx, cancel := context.WithCancel(a.engineCtx())
	a.botMu.Lock()
	if a.botCancels == nil {
		a.botCancels = map[string]context.CancelFunc{}
	}
	a.botCancels[platform] = cancel
	a.botMu.Unlock()
	go a.runBotListener(ctx, platform)
}

func (a *Engine) stopBot(platform string) {
	a.botMu.Lock()
	cancel := a.botCancels[platform]
	delete(a.botCancels, platform)
	for endpoint, inbox := range a.botInboxes {
		if strings.HasPrefix(endpoint, platform+":") {
			if inbox.conv != nil && !a.turnRunningIn(inbox.conv.id) {
				a.letGoOf(inbox.conv)
			}
			delete(a.botInboxes, endpoint)
		}
	}
	a.botMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *Engine) runBotListener(ctx context.Context, platform string) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := botbridge.Listen(ctx, platform, a.receiveBotMessage)
		if ctx.Err() != nil {
			return
		}
		debuglog.Msg("%s bot listener: %v", platform, err)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		maxBackoff := 30 * time.Second
		if platform == botbridge.Discord {
			// Discord limits fresh IDENTIFY calls. A prolonged outage must not
			// turn a helpful reconnect loop into thousands of new sessions.
			maxBackoff = 2 * time.Minute
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (a *Engine) receiveBotMessage(ctx context.Context, msg botbridge.Message) {
	allowed, pairedNow := botbridge.Authorize(msg.Platform, msg.Endpoint, msg.Text)
	if pairedNow {
		_ = msg.Reply(ctx, fmt.Sprintf("อนุญาตแชทนี้แล้วครับ จากนี้ข้อความจะคุยกับผู้ช่วยหลัก “%s” ของ Aetox จริง ผ่านหน้าผู้ช่วยเท่านั้น · ใช้ /new เพื่อเริ่มบทสนทนาใหม่", a.connectedBotAssistantName()))
		return
	}
	if !allowed {
		_ = msg.Reply(ctx, "Bot token เชื่อมแล้ว แต่แชทนี้ยังไม่ได้รับอนุญาต เพื่อกันคนอื่นที่ค้นเจอชื่อบอทเข้าถึงผู้ช่วยบนเครื่อง ให้เปิด Aetox → ของคุณ → การเชื่อมต่อ แล้วส่ง /pair ตามด้วยรหัสที่แสดง · ข้อความจะเข้าเฉพาะหน้าผู้ช่วย")
		return
	}

	a.botMu.Lock()
	if a.botInboxes == nil {
		a.botInboxes = map[string]*channelInbox{}
	}
	inbox := a.botInboxes[msg.Endpoint]
	if inbox == nil {
		inbox = &channelInbox{messages: make(chan botbridge.Message, 16)}
		a.botInboxes[msg.Endpoint] = inbox
		go a.runChannelInbox(ctx, inbox)
	}
	a.botMu.Unlock()
	select {
	case inbox.messages <- msg:
	default:
		_ = msg.Reply(ctx, "มีข้อความรออยู่หลายข้อความแล้ว กรุณารอคำตอบก่อนแล้วลองใหม่ครับ")
	}
}

func (a *Engine) runChannelInbox(ctx context.Context, inbox *channelInbox) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-inbox.messages:
			a.answerBotMessage(ctx, inbox, msg)
		}
	}
}

func (a *Engine) answerBotMessage(ctx context.Context, inbox *channelInbox, msg botbridge.Message) {
	text := strings.TrimSpace(msg.Text)
	switch strings.ToLower(text) {
	case "/start", "/help":
		_ = msg.Reply(ctx, fmt.Sprintf("ที่นี่คุยกับผู้ช่วยหลัก “%s” ของ Aetox จริง ผ่านหน้าผู้ช่วยเท่านั้น (ประวัติแยกสำหรับแชทนี้)\n/new — เริ่มบทสนทนาใหม่\n/help — ดูคำสั่ง", a.connectedBotAssistantName()))
		return
	case "/new":
		if inbox.conv != nil {
			if a.turnRunningIn(inbox.conv.id) {
				_ = msg.Reply(ctx, "กำลังตอบข้อความก่อนหน้าอยู่ รอให้เสร็จก่อนแล้วค่อยใช้ /new ครับ")
				return
			}
			a.letGoOf(inbox.conv)
			inbox.conv = nil
		}
		_ = msg.Reply(ctx, "เริ่มบทสนทนาใหม่แล้วครับ")
		return
	}
	if inbox.conv == nil {
		var err error
		inbox.conv, err = a.newBotConversation(msg.Platform)
		if err != nil {
			_ = msg.Reply(ctx, "เปิดหน้าผู้ช่วยไม่ได้ จึงไม่ส่งข้อความนี้ไปโหมดอื่นแทน: "+err.Error())
			return
		}
	}
	_ = msg.Typing(ctx)
	reply, err := a.sendMessageIn(inbox.conv, text, "")
	answer := strings.TrimSpace(reply.Text)
	if err != nil {
		if answer != "" {
			answer += "\n\n"
		}
		answer += "เกิดข้อผิดพลาด: " + err.Error()
	}
	if answer == "" {
		answer = "Aetox ยังไม่มีข้อความตอบกลับ"
	}
	if sendErr := msg.Reply(ctx, answer); sendErr != nil {
		debuglog.Msg("%s bot reply: %v", msg.Platform, sendErr)
	}
}

func (a *Engine) newBotConversation(platform string) (*conversation, error) {
	conv := newConversation()
	conv.id = newSessionID()
	conv.cfg = a.cfg
	switch platform {
	case botbridge.Telegram:
		conv.transport = prompt.TransportTelegram
	case botbridge.Discord:
		conv.transport = prompt.TransportDiscord
	default:
		return nil, fmt.Errorf("ไม่รู้จักช่องทางสนทนา %q", platform)
	}
	// External messages enter through the ordinary assistant desk, not the
	// unrestricted legacy desk or whichever specialist page is currently open.
	// Fail closed if that desk is unavailable: silently continuing with nil
	// would give an external chat broader tools than the page promises.
	desk, ok := mode.Load(connectedBotDesk)
	if !ok {
		return nil, fmt.Errorf("ไม่พบโต๊ะ %s", connectedBotDesk)
	}
	conv.desk = desk
	a.applyConfig(conv, conv.cfg)
	a.cur()
	a.convs.hold(conv)
	return conv, nil
}
