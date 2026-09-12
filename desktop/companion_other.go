//go:build !windows

package main

import "errors"

// The desktop body is a Win32 layered window (companion_windows.go); the
// other platforms keep the companion inside the app window, and the switch
// that would send it out is answered with "could not" so the window keeps
// its own copy (Companion.svelte falls back on a false).
func openCompanionBody(x, y, size int, sprites func(scale float64) spriteSource, on func(kind string, data map[string]any)) (companionBody, error) {
	return nil, errors.New("companion: no desktop body on this platform")
}
