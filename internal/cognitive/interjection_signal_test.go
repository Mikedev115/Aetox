package cognitive

import "testing"

func TestInterjectionSignalCoalescesAndDrainClearsIt(t *testing.T) {
	agent := NewAgent(AgentConfig{SystemPrompt: "test"})
	select {
	case <-agent.InterjectionSignal():
		t.Fatal("a new agent signalled without an interjection")
	default:
	}

	agent.Interject("first")
	agent.Interject("second")
	got := agent.DrainInterjections()
	if len(got) != 2 || got[0].Text != "first" || got[1].Text != "second" {
		t.Fatalf("drained interjections = %#v, want first and second in order", got)
	}
	select {
	case <-agent.InterjectionSignal():
		t.Fatal("DrainInterjections left a stale wake signal behind")
	default:
	}
}
