package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func linesOf(n int) string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = "line"
	}
	return strings.Join(rows, "\n")
}

// A file ending in a newline has as many lines as it looks like it has. Off by
// one here turns the boundary into a lie in whichever direction it errs.
func TestContentLinesDoesNotInventATrailingLine(t *testing.T) {
	cases := map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\nb\n": 2}
	for input, want := range cases {
		if got := contentLines(input); got != want {
			t.Errorf("contentLines(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestContentLineCapBoundary(t *testing.T) {
	if note := contentLineCapNote("content", linesOf(contentLineCap)); note != "" {
		t.Fatalf("exactly the cap must pass silently, got %q", note)
	}
	note := contentLineCapNote("content", linesOf(contentLineCap+1))
	if note == "" {
		t.Fatal("one line over the cap must be remarked on")
	}
	for _, want := range []string{"301", "300", "written whole", "mode=append"} {
		if !strings.Contains(note, want) {
			t.Errorf("note is missing %q: %s", want, note)
		}
	}
}

// Over the cap is written, whole, and said so. Content that reached the tool
// parsed as JSON, so the cut-off the cap exists to prevent did not happen;
// refusing a finished file bought the same file back in three rounds (§221).
func TestWriteSkillOverCapWritesWholeAndSaysSo(t *testing.T) {
	root := t.TempDir()
	s := &writeSkill{root: root}

	content := linesOf(contentLineCap + 50)
	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"path":    "big.html",
		"content": content,
	})
	if err != nil {
		t.Fatalf("a whole call over the cap must still be written: %v", err)
	}
	if !out.Success {
		t.Error("Success = false on a write that landed")
	}
	data, readErr := os.ReadFile(filepath.Join(root, "big.html"))
	if readErr != nil {
		t.Fatalf("file not written: %v", readErr)
	}
	if string(data) != content {
		t.Fatalf("file on disk is not the content sent (%d bytes vs %d)", len(data), len(content))
	}
	for _, want := range []string{"write done", "350", "mode=append"} {
		if !strings.Contains(out.Content, want) {
			t.Errorf("result is missing %q:\n%s", want, out.Content)
		}
	}
}

// Within the cap there is nothing to say: the note is for the gamble, and a
// call that took none must not read as if it had.
func TestWriteSkillAtCapCarriesNoNote(t *testing.T) {
	root := t.TempDir()
	s := &writeSkill{root: root}

	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"path":    "ok.txt",
		"content": linesOf(contentLineCap),
	})
	if err != nil {
		t.Fatalf("a file exactly at the cap must be written: %v", err)
	}
	if strings.Contains(out.Content, "Note:") {
		t.Errorf("a write within the cap must carry no note:\n%s", out.Content)
	}
}

func TestWriteSkillAtCapStillWrites(t *testing.T) {
	root := t.TempDir()
	s := &writeSkill{root: root}

	if _, err := s.ExecuteTool(context.Background(), map[string]any{
		"path":    "ok.txt",
		"content": linesOf(contentLineCap),
	}); err != nil {
		t.Fatalf("a file exactly at the cap must be written: %v", err)
	}
}

// The other door. A cap that only watched write would be routed around by
// moving the same content into one enormous append, so append carries the
// same note, and lands the same way.
func TestEditAppendOverCapLandsWithNote(t *testing.T) {
	root := t.TempDir()
	path := writeEditFixture(t, root, "cont.txt", "start\n")
	s := &editSkill{root: root}

	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"path":    "cont.txt",
		"replace": linesOf(contentLineCap + 1),
		"mode":    "append",
	})
	if err != nil {
		t.Fatalf("a whole append over the cap must still land: %v", err)
	}
	data, _ := os.ReadFile(path)
	if contentLines(string(data)) != contentLineCap+2 {
		t.Fatalf("append did not land whole, got %d lines", contentLines(string(data)))
	}
	for _, want := range []string{"edit done", "301", "mode=append"} {
		if !strings.Contains(out.Content, want) {
			t.Errorf("result is missing %q:\n%s", want, out.Content)
		}
	}
}

// Replace is deliberately uncapped. One substitution cannot be split in half,
// so a cap there would refuse correct work and offer nothing in its place.
func TestEditReplaceIsNotCapped(t *testing.T) {
	root := t.TempDir()
	path := writeEditFixture(t, root, "big.go", "OLD_BLOCK\n")
	s := &editSkill{root: root}

	if _, err := s.ExecuteTool(context.Background(), map[string]any{
		"path":    "big.go",
		"find":    "OLD_BLOCK",
		"replace": linesOf(contentLineCap + 100),
	}); err != nil {
		t.Fatalf("a large replace must still be allowed: %v", err)
	}
	data, _ := os.ReadFile(path)
	if contentLines(string(data)) != contentLineCap+100 {
		t.Errorf("replacement did not land whole, got %d lines", contentLines(string(data)))
	}
}
