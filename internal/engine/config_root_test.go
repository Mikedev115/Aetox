package engine

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/mode"
)

func TestConversationRebuildKeepsItsProjectRoot(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Engine) error
	}{
		{
			name: "plain rebuild",
			change: func(a *Engine) error {
				_, err := a.SetStance(mode.StancePlan.String())
				return err
			},
		},
		{
			name: "config setting",
			change: func(a *Engine) error {
				return a.SetSpeechModel("")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := bootDeskApp(t, "assistant")
			projectRoot := a.cur().cfg.SandboxRoot

			// This is the state after the new-chat template has been retargeted
			// while an existing conversation remains open in its own project.
			a.cfg.SandboxRoot = t.TempDir()
			if a.cfg.SandboxRoot == projectRoot {
				t.Fatal("test needs distinct current and template roots")
			}

			if err := tt.change(a); err != nil {
				t.Fatalf("change setting: %v", err)
			}
			if got := a.cur().cfg.SandboxRoot; got != projectRoot {
				t.Fatalf("conversation root changed to %q, want %q", got, projectRoot)
			}
		})
	}
}
