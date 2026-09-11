//go:build windows

package main

// The icon path, checked against real executables on this machine.
//
// Everything here is decoration on a row whose text already says what matters,
// so the assertions are about the two ways decoration goes wrong: a picture that
// is not a picture, and a failure that costs something.

import (
	"encoding/base64"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAProgramsOwnIconComesOutAsAPNG(t *testing.T) {
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "charmap.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("no charmap on this machine")
	}
	url := exeIconDataURL(exe)
	if url == "" {
		t.Skip("this machine would not give up charmap's icon")
	}
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(url, prefix) {
		t.Fatalf("the icon is not a data URL: %.40q", url)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, prefix))
	if err != nil {
		t.Fatalf("the icon is not base64: %v", err)
	}
	img, err := png.Decode(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("the icon is not a readable PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != iconPixels || b.Dy() != iconPixels {
		t.Errorf("the icon is %dx%d, want %d square", b.Dx(), b.Dy(), iconPixels)
	}
}

func TestAProgramWithNoIconIsAnEmptyStringNotAnError(t *testing.T) {
	// A path that is not a program at all. The row draws a placeholder; nothing
	// anywhere is told a picture failed, because a program with no icon is a
	// normal thing rather than a fault.
	if got := exeIconDataURL(filepath.Join(t.TempDir(), "nothing-here.exe")); got != "" {
		t.Errorf("a missing program produced %.40q", got)
	}
	if got := exeIconDataURL(""); got != "" {
		t.Errorf("an empty path produced %.40q", got)
	}
}

func TestTheSecondAskIsFree(t *testing.T) {
	// The settings page asks for every open window at once and again on every
	// refresh. Extracting one is a file read, an icon draw and a PNG encode, so
	// without the cache a refresh is a dozen of those.
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "charmap.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("no charmap on this machine")
	}
	first := exeIconDataURL(exe)
	iconMu.Lock()
	_, cached := iconCache[strings.ToLower(exe)]
	iconMu.Unlock()
	if !cached {
		t.Fatal("the icon was not remembered, so every refresh re-extracts every program")
	}
	if second := exeIconDataURL(exe); second != first {
		t.Error("the same program gave two different icons")
	}
}

func TestWindowsVariablesAreResolvedTheWayTheyAreWritten(t *testing.T) {
	// The path table is written in %NAME% because that is how a reader checks it
	// against their own machine; os.ExpandEnv only knows $NAME.
	t.Setenv("AETOX_ICON_TEST", `C:\somewhere`)
	if got := expandWindowsVars(`%AETOX_ICON_TEST%\app.exe`); got != `C:\somewhere\app.exe` {
		t.Errorf("expandWindowsVars gave %q", got)
	}
	// A variable this machine does not set is not a path and must not become a
	// half-resolved one that then gets stat'd.
	if got := expandWindowsVars(`%AETOX_NOT_SET_ANYWHERE%\app.exe`); got != "" {
		t.Errorf("an unset variable resolved to %q", got)
	}
}
