package subagent

// The video agent's `tools:` line, and the two traps a shipped allowlist can
// walk into without anything looking wrong.
//
// It got one on 7 ก.ย. 2569, trimming a 27-tool inheritance down to the job:
// the desk keeps `chairs: files, shell, deliverables` in the room for every
// agent sitting at it, which handed the scene-maker n8n, windmill,
// plugin_install, sheet_write, doc_write, a Cutfile exporter and a repo map.
// None of that is video work, and every one of them is prompt weight on every
// request.

import (
	"slices"
	"testing"

	"github.com/Mikedev115/Aetox/internal/skill"
)

func TestTheVideoAgentsToolkitIsTheJobAndKeepsItsShelf(t *testing.T) {
	isolate(t)
	p, ok := Load("video")
	if !ok {
		t.Fatal("no video profile")
	}
	if len(p.Tools) == 0 {
		t.Fatal("the video agent has no tools: line — it is back to inheriting the whole desk")
	}

	// Trap one: `memory` is dropped inside the filter and re-registered scoped,
	// but KeepsOwnMemory asks the allowlist. An allowlist that forgets it takes
	// the agent's memory away, and the office roster then reports the agent as
	// missing something the file never mentioned.
	if !p.KeepsOwnMemory() {
		t.Error("the tools: line does not name memory, so this agent stops remembering")
	}

	// Trap two: the progressive-loading doors are collected *inside* the same
	// loop that applies the allowlist, so a worker whose tools: never named
	// them does not get them back — and loses its whole skill shelf, which for
	// this agent is the template library and, once installed, twenty more.
	for _, door := range openDoors {
		if !p.AllowsTool(door) {
			t.Errorf("the tools: line does not name %s, so the shelf is unreachable", door)
		}
	}

	// The job itself. `video` is the three actions; `change` and `search` are
	// how a scene is edited rather than rewritten whole; the media readers are
	// how the source material is read at all.
	for _, needed := range []string{"video", "change", "search", "read", "media_read", "web_fetch"} {
		if !p.AllowsTool(needed) {
			t.Errorf("the video agent cannot be handed %s", needed)
		}
	}

	// `github` and `pr` stay: `pr-to-video` is one of the ten HyperFrames
	// routes, and it sizes the clip from the pull request's own diff.
	for _, needed := range []string{"github", "pr"} {
		if !p.AllowsTool(needed) {
			t.Errorf("%s was trimmed, but pr-to-video needs it", needed)
		}
	}

	for _, gone := range []string{"n8n", "windmill", "plugin_install", "doc_write", "sheet_write", "video_project", "codebase"} {
		if p.AllowsTool(gone) {
			t.Errorf("%s is still on the video agent's list and is not video work", gone)
		}
	}

	// And the registry agrees with the file, which is the only claim that
	// matters at dispatch.
	parent := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()})
	held := FilterRegistry(parent, p, nil).Snapshot()
	var names []string
	for name := range held {
		names = append(names, name)
	}
	if !slices.Contains(names, "video-templates") {
		t.Error("the scene library did not reach the agent it belongs to")
	}
	if slices.Contains(names, "windmill") {
		t.Error("the allowlist did not reach the registry")
	}
}

// Trap three, and it is the one an allowlist is worst at: **a name nobody
// registers takes a tool away and says nothing.** `tools:` is an intersection,
// so a typo does not fail to load, does not warn, and does not appear anywhere
// — it just quietly narrows the agent by one, and the next person to notice is
// the model being unable to do something the file says it can.
//
// This is the only bundled profile carrying a `tools:` line, so it is also the
// only one that has to be re-read every time the desk's `chairs:` changes.
// Twenty-one names on 8 ก.ย. 2569, all of them real; the point of the sweep is
// the twenty-second.
func TestEveryToolTheVideoAgentNamesIsAToolThatExists(t *testing.T) {
	isolate(t)
	p, ok := Load("video")
	if !ok {
		t.Fatal("no video profile")
	}

	known := map[string]bool{}
	for _, name := range skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()}).Names() {
		known[name] = true
	}
	// The ones that need a window, so they are registered in desktop/ and never
	// appear in the default registry. Written out rather than discovered
	// because internal/subagent cannot import desktop/ — and a list that has to
	// be edited by hand is the honest cost of a check that would otherwise not
	// exist at all.
	for _, name := range []string{
		"video", "browser", "desk", "desk_terminal",
		"ask_user", "memory", "session_search", "task", "todo_write",
	} {
		known[name] = true
	}

	for _, named := range p.Tools {
		if !known[named] {
			t.Errorf("the video agent's tools: line names %q, which no registry hands out — "+
				"that name removes nothing, adds nothing, and reads like a capability the agent has", named)
		}
	}
}
