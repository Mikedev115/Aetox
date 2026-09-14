package main

// Files and folders across the wire (§248 phase 4). Two directions, one
// rule each.
//
// A path this machine's dialog answered with is a path on THIS machine. With
// the engine a child of this process that is also the engine's disk, and the
// binding takes it as it is. With the engine on a host over ssh it is not:
// the bytes go up first (rpc.UploadFile, through the tunnel, behind the
// token) and the binding is handed where they landed there — onHost. The
// bindings never learn about the wire; every dialog door goes through here.
//
// A folder to pick is the other way round: the folder is on the engine's
// machine, so the native dialog — which browses this one — is the wrong
// question there. pickHostDir asks the window to raise the remote picker
// (RemoteDirPicker.svelte, listing through the engine's ListDir) and waits
// for AnswerHostDir, the way the engine's own questions wait for the screen.
//
// And a path the engine answered with — a folder to reveal, a file to open
// with its program — is a path on the engine's machine, which this machine's
// file manager cannot show. That door stays shut, and says so.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
)

// engineOnHost reports whether the engine the window is on is another
// machine's — the one case in which a path here is not a path there. A host
// over ssh, or an engine attached by hand that named another machine in
// its hello (localEngine.elsewhere).
func (a *App) engineOnHost() bool {
	return a.engine != nil && a.engine.elsewhere()
}

// hostLabel names the host, for the sentences below: the Settings row's
// name, or for an attached engine what it called itself with its OS after
// it — the owner's WSL and his Windows are both named "Mikedev", and
// "อยู่บนเครื่อง Mikedev" on a PC named Mikedev reads as this machine.
func (a *App) hostLabel() string {
	if a.engine == nil {
		return ""
	}
	if h := a.EngineStatus().Host; h != "" {
		return h
	}
	a.engine.mu.Lock()
	defer a.engine.mu.Unlock()
	if a.engine.hello.Hostname != "" {
		return a.engine.hello.Hostname + " (" + a.engine.hello.OS + ")"
	}
	return a.engine.hello.OS
}

// uploadWait bounds one file's trip up the tunnel. Generous: the cap on an
// attachment is 2 GB and a home upstream is slow.
const uploadWait = 30 * time.Minute

// onHost answers with a path the engine can read for a file on this
// machine, and a done that clears what the trip left behind. At home the
// path is the path and done is nothing; on a host the file goes up first.
//
// maxBytes is the cap the binding at the far end will apply (0 for none):
// checked here, before the trip, so a 50 MB photo is refused in the same
// words as at home rather than after 50 MB have crossed the tunnel.
func (a *App) onHost(local string, maxBytes int64) (path string, done func(), err error) {
	local = strings.TrimSpace(local)
	if local == "" || !a.engineOnHost() {
		return local, func() {}, nil
	}
	if maxBytes > 0 {
		info, err := os.Stat(local)
		if err != nil {
			return "", func() {}, fmt.Errorf("ส่งไฟล์ไปเครื่อง %s ไม่ได้: %w", a.hostLabel(), err)
		}
		if info.Size() > maxBytes {
			return "", func() {}, fmt.Errorf("ไฟล์ใหญ่เกินไป (%d MB, สูงสุด %d MB)", info.Size()>>20, maxBytes>>20)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), uploadWait)
	defer cancel()
	up, err := rpc.UploadFile(ctx, a.engineEndpoint, local)
	if err != nil {
		return "", func() {}, fmt.Errorf("ส่งไฟล์ไปเครื่อง %s ไม่ได้: %w", a.hostLabel(), err)
	}
	debuglog.Msg("host files: %s up as %s", filepath.Base(local), up.ID)
	return up.Path, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := rpc.Discard(ctx, a.engineEndpoint, up.ID); err != nil {
			debuglog.Msg("host files: discard %s: %v", up.ID, err) // the sweep is behind it
		}
	}, nil
}

// withHostFile runs the engine's half with a path it can read, and clears
// the trip afterwards. maxBytes as for onHost.
func (a *App) withHostFile(local string, maxBytes int64, bind func(hostPath string) (string, error)) (string, error) {
	path, done, err := a.onHost(local, maxBytes)
	if err != nil {
		return "", err
	}
	defer done()
	return bind(path)
}

// The caps the engine's two attachment doors apply (engine.SaveChatImage,
// engine.SaveChatFile), repeated here so the refusal comes before the trip.
const (
	chatImageCap = 20 << 20
	chatFileCap  = 2 << 30
)

// ---------------------------------------------------------------- attachments

// SaveChatImage is the engine's SaveChatImage with the trip in front of it:
// the picker and the drop both answer with a path on this machine, and on a
// host the engine would find nothing there. The screen owns the name so the
// generator leaves it to this file (engine_forwarders_gen.go).
func (a *App) SaveChatImage(sourcePath string) (string, error) {
	return a.withHostFile(sourcePath, chatImageCap, a.api.SaveChatImage)
}

// SaveChatFile is the same for a clip or a document.
func (a *App) SaveChatFile(sourcePath string) (string, error) {
	return a.withHostFile(sourcePath, chatFileCap, a.api.SaveChatFile)
}

// AddSpaceContextFiles is the engine's, with every picked file — from the
// dialog or dropped on the card — taken up first. Files that could not go
// are named in the error; the ones that could still land, the way the
// engine's own half already answers a list one file at a time.
func (a *App) AddSpaceContextFiles(name string, picked []string) ([]string, error) {
	if !a.engineOnHost() || len(picked) == 0 {
		return a.api.AddSpaceContextFiles(name, picked)
	}
	var (
		there []string
		dones []func()
		errs  []error
	)
	defer func() {
		for _, d := range dones {
			d()
		}
	}()
	for _, p := range picked {
		path, done, err := a.onHost(p, 0)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		there = append(there, path)
		dones = append(dones, done)
	}
	if len(there) == 0 {
		return nil, errors.Join(errs...)
	}
	added, err := a.api.AddSpaceContextFiles(name, there)
	return added, errors.Join(append(errs, err)...)
}

// ---------------------------------------------------------------- folders

// hostDirAsk is one question the window is showing the remote picker for.
type hostDirAsk struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Start string `json:"start"`
}

// hostDirAsks is the questions waiting for AnswerHostDir, by id.
type hostDirAsks struct {
	mu   sync.Mutex
	seq  atomic.Int64
	open map[string]chan string
}

func (h *hostDirAsks) add() (string, chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.open == nil {
		h.open = map[string]chan string{}
	}
	id := strconv.FormatInt(h.seq.Add(1), 10)
	ch := make(chan string, 1)
	h.open[id] = ch
	return id, ch
}

func (h *hostDirAsks) take(id string) (chan string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.open[id]
	delete(h.open, id)
	return ch, ok
}

// pickDirWait is how long a folder question stays open. A dialog is a
// person's to answer, so this is a backstop for a window that reloaded
// under the question, not a pace.
const pickDirWait = 10 * time.Minute

// pickHostDir asks for a folder on the engine's machine: the native dialog
// at home, the window's remote picker on a host. "" is a dismissed dialog,
// which is not a failure and must not raise one.
func (a *App) pickHostDir(title, start string) (string, error) {
	if !a.engineOnHost() {
		return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
			Title:            title,
			DefaultDirectory: start,
		})
	}
	id, ch := a.hostDirs.add()
	a.emitEvent("screen:pickdir", hostDirAsk{ID: id, Title: title, Start: start})
	select {
	case dir := <-ch:
		return strings.TrimSpace(dir), nil
	case <-time.After(pickDirWait):
		a.hostDirs.take(id)
		return "", nil
	}
}

// AnswerHostDir is the remote picker's answer to a screen:pickdir event —
// the folder chosen, or "" for cancelled. An id nobody is waiting on is a
// picker the window kept past its question, and is nothing.
func (a *App) AnswerHostDir(id, dir string) {
	if ch, ok := a.hostDirs.take(id); ok {
		ch <- dir
	}
}

// ---------------------------------------------------------------- reveals

// errOnHost is what a reveal answers when the path is on the engine's
// machine: named, with the path, so the person can go there by hand — the
// terminal pane is on that machine already.
func (a *App) errOnHost(path string) error {
	return fmt.Errorf("อยู่บนเครื่อง %s: %s — เปิดจากหน้าต่างนี้ไม่ได้ ใช้เทอร์มินัลบนเครื่องนั้นแทน", a.hostLabel(), path)
}

// openHostFileCopy fetches one project file from the engine (its /file/
// door, the same one the panes read through) into a folder of this
// machine's temp and opens the copy with its program. A copy, and said to
// be one: an edit made there does not reach the project.
func (a *App) openHostFileCopy(relPath string) error {
	dir, err := os.MkdirTemp("", "aetox-host-")
	if err != nil {
		return err
	}
	local := filepath.Join(dir, filepath.Base(filepath.ToSlash(relPath)))
	ctx, cancel := context.WithTimeout(context.Background(), uploadWait)
	defer cancel()
	if err := rpc.FetchFile(ctx, a.engineEndpoint, relPath, local); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("ดึงสำเนาจากเครื่อง %s ไม่ได้: %w", a.hostLabel(), err)
	}
	return a.revealInFileManager(local)
}
