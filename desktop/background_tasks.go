package main

// The background-tasks panel: what the agent left running, visible to the
// human. shell with run_in_background gave the model dev servers and watch
// builds (internal/skill/shell_background.go); until this panel, the only
// viewer those jobs had was the model itself, and a user who wanted a server
// gone had to ask the chat to please kill it.
//
// Read-through, not a second store: both bindings go straight to the same job
// registry the shell tools share, via skill.Registry. The panel therefore
// shows exactly what the model can reach — same handles, same statuses, same
// kill path — and nothing here can drift from it.

import wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

// BackgroundTask is one background shell command as the panel sees it. The
// UI holds the handle only, like the model does — no pid ever crosses the
// bridge.
type BackgroundTask struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Running bool   `json:"running"`
	Killed  bool   `json:"killed"`
	// ExitError is the failure text of a job that exited on its own with an
	// error, "" for one still running, killed, or finished clean.
	ExitError string `json:"exitError"`
	// ElapsedMs stops growing when the job ends: for a finished job it is how
	// long it ran, not how long ago it started.
	ElapsedMs int64 `json:"elapsedMs"`
}

// BackgroundTasks lists the agent's background commands, running and
// finished, oldest first. The frontend polls this — a job ending is a process
// exiting, which no tool call announces, so there is no event to push.
func (a *App) BackgroundTasks() []BackgroundTask {
	jobs := a.registry.BackgroundJobs()
	// Never nil: §34, a nil slice crashes the frontend.
	out := make([]BackgroundTask, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, BackgroundTask{
			ID:        j.ID,
			Command:   j.Command,
			Running:   !j.Done,
			Killed:    j.Killed,
			ExitError: j.ExitError,
			ElapsedMs: j.Elapsed.Milliseconds(),
		})
	}
	return out
}

// StopBackgroundTask ends one background command — the same path shell_kill
// takes, so the whole process tree dies with it. Stopping a job that already
// finished succeeds quietly: the user asked for "not running", and that holds.
func (a *App) StopBackgroundTask(id string) error {
	return a.registry.StopBackgroundJob(id)
}

// emitBackgroundTasks pushes the current job list to the UI. Wired into the
// engine as OnBackgroundChange (applyConfig), so it fires from a job's own
// goroutines the moment one starts or ends. Same shape as emitTaskChips, and
// for the same reason: the test seam first, the live Wails ctx second, and a
// change with nobody listening is still a change — the list is re-read on the
// next mount.
func (a *App) emitBackgroundTasks() {
	if a.emit != nil {
		a.emit("background:changed", a.BackgroundTasks())
		return
	}
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "background:changed", a.BackgroundTasks())
	}
}
