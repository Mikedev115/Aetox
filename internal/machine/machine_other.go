//go:build !windows && !linux

package machine

// fill has nothing beyond what Go itself knows on this system.
func fill(*Info) {}
