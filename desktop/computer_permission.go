package main

// Who may be driven, decided in advance rather than in the middle of a turn.
//
// The guard (computer_guard.go) answers "is this the kind of thing Aetox does at
// all". This file answers "did this user choose this program". They stay
// separate because the first is a rule the project made and the second is a
// decision only the person at the machine can make.
//
// **Nothing here asks anything.** The first version raised a card on the first
// touch of a program, remembered the yes, and showed it in Settings. The owner
// changed it on 9 ก.ย. after watching it work — "แล้วต้องเลือกด้วยดิ จะให้ตัวไหน
// ควบคุม" — and the new rule is better, not merely different:
//
//	A card raised mid-turn asks somebody to decide in a hurry, about a program
//	they were not thinking about, while an agent waits. Everything in that
//	moment argues for yes — the work is half done, the question is in the way,
//	and the cost of no is starting over. Codex and Claude Desktop both ask
//	exactly then. Choosing on a page whose whole subject is this, with nothing
//	waiting, is a decision rather than a reflex.
//
// What survives from the first version is where the answer LIVES. A yes is a
// `safety.PermissionRule` in permissions.json — §4.3 of the direction doc, and
// the same rule desktop/workspace.go:186 states for folders: a right that is
// not on a list the user can see is a right the panel has stopped describing.
// It is enforced by the gate already running in internal/turn; this file writes
// the rule and never becomes a gate of its own.

import (
	"fmt"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/safety"
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

// requireReachApp answers whether this program may be driven at all.
//
// It does NOT ask. That is the change the owner made on 9 ก.ย. after seeing the
// first version work — *"แล้วต้องเลือกด้วยดิ จะให้ตัวไหนควบคุม"* — and it is a
// better rule than the one it replaced, for a reason worth writing down.
//
// A card raised mid-turn asks a person to decide, in a hurry, about a program
// they were not thinking about, while an agent waits on the answer. Everything
// about that moment argues for yes: the work is half done, the question is in
// the way, and the cost of no is starting again. Both products this was studied
// against ask exactly then, and both of them are asking at the worst possible
// time.
//
// Choosing in advance, on a page whose whole subject is this, is a decision
// rather than a reflex. So the answer here is a refusal that names the page,
// and the granting happens in AllowComputerApp below, from a list of windows
// the user is looking at with nothing waiting on them.
func requireReachApp(t reachTarget) error {
	key := exeKey(t.Exe)
	if key == "" {
		return refuse("ไม่รู้ว่าหน้าต่างนี้เป็นของโปรแกรมไหน จึงไม่แตะ", "")
	}
	if reachGranted(t.Exe) {
		return nil
	}
	return refuse(
		fmt.Sprintf("ยังไม่ได้อนุญาตให้ Aetox ใช้ %s", t.Label()),
		fmt.Sprintf("ผู้ใช้เป็นคนเลือกเองว่าจะให้คุมโปรแกรมไหน — บอกเขาให้ไปที่ ตั้งค่า → การใช้คอมพิวเตอร์ แล้วเพิ่ม %q เข้าไปในรายการ", key))
}

// ComputerAppRow is one program the settings page can offer to allow: a window
// the user has open right now, with what Aetox would be able to do to it.
type ComputerAppRow struct {
	Name    string `json:"name"`    // the program key the grant is written against
	Title   string `json:"title"`   // the window title the user is looking at
	Allowed bool   `json:"allowed"` // already on the list
	Blocked string `json:"blocked"` // non-empty when this kind is never driven
	Warn    string `json:"warn"`    // non-empty when a yes here reaches the whole machine
}

// OpenComputerApps lists what the user could allow, built from the windows
// actually open. A list of every installed program would be a catalogue; a list
// of what is on screen right now is a decision they can make by looking.
func (a *App) OpenComputerApps() []ComputerAppRow {
	out := []ComputerAppRow{}
	windows, err := reachListWindows()
	if err != nil {
		return out
	}
	seen := map[string]bool{}
	for _, w := range windows {
		key := exeKey(w.Exe)
		tier, note := appTier(w.Exe)
		// Never-driven kinds are left out entirely rather than shown greyed:
		// Aetox's own windows are the main case, and offering to allow one is
		// offering something that will never be honoured.
		if key == "" || tier == tierNever || seen[key] {
			continue
		}
		seen[key] = true
		row := ComputerAppRow{Name: key, Title: w.Title, Allowed: reachGranted(w.Exe)}
		if tier == tierElsewhere {
			// Shown, not hidden, and told which tool does it properly. A user
			// looking for Chrome in this list and not finding it learns nothing;
			// one who finds it with "Aetox uses its own browser for this" learns
			// the shape of the product.
			row.Blocked = note
		}
		if tier == tierWarned {
			row.Warn = note
		}
		out = append(out, row)
	}
	return out
}

// AllowComputerApp puts one program on the list.
func (a *App) AllowComputerApp(name string) error {
	key := exeKey(name)
	if key == "" {
		return fmt.Errorf("no program named")
	}
	// The tier is re-checked here rather than trusted from the row the user
	// clicked: the page is a picture of a moment, and the rule is the rule.
	if tier, note := appTier(key); tier == tierNever || tier == tierElsewhere {
		return fmt.Errorf("%s", note)
	}
	if err := config.UpdatePermissions(func(cfg *safety.PermissionConfig) error {
		rule := reachRuleFor(key)
		for _, r := range cfg.Rules {
			if strings.EqualFold(r.Tool, rule.Tool) && r.Pattern == rule.Pattern {
				return nil
			}
		}
		cfg.Rules = append(cfg.Rules, rule)
		return nil
	}); err != nil {
		return err
	}
	a.emitEvent("computer:apps", a.GrantedComputerApps())
	return nil
}
