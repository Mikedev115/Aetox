package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func engineNameForTest() string {
	if runtime.GOOS == "windows" {
		return "aetox-engine.exe"
	}
	return "aetox-engine"
}

func TestDevelopmentEngineUsesSourceAheadOfStaleSibling(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go on PATH")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/Mikedev115/Aetox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "desktop", "build", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(binDir, engineNameForTest())
	if err := os.WriteFile(stale, []byte("stale"), 0o755); err != nil {
		t.Fatal(err)
	}

	bin, dir, prefix, err := engineCommandForExecutable(filepath.Join(binDir, "aetox-dev.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if bin != goBin || dir != root || strings.Join(prefix, " ") != "run ./cmd/aetox-engine" {
		t.Fatalf("dev chose (%q, %q, %q), want current source through go run", bin, dir, prefix)
	}
}

func TestPackagedEngineUsesSiblingBinary(t *testing.T) {
	binDir := t.TempDir()
	want := filepath.Join(binDir, engineNameForTest())
	if err := os.WriteFile(want, []byte("packaged"), 0o755); err != nil {
		t.Fatal(err)
	}

	bin, dir, prefix, err := engineCommandForExecutable(filepath.Join(binDir, "aetox.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if bin != want || dir != "" || prefix != nil {
		t.Fatalf("packaged chose (%q, %q, %q), want sibling %q", bin, dir, prefix, want)
	}
}

// The sentence for an engine that is not beside the app. Packaged, it names
// the likely cause and what to press (§294: Defender quarantined the v1.7.1
// engine and the person read a bare "ไม่พบ" with nothing to do about it);
// in a development tree it stays bare, because nothing was quarantined.
func TestMissingEngineNamesQuarantineOnlyWhenPackaged(t *testing.T) {
	packaged := missingEngine("aetox-engine.exe", `C:\Program Files\Aetox\Aetox`, true).Error()
	wants := []string{"aetox-engine.exe", `C:\Program Files\Aetox\Aetox`, "กัก", "Restore", "Allow on device", "ติดตั้ง Aetox ซ้ำ"}
	if runtime.GOOS == "windows" {
		wants = append(wants, "Windows Security › Protection history")
	}
	for _, want := range wants {
		if !strings.Contains(packaged, want) {
			t.Errorf("packaged sentence lacks %q:\n%s", want, packaged)
		}
	}
	dev := missingEngine("aetox-engine.exe", `D:\Aetox\Aetox\desktop\build\bin`, false).Error()
	if strings.Contains(dev, "กัก") || strings.Contains(dev, "Protection history") {
		t.Errorf("development sentence blames an antivirus:\n%s", dev)
	}
	if !strings.HasPrefix(dev, "ไม่พบ aetox-engine.exe ข้างโปรแกรม") {
		t.Errorf("development sentence changed shape:\n%s", dev)
	}
}
