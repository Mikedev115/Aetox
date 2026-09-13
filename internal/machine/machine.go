// Package machine describes the computer a process runs on — the name, the
// system, the processor, the memory — for a Settings page that lists more
// than one of them (§248 phase 3: this machine, and every host the engine
// can run on). Facts only, read once; nothing here is used to decide
// anything.
package machine

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

// Info is one machine, as its own process sees it.
type Info struct {
	Hostname string `json:"hostname"`
	// OS and Arch are Go's names: windows/linux/darwin, amd64/arm64.
	OS   string `json:"os"`
	Arch string `json:"arch"`
	// Version is the system's own name for itself — "Windows 11 (26200)",
	// "Ubuntu 26.04 LTS" — empty when it could not be read.
	Version string `json:"version"`
	// CPU is the processor's model name when the system says one; CPUs is
	// the count of logical processors Go sees.
	CPU  string `json:"cpu"`
	CPUs int    `json:"cpus"`
	// MemBytes is the physical memory, 0 when unknown.
	MemBytes uint64 `json:"memBytes"`
}

// Line is the one-line spec for a row: "Ubuntu 26.04 LTS · 8 CPU · 16 GB · amd64".
func (i Info) Line() string {
	parts := []string{}
	if i.Version != "" {
		parts = append(parts, i.Version)
	} else if i.OS != "" {
		parts = append(parts, i.OS)
	}
	if i.CPUs > 0 {
		parts = append(parts, fmt.Sprintf("%d CPU", i.CPUs))
	}
	if i.MemBytes > 0 {
		parts = append(parts, fmt.Sprintf("%.0f GB", float64(i.MemBytes)/(1<<30)))
	}
	if i.Arch != "" {
		parts = append(parts, i.Arch)
	}
	return strings.Join(parts, " · ")
}

var (
	once   sync.Once
	cached Info
)

// Collect reads the machine once and answers the same thing after.
func Collect() Info {
	once.Do(func() {
		cached = collect()
	})
	return cached
}

func collect() Info {
	info := Info{OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU()}
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	fill(&info)
	return info
}
