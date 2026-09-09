package oauth

import (
	"context"
	"strings"
	"testing"
)

func TestCopilotMethodIsRestricted(t *testing.T) {
	method, ok := MethodFor("github-copilot")
	if !ok {
		t.Fatal("MethodFor(github-copilot) returned false")
	}
	if method.Risk != RiskRestricted {
		t.Fatalf("method.Risk = %q; want %q", method.Risk, RiskRestricted)
	}
	if method.Kind != "device" {
		t.Fatalf("method.Kind = %q; want device", method.Kind)
	}
	if !strings.Contains(method.Note, "Copilot") {
		t.Fatalf("method.Note = %q; want it to mention Copilot", method.Note)
	}
}

func TestCopilotHeadersIncludeRequiredFields(t *testing.T) {
	headers := CopilotHeaders()
	if headers["User-Agent"] != copilotUserAgent {
		t.Errorf("User-Agent = %q; want %q", headers["User-Agent"], copilotUserAgent)
	}
	if headers["Editor-Version"] != copilotEditorVersion {
		t.Errorf("Editor-Version = %q; want %q", headers["Editor-Version"], copilotEditorVersion)
	}
	if headers["Copilot-Integration-Id"] != copilotIntegration {
		t.Errorf("Copilot-Integration-Id = %q; want %q", headers["Copilot-Integration-Id"], copilotIntegration)
	}
}

func TestRefreshCopilotRequiresRefreshToken(t *testing.T) {
	_, err := refreshCopilot(context.Background(), Credential{Type: "oauth", Access: "token"})
	if err == nil {
		t.Fatal("refreshCopilot succeeded without refresh token")
	}
}
