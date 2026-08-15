package skill

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Mike0165115321/Aetox/internal/model"
)

func TestNewDefaultRegistryRegistersAllBuiltins(t *testing.T) {
	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: t.TempDir()})

	want := []string{
		"help", "echo", "time", "calc", "list", "read", "github_repo_summary",
		"git", "fs", "shell", "shell_output", "shell_kill", "write", "sheet_write", "slides_write", "doc_write", "edit", "grep", "glob", "apply_patch", "notebook_edit", "diagnostics", "symbol", "delete", "plugin_install", "image_ocr", "video_ocr", "pdf_read", "audio_transcribe",
		"web_fetch", "web_search",
		"github_search", "github_read_file", "github_list_files",
		"skills_list", "skill_view",
	}
	for _, name := range want {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("built-in skill %q not registered", name)
		}
		if src, ok := registry.SourceOf(name); !ok || src != SourceBuiltin {
			t.Errorf("SourceOf(%q) = %v, %v, want %v, true", name, src, ok, SourceBuiltin)
		}
	}
	if got := len(registry.Names()); got != len(want) {
		t.Errorf("registry has %d skills, want %d", got, len(want))
	}
}

func TestRegisterDefaultsNilRegistryIsSafe(t *testing.T) {
	RegisterDefaults(nil, RegistryOptions{}) // must not panic
}

// The host's window onto background commands: Registry.BackgroundJobs and
// StopBackgroundJob reach the same job registry the shell tools share, so
// what a panel shows and stops is exactly what the model started.
func TestRegistryBackgroundJobsReachTheShellTools(t *testing.T) {
	isolateAuditLog(t)
	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: t.TempDir()})

	shellTool, _ := registry.Get("shell")
	command := `ping -n 60 127.0.0.1 >nul`
	if runtime.GOOS != "windows" {
		command = `sleep 60`
	}
	out, err := shellTool.(Tool).ExecuteTool(context.Background(), map[string]any{
		"command":           command,
		"run_in_background": true,
	})
	if err != nil {
		t.Fatalf("starting the background command: %v", err)
	}
	id := backgroundIDRe.FindString(out.Content)
	if id == "" {
		t.Fatalf("no handle in %q", out.Content)
	}

	jobs := registry.BackgroundJobs()
	if len(jobs) != 1 || jobs[0].ID != id || jobs[0].Command != command || jobs[0].Done {
		t.Fatalf("BackgroundJobs() = %+v, want %s running %q", jobs, id, command)
	}

	if err := registry.StopBackgroundJob(id); err != nil {
		t.Fatalf("StopBackgroundJob(%s): %v", id, err)
	}
	// Wait for the process to actually die — on Windows t.TempDir cleanup fails
	// while a surviving child still holds the working directory open.
	deadline := time.Now().Add(15 * time.Second)
	for {
		jobs = registry.BackgroundJobs()
		if len(jobs) == 1 && jobs[0].Done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job never died after StopBackgroundJob: %+v", jobs)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !jobs[0].Killed {
		t.Errorf("a stopped job must read as killed: %+v", jobs[0])
	}
}

// Registries without the shell tools — and nil ones — answer emptily rather
// than panicking: the desktop calls these on whatever registry it holds.
func TestRegistryBackgroundJobsWithoutShellTools(t *testing.T) {
	var nilReg *Registry
	if got := nilReg.BackgroundJobs(); got != nil {
		t.Errorf("nil registry: BackgroundJobs() = %v, want nil", got)
	}
	if err := nilReg.StopBackgroundJob("bg_1"); err == nil {
		t.Error("nil registry: stopping must report there is nothing to stop")
	}
	bare := NewRegistry()
	if got := bare.BackgroundJobs(); got != nil {
		t.Errorf("bare registry: BackgroundJobs() = %v, want nil", got)
	}
	if err := bare.StopBackgroundJob("bg_1"); err == nil {
		t.Error("bare registry: stopping must report there is nothing to stop")
	}
}

// A tool description that claims a word — "use this whenever the user wants
// slides" — is routing burned into the tool layer, where it beats any weighing
// the system prompt asks for. That is how "สร้างสไลด์" went straight to a
// .pptx over an explicit "อยากได้เป็นไฟล์ HTML" (2026-08-04): the model's
// thinking quoted the description and never considered anything else.
// Descriptions state capability — "when a real .pptx file is the deliverable"
// — and which shape answers the request stays the model's judgment, taught as
// a principle in internal/prompt.
func TestToolDescriptionsStateCapabilityNotWordTriggers(t *testing.T) {
	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: t.TempDir()})
	for _, name := range registry.Names() {
		s, ok := registry.Get(name)
		if !ok {
			continue
		}
		td, ok := s.(interface{ ToolDefinition() model.ToolDefinition })
		if !ok {
			continue
		}
		desc := strings.ToLower(td.ToolDefinition().Function.Description)
		// "is the deliverable" is the softer second draft of the same trap —
		// "use this when a real .pptx is the deliverable" still tells the model
		// when to pick the tool. The owner's line: capability only, the tool is
		// one option on the shelf.
		for _, trap := range []string{"whenever the user wants", "when the user wants", "whenever the user asks", "when the user asks for", "is the deliverable"} {
			if strings.Contains(desc, trap) {
				t.Errorf("%s claims a word trigger (%q) — describe the capability and leave the routing judgment to the model", name, trap)
			}
		}
	}
}
