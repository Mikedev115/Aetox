package main

// Self-update is the screen's (§248 B1): it downloads a new build of this
// program, and it ends this process to hand over — and the engine, when it is
// a process of its own on another machine, is neither of those things. What
// the engine contributes is one word, ReadyToRestart: whether a turn is using
// the brain right now.

import (
	"context"
	"errors"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/update"
	"github.com/Mikedev115/Aetox/internal/version"
)

// stagedUpdate is the downloaded, verified update waiting for the user to pick
// a moment to restart into (§107). Held on the App rather than in
// internal/update because it is one running app's state, not the package's —
// and guarded because StageUpdate runs on whichever goroutine Wails hands it
// while RestartToUpdate reads it from another.
type stagedUpdate struct {
	mu     sync.Mutex
	staged update.Staged
	// installError is why the previous restart-to-update came back as the
	// same build — the waiter's one line, already worded for the card
	// (update.InstallFailure). "" when it went fine or never happened.
	installError string
}

// StagedInfo is what the window asks about a staged update: which version
// waits, on which channel (the ready sentence differs by it), and whether the
// last attempt to install it failed and why.
type StagedInfo struct {
	Version      string `json:"version"`
	Channel      string `json:"channel"`
	InstallError string `json:"installError"`
}

// CheckForUpdate asks GitHub whether a newer release exists. Explicitly, from
// the button in Settings → About — nothing calls it on a timer yet.
//
// ErrDisabled is folded into the returned Status rather than raised: the user
// switching the check off is not a failure, and rendering it as one would put
// a red error under a setting they chose. Every other failure does reject, so
// "could not reach GitHub" reads as what it is.
func (a *App) CheckForUpdate() (update.Status, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	st, err := update.Check(ctx, version.Current)
	if errors.Is(err, update.ErrDisabled) {
		return st, nil
	}
	return st, err
}

// StageUpdate downloads the newer release, verifies its signature and its
// bytes, and puts this machine one restart away from running it — without
// touching the window. Progress rides out as `update:progress` in bytes, which
// is what the download actually knows; turning that into a percentage or a
// megabyte count is the UI's business.
//
// It does not restart, on purpose: see internal/update's Stage. Downloading is
// cheap for the user, closing their window is not, and one act that did both
// would spend the second without asking.
//
// Deliberately not refused mid-turn. Nothing here interrupts anything — the
// agent keeps working while the bytes come down, and the gate belongs on
// RestartToUpdate, which is where the process actually ends.
func (a *App) StageUpdate() error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	debuglog.Msg("self-update: staging from %s", version.Current)
	staged, err := update.Stage(ctx, version.Current, func(done, total int64) {
		a.emitEvent("update:progress", map[string]int64{"done": done, "total": total})
	})
	if err != nil {
		debuglog.Msg("self-update: stage failed: %v", err)
		return err
	}
	debuglog.Msg("self-update: staged %s (%s)", staged.Version, staged.Channel)
	a.staged.mu.Lock()
	a.staged.staged = staged
	a.staged.installError = ""
	a.staged.mu.Unlock()
	return nil
}

// StagedUpdate is the update waiting for a restart, or the zero value if none
// is. What a window that just reloaded asks, so a staged update survives the
// webview coming back without the Go side having lost it — and what the
// window asks on every launch, since adoptStagedUpdate may have found one the
// previous run left behind.
func (a *App) StagedUpdate() StagedInfo {
	a.staged.mu.Lock()
	defer a.staged.mu.Unlock()
	return StagedInfo{
		Version:      a.staged.staged.Version,
		Channel:      string(a.staged.staged.Channel),
		InstallError: a.staged.installError,
	}
}

// adoptStagedUpdate is the startup half of "later is a real answer" on the
// installer channel: read how the previous hand-off went, pick up an
// installer that was downloaded and verified but never restarted into, and
// only then sweep the leftovers. In that order — the sweep used to run first
// and unconditionally, so closing the window on a staged update deleted it,
// and the next launch offered the same 24 MB download again.
//
// Runs off the startup path (hashing a 24 MB file is ~100 ms the first paint
// does not need), and announces what it found as `update:staged` because the
// window may already have asked StagedUpdate and been told "nothing".
func (a *App) adoptStagedUpdate() {
	if line := update.ReadRestartLog(); line != "" {
		debuglog.Msg("self-update: previous hand-off: %s", line)
		a.staged.mu.Lock()
		a.staged.installError = update.InstallFailure(line)
		a.staged.mu.Unlock()
	}
	staged, ok := update.Adopt(version.Current)
	if ok {
		debuglog.Msg("self-update: adopted staged %s (%s) from a previous run", staged.Version, staged.Channel)
		a.staged.mu.Lock()
		a.staged.staged = staged
		a.staged.mu.Unlock()
	}
	update.RemoveLeftovers(ok)
	if info := a.StagedUpdate(); info.Version != "" || info.InstallError != "" {
		a.emitEvent("update:staged", info)
	}
}

// RestartToUpdate is the half the user times: close this build, let the waiter
// bring the new one up.
//
// Refused mid-turn for the same reason every session switch is — this ends the
// process, and the process is where the turn lives. Its own sentence
// (errTurnBusyUpdate) because the shared one ends in advice about switching
// chats, which is not the door the user is standing in.
func (a *App) RestartToUpdate() error {
	if err := a.api.ReadyToRestart(); err != nil {
		return err
	}
	a.staged.mu.Lock()
	staged := a.staged.staged
	a.staged.mu.Unlock()
	debuglog.Msg("self-update: restarting into %s (%s)", staged.Version, staged.Channel)
	if err := staged.Restart(); err != nil {
		debuglog.Msg("self-update: hand-off failed: %v", err)
		return err
	}
	// The relauncher is waiting on this process. Quit on a short delay rather
	// than here: the frontend's await deserves its resolution first, so the
	// button can honestly say "restarting" instead of the window vanishing
	// mid-click.
	go func() {
		time.Sleep(400 * time.Millisecond)
		if a.ctx != nil {
			wailsruntime.Quit(a.ctx)
		}
	}()
	return nil
}
