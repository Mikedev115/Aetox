package machine

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// memoryStatusEx is MEMORYSTATUSEX, which x/sys/windows does not carry.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

// fill reads the version from the kernel, the memory from kernel32 and the
// processor's name from the registry — the three places Windows keeps them.
func fill(info *Info) {
	if v := windows.RtlGetVersion(); v != nil {
		name := "Windows"
		switch {
		case v.MajorVersion == 10 && v.BuildNumber >= 22000:
			name = "Windows 11"
		case v.MajorVersion == 10:
			name = "Windows 10"
		}
		info.Version = fmt.Sprintf("%s (%d)", name, v.BuildNumber)
	}
	var m memoryStatusEx
	m.Length = uint32(unsafe.Sizeof(m))
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); r != 0 {
		info.MemBytes = m.TotalPhys
	}
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE); err == nil {
		if s, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
			info.CPU = trimSpaces(s)
		}
		k.Close()
	}
}

func trimSpaces(s string) string {
	out := make([]byte, 0, len(s))
	space := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if !space && len(out) > 0 {
				out = append(out, ' ')
			}
			space = true
			continue
		}
		space = false
		out = append(out, s[i])
	}
	for len(out) > 0 && out[len(out)-1] == ' ' {
		out = out[:len(out)-1]
	}
	return string(out)
}
