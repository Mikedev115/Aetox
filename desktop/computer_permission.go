package main

// Asking once, and writing the answer down.
//
// This is the second door — the guard (computer_guard.go) answers "is this the
// kind of thing Aetox does at all", and this one answers "does this user want it
// done to this program". They are separate because a card offered for something
// that would be refused afterwards is the failure askWorkspaceWiden names:
// "a question whose yes cannot be honoured trains the user to distrust the
// question."
//
// The shape is copied from askWorkspaceWiden deliberately, including the part
// both rivals do differently. Codex and Claude Desktop both scope an app
// approval to the session — Claude's expires when the session ends, Codex's
// lives in a per-app allowlist you can only see by going looking for it. Ours
// persists and shows up in Settings, because the house already argued this out:
//
//	"There is deliberately no 'allow once': a right that is not on the list is
//	 a right the panel has stopped describing."  — desktop/workspace.go:186
//
// A session-scoped grant is an "allow once" with a longer fuse. It disappears
// without being revoked, which means the user never learns what they granted,
// and it re-asks later for something they already decided — the two failure
// modes of a permission prompt, in one design.
//
// The rule written is safety.PermissionRule{Tool: "computer_*", Pattern:
// "<exe>*"}, which is exactly the shape §4.3 of the direction doc specified,
// and it is enforced by the gate that already runs in internal/turn — this file
// writes the rule and never becomes a fourth gate of its own.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/safety"
)

const (
	reachAllow  = "อนุญาต / Allow"
	reachRefuse = "ไม่ / No"
)

// computerToolPrefix is the tool-name half of every rule this file writes. The
// per-action names (computer_read, computer_click, …) all match it, so one yes
// covers the program rather than the verb: a user deciding "yes, Aetox may
// drive Notepad" is not deciding it four more times for click, type and close.
const computerToolPrefix = "computer_*"

// reachRuleFor builds the rule a yes becomes. Pattern is the program key with a
// trailing star because toolCallToArgs puts the program first in the args it
// hands safety, and everything after it is the action's own detail.
func reachRuleFor(exePath string) safety.PermissionRule {
	return safety.PermissionRule{
		Tool:    computerToolPrefix,
		Pattern: exeKey(exePath) + "*",
		Action:  safety.PermissionAllow,
	}
}

// reachGranted reports whether the user has already said yes to this program.
func reachGranted(exePath string) bool {
	key := exeKey(exePath)
	if key == "" {
		return false
	}
	cfg, err := config.LoadPermissions()
	if err != nil {
		return false
	}
	action, matched := cfg.Resolve("computer_read", []string{key})
	return matched && action == safety.PermissionAllow
}

// GrantedComputerApps lists the programs the user has said yes to, for the
// settings page. The list is the whole point of persisting: a grant nobody can
// see is a grant nobody can take back.
func (a *App) GrantedComputerApps() []string {
	out := []string{}
	cfg, err := config.LoadPermissions()
	if err != nil {
		return out
	}
	seen := map[string]bool{}
	for _, r := range cfg.Rules {
		if !strings.EqualFold(strings.TrimSpace(r.Tool), computerToolPrefix) {
			continue
		}
		if safety.NormalizePermissionAction(string(r.Action)) != safety.PermissionAllow {
			continue
		}
		name := strings.TrimSuffix(strings.TrimSpace(r.Pattern), "*")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// RevokeComputerApp takes a program back off the list.
func (a *App) RevokeComputerApp(name string) error {
	key := exeKey(name)
	if key == "" {
		return fmt.Errorf("no program named")
	}
	return config.UpdatePermissions(func(cfg *safety.PermissionConfig) error {
		kept := cfg.Rules[:0]
		for _, r := range cfg.Rules {
			if strings.EqualFold(strings.TrimSpace(r.Tool), computerToolPrefix) &&
				strings.TrimSuffix(strings.TrimSpace(r.Pattern), "*") == key {
				continue
			}
			kept = append(kept, r)
		}
		cfg.Rules = kept
		return nil
	})
}

// askReachApp puts the card up and, on a yes, writes the grant.
//
// Returns nil when the program may be driven — either because it already was
// granted or because the user just said so — and a worded refusal otherwise.
// Never returns a bare boolean: the caller is a tool action that has to report
// what happened, and "the user said no" and "there was nobody to ask" are
// different sentences.
func (a *App) askReachApp(ctx context.Context, conv *conversation, t reachTarget) error {
	key := exeKey(t.Exe)
	if key == "" {
		return refuse("ไม่รู้ว่าหน้าต่างนี้เป็นของโปรแกรมไหน จึงไม่ขออนุญาตให้", "")
	}
	if reachGranted(t.Exe) {
		return nil
	}

	// No window, no question — the same test emitEvent makes before it reaches
	// the Wails runtime, and the same one askWorkspaceWiden makes. A headless
	// run has nobody to ask, and parking a tool call on an answer that can
	// never arrive is worse than a refusal the model can report.
	if a.emit == nil && a.ctx == nil {
		return refuse(
			fmt.Sprintf("ยังไม่ได้อนุญาตให้ Aetox ใช้ %s และตอนนี้ไม่มีหน้าต่างให้ถาม", t.Label()),
			"")
	}

	question := fmt.Sprintf(
		"งานนี้ต้องใช้หน้าต่างของโปรแกรมอื่น\n\n**%s**\n\nให้ Aetox อ่านและกดในโปรแกรมนี้ไหม (ถอนได้ทีหลังที่ ตั้งค่า → การใช้คอมพิวเตอร์)",
		t.Label())
	if warn := broadReachWarning(t.Exe); warn != "" {
		// The extra sentence a program with machine-wide reach earns. Both
		// rivals show one on the same list, and the reason is that a yes here
		// is worth more than a yes to a text editor — a card that reads the
		// same for both is quietly lying about one of them.
		question += "\n\n⚠️ " + warn
	}

	ch, err := a.beginUserQuestion(conv, question, []string{reachAllow, reachRefuse})
	if err != nil {
		return refuse(
			"มีคำถามอื่นค้างอยู่ในแชตนี้ จึงยังขออนุญาตเรื่องนี้ไม่ได้",
			"ตอบคำถามนั้นก่อน แล้วสั่งใหม่อีกครั้ง")
	}
	defer a.endUserQuestion(conv)

	select {
	case answer := <-ch:
		if answer != reachAllow {
			return refuse(fmt.Sprintf("ผู้ใช้ไม่อนุญาตให้ใช้ %s", t.Label()), "")
		}
	case <-ctx.Done():
		return refuse("เทิร์นถูกหยุดก่อนได้คำตอบ", "")
	}

	if err := config.UpdatePermissions(func(cfg *safety.PermissionConfig) error {
		rule := reachRuleFor(t.Exe)
		for _, r := range cfg.Rules {
			if strings.EqualFold(r.Tool, rule.Tool) && r.Pattern == rule.Pattern {
				return nil // already there; a second yes is not a second rule
			}
		}
		cfg.Rules = append(cfg.Rules, rule)
		return nil
	}); err != nil {
		// The user said yes and the machine would not write it down. Refusing
		// is the honest answer: proceeding would drive the program on a grant
		// that does not exist anywhere, which is the session-scoped grant this
		// design rejected, arrived at by accident.
		return refuse(
			fmt.Sprintf("บันทึกสิทธิ์ไม่สำเร็จ จึงยังไม่ใช้ %s: %v", t.Label(), err),
			"")
	}
	a.emitEvent("computer:apps", a.GrantedComputerApps())
	return nil
}
