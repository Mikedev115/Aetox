package turn

import "testing"

// A timeline row has to be readable at a glance: the raw JSON truncated at 40
// characters used to cut mid-key, which named neither the file nor the tool's
// target.
func TestToolCallSubject(t *testing.T) {
	cases := []struct{ name, args, want string }{
		{"write", `{"path":"internal/skill/edit.go","content":"package skill\n..."}`, "internal/skill/edit.go"},
		{"edit alt key", `{"file_path":"desktop/app.go","find":"a","replace":"b"}`, "desktop/app.go"},
		{"fetch", `{"url":"https://ollama.com"}`, "https://ollama.com"},
		{"bash", `{"command":"go build ./..."}`, "go build ./..."},
		// A row reading only "ถามคุณ" has forgotten the one fact worth keeping
		// about a question (owner, 7 ก.ย.). The answer rides the result event;
		// the question is right here in the arguments.
		{"ask_user", `{"question":"เอาทางไหนดี","options":["a","b"]}`, "เอาทางไหนดี"},
		{"no nameable arg", `{"limit":10}`, ""},
		{"blank value falls through", `{"path":"  ","url":"https://x.dev"}`, "https://x.dev"},
		{"not json keeps old behaviour", `{"path":"truncated`, `{"path":"truncated`},
	}
	for _, c := range cases {
		if got := toolCallSubject(c.args); got != c.want {
			t.Errorf("%s: toolCallSubject(%q) = %q, want %q", c.name, c.args, got, c.want)
		}
	}
}
