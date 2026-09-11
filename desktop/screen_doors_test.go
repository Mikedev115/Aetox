package main

import (
	"reflect"
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// Every door that needs a window has a twin that does not (§248 A5): the
// dialog or the reveal on the screen, the work — a path in, or bytes out —
// with the engine. Since B1 the two live in different packages, which is the
// whole point; this table is the contract, and a door added on one side
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
		// reveals → the engine half that answers with the host path
		"OpenFileExternally":    "ProjectFilePath",
		"OpenArtifact":          "ArtifactPath",
		"OpenExport":            "ExportPath",
		"OpenMCPFolder":         "MCPFolderPath",
		"OpenMemoryFolder":      "MemoryFolderPath",
		"OpenPromptsFolder":     "PromptsFolderPath",
		"OpenSkillsFolder":      "SkillsFolderPath",
		"OpenSpaceFolder":       "SpaceFolderPath",
		"OpenSubagentsFolder":   "SubagentsFolderPath",
		"OpenAgentsFolder":      "AgentsFolderPath",
		"OpenAgentSkillsFolder": "AgentSkillsFolderPath",
		"OpenAgentHome":         "AgentHomePath",
		"RevealSpeechModel":     "SpeechModelFolderPath",
		"OpenSpeechModelDir":    "SpeechModelDirPath",
	}
	screen := reflect.TypeOf(&App{})
	eng := reflect.TypeOf(&engine.Engine{})
	for door, twin := range twins {
		if _, ok := screen.MethodByName(door); !ok {
			t.Errorf("screen door %s is gone", door)
		}
		if _, ok := eng.MethodByName(twin); !ok {
			t.Errorf("%s has no engine twin %s", door, twin)
		}
	}
}

// A reveal goes to the OS door with the path the engine answered, and stops
// where the engine refuses — the screen adds no judgment of its own.
func TestARevealOpensWhatTheEngineAnswers(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := &App{eng: engine.NewEngine()}
	var opened string
	a.openDir = func(p string) error { opened = p; return nil }

	if err := a.OpenMemoryFolder(); err != nil {
		t.Fatalf("OpenMemoryFolder: %v", err)
	}
	want, err := a.eng.MemoryFolderPath()
	if err != nil {
		t.Fatal(err)
	}
	if opened != want {
		t.Errorf("opened %q, want the folder the engine answered %q", opened, want)
	}

	opened = ""
	if err := a.OpenSpaceFolder("no-such-project-anywhere"); err == nil {
		t.Error("a project that does not exist was opened")
	}
	if opened != "" {
		t.Errorf("the OS door was opened at %q although the engine refused", opened)
	}
}
