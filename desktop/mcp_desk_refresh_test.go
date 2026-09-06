package main

// A placement the user makes has to reach the session they are looking at.
//
// mode.Mode is a snapshot and two of its lists — MCP and Connections — are
// resolved when it is loaded, not written in the manifest. The session holds
// one from the moment it opened, so ticking a desk on a server rebuilt the
// engine, connected the server and registered its tools into a registry whose
// desk then filtered every one of them straight back off. Nothing on screen
// said so: the row read connected with its tool count, and the assistant
// answered that it had no MCP tools.

import (
	"slices"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

func TestTickingADeskOnAServerReachesTheOpenSession(t *testing.T) {
	a := bootDeskApp(t, "assistant")

	if err := a.SaveMCPServer("", config.MCPServerConfig{Name: "canva", URL: "https://example.invalid/mcp"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := a.SetMCPServerTargets("canva", []string{"assistant"}); err != nil {
		t.Fatalf("place on the desk: %v", err)
	}
	if !a.cur().desk.AllowsServer("canva") {
		t.Fatalf("the open session's desk does not carry the server just ticked onto it: MCP=%v", a.cur().desk.MCP)
	}

	// And the other way: unticking has to land on the same session too, or a
	// server switched off goes on answering until the next launch.
	if err := a.SetMCPServerTargets("canva", nil); err != nil {
		t.Fatalf("take off the desk: %v", err)
	}
	if a.cur().desk.AllowsServer("canva") {
		t.Fatalf("the open session's desk still carries a server taken off it: MCP=%v", a.cur().desk.MCP)
	}
}

// The other half of the same complaint: เพิ่มแล้ว has to mean the assistant can
// use it. A preset added from the shelf carries no placement of its own, and
// the app's answer for that has always been config.MCPDefaultDesks — but only
// the migration applied it, and the migration could not see a server saved this
// way (`"for": null` is a key that is present). So the shelf said added, the row
// said connected, and no desk carried a single tool until the user went and
// ticked the desk the app would have chosen anyway.
func TestAddingAServerLandsItOnTheGeneralDesks(t *testing.T) {
	a := bootDeskApp(t, "assistant")

	if err := a.SaveMCPServer("", config.MCPServerConfig{Name: "canva", URL: "https://example.invalid/mcp"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if !a.cur().desk.AllowsServer("canva") {
		t.Fatalf("a freshly added server reaches no desk: MCP=%v", a.cur().desk.MCP)
	}
	servers, err := config.LoadMCPServers()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, s := range servers {
		if s.Name != "canva" {
			continue
		}
		// Written into the entry, not applied at read time: a default the user
		// cannot see on the row is one they cannot switch off.
		if !slices.Equal(s.For, config.MCPDefaultDesks) {
			t.Fatalf("the default placement is not in the file: for=%v", s.For)
		}
	}

	// And "shown to nobody" still means it. Absent is the caller having nothing
	// to say; an empty list is a decision, and the default must not overrule it.
	if err := a.SaveMCPServer("", config.MCPServerConfig{Name: "quiet", URL: "https://example.invalid/mcp", For: []string{}}); err != nil {
		t.Fatalf("add placed nowhere: %v", err)
	}
	if a.cur().desk.AllowsServer("quiet") {
		t.Fatalf("a server placed nowhere was handed to the desk anyway: MCP=%v", a.cur().desk.MCP)
	}
}
