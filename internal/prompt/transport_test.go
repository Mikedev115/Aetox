package prompt

import (
	"strings"
	"testing"
)

func TestNoTransportLeavesTheMainPromptUntouched(t *testing.T) {
	base := "main prompt\n"
	if got := WithTransport(base, ""); got != base {
		t.Fatalf("zero transport changed the main prompt: %q", got)
	}
	if got := WithTransport(base, "unknown"); got != base {
		t.Fatalf("unknown transport changed the main prompt: %q", got)
	}
}

func TestTelegramTransportIsASeparateDeliveryLayer(t *testing.T) {
	base := "main identity and desk prompt"
	got := WithTransport(base, TransportTelegram)
	if !strings.HasPrefix(got, base+"\n\n") {
		t.Fatal("transport replaced or entered the main prompt instead of following it")
	}
	for _, want := range []string{
		"Conversation channel: Telegram",
		"same Aetox main assistant",
		"plain text without a Markdown parse mode",
		"Ask necessary follow-up questions directly",
		"name its path",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Telegram layer is missing %q:\n%s", want, got)
		}
	}
}

func TestDiscordTransportNamesItsRealPresentationConstraints(t *testing.T) {
	got := WithTransport("main", TransportDiscord)
	for _, want := range []string{
		"Conversation channel: Discord",
		"direct message or a server channel",
		"Discord Markdown is available",
		"bot mention is removed",
		"same Aetox main assistant",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Discord layer is missing %q:\n%s", want, got)
		}
	}
}
