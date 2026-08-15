package skill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mike0165115321/Aetox/internal/model"
	"github.com/Mike0165115321/Aetox/internal/proc"
)

// Background commands: the third of the three tools a coding agent needs from a
// terminal and the only one Aetox had no answer for at all.
//
// `shell` runs a command and waits, abandoned at 60 seconds (ARCHITECTURE.md
// §50 made that adjustable). That covers a test run and a build. It cannot
// cover the other half of development — a dev server, a watch build, a log
// tail — because those never exit, and "never exits" is not a slow command, it
// is a different shape of work. Before this the model simply could not start
// one: it would hang until the deadline and be killed.
//
// Three tools sharing one runner, the same arrangement §44 uses for `task`:
//
//   - shell with run_in_background:true starts it and returns a handle at once
//   - shell_output redeems the handle for whatever has been printed since the
//     last read
//   - shell_kill ends it
//
// The handle, not the process, is what the model holds. It never sees a pid,
// so it cannot kill something Aetox did not start.

// maxBackgroundShells bounds how many commands can be running unattended. A
// model that starts a hundred servers has misunderstood something, and the
// tenth is where saying so is still cheap.
const maxBackgroundShells = 8

// backgroundOutputCap is per job, and separate from toolOutputByteCap because
// a log tail is precisely the thing that produces megabytes: the buffer keeps
// the newest bytes rather than the oldest, which is the opposite of what a
// finished command wants and the only useful answer for a running one.
const backgroundOutputCap = 256 << 10

type backgroundShells struct {
	mu   sync.Mutex
	seq  int
	jobs map[string]*backgroundJob

	// notify fires after the set or state of jobs changes: a start, an exit, a
	// kill (which is an exit). Set once by RegisterDefaults before any job can
	// exist, never during use. It exists for a host UI that mirrors this
	// registry — a finished process is not a tool call, so without a push
	// there is nothing for a panel to subscribe to but a clock.
	notify func()
}

// changed announces a state change to whoever is mirroring the registry.
// Called without any lock held: notify re-enters through Jobs(), which takes
// b.mu and every job's mu.
func (b *backgroundShells) changed() {
	if b.notify != nil {
		b.notify()
	}
}

func newBackgroundShells() *backgroundShells {
	return &backgroundShells{jobs: make(map[string]*backgroundJob)}
}

type backgroundJob struct {
	id      string
	command string
	cancel  context.CancelFunc
	started time.Time

	mu       sync.Mutex
	buf      []byte
	cursor   int // bytes already handed to shell_output
	overflow bool
	done     bool
	exitErr  error
	ended    time.Time
	killed   bool
}

// Write is the job's stdout and stderr. It keeps the tail, not the head: for a
// server or a watcher the interesting line is always the most recent one.
func (j *backgroundJob) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.buf = append(j.buf, p...)
	if len(j.buf) > backgroundOutputCap {
		drop := len(j.buf) - backgroundOutputCap
		j.buf = j.buf[drop:]
		j.overflow = true
		// The cursor indexes the same slice, so it has to slide with it or the
		// next read replays bytes the caller already has.
		if j.cursor -= drop; j.cursor < 0 {
			j.cursor = 0
		}
	}
	return len(p), nil
}

// drain returns what has arrived since the last drain and marks it read.
func (j *backgroundJob) drain() (string, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	fresh := string(j.buf[j.cursor:])
	j.cursor = len(j.buf)
	overflowed := j.overflow
	j.overflow = false
	return fresh, overflowed
}

func (j *backgroundJob) status() (done, killed bool, exitErr error, since time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.done {
		return true, j.killed, j.exitErr, j.ended.Sub(j.started)
	}
	return false, false, nil, time.Since(j.started)
}

// kill marks the job killed and cancels it. cancel is what proc.KillOnCancel
// watches, so this takes the whole process tree — npm and the node it spawned,
// not just the shell in front of them. The mark comes first: the Wait
// goroutine can observe the death at any moment after cancel, and a job that
// reads as "exited with an error" for that instant would tell whoever asked
// that the command failed on its own.
func (j *backgroundJob) kill() {
	j.mu.Lock()
	j.killed = true
	j.mu.Unlock()
	j.cancel()
}

// BackgroundJob is the snapshot of one job that leaves the package: enough to
// draw a row in a panel and address the job by handle, and nothing that could
// be misused — no pid, no process, no buffer. Same rule the model lives
// under: the handle is what callers hold.
type BackgroundJob struct {
	ID      string
	Command string
	Done    bool
	Killed  bool
	// ExitError is the failure of a job that ended on its own with a non-zero
	// exit, "" for one still running, killed, or finished clean.
	ExitError string
	// Elapsed is how long the job has been running, or ran: it stops growing
	// when the job ends, exactly as status() reports it.
	Elapsed time.Duration
}

// Jobs snapshots every job still remembered — running and finished both,
// oldest first. Finished jobs stay listed until forgetFinishedLocked reclaims
// them, which is the same window shell_output has to read them.
func (b *backgroundShells) Jobs() []BackgroundJob {
	b.mu.Lock()
	jobs := make([]*backgroundJob, 0, len(b.jobs))
	for _, j := range b.jobs {
		jobs = append(jobs, j)
	}
	b.mu.Unlock()
	// Start order, not map order or lexical id order — bg_10 belongs after
	// bg_9, and started is the one field that says so without parsing ids.
	sort.Slice(jobs, func(i, k int) bool { return jobs[i].started.Before(jobs[k].started) })
	out := make([]BackgroundJob, 0, len(jobs))
	for _, j := range jobs {
		done, killed, exitErr, elapsed := j.status()
		snap := BackgroundJob{ID: j.id, Command: j.command, Done: done, Killed: killed, Elapsed: elapsed}
		// A killed process reports a Wait error too, but that death was asked
		// for — "killed" is the whole story, same precedence shell_output's
		// header uses.
		if exitErr != nil && !killed {
			snap.ExitError = exitErr.Error()
		}
		out = append(out, snap)
	}
	return out
}

// stop ends one job by handle, the same path shell_kill takes. A job that
// already finished is a success, not an error: the caller asked for a state —
// "not running" — and that state holds.
func (b *backgroundShells) stop(id string) error {
	job, ok := b.get(id)
	if !ok {
		return fmt.Errorf("no background command %q — %s", id, describeRunning(b))
	}
	if done, _, _, _ := job.status(); done {
		return nil
	}
	job.kill()
	return nil
}

// start launches the command detached from the turn.
//
// context.Background, deliberately: the turn's context is cancelled the moment
// the turn ends, and a dev server that dies with the answer that started it is
// not a background command. What still stops them is the job object every
// child is already in (§24), so nothing here outlives the app.
func (b *backgroundShells) start(backend proc.Backend, workDir, commandLine string) (*backgroundJob, error) {
	b.mu.Lock()
	if len(b.jobs) >= maxBackgroundShells {
		running := 0
		for _, j := range b.jobs {
			if done, _, _, _ := j.status(); !done {
				running++
			}
		}
		if running >= maxBackgroundShells {
			b.mu.Unlock()
			return nil, fmt.Errorf("%d background commands are already running — read or kill one before starting another", running)
		}
		// Only finished jobs are holding the slots; forget the oldest of them
		// rather than refusing to run.
		b.forgetFinishedLocked()
	}
	b.seq++
	id := "bg_" + strconv.Itoa(b.seq)
	b.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cmd := backend.Command(ctx, commandLine, workDir)

	job := &backgroundJob{id: id, command: commandLine, cancel: cancel, started: time.Now()}
	cmd.Stdout = job
	cmd.Stderr = job

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	b.mu.Lock()
	b.jobs[id] = job
	b.mu.Unlock()
	b.changed()

	go func() {
		err := cmd.Wait()
		cancel()
		job.mu.Lock()
		job.done, job.exitErr, job.ended = true, err, time.Now()
		job.mu.Unlock()
		b.changed()
	}()
	return job, nil
}

// forgetFinishedLocked drops completed jobs whose output has been read. Called
// with b.mu held.
func (b *backgroundShells) forgetFinishedLocked() {
	ids := make([]string, 0, len(b.jobs))
	for id := range b.jobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		job := b.jobs[id]
		if done, _, _, _ := job.status(); !done {
			continue
		}
		job.mu.Lock()
		unread := job.cursor < len(job.buf)
		job.mu.Unlock()
		if !unread {
			delete(b.jobs, id)
		}
	}
}

func (b *backgroundShells) get(id string) (*backgroundJob, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	job, ok := b.jobs[strings.TrimSpace(id)]
	return job, ok
}

// running lists the ids still going, so a lost handle can be recovered from an
// error message instead of leaving the process unreachable forever.
func (b *backgroundShells) running() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ids := make([]string, 0, len(b.jobs))
	for id, job := range b.jobs {
		if done, _, _, _ := job.status(); !done {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// --- shell_output -----------------------------------------------------------

type shellOutputSkill struct{ shells *backgroundShells }

func (*shellOutputSkill) Name() string { return "shell_output" }

func (*shellOutputSkill) Description() string {
	return "อ่านผลลัพธ์ใหม่จากคำสั่งที่รันอยู่เบื้องหลัง"
}

func (*shellOutputSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shell_id": map[string]any{
				"type":        "string",
				"description": "The handle shell returned when it started the command in the background.",
			},
			"filter": map[string]any{
				"type":        "string",
				"description": "Keep only lines matching this regular expression. Use it on a chatty log so the one line you are waiting for is not buried.",
			},
		},
		"required":             []string{"shell_id"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name: "shell_output",
			Description: "Read whatever a background command has printed since you last read it, and whether it is still running. " +
				"Output is consumed: each call returns only what is new, so nothing has to be re-read to find the end. " +
				"A server that is still starting may have printed nothing yet — that is not a failure, call again.",
			Parameters: payload,
		},
	}
}

func (s *shellOutputSkill) Execute(ctx context.Context, input Input) (Output, error) {
	args := stringSlice(input["args"])
	if len(args) == 0 {
		err := errors.New("usage: shell_output <shell_id>")
		return newToolOutput("shell_output", "shell_output", "", time.Now(), false, err), err
	}
	return s.ExecuteTool(ctx, map[string]any{"shell_id": args[0]})
}

func (s *shellOutputSkill) ExecuteTool(_ context.Context, args map[string]any) (Output, error) {
	start := time.Now()
	id, _ := args["shell_id"].(string)
	id = strings.TrimSpace(id)
	command := "shell_output " + id
	if id == "" {
		err := errors.New("shell_id is required")
		return newToolOutput("shell_output", "shell_output", "", start, false, err), err
	}
	job, ok := s.shells.get(id)
	if !ok {
		err := fmt.Errorf("no background command %q — %s", id, describeRunning(s.shells))
		return newToolOutput("shell_output", command, "", start, false, err), err
	}

	fresh, overflowed := job.drain()
	if pattern, _ := args["filter"].(string); strings.TrimSpace(pattern) != "" {
		filtered, err := filterLines(fresh, pattern)
		if err != nil {
			return newToolOutput("shell_output", command, "", start, false, err), err
		}
		fresh = filtered
	}

	done, killed, exitErr, elapsed := job.status()
	header := fmt.Sprintf("[%s] %s — running for %s", id, job.command, elapsed.Round(time.Second))
	switch {
	case killed:
		header = fmt.Sprintf("[%s] %s — killed after %s", id, job.command, elapsed.Round(time.Second))
	case done && exitErr != nil:
		header = fmt.Sprintf("[%s] %s — exited with %v after %s", id, job.command, exitErr, elapsed.Round(time.Second))
	case done:
		header = fmt.Sprintf("[%s] %s — finished in %s", id, job.command, elapsed.Round(time.Second))
	}
	if overflowed {
		header += "\n(earlier output was dropped — this command prints faster than it is being read)"
	}

	body := strings.TrimRight(fresh, "\r\n")
	if body == "" {
		body = "(nothing new since the last read)"
	}
	body, truncated := limitLines(body, defaultToolOutputLineLimit)
	// Not an error even when the command failed: shell_output succeeded at its
	// own job, which is reporting. The header carries the command's fate.
	return newToolOutput("shell_output", command, header+"\n\n"+body, start, truncated, nil), nil
}

// --- shell_kill -------------------------------------------------------------

type shellKillSkill struct{ shells *backgroundShells }

func (*shellKillSkill) Name() string { return "shell_kill" }

func (*shellKillSkill) Description() string {
	return "หยุดคำสั่งที่รันอยู่เบื้องหลัง"
}

func (*shellKillSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shell_id": map[string]any{
				"type":        "string",
				"description": "The handle of the background command to stop.",
			},
		},
		"required":             []string{"shell_id"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name: "shell_kill",
			Description: "Stop a background command and everything it started. " +
				"Kill a dev server when you are done with it rather than leaving it holding a port.",
			Parameters: payload,
		},
	}
}

func (s *shellKillSkill) Execute(ctx context.Context, input Input) (Output, error) {
	args := stringSlice(input["args"])
	if len(args) == 0 {
		err := errors.New("usage: shell_kill <shell_id>")
		return newToolOutput("shell_kill", "shell_kill", "", time.Now(), false, err), err
	}
	return s.ExecuteTool(ctx, map[string]any{"shell_id": args[0]})
}

func (s *shellKillSkill) ExecuteTool(_ context.Context, args map[string]any) (Output, error) {
	start := time.Now()
	id, _ := args["shell_id"].(string)
	id = strings.TrimSpace(id)
	command := "shell_kill " + id
	if id == "" {
		err := errors.New("shell_id is required")
		return newToolOutput("shell_kill", "shell_kill", "", start, false, err), err
	}
	job, ok := s.shells.get(id)
	if !ok {
		err := fmt.Errorf("no background command %q — %s", id, describeRunning(s.shells))
		return newToolOutput("shell_kill", command, "", start, false, err), err
	}
	if done, _, _, elapsed := job.status(); done {
		return newToolOutput("shell_kill", command,
			fmt.Sprintf("[%s] had already finished, after %s", id, elapsed.Round(time.Second)), start, false, nil), nil
	}
	job.kill()
	return newToolOutput("shell_kill", command, "["+id+"] killed: "+job.command, start, false, nil), nil
}

func describeRunning(shells *backgroundShells) string {
	ids := shells.running()
	if len(ids) == 0 {
		return "nothing is running in the background"
	}
	return "currently running: " + strings.Join(ids, ", ")
}
