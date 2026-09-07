package subagent

// The third shelf: knowledge that arrived as a pinned download.
//
// It exists because hyperframes ships its authoring knowledge as 20 Agent
// Skills and its npm package carries two of them — the rest are fetched over
// the network at first use, which is a road Aetox does not have open. They
// travel as a pinned zip instead, and this file is what says the video agent
// can actually read one after it lands.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// writeInstalledSkill drops one SKILL.md where a pinned download would put it.
func writeInstalledSkill(t *testing.T, agent, name, description, body string) string {
	t.Helper()
	root, err := config.AgentInstalledSkillsPath(agent)
	if err != nil {
		t.Fatalf("installed skills path for %s: %v", agent, err)
	}
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	doc := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}

// The claim: a downloaded shelf reaches the worker it was installed for, is
// invisible to every other worker, and never reaches the main assistant — the
// same three sentences the worker's own folder has to satisfy.
func TestADownloadedSkillReachesItsOwnWorkerAndNoOther(t *testing.T) {
	isolate(t)
	parent := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()})
	writeInstalledSkill(t, "video", "hyperframes-core", "composition structure and timing attributes", "data-duration")

	if _, ok := FilterRegistry(parent, agent("video"), nil).Get("hyperframes-core"); !ok {
		t.Error("video cannot reach a skill installed for it")
	}
	if _, ok := FilterRegistry(parent, agent("doc"), nil).Get("hyperframes-core"); ok {
		t.Error("a shelf installed for video reached doc")
	}
	if _, ok := parent.Get("hyperframes-core"); ok {
		t.Error("an installed shelf leaked into the main assistant's registry")
	}
}

// The order the three addresses resolve in, asserted where it matters: a
// download must not be able to answer to a name the user already used.
//
// The user's folder wins because editing a shipped worker means copying it out
// and the copy has to be the one that runs. A download loses to both because it
// is the only one of the three nobody in this office wrote.
func TestWhatTheUserWroteBeatsWhatWasDownloaded(t *testing.T) {
	isolate(t)
	writeSkill(t, "video", "hyperframes-cli", "mine", "the user's own account")
	writeInstalledSkill(t, "video", "hyperframes-cli", "theirs", "upstream's account")

	own, errs := OwnSkills("video")
	for _, err := range errs {
		t.Errorf("OwnSkills: %v", err)
	}
	seen := 0
	for _, s := range own {
		if s.Name != "hyperframes-cli" {
			continue
		}
		seen++
		if s.Description != "mine" {
			t.Errorf("description is %q, want the user's own", s.Description)
		}
	}
	if seen != 1 {
		t.Errorf("hyperframes-cli appears %d times, want 1 — a shadowed skill must not also ship beside the one that shadowed it", seen)
	}
}

// A skill nobody installed is not an error. It is the normal state for every
// worker but the one that makes video, and a scan that reported it would put a
// line in the debug log on every dispatch for a folder that was never meant to
// be there.
func TestNoDownloadedShelfIsNotAFailure(t *testing.T) {
	isolate(t)
	if _, errs := OwnSkills("video"); len(errs) != 0 {
		t.Errorf("OwnSkills complained about an absent download: %v", errs)
	}
}

// The level-3 read, which is the whole point of carrying these files rather
// than a summary of them. Every hyperframes SKILL.md is a router that names a
// `references/` file per decision; a shelf whose bodies open but whose
// references do not is a shelf that answers every question with a filename.
func TestADownloadedSkillsReferencesAreReadable(t *testing.T) {
	isolate(t)
	dir := writeInstalledSkill(t, "video", "hyperframes", "the router", "read references/routes/slideshow.md")
	routes := filepath.Join(dir, "references", "routes")
	if err := os.MkdirAll(routes, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", routes, err)
	}
	if err := os.WriteFile(filepath.Join(routes, "slideshow.md"), []byte("author a deck"), 0o644); err != nil {
		t.Fatalf("write route: %v", err)
	}

	data, err := ReadSkillFile("video", "hyperframes", "references/routes/slideshow.md")
	if err != nil {
		t.Fatalf("ReadSkillFile: %v", err)
	}
	if string(data) != "author a deck" {
		t.Errorf("read %q", data)
	}

	names, err := ListSkillDir("video", "hyperframes", "references/routes")
	if err != nil {
		t.Fatalf("ListSkillDir: %v", err)
	}
	if len(names) != 1 || names[0] != "slideshow.md" {
		t.Errorf("listed %v, want [slideshow.md]", names)
	}
}

// The copy, for the same reason it exists for the scene library: these skills
// ship png, woff2 and mp3 beside their HTML, and a model asked to reproduce
// those by typing them out is a model asked to invent them.
func TestADownloadedSkillsFolderCanBeCopiedOut(t *testing.T) {
	isolate(t)
	dir := writeInstalledSkill(t, "video", "music-to-video", "beat-synced video", "copy a template")
	tpl := filepath.Join(dir, "references", "templates", "card-flyby")
	if err := os.MkdirAll(tpl, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", tpl, err)
	}
	for name, body := range map[string]string{"index.html": "<main></main>", "program.json": "{}"} {
		if err := os.WriteFile(filepath.Join(tpl, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	dest := filepath.Join(t.TempDir(), "project")
	n, err := CopySkillDir("video", "music-to-video", "references/templates/card-flyby", dest)
	if err != nil {
		t.Fatalf("CopySkillDir: %v", err)
	}
	if n != 2 {
		t.Errorf("copied %d files, want 2", n)
	}
	if _, err := os.Stat(filepath.Join(dest, "program.json")); err != nil {
		t.Errorf("the folder arrived without the file beside its markup: %v", err)
	}
}

// A download lands in its own directory rather than in the worker's home, and
// this is the reason spelled as a test: internal/capability removes a
// component's Dest before every unpack, so a shelf pointed at the home would
// delete the skills the user wrote there the first time the pin moved.
func TestTheDownloadedShelfIsNotInsideTheWorkersHome(t *testing.T) {
	isolate(t)
	home, err := config.AgentSkillsPath("video")
	if err != nil {
		t.Fatalf("skills path: %v", err)
	}
	installed, err := config.AgentInstalledSkillsPath("video")
	if err != nil {
		t.Fatalf("installed skills path: %v", err)
	}
	rel, err := filepath.Rel(home, installed)
	if err == nil && rel != ".." && !filepath.IsAbs(rel) && rel[0] != '.' {
		t.Fatalf("%s is inside %s — a component's Dest is wiped on every install, and that folder is the user's", installed, home)
	}
	if installed == home {
		t.Fatal("the downloaded shelf and the user's own are the same folder")
	}
}
