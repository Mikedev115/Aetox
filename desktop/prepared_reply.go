package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
)

// The answer the user has not typed yet, typed for them (owner's call,
// 7 ก.ย.: *"มันคือการเตรียมคำต่อล่วงหน้า"*).
//
// A turn that ends by handing the user a fork — "ทำ a หรือ b" — ends with the
// user's next message already decided in everything but the typing. They read
// two options, pick the better one, and then have to say so in words. This
// writes those words into the composer, dimmed, before they reach the keyboard:
// Tab takes them, Tab again swaps to the other option's wording, typing
// anything makes them vanish. Nothing is ever sent, and nothing is ever put in
// the box that a keystroke did not ask for.
//
// **Two rules decide when this is allowed to spend anything**, and both come
// from the owner: *"มันไม่ควรเป็นช้อยบังคับ… บางงานมันไม่จำเป็นต้องสร้างคำตอบรอ"*.
//
//  1. offersChoice reads the finished answer with no model at all. A turn that
//     ends "แก้ไฟล์ให้แล้ว เทสต์ผ่าน" leaves no decision to prepare for, and no
//     call is made — not made and discarded, not made at all.
//  2. What survives that gate costs ONE small call, after the answer is already
//     on screen, off the turn's critical path.
//
// **Why a second call rather than a tool the answering model calls itself.**
// That was the first design and it is the one this app has already run: this is
// exactly the shape of `suggest_task`, which was cut on 2026-08-29 after 270
// tokens of tool block in every request on every desk bought zero calls in
// 6,253 runs (app.go, everySessionSkills). A tool is a standing tax on every
// request in the hope the model reaches for it; this is a charge on the turns
// that actually ended in a question, decided by a function that can be read and
// tested. The precedent is close enough that repeating it would have been a
// choice, not an accident.

// PreparedReply is one wording the user might send next, ready to be taken.
// Text is what would land in the composer verbatim — first person, the user's
// own voice, because it IS the user's message.
type PreparedReply struct {
	Text string `json:"text"`
}

// preparedReplyOn reads the switch. Positive-by-absence, unlike SkillTuneAuto
// next door, and the difference is where the spend lands: that one drafts in
// the background on its own schedule, which is what "absent means off" is for,
// while this one only ever spends on a turn the user just watched finish, and
// shows the result in the one place they are already looking. A feature nobody
// can see is a feature nobody switches on.
func preparedReplyOn() bool {
	pref, ok, _ := config.LoadModelPreference()
	if !ok {
		return true
	}
	return !pref.PreparedReplyOff
}

// PreparedReplyOn reports the switch for the settings page.
func (a *App) PreparedReplyOn() bool { return preparedReplyOn() }

// SetPreparedReplyOn persists it.
func (a *App) SetPreparedReplyOn(on bool) error {
	return config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.PreparedReplyOff = !on
		return nil
	})
}

// choiceMarks are the ways an answer hands the decision back, in the languages
// this app is written in. Matched against the TAIL of the answer only (see
// offersChoice), so a question asked in passing halfway through a long
// explanation does not count — what matters is how the turn ENDED.
//
// Deliberately short and deliberately explicit. The temptation here is a
// cleverer test — sentence parsing, a classifier, asking a model — and every
// version of that costs the thing this gate exists to protect: it has to be
// free, and it has to be readable by whoever next wonders why their turn did or
// did not prepare anything.
var choiceMarks = []string{
	// Thai. The particles that turn a sentence into a question, plus the two
	// phrasings that offer a fork without one ("เอาแบบไหน", "หรือจะ").
	"ไหม", "มั้ย", "หรือเปล่า", "รึเปล่า", "หรือไม่",
	"แบบไหน", "อันไหน", "ทางไหน", "อย่างไหน", "ยังไงดี", "ไงดี",
	"หรือจะ", "หรือว่า", "ให้ผมทำ", "จะเอา", "เลือก",
	// English.
	"which ", "would you like", "do you want", "should i", "shall i",
	"let me know", "your call", "or should", "prefer",
}

// offersChoice reports whether a finished answer leaves the user something to
// decide — the whole gate between "one small call" and "no call at all".
//
// It reads the END of the answer and nothing else, because that is where a turn
// hands control back. The distinction it has to draw is between a question that
// is being ASKED and one that was asked and then answered: an answer opening
// "คำถามคือไฟล์ไหนที่พัง? คำตอบคือ…" and going on to explain for a paragraph is
// not waiting for anybody.
//
// Two joined rules, and both are needed:
//
//   - the last three non-empty lines, so a closing question still counts when it
//     sits under the bulleted list of what was done — which is how most answers
//     in this app are shaped — and so a question does not stop counting because
//     the two options were spelled out as bullets beneath it.
//   - capped to the last 300 runes of that, so one long unbroken Thai paragraph
//     (which is one "line", and routinely 400+ runes) is judged on how it ends
//     rather than on anything it happened to say on the way there.
//
// Runes throughout: 300 bytes of Thai is 100 characters, a third of the window
// this is meant to be.
func offersChoice(answer string) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	lines := []string{}
	for _, line := range strings.Split(answer, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	tail := []rune(strings.Join(lines, "\n"))
	if len(tail) > 300 {
		tail = tail[len(tail)-300:]
	}
	lower := strings.ToLower(string(tail))
	if strings.ContainsAny(lower, "?？") {
		return true
	}
	for _, mark := range choiceMarks {
		if strings.Contains(lower, mark) {
			return true
		}
	}
	return false
}

// maybePrepareReply is the entry point every finished turn calls. It decides
// nothing itself beyond the gates, and it never blocks the turn: the answer is
// already on screen by the time this runs, and the wording arrives when it
// arrives.
func (a *App) maybePrepareReply(conv *conversation, question, answer string) {
	if conv == nil || !preparedReplyOn() || !preparedReplies {
		return
	}
	if strings.TrimSpace(answer) == "" || !offersChoice(answer) {
		return
	}
	// One at a time across the whole app. Two chats finishing together would
	// otherwise open two calls, and the second one's wording would land in a
	// composer belonging to the first — the event is stamped with a session, but
	// the spend is not worth the race.
	if !a.preparing.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer a.preparing.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		replies, err := a.prepareReplies(ctx, conv, question, answer)
		if err != nil {
			debuglog.Msg("prepared reply: %v", err)
			return
		}
		if len(replies) == 0 {
			return
		}
		// A turn that started while the wording was being written has taken the
		// composer somewhere else: the user has already said their next thing,
		// and dropping a prepared answer to the previous question into the box
		// they are typing in is the one behaviour this feature must never have.
		if a.turnRunningIn(conv.id) {
			return
		}
		a.emitEvent("composer:prepared", sessionEvent[[]PreparedReply]{SessionID: conv.id, Data: replies})
	}()
}

// preparedReplies is the test seam, matching autoTuneSkills next door: a unit
// test that finishes a turn ending in a question would otherwise reach for a
// real model. False in the test harness, true in the app.
var preparedReplies = true

const prepareInstructions = `คุณกำลังช่วย "เตรียมคำ" ให้ผู้ใช้ ไม่ใช่ตอบคำถาม

ด้านล่างคือบทสนทนาที่เพิ่งจบหนึ่งรอบ ถ้าคำตอบของผู้ช่วยทิ้งทางเลือกหรือคำถามไว้ให้ผู้ใช้ตัดสินใจ
ให้เขียน "ประโยคที่ผู้ใช้จะพิมพ์ตอบ" ของแต่ละทางเลือก แล้วเรียกเครื่องมือ prepared_reply

กติกา
- เขียนในมุมของผู้ใช้ (เป็นคนสั่ง) ไม่ใช่มุมผู้ช่วย เช่น "เอาทางแรก แต่ขอให้กรองก่อน" ไม่ใช่ "ผมจะทำทางแรกให้"
- ภาษาเดียวกับที่ผู้ใช้ใช้
- สั้น หนึ่งบรรทัด พิมพ์เองแล้วส่งได้ทันที
- เรียงจากทางที่ควรเลือกที่สุดก่อน มากสุด 3 อัน
- ถ้าคำตอบไม่ได้ทิ้งอะไรให้ตัดสินใจ ส่ง replies เป็นลิสต์ว่าง`

// preparedReplyTool forces the answer through a tool call rather than asking for
// JSON in prose — the same lesson skilltune_gen.go writes up: a model that wants
// to explain itself will put the explanation and the JSON in one string, and one
// tool with ToolChoice "required" is the one shape both wire formats honour.
var preparedReplyTool = model.ToolDefinition{
	Type: "function",
	Function: model.ToolFunction{
		Name:        "prepared_reply",
		Description: "ส่งประโยคที่ผู้ใช้น่าจะพิมพ์ตอบ เรียงจากทางที่ควรเลือกที่สุด",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{
				"replies":{
					"type":"array",
					"maxItems":3,
					"items":{"type":"string"},
					"description":"ประโยคของผู้ใช้ หนึ่งบรรทัดต่อทางเลือก ลิสต์ว่างถ้าไม่มีอะไรให้ตัดสินใจ"
				}
			},
			"required":["replies"]
		}`),
	},
}

// prepareReplies makes the one call. Through oneShotProvider, so the wording is
// written by the model the user is already paying for and already talking to —
// a cheaper second model would answer in a different voice than the
// conversation it is finishing a sentence for.
func (a *App) prepareReplies(ctx context.Context, conv *conversation, question, answer string) ([]PreparedReply, error) {
	p, modelName, err := a.oneShotProviderFor(conv)
	if err != nil {
		return nil, err
	}
	// Capped, and the answer from its end: what the user has to decide is at the
	// bottom of it, and a long turn's opening paragraphs buy nothing here.
	resp, err := p.Complete(ctx, model.Request{
		Model: modelName,
		Messages: []model.Message{{Role: model.RoleUser, Content: fmt.Sprintf(
			"%s\n\n=== ผู้ใช้ถาม ===\n%s\n\n=== ผู้ช่วยตอบ ===\n%s",
			prepareInstructions, lastRunes(question, 600), lastRunes(answer, 1500))}},
		Tools:      []model.ToolDefinition{preparedReplyTool},
		ToolChoice: "required",
		MaxTokens:  400,
	})
	if err != nil {
		return nil, err
	}
	raw := ""
	if len(resp.ToolCalls) > 0 {
		raw = resp.ToolCalls[0].Function.Arguments
	} else {
		raw = extractJSONObject(resp.Text)
	}
	if raw == "" {
		return nil, nil
	}
	var out struct {
		Replies []string `json:"replies"`
	}
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil, fmt.Errorf("the prepared reply did not parse")
	}
	return cleanReplies(out.Replies), nil
}

// cleanReplies is the last gate before the wording reaches a composer. Every
// rule here exists because the box on the other side is the user's own message:
// a blank one would offer a Tab that types nothing, a duplicate would make the
// second Tab press look broken, and a paragraph would arrive as a wall of dim
// text over a control that is one line tall.
func cleanReplies(in []string) []PreparedReply {
	out := []PreparedReply{}
	seen := map[string]bool{}
	for _, text := range in {
		// One line: a model that ignored "สั้น หนึ่งบรรทัด" and wrote a plan
		// gets its first line taken rather than the whole thing dropped.
		if nl := strings.IndexByte(text, '\n'); nl >= 0 {
			text = text[:nl]
		}
		text = strings.TrimSpace(text)
		if text == "" || seen[text] {
			continue
		}
		if len([]rune(text)) > 220 {
			continue
		}
		seen[text] = true
		out = append(out, PreparedReply{Text: text})
		if len(out) == 3 {
			break
		}
	}
	return out
}

// lastRunes keeps the END of a text, which is the opposite of every other
// truncation in this file's neighbourhood and is the point: the decision an
// answer hands back is in its closing lines.
func lastRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return "…" + string(r[len(r)-max:])
}
