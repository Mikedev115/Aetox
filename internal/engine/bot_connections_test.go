package engine

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/botbridge"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/prompt"
)

// The settings page must describe the route the listener actually uses. This
// test guards the seam between the generic connection catalog and the engine:
// changing the bot desk, identity name, or default model cannot leave the UI
// confidently naming a different assistant.
func TestConnectionsReportTheRealBotAssistantRoute(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	if err := config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.HeadNames = map[string]string{"assistant": "มะลิ"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	a := &Engine{cfg: config.Config{ModelProvider: "openai", ModelName: "gpt-test"}}
	rows := a.Connections()
	channels := 0
	for _, row := range rows {
		if !row.Channel {
			if row.ChannelDesk != "" || row.ChannelAssistant != "" || row.ChannelModel != "" {
				t.Fatalf("non-channel %s was given a bot route: %+v", row.ID, row)
			}
			continue
		}
		channels++
		if row.ChannelDesk != connectedBotDesk || row.ChannelAssistant != "มะลิ" {
			t.Fatalf("%s route = %q/%q, want %q/มะลิ", row.ID, row.ChannelDesk, row.ChannelAssistant, connectedBotDesk)
		}
		if row.ChannelProvider != "openai" || row.ChannelModel != "gpt-test" {
			t.Fatalf("%s model = %q/%q", row.ID, row.ChannelProvider, row.ChannelModel)
		}
	}
	if channels != 2 {
		t.Fatalf("channel rows = %d, want Telegram and Discord", channels)
	}
}

func TestBotConversationEntersOnlyTheAssistantDesk(t *testing.T) {
	a := bootFreshApp(t)
	conv, err := a.newBotConversation(botbridge.Telegram)
	if err != nil {
		t.Fatal(err)
	}
	if got := conv.desk.DeskName(); got != connectedBotDesk {
		t.Fatalf("bot desk = %q, want %q", got, connectedBotDesk)
	}
	if conv.chair != "" {
		t.Fatalf("bot opened specialist chair %q; want the main assistant", conv.chair)
	}
	if conv.cfg.ModelProvider != a.cfg.ModelProvider || conv.cfg.ModelName != a.cfg.ModelName {
		t.Fatalf("bot config = %s/%s, new-chat config = %s/%s",
			conv.cfg.ModelProvider, conv.cfg.ModelName, a.cfg.ModelProvider, a.cfg.ModelName)
	}
	if conv.transport != prompt.TransportTelegram {
		t.Fatalf("bot transport = %q, want Telegram", conv.transport)
	}
	messages := conv.agent.ContextMessages()
	if len(messages) == 0 || !strings.Contains(messages[0].Content, "Conversation channel: Telegram") {
		t.Fatalf("bot system prompt has no Telegram delivery layer: %+v", messages)
	}
	if ordinary := a.cur().agent.ContextMessages(); len(ordinary) > 0 && strings.Contains(ordinary[0].Content, "Conversation channel:") {
		t.Fatal("the bot-only transport layer leaked into the ordinary Aetox conversation")
	}
}

func TestBotConversationRejectsAnUnknownTransport(t *testing.T) {
	a := bootFreshApp(t)
	if _, err := a.newBotConversation("carrier-pigeon"); err == nil {
		t.Fatal("an unknown channel created a conversation with invented context")
	}
}

func TestBotTransportSurvivesSessionReload(t *testing.T) {
	a := bootFreshApp(t)
	conv, err := a.newBotConversation(botbridge.Discord)
	if err != nil {
		t.Fatal(err)
	}
	if !a.openTurn(conv, SessionMessage{Role: "user", Text: "hello from Discord", Time: "now"}) {
		t.Fatal("bot turn was not persisted")
	}

	// Drop the live engine so LoadSession must reconstruct the coordinate from
	// SQLite rather than succeeding because the same object is still in memory.
	a.letGoOf(conv)
	if _, err := a.LoadSession(conv.id); err != nil {
		t.Fatal(err)
	}
	if got := a.cur().transport; got != prompt.TransportDiscord {
		t.Fatalf("reloaded transport = %q, want Discord", got)
	}
	state, err := a.OpenSession(conv.id, DeskFilter{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if state.Transport != string(prompt.TransportDiscord) {
		t.Fatalf("open state transport = %q, want Discord", state.Transport)
	}
}
