package main

// The doors with the engine on a host (host_files.go): a file picked here
// reaches the engine's binding as a path THERE and the trip is cleared
// after; a folder is asked for through the window's picker and the answer
// comes back to the waiting door; a reveal names where the path is instead
// of opening this machine's file manager on it; a file opened with its
// program is a fetched copy. And at home none of this happens — the same
// doors take the path as it is.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/rpc"
)

// projectOpen opens a project in the test's engine, and answers its root.
func projectOpen(t *testing.T, a *App) string {
	t.Helper()
	root := t.TempDir()
	if _, err := a.api.OpenProjectPath(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestOnAHostAnAttachmentGoesUpTheWireFirst(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	root := projectOpen(t, a)

	// A "Windows path" as far as the engine is concerned: a file in a folder
	// the project cannot see, which is what every dialog on this machine
	// answers with when the engine is elsewhere.
	local := filepath.Join(t.TempDir(), "screenshot.png")
	if err := os.WriteFile(local, []byte("\x89PNG not really"), 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := a.SaveChatImage(local)
	if err != nil {
		t.Fatalf("SaveChatImage on a host: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("the attachment is not in the project at %q: %v", rel, err)
	}
	if string(got) != "\x89PNG not really" {
		t.Errorf("the bytes changed on the way: %q", got)
	}
	if !strings.HasSuffix(rel, ".png") {
		t.Errorf("landed as %q, want the picked file's extension kept", rel)
	}
	// The trip is cleared: nothing is left in the engine's inbox.
	inbox := filepath.Join(os.Getenv("AETOX_DATA_ROOT"), "inbox")
	if entries, _ := os.ReadDir(inbox); len(entries) != 0 {
		t.Errorf("%d landings left in the inbox after the binding took its copy", len(entries))
	}

	// The same for a clip through SaveChatFile.
	clip := filepath.Join(t.TempDir(), "take 1.mp4")
	if err := os.WriteFile(clip, []byte("mp4"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rel, err := a.SaveChatFile(clip); err != nil || !strings.HasSuffix(rel, ".mp4") {
		t.Errorf("SaveChatFile on a host = %q, %v", rel, err)
	}
}

func TestAtHomeAnAttachmentIsTheEnginesOwnCopy(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t)
	projectOpen(t, a)
	local := filepath.Join(t.TempDir(), "photo.jpg")
	if err := os.WriteFile(local, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rel, err := a.SaveChatImage(local); err != nil || !strings.HasSuffix(rel, ".jpg") {
		t.Errorf("SaveChatImage at home = %q, %v", rel, err)
	}
	if entries, _ := os.ReadDir(filepath.Join(os.Getenv("AETOX_DATA_ROOT"), "inbox")); len(entries) != 0 {
		t.Error("at home the file must not take the trip through the inbox")
	}
}

func TestOnAHostAFileThatIsNotThereIsNamedNotCopied(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	projectOpen(t, a)
	_, err := a.SaveChatImage(filepath.Join(t.TempDir(), "gone.png"))
	if err == nil {
		t.Fatal("a file that does not exist here was attached")
	}
	if !strings.Contains(err.Error(), "wsl") {
		t.Errorf("the error does not name the host: %v", err)
	}
}

func TestOnAHostProjectContextFilesGoUpTogether(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	if _, err := a.api.CreateSpace("งานวิจัย"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var picked []string
	for _, name := range []string{"a.pdf", "b.md"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		picked = append(picked, p)
	}
	picked = append(picked, filepath.Join(dir, "missing.txt"))
	files, err := a.AddSpaceContextFiles("งานวิจัย", picked)
	if err == nil {
		t.Error("the missing file was not named")
	}
	if len(files) != 2 {
		t.Errorf("got %d context files, want the two that exist to land regardless: %v", len(files), files)
	}
}

func TestOnAHostAFolderIsAskedThroughTheWindow(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	rec := captureEvents(a)
	projectOpen(t, a)

	// The door blocks on the window's answer, the way the engine's own
	// questions do; the test is the window.
	there := t.TempDir()
	type answer struct {
		folders int
		err     error
	}
	done := make(chan answer, 1)
	go func() {
		folders, err := a.AddWorkspaceFolder()
		done <- answer{len(folders), err}
	}()
	var ask hostDirAsk
	deadline := time.Now().Add(5 * time.Second)
	for ask.ID == "" && time.Now().Before(deadline) {
		for _, e := range rec.all() {
			if e.Name == "screen:pickdir" && len(e.Data) == 1 {
				ask = e.Data[0].(hostDirAsk)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ask.ID == "" {
		t.Fatal("no screen:pickdir was raised for a folder door on a host")
	}
	if ask.Title == "" {
		t.Error("the picker was raised with no title")
	}
	a.AnswerHostDir(ask.ID, there)
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("AddWorkspaceFolder: %v", got.err)
		}
		if got.folders != 1 {
			t.Errorf("%d workspace folders, want the one the picker answered", got.folders)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the door did not come back after the picker answered")
	}

	// An answer nobody waits for is nothing — not a panic, not a leak.
	a.AnswerHostDir("no-such-ask", there)
	// A dismissed picker is not a failure.
	go func() {
		_, err := a.BrowseFolder()
		done <- answer{0, err}
	}()
	var second hostDirAsk
	deadline = time.Now().Add(5 * time.Second)
	for second.ID == "" && time.Now().Before(deadline) {
		for _, e := range rec.all() {
			if e.Name == "screen:pickdir" && len(e.Data) == 1 {
				if got := e.Data[0].(hostDirAsk); got.ID != ask.ID {
					second = got
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.AnswerHostDir(second.ID, "")
	select {
	case got := <-done:
		// BrowseFolder is refused while a project is focused — the engine's
		// own refusal, which is fine; what must not happen is a hang.
		_ = got
	case <-time.After(5 * time.Second):
		t.Fatal("a dismissed picker left the door waiting")
	}
}

func TestOnAHostARevealNamesWhereThePathIs(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	var opened string
	a.openDir = func(p string) error { opened = p; return nil }

	err := a.OpenMemoryFolder()
	if err == nil {
		t.Fatal("a folder on the host was 'opened' in this machine's file manager")
	}
	want, _ := a.api.MemoryFolderPath()
	if !strings.Contains(err.Error(), "wsl") || !strings.Contains(err.Error(), want) {
		t.Errorf("the error must name the host and the path: %v", err)
	}
	if opened != "" {
		t.Errorf("the file manager was opened on %q anyway", opened)
	}
}

func TestOnAHostAFileOpensAsAFetchedCopy(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	root := projectOpen(t, a)
	if err := os.WriteFile(filepath.Join(root, "แผน 2569.pdf"), []byte("%PDF plan"), 0o644); err != nil {
		t.Fatal(err)
	}
	var opened string
	a.openDir = func(p string) error { opened = p; return nil }
	if err := a.OpenFileExternally("แผน 2569.pdf"); err != nil {
		t.Fatalf("OpenFileExternally on a host: %v", err)
	}
	if opened == "" || strings.HasPrefix(opened, root) {
		t.Fatalf("opened %q, want a copy on this machine, not the project's own file", opened)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(opened)) })
	if filepath.Base(opened) != "แผน 2569.pdf" {
		t.Errorf("the copy is named %q, want the file's own name", filepath.Base(opened))
	}
	if got, _ := os.ReadFile(opened); string(got) != "%PDF plan" {
		t.Errorf("the copy's bytes differ: %q", got)
	}
	// Outside the sandbox stays the engine's refusal, before any fetch.
	if err := a.OpenFileExternally("../elsewhere.pdf"); err == nil {
		t.Error("a path outside the project was fetched")
	}
}

func TestAnAttachedEngineIsElsewhereOnlyWhenItsHelloNamesAnotherMachine(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	t.Setenv("AETOX_ENGINE_ADDR", "tcp:"+addr)
	here, _ := os.Hostname()
	a.engine = &localEngine{token: testToken, target: engineTarget{mode: modeLocal}, proc: &engineProcess{network: "tcp", address: addr}}

	a.engine.hello = rpc.HelloResult{OS: runtime.GOOS, Hostname: here}
	if a.engineOnHost() {
		t.Error("an engine attached on this very machine was taken for another machine's")
	}
	a.engine.hello = rpc.HelloResult{OS: "linux", Hostname: "box"}
	if !a.engineOnHost() {
		t.Error("an engine that named another machine was taken for this one")
	}
	if a.hostLabel() != "box (linux)" {
		t.Errorf("hostLabel = %q, want the name the engine gave with its OS", a.hostLabel())
	}
	// An older engine says no hostname: the OS decides.
	a.engine.hello = rpc.HelloResult{OS: "linux"}
	if runtime.GOOS != "linux" && !a.engineOnHost() {
		t.Error("an engine on another OS, with no hostname to give, was taken for this machine")
	}
	// Two machines with one name (the owner's WSL and Windows are both
	// "Mikedev"): the OS decides before the name.
	a.engine.hello = rpc.HelloResult{OS: "linux", Hostname: here}
	if runtime.GOOS != "linux" && !a.engineOnHost() {
		t.Error("a WSL engine wearing this machine's hostname was taken for this machine")
	}
	// And without AETOX_ENGINE_ADDR the child is this machine's whatever it said.
	t.Setenv("AETOX_ENGINE_ADDR", "")
	if a.engineOnHost() {
		t.Error("a child of this process was taken for another machine's")
	}
}

func TestOnAHostAFileOverTheCapIsRefusedBeforeTheTrip(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a, addr := newTestAppAt(t)
	onAHost(a, addr)
	projectOpen(t, a)
	big := filepath.Join(t.TempDir(), "huge.png")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(chatImageCap + 1); err != nil { // sparse: no 20 MB written
		t.Fatal(err)
	}
	f.Close()
	_, err = a.SaveChatImage(big)
	if err == nil || !strings.Contains(err.Error(), "ไฟล์ใหญ่เกินไป") {
		t.Fatalf("SaveChatImage over the cap = %v, want the engine's own refusal", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(os.Getenv("AETOX_DATA_ROOT"), "inbox")); len(entries) != 0 {
		t.Error("the file crossed the wire before it was refused")
	}
}
