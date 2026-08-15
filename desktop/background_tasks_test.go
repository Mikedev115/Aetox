package main

import (
	"context"
	"regexp"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Mike0165115321/Aetox/internal/skill"
)

// BackgroundTasks must never return nil — §34, a nil slice crashes the
// frontend. An App with no engine yet is the state the panel first mounts in.
func TestBackgroundTasksEmptyIsNotNil(t *testing.T) {
	a := &App{}
	if a.BackgroundTasks() == nil {
		t.Fatalf("empty task list marshaled as null")
	}
}

// The panel's whole contract, end to end through the real registry: a command
// the model starts in the background shows up in the list, the stop button
// kills the real process, and each transition pushes background:changed so
// the UI never has to poll.
func TestBackgroundTasksListStopAndEmit(t *testing.T) {
	isolateUserDirs(t)
	a := &App{}
	var mu sync.Mutex
	var events []string
	a.emit = func(event string, _ ...any) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}
	// The same wiring applyConfig asks bootstrap for: the job registry's
	// change hook is this App's emitter.
	a.registry = skill.NewDefaultRegistry(skill.RegistryOptions{
		SandboxRoot:        t.TempDir(),
		OnBackgroundChange: a.emitBackgroundTasks,
	})

	command := `ping -n 60 127.0.0.1 >nul`
	if runtime.GOOS != "windows" {
		command = `sleep 60`
	}
	shellTool, _ := a.registry.Get("shell")
	out, err := shellTool.(skill.Tool).ExecuteTool(context.Background(), map[string]any{
		"command":           command,
		"run_in_background": true,
	})
	if err != nil {
		t.Fatalf("starting the background command: %v", err)
	}
	id := regexp.MustCompile(`\bbg_\d+\b`).FindString(out.Content)
	if id == "" {
		t.Fatalf("no handle in %q", out.Content)
	}

	tasks := a.BackgroundTasks()
	if len(tasks) != 1 {
		t.Fatalf("BackgroundTasks() = %+v, want the one job just started", tasks)
	}
	if tasks[0].ID != id || tasks[0].Command != command || !tasks[0].Running {
		t.Errorf("task = %+v, want %s running %q", tasks[0], id, command)
	}

	if err := a.StopBackgroundTask("bg_999"); err == nil {
		t.Error("stopping an unknown handle must fail rather than report a kill that never happened")
	}

	if err := a.StopBackgroundTask(id); err != nil {
		t.Fatalf("StopBackgroundTask(%s): %v", id, err)
	}
	// Wait for the real process to die — on Windows t.TempDir cleanup fails
	// while a surviving child still holds the working directory open.
	deadline := time.Now().Add(15 * time.Second)
	var task BackgroundTask
	for {
		task = a.BackgroundTasks()[0]
		if !task.Running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job still running after StopBackgroundTask: %+v", task)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !task.Killed || task.ExitError != "" {
		t.Errorf("a stopped job must read as killed, not failed: %+v", task)
	}

	// Two pushes minimum: the start and the death. The panel's badge and list
	// are fed entirely by these.
	mu.Lock()
	defer mu.Unlock()
	changed := 0
	for _, e := range events {
		if e == "background:changed" {
			changed++
		}
	}
	if changed < 2 {
		t.Errorf("background:changed fired %d times, want one for the start and one for the death (events: %v)", changed, events)
	}
}
