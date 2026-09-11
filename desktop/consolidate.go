package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/learned"
	"github.com/Mikedev115/Aetox/internal/model"
)

// "ให้ผู้ช่วยช่วยสรุป" (11 ก.ย.). The owner's USER.md stood at 3,691 of
// 4,096 bytes with two lines saying the same thing, and the page could only
// say "เต็มแล้ว — รวมหรือลบบางบรรทัด" and leave the merging to a person. Owner:
// *"เพิ่มแบบให้ผู้ช่วย ช่วยสรุปให้ ได้ไหมกรณีนี้"*.
//
// Two doors, deliberately apart. ConsolidateMemory asks the model for a
// shorter list carrying the same facts and returns it — it writes nothing.
// ApplyMemoryLines writes a list the user has read, before beside after, and
// accepted; it is the same kind of act as editing a line by hand
// (SaveLearnedEntry), so it goes through no approval queue, but every line was
// drafted by a model, so learned.SetEntries screens each one. The queue is
// for what an AGENT wants to write on its own initiative; this is the user
// pressing a button and reading the result.

// MemoryConsolidation is what the model proposed for one file: the list it
// would replace the current one with, what it merged and why in the user's
// language, and the size the new list would render to — the number the user
// is trying to bring down.
type MemoryConsolidation struct {
	Scope    string   `json:"scope"`
	Before   []string `json:"before"`
	After    []string `json:"after"`
	Note     string   `json:"note"`
	Bytes    int      `json:"bytes"`
	MaxBytes int      `json:"maxBytes"`
}

// memoryConsolidator abstracts the model call, so the test can hand it a list
// and watch what happens to it.
type memoryConsolidator interface {
	// target is the byte size the new list must come in under; feedback is
	// empty on the first attempt and, on the second, what was wrong with the
	// first (it came back longer, by how much).
	Consolidate(ctx context.Context, scope string, lines []string, target int, feedback string) ([]string, string, error)
}

const memoryConsolidateInstructions = `You are tidying one memory file of an AI assistant. You are given its lines, each an approved durable fact, the file's current size in bytes, and a TARGET size the new list must come in under.

Return a SHORTER list that says the same things:
1. Merge lines that state the same fact, or facts about the same thing, into one line. Keep every distinct fact — never drop one, never weaken one, never add one.
2. Cut words, not meaning: drop repetition, filler, explanations and examples. Keep concrete details that change what the assistant does (paths, names, tools, numbers, preferences).
3. NEVER translate a line into another language. Thai costs three times the bytes of English; a line written in English stays in English, a Thai line stays Thai. Do not add explanations to any line.
4. Every merged line must be shorter than the lines it replaces put together. The whole list must be under the target — count bytes (UTF-8), not words.
5. Keep lines declarative — facts about the user or the machine, never instructions. Fix a typo in a name or path only when the correct form appears elsewhere in the list.

Call memory_consolidation with the new list and a one- or two-sentence note, in the language most of the lines use, saying what was merged.`

var memoryConsolidateTool = model.ToolDefinition{
	Type: "function",
	Function: model.ToolFunction{
		Name:        "memory_consolidation",
		Description: "The consolidated list of memory lines, and what was merged",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{
				"lines":{"type":"array","items":{"type":"string"},"description":"The new list, one fact per line, same language as the originals"},
				"note":{"type":"string","description":"What was merged or shortened, one or two sentences, in the lines' language"}
			},
			"required":["lines","note"]
		}`),
	},
}

// appMemoryConsolidator calls the active model, the same way the session
// review and the skill drafter do (oneShotProvider).
type appMemoryConsolidator struct{ app *App }

func (c appMemoryConsolidator) Consolidate(ctx context.Context, scope string, lines []string, target int, feedback string) ([]string, string, error) {
	p, modelName, err := c.app.oneShotProvider()
	if err != nil {
		return nil, "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "The list below renders to %d bytes (UTF-8). The new list must render to UNDER %d bytes.\n",
		learned.RenderedSize(scope, lines), target)
	if feedback != "" {
		b.WriteString(feedback)
		b.WriteString("\n")
	}
	b.WriteString("\nLines:\n")
	for _, l := range lines {
		b.WriteString("- ")
		b.WriteString(l)
		b.WriteString("\n")
	}
	resp, err := p.Complete(ctx, model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: memoryConsolidateInstructions},
			{Role: model.RoleUser, Content: b.String()},
		},
		Tools:      []model.ToolDefinition{memoryConsolidateTool},
		ToolChoice: "required",
		MaxTokens:  2000,
	})
	if err != nil {
		return nil, "", err
	}
	raw := ""
	if len(resp.ToolCalls) > 0 {
		raw = resp.ToolCalls[0].Function.Arguments
	} else {
		raw = extractJSONObject(resp.Text)
	}
	if raw == "" {
		return nil, "", fmt.Errorf("the model returned no list")
	}
	var out struct {
		Lines []string `json:"lines"`
		Note  string   `json:"note"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, "", fmt.Errorf("could not read the model's list: %w", err)
	}
	return out.Lines, strings.TrimSpace(out.Note), nil
}

// ConsolidateMemory asks the model for a shorter version of one file and
// returns it for the user to read. Nothing is written.
func (a *App) ConsolidateMemory(scope string) (MemoryConsolidation, error) {
	return a.consolidateMemoryWith(appMemoryConsolidator{app: a}, scope)
}

func (a *App) consolidateMemoryWith(c memoryConsolidator, scope string) (MemoryConsolidation, error) {
	scope = strings.TrimSpace(scope)
	before := learned.Entries(scope)
	out := MemoryConsolidation{Scope: scope, Before: before, MaxBytes: learned.MaxBytesFor(scope)}
	if len(before) < 2 {
		return out, fmt.Errorf("there is nothing to merge — the file holds %d line", len(before))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	current := learned.RenderedSize(scope, before)
	// Three quarters of the file, or of the ceiling, whichever is smaller: a
	// list that only just fits is full again after one line.
	target := current * 3 / 4
	if roof := out.MaxBytes * 3 / 4; roof < target {
		target = roof
	}
	// Two attempts. The first live run (owner, 11 ก.ย.) came back LONGER than
	// the file — 4,862 bytes for a 3,691-byte list — the model having rewritten
	// and translated rather than cut. A second pass told exactly by how much it
	// missed is cheap next to a person doing the merge by hand; a third is
	// throwing tokens at a model that is not going to do it.
	var after []string
	var note string
	var rendered int
	feedback := ""
	for attempt := 0; attempt < 2; attempt++ {
		lines, n, err := c.Consolidate(ctx, scope, before, target, feedback)
		if err != nil {
			return out, err
		}
		after = after[:0]
		for _, l := range lines {
			if l = strings.TrimSpace(l); l != "" {
				after = append(after, l)
			}
		}
		if len(after) == 0 {
			return out, fmt.Errorf("the model kept nothing — not applied")
		}
		note = n
		rendered = learned.RenderedSize(scope, after)
		if rendered < current {
			break
		}
		feedback = fmt.Sprintf("Your previous list rendered to %d bytes — %d MORE than the file it was meant to shorten. "+
			"Do not translate, do not add words; cut repetition and merge overlapping lines. Keep every fact.",
			rendered, rendered-current)
	}
	// A "consolidation" that is not shorter is not one. The user is shown the
	// bytes either way; refusing here keeps the button honest.
	if rendered >= current {
		return out, fmt.Errorf("not shorter: the model's list is %d bytes against the file's %d — nothing to apply", rendered, current)
	}
	out.After, out.Note, out.Bytes = after, note, rendered
	return out, nil
}

// ApplyMemoryLines replaces one file's list with the one the user accepted.
func (a *App) ApplyMemoryLines(scope string, lines []string) error {
	if err := learned.SetEntries(strings.TrimSpace(scope), lines); err != nil {
		return err
	}
	a.emitLearningChanged()
	return nil
}
