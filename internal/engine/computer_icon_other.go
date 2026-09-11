//go:build !windows

package engine

// No icons off Windows, for the same reason there is no reach off Windows.
// Empty is the answer the row already knows how to draw.

func exeIconDataURL(string) string { return "" }

func (a *Engine) ProgramIcon(string) string { return "" }
