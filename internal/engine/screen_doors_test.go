package engine

import (
	"archive/zip"
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Every door that needs a window has a twin that does not (§248 A5): the
// dialog or the reveal on the screen, the work — a path in, or bytes out —
// with the engine. The table is the contract; a door added to one side
// without the other fails here before it fails on a remote host.
func TestEveryScreenDoorHasItsEngineTwin(t *testing.T) {
	twins := map[string]string{
		// dialogs → the engine half that takes a path or hands back bytes
		"ExportAgentPackage":  "AgentPackageBytes",
		"ExportSession":       "SessionExportBytes",
		"SavePicture":         "PictureBytes",
		"ImportSession":       "ImportSessionFrom",
		"PickPresetImage":     "SetPresetImageFrom",
		"InstallSkillFromZip": "InstallSkillsFromZipAt",
		"AddSpaceContext":     "AddSpaceContextFiles",
		"AddWorkspaceFolder":  "AddWorkspaceFolderAt",
		"BrowseFolder":        "BrowseFolderAt",
		"OpenProjectFolder":   "OpenProjectPath",
		// reveals → the engine half that answers with the host path
		"OpenFileExternally":    "ProjectFilePath",
		"OpenArtifact":          "ArtifactPath",
		"OpenMCPFolder":         "MCPFolderPath",
		"OpenMemoryFolder":      "MemoryFolderPath",
		"OpenPromptsFolder":     "PromptsFolderPath",
		"OpenSkillsFolder":      "SkillsFolderPath",
		"OpenSpaceFolder":       "SpaceFolderPath",
		"OpenSubagentsFolder":   "SubagentsFolderPath",
		"OpenAgentsFolder":      "AgentsFolderPath",
		"OpenAgentSkillsFolder": "AgentSkillsFolderPath",
		"OpenAgentHome":         "AgentHomePath",
	}
	typ := reflect.TypeOf(&Engine{})
	for door, twin := range twins {
		if _, ok := typ.MethodByName(door); !ok {
			t.Errorf("screen door %s is gone", door)
		}
		if _, ok := typ.MethodByName(twin); !ok {
			t.Errorf("%s has no engine twin %s", door, twin)
		}
	}
}

// The engine renders; the screen decides where it goes. A rendered export
// carries the name the dialog will suggest, extension included, and nothing
// about a path.
func TestSessionExportBytesNamesTheFileForTheDialog(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	conv := a.cur()
	a.appendTurn(conv,
		SessionMessage{Role: "user", Text: "สวัสดี", Time: "10:00"},
		SessionMessage{Role: "agent", Text: "hello", Time: "10:00"},
	)
	for format, ext := range map[string]string{"json": ".json", "markdown": ".md"} {
		file, err := a.SessionExportBytes(conv.id, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if !strings.HasSuffix(file.Name, ext) || !strings.HasPrefix(file.Name, "aetox-chat-") {
			t.Errorf("%s: name %q, want aetox-chat-…%s", format, file.Name, ext)
		}
		if !bytes.Contains(file.Data, []byte("hello")) {
			t.Errorf("%s: the export does not carry the conversation", format)
		}
	}
	if _, err := a.SessionExportBytes(conv.id, "docx"); err == nil {
		t.Error("an unknown format must be refused, not rendered as something else")
	}
}

// A packed agent is a zip in memory with the summary beside it, so the screen
// can save the one and show the other — and a name that is not an agent is
// refused before anything is packed.
func TestAgentPackageBytesRefusesWhatIsNotAnAgent(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if _, err := a.AgentPackageBytes(""); err == nil {
		t.Error("an empty name must be refused")
	}
	if file, err := a.AgentPackageBytes("no-such-agent-anywhere"); err == nil {
		if _, zerr := zip.NewReader(bytes.NewReader(file.Data), int64(len(file.Data))); zerr != nil {
			t.Errorf("a package that was produced is not a zip: %v", zerr)
		}
		t.Error("a name nobody has should not pack")
	}
}

// The tree's root is a decision the engine keeps: a dismissed dialog keeps
// what was showing, a folder that is not one is refused, and a focused project
// refuses a second root altogether.
func TestBrowseFolderAtKeepsTheRules(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.projectFocused = false
	dir := t.TempDir()
	if got, err := a.BrowseFolderAt(dir); err != nil || got != dir {
		t.Fatalf("BrowseFolderAt(dir) = %q, %v", got, err)
	}
	if got, err := a.BrowseFolderAt(""); err != nil || got != dir {
		t.Errorf("a dismissed dialog must keep the root: got %q, %v", got, err)
	}
	if _, err := a.BrowseFolderAt(dir + "-missing"); err == nil {
		t.Error("a folder that is not there must be refused")
	}
	a.projectFocused = true
	if _, err := a.BrowseFolderAt(dir); err == nil {
		t.Error("a focused project must refuse a second root")
	}
}
