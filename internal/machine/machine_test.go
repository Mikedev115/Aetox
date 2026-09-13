package machine

import (
	"runtime"
	"strings"
	"testing"
)

func TestCollectSaysWhatThisMachineIs(t *testing.T) {
	i := Collect()
	if i.OS != runtime.GOOS || i.Arch != runtime.GOARCH || i.CPUs < 1 || i.Hostname == "" {
		t.Errorf("Collect = %+v", i)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "linux" {
		if i.Version == "" || i.MemBytes == 0 {
			t.Errorf("the system's own name or memory was not read: %+v", i)
		}
	}
	line := i.Line()
	if !strings.Contains(line, "CPU") || !strings.Contains(line, i.Arch) {
		t.Errorf("Line = %q", line)
	}
	if Collect() != i {
		t.Error("a second Collect answered differently")
	}
}

func TestLineLeavesOutWhatIsUnknown(t *testing.T) {
	if got := (Info{OS: "linux", Arch: "arm64", CPUs: 4}).Line(); got != "linux · 4 CPU · arm64" {
		t.Errorf("Line = %q", got)
	}
	if got := (Info{Version: "Ubuntu 26.04 LTS", Arch: "amd64", CPUs: 8, MemBytes: 16 << 30}).Line(); got != "Ubuntu 26.04 LTS · 8 CPU · 16 GB · amd64" {
		t.Errorf("Line = %q", got)
	}
}
