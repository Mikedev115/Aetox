package machine

import (
	"os"
	"strconv"
	"strings"
)

// fill reads what Linux publishes as files: the distribution's own name,
// the processor's, and the memory line.
func fill(info *Info) {
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
				info.Version = strings.Trim(strings.TrimSpace(v), `"`)
				break
			}
		}
	}
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			if k == "model name" || k == "Model" {
				info.CPU = strings.TrimSpace(v)
				break
			}
		}
	}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(line, "MemTotal:"); ok {
				fields := strings.Fields(v)
				if len(fields) > 0 {
					if kb, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
						info.MemBytes = kb * 1024
					}
				}
				break
			}
		}
	}
}
