//go:build !windows

package main

// No overlay off Windows, because there is no reach off Windows to warn about
// (computer_reach_other.go). Silent no-ops rather than refusals: unlike the
// tool's own verbs, nothing here is something a model asked for, so there is
// nobody to tell.

type screenOverlay struct{}

func (o *screenOverlay) show() {}
func (o *screenOverlay) hide() {}

var overlay = &screenOverlay{}
